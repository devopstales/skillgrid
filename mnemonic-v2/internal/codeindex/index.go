// Package codeindex is the live symbol/edge graph of the project's working
// tree: the scan → per-file extract tx → orphan prune → index_freshness
// driver (the write path, spec 02-code-index 001 §6) and the extraction /
// graph / search / watch functions that thicken it (TICKET-03…08).
package codeindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/extract"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// excludedDirs are the walk's always-excluded directory names (spec 001
// §6 / T1: .git / node_modules / vendor / build output / the KB vault —
// the KB has its own kb_index and is not code).
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".skillgrid":   true,
}

// extractFn is the per-file extract hook (the Task 1 test-double seam —
// Task 3 wires the real tree-sitter ladder into the same hook). path is
// the repo-relative path; content is the file body.
type extractFn func(path, content string) (store.FileGraph, error)

// IndexOpts configures a run. Project scopes the index_freshness record
// (one DB per project — CI-O2 — so the record's PK is the project id).
// ExtractFn is required at Task 1 time (the real ladder lands in
// TICKET-03); Stamp is the extractor stamp (D-STAMP — TICKET-03's real
// content hash; a placeholder const at Task 1 time, ponytail: the stamp's
// algorithm is TICKET-03's door, not the driver's).
type IndexOpts struct {
	Project   string
	ExtractFn extractFn
	Stamp     string
	// LSP is the optional LSP tier (TICKET-08b, CI-6, off by default). When
	// non-nil the run resolves member calls through it after the core index:
	// a resolved call → an LSP_RESOLVED edge (the 4th confidence tier).
	// Best-effort: a failing/timing-out server warns + continues (the static
	// name-only edge stands, never a partial edge set).
	LSP *extract.LSPConfig
}

// IndexReport is the run's first-class outcomes (T1: oversized files are
// counted, not silent; deleted files are counted, not silent).
type IndexReport struct {
	FilesIndexed   int
	FilesDeleted   int
	FilesOversized int
}

// Index walks root, extracts every eligible file, writes each file's
// target state through the Store inside ONE per-file tx (K1 — the
// idempotency claim rests on it), prunes the orphan symbols the run
// declared gone, and records index_freshness. A per-file extract or write
// failure returns an error (the run is not partial-silent); the watch
// loop's failure isolation (TICKET-06) is the caller's concern.
func Index(ctx context.Context, root string, st store.Store, opts IndexOpts) (IndexReport, error) {
	// The ExtractFn seam (TICKET-01's test double, TICKET-03's real ladder):
	// a nil ExtractFn defaults to the real 3-tier ladder (TICKET-03 wires it
	// in; the store-level tests still inject a test double).
	fn := opts.ExtractFn
	if fn == nil {
		fn = extract.DefaultExtractFn
	}
	stamp := opts.Stamp
	if stamp == "" {
		stamp = extract.Stamp() // D-STAMP: the real content hash (TICKET-03)
	}
	var rep IndexReport

	// walk the tree (deterministic order for deterministic reports)
	var rels []string
	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && excludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // symlinks are not indexed (v1 stance; ponytail: the walk follows them for regular files only)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rels = append(rels, rel)
		return nil
	})
	if walkErr != nil {
		return rep, fmt.Errorf("codeindex.Index: walk: %w", walkErr)
	}
	sort.Strings(rels)

	// the target uid set (the orphan prune's input, spec 002 §9)
	var targetUIDs []string
	var symbolsCount, edgesCount, chunksCount, unresolvedCount int

	for _, rel := range rels {
		full := filepath.Join(root, rel)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return rep, fmt.Errorf("codeindex.Index: read %s: %w", rel, err)
		}
		lang := languageOf(rel)
		if lang == "" {
			continue // unmapped extension: skipped (not indexed, not counted)
		}
		sum := sha256.Sum256(content)
		hash := hex.EncodeToString(sum[:])

		// the incremental guard (spec 001 §2): an unchanged file (same
		// content_hash) is a no-op (idempotent upsert, T1). Its existing
		// uid set still joins the prune target (spec 002 §9: the target is
		// "the new files' + the skipped files' UIDs" — a no-op run must not
		// prune the graph it is not touching).
		if f, err := st.FileByPath(ctx, rel); err == nil && f.ContentHash == hash {
			targetUIDs = append(targetUIDs, fileUids(ctx, st, f.ID)...)
			continue
		}

		g, err := fn(rel, string(content))
		if err != nil {
			return rep, fmt.Errorf("codeindex.Index: extract %s: %w", rel, err)
		}
		// The oversized outcome (T1): a file over the max is not indexed —
		// counted in files_oversized, not silent. The ladder marks it via the
		// graph's Oversized flag; the driver counts it (a zero-value graph
		// from a test double has Oversized=false → treated as indexed).
		if g.Oversized {
			rep.FilesOversized++
			continue
		}

		// K1: the five per-file writes in ONE tx — all commit or all roll
		// back (a partial file is not a graph state).
		tx, err := st.Begin(ctx)
		if err != nil {
			return rep, fmt.Errorf("codeindex.Index: begin %s: %w", rel, err)
		}
		fid, err := tx.UpsertFile(ctx, store.FileRow{
			Path: rel, MtimeNS: info.ModTime().UnixNano(), Size: info.Size(),
			ContentHash: hash, Language: lang,
		})
		if err != nil {
			_ = tx.Rollback(ctx)
			return rep, fmt.Errorf("codeindex.Index: upsert file %s: %w", rel, err)
		}
		if _, err := tx.UpsertSymbols(ctx, fid, g.Symbols); err != nil {
			_ = tx.Rollback(ctx)
			return rep, fmt.Errorf("codeindex.Index: upsert symbols %s: %w", rel, err)
		}
		// ponytail: edge target resolution (name → symbols.id) is the
		// extraction ladder's job (TICKET-03); the driver writes what the
		// extract declares (CI-INV-3: an unbound reference is name-only +
		// an unresolved ledger row, never a guess).
		if err := tx.ReplaceEdgesForFile(ctx, fid, g.Edges); err != nil {
			_ = tx.Rollback(ctx)
			return rep, fmt.Errorf("codeindex.Index: replace edges %s: %w", rel, err)
		}
		if err := tx.ReplaceUnresolvedForFile(ctx, fid, g.Unresolved); err != nil {
			_ = tx.Rollback(ctx)
			return rep, fmt.Errorf("codeindex.Index: replace unresolved %s: %w", rel, err)
		}
		if err := tx.ReplaceChunksForFile(ctx, fid, g.Chunks); err != nil {
			_ = tx.Rollback(ctx)
			return rep, fmt.Errorf("codeindex.Index: replace chunks %s: %w", rel, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return rep, fmt.Errorf("codeindex.Index: commit %s: %w", rel, err)
		}

		rep.FilesIndexed++
		targetUIDs = append(targetUIDs, uidsOf(g.Symbols)...)
		symbolsCount += len(g.Symbols)
		edgesCount += len(g.Edges)
		chunksCount += len(g.Chunks)
		unresolvedCount += len(g.Unresolved)
	}

	// absent-on-disk = delete (M10): a files row whose path the walk did
	// not see is removed (the FK cascades take its symbols/edges/chunks).
	if rows, ok := st.(interface {
		RawQuery(q string, args ...any) ([][]any, error)
	}); ok {
		seen := map[string]bool{}
		for _, rel := range rels {
			seen[rel] = true
		}
		fileRows, err := rows.RawQuery(`SELECT path FROM files`)
		if err != nil {
			return rep, fmt.Errorf("codeindex.Index: read files: %w", err)
		}
		for _, r := range fileRows {
			path, ok := r[0].(string)
			if !ok || seen[path] {
				continue
			}
			if err := st.DeleteFile(ctx, path); err != nil {
				return rep, fmt.Errorf("codeindex.Index: delete %s: %w", path, err)
			}
			rep.FilesDeleted++
		}
	}

	// the orphan prune (spec 002 §9): a symbol whose uid the run did not
	// declare is gone.
	if err := st.PruneOrphanSymbols(ctx, targetUIDs); err != nil {
		return rep, fmt.Errorf("codeindex.Index: prune orphans: %w", err)
	}

	// the LSP tier (TICKET-08b, CI-6 — opt-in, off by default): the member
	// calls the static pass could not type are resolved through the language
	// server; a resolved call → an LSP_RESOLVED edge (the 4th confidence
	// tier). Best-effort: a failing/timing-out server warns + continues
	// (the static name-only edge stands, never a partial edge set).
	lspRan := false
	if opts.LSP != nil {
		if n, warnings := RunLSPPass(ctx, root, st, opts.LSP); warnings != nil {
			// The tier's degradation notes (the warn + continue contract —
			// the handler's Index surfaces them on the run result).
			if opts.LSP.Warn != nil {
				for _, w := range warnings {
					opts.LSP.Warn(w)
				}
			}
			_ = n
		}
		lspRan = true
	}

	// the freshness record (spec 001 §4 — the query-path staleness
	// contract's data source; the counts are the run's; capabilities.lsp
	// reflects the tier's wiring — true when --lsp ran, false otherwise).
	now := nowStampUTC()
	lspCap := "false"
	if lspRan {
		lspCap = "true"
	}
	caps := `{"graph":true,"fts":true,"vector":true,"lsp":` + lspCap + `,"pdg":false,"communities":false,"process":false}`
	if err := st.UpdateIndexFreshness(ctx, store.FreshnessRecord{
		Project:         opts.Project,
		ExtractorStamp:  stamp,
		LastIndexedAt:   &now,
		FilesIndexed:    rep.FilesIndexed,
		FilesDeleted:    rep.FilesDeleted,
		SymbolsCount:    symbolsCount,
		EdgesCount:      edgesCount,
		ChunksCount:     chunksCount,
		UnresolvedCount: unresolvedCount,
		Capabilities:    caps,
	}); err != nil {
		return rep, fmt.Errorf("codeindex.Index: freshness: %w", err)
	}
	return rep, nil
}

// RunLSPPass is the LSP tier pass (TICKET-08b, CI-6): it resolves the member
// calls the static extract could not type (the language server on PATH, or
// the injected LSPResolver seam) and writes the resolved calls as
// LSP_RESOLVED edges over the Store. It runs after the core index (the
// edges/symbols it resolves over are committed). Best-effort: a missing
// server, a failing/timing-out server (the per-call 30 s timeout), or a
// resolution error returns the warnings and NO edges — the static name-only
// edge stands, never a partial edge set (all-or-nothing per call site).
// Returns the number of LSP_RESOLVED edges written + the degradation notes.
func RunLSPPass(ctx context.Context, root string, st store.Store, cfg *extract.LSPConfig) (int, []string) {
	var warnings []string
	warn := func(s string) { warnings = append(warnings, s) }

	// The scanned file set (the LSP-covered languages — the walk mirrors the
	// core index's exclusion set).
	var files []extract.LSPFile
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // a walk error is a skip, not a failure (best-effort pass)
		}
		if d.IsDir() {
			if p != root && excludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		if extract.LSPLanguageForPath(rel) == "" {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		files = append(files, extract.LSPFile{Path: rel, Contents: content})
		return nil
	})
	if len(files) == 0 {
		return 0, warnings
	}

	// The client: the injected resolver (the test seam) or the real server
	// binaries on PATH (a missing server → the warn + continue no-op).
	var client *extract.LSPClient
	if cfg.Resolver != nil {
		client = extract.NewLSPClientForTest(extract.LSPClientOptions{
			Root: root, Files: func() []extract.LSPFile { return files },
			Timeout: cfg.Timeout, Warn: warn,
		}, cfg.Resolver)
	} else {
		c, err := extract.NewLSPClient(extract.LSPClientOptions{
			Root: root, Files: func() []extract.LSPFile { return files },
			Timeout: cfg.Timeout, Warn: warn,
		})
		if err != nil {
			warn("lsp: " + err.Error() + " (static index unchanged)")
			return 0, warnings
		}
		client = c
	}

	// The resolution (all-or-nothing: an error → NO edges, the static index
	// stands — the v1 lspPass's "warn + continue, no partial edge set").
	edges, err := client.ResolveMemberCalls(ctx)
	if err != nil {
		warn("lsp: " + err.Error() + " (static index unchanged)")
		return 0, warnings
	}
	if len(edges) == 0 {
		return 0, warnings
	}

	// The persistence: each resolved call → an LSP_RESOLVED edge upsert (the
	// v1 PersistResolvedEdges shape over the Store's raw seams).
	written := 0
	for _, e := range edges {
		fid, ok := fileIDByPath(ctx, st, e.FilePath)
		if !ok {
			continue // the file is not indexed — no edge source (not fabricated)
		}
		fromID := enclosingSymbolID(ctx, st, fid, e.Line)
		if fromID == 0 {
			continue // no enclosing symbol — no edge source (not fabricated)
		}
		var toID *int64
		if rows, err := st.SymbolByName(ctx, e.Callee); err == nil && len(rows) > 0 {
			toID = &rows[0].ID
		}
		if err := upsertLSPEdge(ctx, st, fid, fromID, toID, e.Callee, e.Line); err != nil {
			warn(fmt.Sprintf("lsp: persist %s:%d: %v", e.FilePath, e.Line, err))
			continue
		}
		written++
	}
	return written, warnings
}

// fileIDByPath resolves a file's id by its repo-relative path ("" → not
// found). A query error is not-found (the best-effort pass never errors).
func fileIDByPath(ctx context.Context, st store.Store, path string) (int64, bool) {
	f, err := st.FileByPath(ctx, path)
	if err != nil {
		return 0, false
	}
	return f.ID, true
}

// enclosingSymbolID resolves the symbol enclosing (file, line) — the
// function/method whose span contains the line (the narrowest such symbol,
// the v1 enclosingSymbolForLine shape). 0 = none (no edge source).
func enclosingSymbolID(ctx context.Context, st store.Store, fileID int64, line int) int64 {
	syms, err := st.SymbolsByFile(ctx, fileID)
	if err != nil {
		return 0
	}
	var best int64
	bestSpan := int64(1 << 30)
	for _, s := range syms {
		if s.Kind != "function" && s.Kind != "method" {
			continue
		}
		if s.StartLine <= line && line <= s.EndLine {
			span := int64(s.EndLine - s.StartLine)
			if span < bestSpan {
				bestSpan = span
				best = s.ID
			}
		}
	}
	return best
}

// upsertLSPEdge writes the LSP_RESOLVED edge (a kind=calls row at the call
// site's line). An existing edge at the same (from, line, kind) is
// promoted: its confidence becomes LSP_RESOLVED (the v1 ON CONFLICT
// promotion — the static edge is superseded, not duplicated). A missing raw
// seam is a no-op (the dialect does not expose it).
func upsertLSPEdge(ctx context.Context, st store.Store, fileID, fromID int64, toID *int64, toName string, line int) error {
	re, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return nil
	}
	return re.RawExec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line, context)
		VALUES ('calls', $1, $2, $3, $4, 'LSP_RESOLVED', $5, 'lsp')
		ON CONFLICT(kind, from_id, file_id, to_id, to_name, line) DO UPDATE SET
			confidence = 'LSP_RESOLVED',
			to_id = $3,
			to_name = $4,
			context = 'lsp'`, fromID, fileID, nullInt64(toID), toName, line)
}

// nullInt64 renders a *int64 for a raw SQL arg (nil → NULL).
func nullInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// uidsOf collects a symbol set's uids (the prune target's input).
func uidsOf(syms []store.Symbol) []string {
	var out []string
	for _, s := range syms {
		if s.UID != "" {
			out = append(out, s.UID)
		}
	}
	return out
}

// fileUids reads the stored uid set of one file (the no-op guard's prune-
// target input — ponytail: a SQLite-specific read seam, the M10 delete read
// 's twin; the dialect-neutral form (a FileSymbols or uid-list method) is
// TICKET-02's parity-table-set decision).
func fileUids(ctx context.Context, st store.Store, fileID int64) []string {
	raw, ok := st.(interface {
		RawQuery(q string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil
	}
	rows, err := raw.RawQuery(`SELECT uid FROM symbols WHERE file_id = ?`, fileID)
	if err != nil {
		return nil
	}
	var out []string
	for _, r := range rows {
		if uid, ok := r[0].(string); ok && uid != "" {
			out = append(out, uid)
		}
	}
	return out
}

// languageOf maps a file path to its detected language (the v1 ext map,
// kept — ponytail: the extension table is TICKET-03's data; Task 1 carries
// the minimum set the door's fixtures need). An unmapped extension is
// skipped (not indexed).
func languageOf(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".js":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp":
		return "cpp"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".md":
		return "markdown"
	case ".yaml", ".yml", ".toml", ".json":
		return "config"
	case ".sql":
		return "sql"
	}
	return ""
}

// nowStampUTC is the freshness record's wall-clock (the store's own
// timestamp format, the memory change's nowStamp shape — last_indexed_at
// is compared as a string, never parsed, so the two formats agree).
func nowStampUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
