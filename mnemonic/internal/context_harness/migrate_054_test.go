package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestMigration054_TablesExist confirms migration 054 creates the indexed_files
// base table, its FTS5 shadow, and the project index. A fresh store.Open applies
// every embedded migration, so 054 lands by filename sort after 053.
func TestMigration054_TablesExist(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	// indexed_files is a plain table; indexed_files_fts is a virtual table that
	// sqlite_master records as type='table' (plus its -content/-data shadow
	// tables, so its count is >1 — we only assert presence).
	for _, table := range []string{"indexed_files", "indexed_files_fts"} {
		var n int
		if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n < 1 {
			t.Fatalf("table %s not created (count=%d)", table, n)
		}
	}
	var idx int
	if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_indexed_files_project'").Scan(&idx); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if idx != 1 {
		t.Fatalf("idx_indexed_files_project not created (count=%d)", idx)
	}
}

// TestMigration054_FTSInsertTrigger confirms the AFTER INSERT trigger keeps the
// FTS shadow in sync: an indexed file inserted by column name is immediately
// retrievable via MATCH on its content.
func TestMigration054_FTSInsertTrigger(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	_, err = st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, content)
		VALUES ('ctxproj', 'authflow.go', 'file', 'go', 'authflow file content hello indexed')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	var c int
	if err := st.DB.QueryRow(`SELECT count(*) FROM indexed_files_fts WHERE indexed_files_fts MATCH 'indexed'`).Scan(&c); err != nil {
		t.Fatalf("fts match: %v", err)
	}
	if c != 1 {
		t.Fatalf("expected 1 fts hit, got %d", c)
	}
}

// TestMigration054_Idempotent reopens a fresh store and confirms the base table
// is exactly one — 054's CREATE TABLE IF NOT EXISTS must not duplicate on a
// re-run (the runner also tracks it in index_meta, so it is skipped).
func TestMigration054_Idempotent(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir, "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()
	// Reopen the SAME data dir so the already-applied migration is re-checked.
	st, err = store.Open(dir, "ctxproj")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st.Close()
	var n int
	if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='indexed_files'").Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 indexed_files table after reopen, got %d", n)
	}
}
