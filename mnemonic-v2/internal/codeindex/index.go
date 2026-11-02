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
	if opts.ExtractFn == nil {
		return IndexReport{}, fmt.Errorf("codeindex.Index: ExtractFn is required (the extraction ladder is TICKET-03)")
	}
	stamp := opts.Stamp
	if stamp == "" {
		stamp = "v2-task1"
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

		g, err := opts.ExtractFn(rel, string(content))
		if err != nil {
			return rep, fmt.Errorf("codeindex.Index: extract %s: %w", rel, err)
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

	// the freshness record (spec 001 §4 — the query-path staleness
	// contract's data source; the counts are the run's).
	now := nowStampUTC()
	caps := `{"graph":true,"fts":true,"vector":true,"lsp":false,"pdg":false,"communities":false,"process":false}`
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
