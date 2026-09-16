package store

import (
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

var hybridTables = []string{"symbols", "edges", "rationale", "symbol_fts", "embeddings", "embed_meta", "lsh_buckets", "index_freshness"}

// TestStoreOpenPreservesExistingChunkIndex covers @step-01 failure: a store
// that already has files+chunks opens with the new graph schema, keeps
// files/chunks intact, and chunk search still returns prior hits. This is the
// highest-priority contract for step 01 — it gates everything.
func TestStoreOpenPreservesExistingChunkIndex(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "legacyproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('legacy.go', 1, 100, 'h1', ?)`, now); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	legacyText := "package main\n\nfunc legacyHelper() {\n\tprintln(\"needle-legacy\")\n}\n"
	if _, err := st.DB.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (1, 1, 5, ?, 'c1')`, legacyText); err != nil {
		t.Fatalf("seed chunk: %v", err)
	}
	st.Close()

	// Simulate a newer process opening the same DB (011 migration applies here).
	st2, err := Open(dir, "legacyproj")
	if err != nil {
		t.Fatalf("re-open with 011: %v", err)
	}
	defer st2.Close()

	for _, name := range hybridTables {
		if !tableExists(t, st2.DB, name) {
			t.Fatalf("expected table %s after open", name)
		}
	}
	if countMigration(t, st2.DB, "011_hybrid_code_intel.sql") != 1 {
		t.Fatalf("expected 011 migration recorded once")
	}

	var files, chunks int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&files); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&chunks); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if files != 1 {
		t.Errorf("expected 1 file intact, got %d", files)
	}
	if chunks != 1 {
		t.Errorf("expected 1 chunk intact, got %d", chunks)
	}
	var path string
	if err := st2.DB.QueryRow(`SELECT path FROM files WHERE id = 1`).Scan(&path); err != nil {
		t.Fatalf("file path: %v", err)
	}
	if path != "legacy.go" {
		t.Errorf("file rewritten: %q", path)
	}

	hits, err := search.CodeSearch(st2.DB, "needle-legacy", 5)
	if err != nil {
		t.Fatalf("chunk search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 prior chunk hit, got %d", len(hits))
	}
	if hits[0].Path != "legacy.go" {
		t.Errorf("chunk hit path: %q", hits[0].Path)
	}
}
