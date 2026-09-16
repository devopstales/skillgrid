// Rename plans and (opt-in) applies a symbol rename on the existing graph.
// The plan splits into graph edits (high-confidence: the definition plus the
// typed references from the edges table — resolved endpoints only) and
// text-search edits (lower-confidence: every stored occurrence of the old
// name outside the graph bucket, flagged "review carefully"). dry_run
// defaults true: without Apply the plan is returned and nothing is written.
package affected

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
)

// RenameOptions tunes a rename.
type RenameOptions struct {
	Old, New string
	// File/UID/Kind narrow the resolution (005 disambiguation).
	File string
	UID  string
	Kind string
	// DryRun returns the plan without writing (default true; Apply forces
	// the non-dry path).
	DryRun bool
	// Apply writes the planned edits (only files listed in the plan).
	Apply bool
}

// Candidate is one ranked rename candidate (ambiguous target).
type Candidate struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
	Path string `json:"path"`
}

// Edit is one planned rename edit: the file, the old/new text, the source
// (graph | text-search) and a confidence label. Every edit carries one.
type Edit struct {
	Path       string `json:"path"`
	Old        string `json:"old"`
	New        string `json:"new"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	// Note flags the text-search bucket "review carefully".
	Note string `json:"note,omitempty"`
}

// RenamePlan is the code_rename answer. Ambiguous targets carry a ranked
// candidate list (never a silent pick) and no edits.
type RenamePlan struct {
	Old    string `json:"old"`
	New    string `json:"new"`
	// Target is the resolved symbol (empty when ambiguous/not-found).
	Target       map[string]any `json:"target,omitempty"`
	Ambiguous    bool           `json:"ambiguous,omitempty"`
	NotFound     bool           `json:"not_found,omitempty"`
	Candidates   []Candidate  `json:"candidates,omitempty"`
	GraphEdits   []Edit         `json:"graph_edits"`
	TextSearchEdits []Edit      `json:"text_search_edits"`
	FilesAffected int            `json:"files_affected"`
	TotalEdits   int            `json:"total_edits"`
	DryRun       bool           `json:"dry_run"`
	Applied      bool           `json:"applied,omitempty"`
	// EditedFiles lists the files actually written (non-dry runs only).
	EditedFiles []string `json:"edited_files,omitempty"`
}

// renameRefKinds are the edge kinds that count as a typed reference of the
// target (the graph bucket). tests_for/tested_by keep their own edges, so
// their reference sites are graph edits too.
var renameRefKinds = []string{`"calls"`, `"imports"`, `"reference"`, `"route"`, `"extends"`, `"implements"`, `"references"`, `"tests_for"`, `"tested_by"`}

// Rename resolves Old via 005 disambiguation (ambiguous → ranked candidates,
// no silent pick) and builds the edit plan. With Apply=true it writes exactly
// the planned files (no commit/push); otherwise it returns the plan with
// dry_run=true and touches nothing.
func Rename(ctx context.Context, db *sql.DB, opts RenameOptions) (RenamePlan, error) {
	plan := RenamePlan{Old: opts.Old, New: opts.New, GraphEdits: []Edit{}, TextSearchEdits: []Edit{}}
	if opts.Old == "" || opts.New == "" || opts.Old == opts.New {
		return plan, &RenameError{Msg: "rename: provide distinct old and new names"}
	}
	if opts.Apply {
		plan.DryRun = false
	} else {
		plan.DryRun = true
	}

	res, err := resolveSymbol(ctx, db, opts.Old, resolveFilter{File: opts.File, UID: opts.UID, Kind: opts.Kind})
	if err != nil {
		return plan, err
	}
	if res.NotFound {
		plan.NotFound = true
		return plan, nil
	}
	if res.Ambiguous {
		plan.Ambiguous = true
		for _, c := range rankCandidates(ctx, db, res.Matches) {
			plan.Candidates = append(plan.Candidates, Candidate{Name: c.Name, UID: c.UID, Path: c.Path})
		}
		return plan, nil
	}
	target := res.Target
	plan.Target = symbolDTO(target)

	// Graph bucket: the definition site + every typed reference (resolved
	// edge endpoints only — dropped references have no edge, so they are not
	// graph edits; they surface as text-search edits at most).
	graphPaths := map[string]bool{target.Path: true}
	kindList := strings.Join(renameRefKinds, ",")
	rows, err := db.QueryContext(ctx, `
		SELECT f.path, s.name
		FROM edges e
		JOIN symbols s ON s.id = e.from_id
		JOIN files f ON f.id = s.file_id
		WHERE e.to_id = ? AND e.kind IN (`+kindList+`)
		ORDER BY e.id`, target.ID)
	if err != nil {
		return plan, err
	}
	type ref struct {
		path, name string
	}
	var refs []ref
	for rows.Next() {
		var r ref
		if err := rows.Scan(&r.path, &r.name); err != nil {
			rows.Close()
			return plan, err
		}
		refs = append(refs, r)
		graphPaths[r.path] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return plan, err
	}
	rows.Close()

	// The definition edit: the symbol name at its definition site.
	plan.GraphEdits = append(plan.GraphEdits, Edit{
		Path: target.Path, Old: opts.Old, New: opts.New,
		Source: "definition", Confidence: "high",
	})
	// Typed reference edits: one per stored reference occurrence (name-based
	// count over the stored symbols/edges occurrences of the old name).
	for _, r := range refs {
		count := countNameInFile(db, r.path, opts.Old)
		plan.GraphEdits = append(plan.GraphEdits, Edit{
			Path: r.path, Old: opts.Old, New: opts.New,
			Source: "typed-reference", Confidence: "high",
			Note: fmt.Sprintf("occurrences: %d", count),
		})
	}

	// Text-search bucket: every other stored file that has a symbol whose
	// name matches the old name (exact or substring) outside the graph
	// bucket. Lower-confidence: string matches can be coincidental (comments,
	// other languages, substrings) — flagged "review carefully".
	rows2, err := db.QueryContext(ctx, `
		SELECT DISTINCT f.path
		FROM symbols s3
		JOIN files f ON f.id = s3.file_id
		WHERE s3.name = ?
		ORDER BY f.path`, opts.Old)
	if err != nil {
		return plan, err
	}
	for rows2.Next() {
		var p string
		if err := rows2.Scan(&p); err != nil {
			rows2.Close()
			return plan, err
		}
		if graphPaths[p] {
			continue
		}
		plan.TextSearchEdits = append(plan.TextSearchEdits, Edit{
			Path: p, Old: opts.Old, New: opts.New,
			Source: "text-search", Confidence: "low",
			Note: "review carefully (string match)",
		})
	}
	if err := rows2.Err(); err != nil {
		rows2.Close()
		return plan, err
	}
	rows2.Close()

	// De-dup against the graph bucket (the definition + typed refs already
	// cover those files at higher confidence).
	var ts []Edit
	for _, e := range plan.TextSearchEdits {
		if graphPaths[e.Path] {
			continue
		}
		ts = append(ts, e)
	}
	plan.TextSearchEdits = ts

	files := map[string]bool{}
	for _, e := range plan.GraphEdits {
		files[e.Path] = true
	}
	for _, e := range plan.TextSearchEdits {
		files[e.Path] = true
	}
	plan.FilesAffected = len(files)
	plan.TotalEdits = len(plan.GraphEdits) + len(plan.TextSearchEdits)
	sort.SliceStable(plan.GraphEdits, func(i, j int) bool { return plan.GraphEdits[i].Path < plan.GraphEdits[j].Path })
	sort.SliceStable(plan.TextSearchEdits, func(i, j int) bool { return plan.TextSearchEdits[i].Path < plan.TextSearchEdits[j].Path })

	if !plan.DryRun {
		// Apply: write exactly the planned files (string replace of the old
		// name with the new one). No commit, no push. A planned file that is
		// absent on disk (indexed but deleted) is skipped, not an error —
		// the apply path only writes files that exist.
		var edited []string
		for p := range files {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			if err := applyEdit(ctx, db, p, opts.Old, opts.New); err != nil {
				return plan, err
			}
			edited = append(edited, p)
		}
		sort.Strings(edited)
		plan.EditedFiles = edited
		plan.Applied = true
	}
	return plan, nil
}

// RenameError is a clear validation error (distinct old/new names).
type RenameError struct{ Msg string }

func (e *RenameError) Error() string { return e.Msg }

// applyEdit replaces every occurrence of old with new in the file's indexed
// content (chunks), writing back the file at its indexed path. It is the
// non-dry rename write path: only the planned file is touched.
func applyEdit(ctx context.Context, db *sql.DB, path, old, new string) error {
	var content []byte
	if err := readFileInto(db, path, &content); err != nil {
		return err
	}
	updated := strings.ReplaceAll(string(content), old, new)
	if updated == string(content) {
		return nil
	}
	return writeFileFromDB(db, path, []byte(updated))
}

// countNameInFile counts stored occurrences of name in a file's indexed
// symbols/chunks (best-effort; 0 when the file is not indexed).
func countNameInFile(db *sql.DB, path, name string) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.path = ? AND s.name = ?`, path, name).Scan(&n)
	return n
}

