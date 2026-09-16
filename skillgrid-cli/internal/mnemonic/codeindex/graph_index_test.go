package codeindex

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

var graphTables = []string{"symbols", "edges", "rationale", "symbol_fts", "embeddings", "embed_meta", "lsh_buckets", "index_freshness"}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, name).Scan(&n); err != nil {
		t.Fatalf("tableExists %s: %v", name, err)
	}
	return n > 0
}

func writeFixtureProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nimport \"fmt\"\n\nfunc main() {\n\thelper()\n}\n\nfunc helper() {\n\tfmt.Println(\"hello\")\n}\n")
	mustWrite(t, filepath.Join(root, "lib.ts"), "import { alpha } from './alpha';\n\nexport function beta() { return alpha(); }\n\nclass Gamma extends Alpha {}\n")
	mustWrite(t, filepath.Join(root, "alpha.ts"), "export const alpha = 1;\nexport class Alpha {}\n")
	mustWrite(t, filepath.Join(root, "app.tsx"), "import React from 'react';\n\nexport function Beta() { return <div onClick={() => onClick()} />; }\n\nfunction onClick() {}\n")
	mustWrite(t, filepath.Join(root, "logic.py"), "import os\n\ndef alpha():\n    return beta()\n\ndef beta():\n    return 1\n\nclass Alpha(os.PathLike):\n    pass\n")
	mustWrite(t, filepath.Join(root, "calc.rs"), "use std::fmt;\n\npub fn alpha() -> i32 {\n    beta()\n}\n\nfn beta() -> i32 { 1 }\n")
	mustWrite(t, filepath.Join(root, "App.java"), "package app;\n\nimport java.util.List;\n\npublic class Alpha {\n    public int run() { return beta(); }\n}\n")
	return root
}

var fixtureCfg = Config{
	Include:      []string{"**/*.go", "**/*.ts", "**/*.tsx", "**/*.py", "**/*.rs", "**/*.java"},
	Exclude:      []string{"**/node_modules/**", "**/.git/**"},
	ChunkLines:   80,
	ChunkOverlap: 10,
}

// TestIndexYieldsGraphTables covers @step-01 happy: after indexing, the graph
// tables exist without rewriting files/chunks.
func TestIndexYieldsGraphTables(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, name := range graphTables {
		if !tableExists(t, idx.store.DB, name) {
			t.Errorf("expected table %s after index", name)
		}
	}
}

// TestIndexYieldsQueryableSymbolsAndEdges covers @step-01 happy: symbols and
// edges are queryable for the indexed languages.
func TestIndexYieldsQueryableSymbolsAndEdges(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	symbolCount := func(like string) int {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name LIKE ?`, like).Scan(&n); err != nil {
			t.Fatalf("count symbols %q: %v", like, err)
		}
		return n
	}
	for _, name := range []string{"helper", "alpha", "beta", "Alpha", "beta", "alpha", "run", "Beta"} {
		if symbolCount("%"+name+"%") == 0 {
			t.Errorf("expected a symbol matching %q to be indexed", name)
		}
	}

	var calls int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'calls'`).Scan(&calls); err != nil {
		t.Fatalf("count call edges: %v", err)
	}
	if calls == 0 {
		t.Errorf("expected call edges, got 0")
	}
	// A cross-file call (main.go's helper is called by main in main.go; alpha.tsx onClick is local).
	// At minimum, main() calls helper() in main.go.
	var mainCallsHelper int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols s ON s.id = e.from_id
		WHERE e.kind = 'calls' AND s.name = 'main'
			AND EXISTS (SELECT 1 FROM symbols c WHERE c.id = e.to_id AND c.name = 'helper')
	`).Scan(&mainCallsHelper)
	if err != nil {
		t.Fatalf("main->helper: %v", err)
	}
	if mainCallsHelper == 0 {
		t.Errorf("expected a calls edge from main to helper in main.go")
	}

	var heri int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind IN ('extends', 'implements')`).Scan(&heri); err != nil {
		t.Fatalf("count heritage edges: %v", err)
	}
	if heri == 0 {
		t.Errorf("expected heritage edges (TS class Gamma extends Alpha), got 0")
	}
}

// TestIndexGraphEdgesCarryConfidence covers every Edge carries a Confidence
// Label (EXTRACTED | INFERRED | AMBIGUOUS).
func TestIndexGraphEdgesCarryConfidence(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var bad int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind IS NULL OR kind = '' OR kind NOT IN ('calls', 'extends', 'implements', 'imports', 'references', 'inherits')`).Scan(&bad); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if bad != 0 {
		t.Errorf("expected only known edge kinds, found %d others", bad)
	}
	var confs []string
	rows, err := idx.store.DB.Query(`SELECT DISTINCT confidence FROM edges`)
	if err != nil {
		t.Fatalf("query confidences: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan confidence: %v", err)
		}
		confs = append(confs, c)
	}
	if rows.Err() != nil {
		t.Fatalf("rows: %v", rows.Err())
	}
	allowed := map[string]bool{"EXTRACTED": true, "INFERRED": true, "AMBIGUOUS": true}
	for _, c := range confs {
		if !allowed[c] {
			t.Errorf("unexpected confidence label %q", c)
		}
	}
	if len(confs) == 0 {
		t.Errorf("expected at least one edge with a confidence label")
	}
}

// TestIndexPreservesFilesAndChunks covers @step-01 happy: graph indexing does
// not rewrite the existing files/chunks contract.
func TestIndexPreservesFilesAndChunks(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB
	var files, chunks int
	if err := db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&files); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&chunks); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if files != 7 {
		t.Errorf("expected 7 files (all fixture files), got %d", files)
	}
	if chunks == 0 {
		t.Errorf("expected chunks to still be indexed alongside the graph")
	}
	// The existing chunk FTS search must still return prior hits.
	hits, err := search.CodeSearch(db, "hello", 5)
	if err != nil {
		t.Fatalf("chunk search: %v", err)
	}
	if len(hits) == 0 {
		t.Errorf("expected chunk search for 'hello' to return hits")
	}
}

// TestIndexOversizedFileSkippedAndCounted covers @step-01 edge: a file larger
// than max_file_size is skipped (in stats), not an error, not a fallback.
func TestIndexOversizedFileSkippedAndCounted(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	big := filepath.Join(root, "bundle.go")
	data := make([]byte, MaxFileSize+1)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(big, data, 0o644); err != nil {
		t.Fatalf("write big: %v", err)
	}
	stats, err := idx.Run(context.Background(), root, fixtureCfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.FilesOversized != 1 {
		t.Errorf("expected 1 oversized file counted, got %d", stats.FilesOversized)
	}
	// The oversized file must not be indexed as a file row.
	var n int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE path = 'bundle.go'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("oversized file should not be a files row")
	}
	// And it must not have produced a symbol via fallback.
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM symbols s JOIN files f ON f.id = s.file_id WHERE f.path = 'bundle.go'`).Scan(&n); err != nil {
		t.Fatalf("count symbols: %v", err)
	}
	if n != 0 {
		t.Errorf("oversized file should not yield symbols (skip, not fallback), got %d", n)
	}
}

// TestIndexPrunesDeletedFileFootprint covers @step-01 edge: deleting a file
// prunes its whole graph footprint in one pass, no orphans.
func TestIndexPrunesDeletedFileFootprint(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	assertFileHasSymbols(t, idx.store.DB, "calc.rs", "alpha")
	if err := os.Remove(filepath.Join(root, "calc.rs")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	assertNoFootprint(t, idx.store.DB, "calc.rs")
}

// TestIndexPrunesRemovedFunctionFootprint covers @step-01 edge: removing a
// function within a file prunes that symbol's footprint, not the whole file.
func TestIndexPrunesRemovedFunctionFootprint(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	assertSymbolPresent(t, idx.store.DB, "main.go", "helper")
	// Remove the helper function but keep main().
	newMain := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(newMain), 0o644); err != nil {
		t.Fatalf("rewrite main.go: %v", err)
	}
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	assertSymbolAbsent(t, idx.store.DB, "main.go", "helper")
	// main() must survive; its call edge to helper must be gone.
	assertSymbolPresent(t, idx.store.DB, "main.go", "main")
	var calls int
	if err := idx.store.DB.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols c ON c.id = e.to_id
		WHERE e.kind = 'calls' AND c.name = 'helper'
			AND EXISTS (SELECT 1 FROM symbols f WHERE f.id = e.from_id AND f.file_id = (SELECT id FROM files WHERE path = 'main.go'))
	`).Scan(&calls); err != nil {
		t.Fatalf("count edges: %v", err)
	}
	if calls != 0 {
		t.Errorf("expected 0 call edges into removed helper, got %d", calls)
	}
}

// TestIndexOrphanPruneNoLingering covers the target-state invariant: after a
// prune pass, no symbol/edge/vector/bucket row references a missing symbol or
// file.
func TestIndexOrphanPruneNoLingering(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeFixtureProject(t)
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "App.java")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	db := idx.store.DB
	orphanChecks := map[string]string{
		"symbols file_id": `SELECT COUNT(*) FROM symbols s WHERE NOT EXISTS (SELECT 1 FROM files f WHERE f.id = s.file_id)`,
		"edges from_id":   `SELECT COUNT(*) FROM edges e WHERE NOT EXISTS (SELECT 1 FROM symbols s WHERE s.id = e.from_id)`,
		"edges to_id":     `SELECT COUNT(*) FROM edges e WHERE e.to_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM symbols s WHERE s.id = e.to_id)`,
		"embeddings":      `SELECT COUNT(*) FROM embeddings v WHERE NOT EXISTS (SELECT 1 FROM symbols s WHERE s.id = v.symbol_id)`,
	}
	for name, q := range orphanChecks {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if n != 0 {
			t.Errorf("orphan rows in %s after prune: %d", name, n)
		}
	}
}

func assertFileHasSymbols(t *testing.T, db *sql.DB, path, name string) {
	t.Helper()
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE f.path = ? AND s.name = ?`, path, name).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n == 0 {
		t.Fatalf("expected symbol %q in %s", name, path)
	}
}

func assertNoFootprint(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols s JOIN files f ON f.id = s.file_id WHERE f.path = ?`, path).Scan(&n); err != nil {
		t.Fatalf("symbols: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 symbols for deleted file, got %d", n)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		WHERE e.from_id IN (SELECT s.id FROM symbols s JOIN files f ON f.id = s.file_id WHERE f.path = ?)
		   OR e.to_id IN (SELECT s.id FROM symbols s JOIN files f ON f.id = s.file_id WHERE f.path = ?)`, path, path).Scan(&n); err != nil {
		t.Fatalf("edges: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 edges for deleted file, got %d", n)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM embeddings v
		WHERE v.symbol_id IN (SELECT s.id FROM symbols s JOIN files f ON f.id = s.file_id WHERE f.path = ?)`, path).Scan(&n); err != nil {
		t.Fatalf("embeddings: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 embeddings for deleted file, got %d", n)
	}
}

func assertSymbolPresent(t *testing.T, db *sql.DB, path, name string) {
	t.Helper()
	assertFileHasSymbols(t, db, path, name)
}

func assertSymbolAbsent(t *testing.T, db *sql.DB, path, name string) {
	t.Helper()
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE f.path = ? AND s.name = ?`, path, name).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("expected symbol %q to be pruned from %s", name, path)
	}
}
