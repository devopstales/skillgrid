package store

import (
	"os"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dataDir := t.TempDir()
	st, err := Open(dataDir, "test-project")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

const testNow = "2026-01-01T00:00:00Z"

func TestOpenCreatesRequiredTables(t *testing.T) {
	st := openTestStore(t)
	want := []string{
		"sessions",
		"observations",
		"files",
		"chunks",
		"web_cache",
		"index_meta",
	}
	for _, name := range want {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_list WHERE name = ?`, name).Scan(&n); err != nil {
			t.Fatalf("pragma_table_list query %q: %v", name, err)
		}
		if n == 0 {
			t.Errorf("expected table %q to exist", name)
		}
	}
}

func TestOpenCreatesFTSTables(t *testing.T) {
	st := openTestStore(t)
	want := []string{"observations_fts", "chunks_fts", "web_cache_fts"}
	for _, name := range want {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_list WHERE name = ?`, name).Scan(&n); err != nil {
			t.Fatalf("pragma_table_list query %q: %v", name, err)
		}
		if n == 0 {
			t.Errorf("expected FTS table %q to exist", name)
		}
	}
}

// Migrations are wrapped in IF NOT EXISTS so re-runs are no-ops.
func TestMigrationsAreIdempotent(t *testing.T) {
	dataDir := t.TempDir()
	st1, err := Open(dataDir, "idem")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	var n1 int
	if err := st1.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts`).Scan(&n1); err != nil {
		t.Fatalf("count: %v", err)
	}
	if _, err := st1.DB.Exec(`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status) VALUES ('x','idem','/tmp','2026-01-01T00:00:00Z','active')`); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	st1.Close()

	st2, err := Open(dataDir, "idem")
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()
	var n2 int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts`).Scan(&n2); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n1 != n2 {
		t.Errorf("FTS row count changed across re-open: %d -> %d", n1, n2)
	}
}

func TestOpenRejectsBadProjectID(t *testing.T) {
	dataDir := t.TempDir()
	if _, err := Open(dataDir, ""); err == nil {
		t.Errorf("expected empty project id to be rejected")
	}
	if _, err := Open(dataDir, "a/../b"); err == nil {
		t.Errorf("expected '..' project id to be rejected")
	}
}

func TestSchemaVersionMeta(t *testing.T) {
	st := openTestStore(t)
	var v int
	if err := st.DB.QueryRow(`SELECT schema_version FROM index_meta WHERE key = 'schema_version'`).Scan(&v); err != nil {
		t.Fatalf("schema_version: %v", err)
	}
	if v != 1 {
		t.Errorf("expected schema_version 1, got %d", v)
	}
}

// TestFTSSyncInsertUpdateDelete verifies the FTS sync triggers fire on insert,
// update, and delete for observations.
func TestFTSSyncInsertUpdateDelete(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1','test-project','/tmp/x', ?, 'active')`, testNow); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	res, err := st.DB.Exec(`
		INSERT INTO observations
			(session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at)
		VALUES ('s1','decision','title-one','body-aaa','test-project','project','h1',0,?,?)`, testNow, testNow)
	if err != nil {
		t.Fatalf("insert observation: %v", err)
	}
	id, _ := res.LastInsertId()

	var matchNew int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"title-one"`).Scan(&matchNew); err != nil {
		t.Fatalf("fts match new: %v", err)
	}
	if matchNew == 0 {
		t.Errorf("expected FTS to find inserted observation")
	}

	if _, err := st.DB.Exec(`UPDATE observations SET title = 'title-two' WHERE id = ?`, id); err != nil {
		t.Fatalf("update: %v", err)
	}
	var matchTwo int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"title-two"`).Scan(&matchTwo); err != nil {
		t.Fatalf("fts match two: %v", err)
	}
	if matchTwo == 0 {
		t.Errorf("expected FTS to reflect updated title")
	}

	var matchOld int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"title-one"`).Scan(&matchOld); err != nil {
		t.Fatalf("fts match old: %v", err)
	}
	if matchOld != 0 {
		t.Errorf("expected old title gone, found %d", matchOld)
	}

	if _, err := st.DB.Exec(`DELETE FROM observations WHERE id = ?`, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var matchAfterDelete int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"title-two"`).Scan(&matchAfterDelete); err != nil {
		t.Fatalf("fts match after delete: %v", err)
	}
	if matchAfterDelete != 0 {
		t.Errorf("expected FTS to not see deleted observation, found %d", matchAfterDelete)
	}
}

func TestWALJournalMode(t *testing.T) {
	st := openTestStore(t)
	var mode string
	if err := st.DB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", mode)
	}
}

func TestDataDirCreated(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "nested", "data")
	st, err := Open(dataDir, "proj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()
	if _, err := os.Stat(filepath.Join(dataDir, "proj.sqlite")); err != nil {
		t.Errorf("expected created db file: %v", err)
	}
}

