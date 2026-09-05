package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

var tierTables = []string{
	"tiered_contents",
	"long_term_memories",
	"retrieval_trails",
	"path_embeddings",
}

func seedObservation(t *testing.T, st *Store, title string) int64 {
	t.Helper()
	sid := "sess-tier-schema"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'tierproj', '/tmp', ?, 'active')`, sid, now); err != nil {
		t.Fatalf("session: %v", err)
	}
	res, err := st.DB.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, scope, created_at, updated_at)
		VALUES (?, 'discovery', ?, 'body', 'tierproj', 'project', ?, ?)`,
		sid, title, now, now)
	if err != nil {
		t.Fatalf("insert observation: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last id: %v", err)
	}
	return id
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
	).Scan(&n); err != nil {
		t.Fatalf("sqlite_master %s: %v", name, err)
	}
	return n == 1
}

func countMigration(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM index_meta WHERE key = ?`, "migration:"+name,
	).Scan(&n); err != nil {
		t.Fatalf("count migration %s: %v", name, err)
	}
	return n
}

// TestStoreOpenAddsTierTablesWithoutRewritingRows covers @step-01 happy path.
func TestStoreOpenAddsTierTablesWithoutRewritingRows(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "tierproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	id := seedObservation(t, st, "pre-tier")
	st.Close()

	st2, err := Open(dir, "tierproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()

	for _, name := range tierTables {
		if !tableExists(t, st2.DB, name) {
			t.Fatalf("expected table %s after open", name)
		}
	}
	if countMigration(t, st2.DB, "010_tiered_context.sql") != 1 {
		t.Fatalf("expected 010 migration recorded once")
	}

	var title string
	if err := st2.DB.QueryRow(`SELECT title FROM observations WHERE id = ?`, id).Scan(&title); err != nil {
		t.Fatalf("read observation: %v", err)
	}
	if title != "pre-tier" {
		t.Fatalf("observation rewritten: got %q", title)
	}
}

// TestUpgradeFrom008IdempotentTo010 covers @step-01 edge: upgrade + re-open.
func TestUpgradeFrom008IdempotentTo010(t *testing.T) {
	dir := t.TempDir()
	db := rawSQLDB(t, dir, "from008")
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}
	// Seed an observation-shaped row and a code-index file so we can assert intactness.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'from008', '/tmp', ?, 'active')`, now); err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, scope, revision_count, created_at, updated_at)
		VALUES ('s1', 'discovery', 'keep-me', 'body', 'from008', 'project', 0, ?, ?)`, now, now); err != nil {
		t.Fatalf("obs: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('a.go', 1, 1, 'h', ?)`, now); err != nil {
		t.Fatalf("file: %v", err)
	}
	db.Close()

	st, err := Open(dir, "from008")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, name := range tierTables {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("missing %s after first open", name)
		}
	}
	st.Close()

	st2, err := Open(dir, "from008")
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "010_tiered_context.sql") != 1 {
		t.Fatalf("010 applied more than once")
	}

	var obs, files, fts int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE title='keep-me'`).Scan(&obs); err != nil {
		t.Fatalf("obs count: %v", err)
	}
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE path='a.go'`).Scan(&files); err != nil {
		t.Fatalf("files count: %v", err)
	}
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='observations_fts'`).Scan(&fts); err != nil {
		t.Fatalf("fts: %v", err)
	}
	if obs != 1 || files != 1 || fts != 1 {
		t.Fatalf("intactness failed: obs=%d files=%d fts=%d", obs, files, fts)
	}
}

// TestMigrationFailLeavesPriorDataIntact covers @step-01 failure: a failed
// DDL abort must not make prior observations/indexes unreadable.
func TestMigrationFailLeavesPriorDataIntact(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "failproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	id := seedObservation(t, st, "survive-fail")
	path := filepath.Join(dir, "failproj.sqlite")

	// Simulate a mid-migration abort on the live handle (same connection
	// class migrate uses). Prior rows must remain readable afterwards.
	_, execErr := st.DB.Exec(`
		CREATE TABLE IF NOT EXISTS __tier_fail_probe (id INTEGER PRIMARY KEY);
		CREATE TABLE __tier_fail_probe (id INTEGER PRIMARY KEY);
	`)
	if execErr == nil {
		t.Fatal("expected failing DDL")
	}

	var title string
	if err := st.DB.QueryRow(`SELECT title FROM observations WHERE id = ?`, id).Scan(&title); err != nil {
		t.Fatalf("observation unreadable after failed DDL: %v", err)
	}
	if title != "survive-fail" {
		t.Fatalf("title=%q", title)
	}
	var fts, chunks int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='observations_fts'`).Scan(&fts); err != nil {
		t.Fatalf("fts: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='chunks_fts'`).Scan(&chunks); err != nil {
		t.Fatalf("chunks: %v", err)
	}
	if fts != 1 || chunks != 1 {
		t.Fatalf("indexes missing after fail: fts=%d chunks=%d", fts, chunks)
	}
	st.Close()

	// Re-open must still succeed (010 already applied; probe table optional).
	st2, err := Open(filepath.Dir(path), "failproj")
	if err != nil {
		t.Fatalf("re-open after fail: %v", err)
	}
	defer st2.Close()
	if err := st2.DB.QueryRow(`SELECT title FROM observations WHERE id = ?`, id).Scan(&title); err != nil {
		t.Fatalf("re-open read: %v", err)
	}
}
