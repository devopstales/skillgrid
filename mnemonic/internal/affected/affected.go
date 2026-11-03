// Package affected answers "which tests do I run after this change?": a
// read-only traversal of the 005 symbols/edges graph from a set of changed
// files out to the test files that depend on them (import + tests_for edges),
// depth-capped. It adds no nodes or edges and traverses ONLY stored (resolved)
// edges: step-01's drop-not-guess policy means a dropped or ambiguous
// reference has no stored edge, so it can never inflate the reported radius.
package affected

import (
	"bufio"
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultDepth caps the blast-radius traversal when Options.Depth is unset.
const DefaultDepth = 5

// Relationship is one stored edge hop in the blast radius: the changed (or
// transitively reached) symbol, the dependent it points to, and the edge kind
// + confidence that connects them. It is a path/relationship from the existing
// graph — never an invented edge.
type Relationship struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
	Depth      int    `json:"depth"`
	// Via is the UID of the dependent symbol (deterministic identity; the
	// dependent's path is the symbol's file path).
	Via string `json:"via,omitempty"`
	// DepPath is the dependent symbol's file path.
	DepPath string `json:"dep_path"`
}

// Result is the code_affected answer: the affected test files (deduped,
// sorted), every relationship hop that produced them, and a clear message for
// the empty-changed-set case (not an error).
type Result struct {
	Changed       []string       `json:"changed"`
	TestFiles     []string       `json:"test_files"`
	Relationships []Relationship `json:"relationships"`
	Depth         int            `json:"depth"`
	Filter        string         `json:"filter,omitempty"`
	// Message is set when there are no changed files (empty result, not an
	// error) or no affected tests were found.
	Message string `json:"message,omitempty"`
}

// Options tunes a code_affected traversal.
type Options struct {
	// Changed is the set of changed file paths (repo-relative, as files.path).
	Changed []string
	// Depth caps the traversal (<= 0 uses DefaultDepth).
	Depth int
	// Filter restricts reported test files to a substring/segment match.
	Filter string
}

// isTestPath reports whether a file path is a test file (any language).
func isTestPath(path string) bool {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.Contains(base, "test_") && strings.HasSuffix(base, ".py"):
		return true
	case strings.HasSuffix(base, "_test.rb"):
		return true
	case strings.HasSuffix(base, ".test.js"), strings.HasSuffix(base, ".test.ts"),
		strings.HasSuffix(base, ".test.tsx"), strings.HasSuffix(base, ".test.jsx"):
		return true
	case strings.Contains(base, "test.") && (strings.HasSuffix(base, ".js") || strings.HasSuffix(base, ".ts")):
		return true
	}
	return false
}

// matchFilter reports whether a test path passes the filter (substring or
// path-segment match; empty filter passes everything).
func matchFilter(path, filter string) bool {
	if filter == "" {
		return true
	}
	if strings.Contains(path, filter) {
		return true
	}
	segs := strings.Split(path, "/")
	for _, s := range segs {
		if s == filter {
			return true
		}
	}
	return false
}

// traverseKinds are the resolved edge kinds the blast-radius math walks.
// calls/imports are 005's dependency kinds; reference/route/extends/implements
// are 005 heritage/call kinds; references/navigates are step-01's
// framework-route kinds (resolved only — dropped refs have no stored edge).
// tests_for/tested_by connect code to its tests.
var traverseKinds = []string{`"calls"`, `"imports"`, `"reference"`, `"route"`, `"extends"`, `"implements"`, `"references"`, `"navigates"`, `"tests_for"`, `"tested_by"`}

// Affected runs the depth-capped traversal from the changed files and returns
// the affected test files plus every relationship hop. Query only: it reads
// the symbols/edges tables and invents nothing. An empty changed set returns
// an empty result with a message (not an error).
func Affected(ctx context.Context, db *sql.DB, opts Options) (Result, error) {
	depth := opts.Depth
	if depth <= 0 {
		depth = DefaultDepth
	}
	res := Result{Depth: depth, Filter: opts.Filter,
		TestFiles: []string{}, Relationships: []Relationship{}}
	normalized := make([]string, len(opts.Changed))
	for i, p := range opts.Changed {
		normalized[i] = normPath(p)
	}
	changed := dedupeSorted(normalized)
	res.Changed = changed
	if len(changed) == 0 {
		res.Message = "no changed files: nothing to analyze (pass a file list, --stdin, or --base)"
		return res, nil
	}

	// Seed: every symbol defined in a changed file.
	seedKinds := strings.Join(traverseKinds, ",")
	seen := map[int64]bool{}
	var queue []struct {
		id    int64
		depth int
	}
	args := make([]any, 0, len(changed))
	placeholders := make([]string, 0, len(changed))
	for _, p := range changed {
		placeholders = append(placeholders, "?")
		args = append(args, p)
	}
	rows, err := db.QueryContext(ctx, `
		SELECT s.id FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.path IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY s.id`, args...)
	if err != nil {
		return res, err
	}
	var seeded int
	for rows.Next() {
		seeded++
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return res, err
		}
		seen[id] = true
		queue = append(queue, struct {
			id    int64
			depth int
		}{id: id, depth: 0})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return res, err
	}
	rows.Close()
	if seeded == 0 && len(changed) > 0 {
		res.Message = "changed files are not indexed (run code_index first): " + strings.Join(changed, ", ")
		return res, nil
	}

	testSeen := map[string]bool{}
	pathCache := map[int64]string{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= depth {
			continue
		}
		// Dependents of cur.id: edges that point AT it, either by resolved
		// id (e.to_id = cur.id — 005 calls/imports + step-01 resolved
		// references) or by a name-only endpoint (e.to_id IS NULL AND
		// e.to_name matches — 005 heritage / navigates / unresolved-by-name).
		// Only stored edges are traversed: a dropped reference has none.
		hops, err := dependentsOf(ctx, db, cur.id, seedKinds)
		if err != nil {
			return res, err
		}
		for _, h := range hops {
			sym, err := symbolByID(ctx, db, h.fromID)
			if err != nil {
				continue
			}
			rel := Relationship{
				From: curPathCache(ctx, db, cur.id, pathCache),
				To:   sym.Name,
				Kind: h.kind, Confidence: h.conf,
				Depth: cur.depth + 1, Via: sym.UID, DepPath: sym.Path,
			}
			res.Relationships = append(res.Relationships, rel)
			if isTestPath(sym.Path) && matchFilter(sym.Path, opts.Filter) && !testSeen[sym.Path] {
				testSeen[sym.Path] = true
				res.TestFiles = append(res.TestFiles, sym.Path)
			}
			if !seen[h.fromID] {
				seen[h.fromID] = true
				queue = append(queue, struct {
					id    int64
					depth int
				}{id: h.fromID, depth: cur.depth + 1})
			}
		}
	}
	sort.Strings(res.TestFiles)
	sort.SliceStable(res.Relationships, func(i, j int) bool {
		if res.Relationships[i].Depth != res.Relationships[j].Depth {
			return res.Relationships[i].Depth < res.Relationships[j].Depth
		}
		if res.Relationships[i].From != res.Relationships[j].From {
			return res.Relationships[i].From < res.Relationships[j].From
		}
		return res.Relationships[i].Via < res.Relationships[j].Via
	})
	if len(res.TestFiles) == 0 {
		res.Message = "no affected test files found for the changed set"
	}
	return res, nil
}

// AffectedFromStdin reads a git-diff --name-only-style file list (one path per
// line) from r and runs the same traversal. An empty list is an empty result
// with a clear message (not an error).
func AffectedFromStdin(ctx context.Context, db *sql.DB, stdin string, opts Options) (Result, error) {
	changed, err := ParseFileList(stdin)
	if err != nil {
		return Result{}, err
	}
	opts.Changed = changed
	return Affected(ctx, db, opts)
}

// ParseFileList parses a newline-separated file list (git diff --name-only
// style), trimming blank lines and CR. Never errors on empty input.
func ParseFileList(s string) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type hop struct {
	fromID int64
	kind   string
	conf   string
}

// dependentsOf returns the stored edges pointing at symID (resolved id or
// name-only endpoint), of the traversable kinds.
func dependentsOf(ctx context.Context, db *sql.DB, symID int64, kinds string) ([]hop, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT e.from_id, e.kind, e.confidence
		FROM edges e
		WHERE (e.to_id = ? OR (e.to_id IS NULL AND e.to_name IN (SELECT name FROM symbols WHERE id = ?)))
		  AND e.kind IN (`+kinds+`)
		ORDER BY e.id`, symID, symID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []hop
	for rows.Next() {
		var h hop
		if err := rows.Scan(&h.fromID, &h.kind, &h.conf); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// symbolByID loads a symbol node for the traversal.
func symbolByID(ctx context.Context, db *sql.DB, id int64) (sym, error) {
	var s sym
	err := db.QueryRowContext(ctx, `
		SELECT s.id, s.name, s.kind, s.uid, f.path
		FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE s.id = ?`, id).Scan(&s.ID, &s.Name, &s.Kind, &s.UID, &s.Path)
	if err != nil {
		return sym{}, err
	}
	return s, nil
}

type sym struct {
	ID   int64
	Name string
	Kind string
	UID  string
	Path string
}

// curPathCache memoizes symbol->path lookups for the relationship report. The
// cache is scoped to the calling Affected invocation (passed in), so
// concurrent calls over different stores cannot collide on a symbol id or
// read a stale path from another call.
func curPathCache(ctx context.Context, db *sql.DB, id int64, pathCache map[int64]string) string {
	if p, ok := pathCache[id]; ok {
		return p
	}
	s, err := symbolByID(ctx, db, id)
	if err != nil {
		pathCache[id] = ""
		return ""
	}
	pathCache[id] = s.Path
	return s.Path
}

func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// normPath strips a single leading "./" so a git-diff path ("src/base.go")
// matches the indexed files.path ("src/base.go") even when the input is
// "./src/base.go" (git diff emits both forms depending on invocation).
func normPath(p string) string {
	return strings.TrimPrefix(strings.TrimSpace(p), "./")
}
