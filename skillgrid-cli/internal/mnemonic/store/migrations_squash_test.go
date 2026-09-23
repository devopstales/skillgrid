package store

import (
	"testing"
)

// TestSquashShimFresh verifies a fresh database executes 001_schema.sql
// (the schema lands) and records it, then applies 040 and 041.
func TestSquashShimFresh(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "freshsquash")
	if err != nil {
		t.Fatalf("open fresh: %v", err)
	}
	defer st.Close()
	if countMigration(t, st.DB, "001_schema.sql") != 1 {
		t.Fatalf("expected 001_schema.sql recorded once on fresh DB")
	}
	if !tableExists(t, st.DB, "session_events") {
		t.Fatalf("session_events missing after fresh migrate")
	}
	if !columnExists(t, st.DB, "sessions", "agent_session_id") {
		t.Fatalf("sessions.agent_session_id missing after fresh migrate")
	}
	// 041 dropped the Hub tables even on a fresh DB.
	if tableExists(t, st.DB, "change_snapshots") {
		t.Errorf("change_snapshots should be dropped by 041 on fresh DB")
	}
}

// TestSquashShimUpgrade verifies a pre-squash database (created by the OLD
// runner, which recorded individual migration:<name> rows and a
// schema_version marker but no migration:001_schema.sql) migrates WITHOUT
// re-executing 001_schema.sql. The shim records it and skips execution; 040
// and 041 still run. Existing tables and data survive untouched.
func TestSquashShimUpgrade(t *testing.T) {
	dir := t.TempDir()
	db := rawSQLDB(t, dir, "presquash")
	defer db.Close()

	// Minimal pre-squash footprint: index_meta with a schema_version marker
	// and an old-style individual migration row, plus the two tables the
	// squashed 001_schema.sql would re-create (so a re-run would error or,
	// worse, the data-bearing tables already exist).
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS index_meta (
			key TEXT PRIMARY KEY,
			schema_version INTEGER NOT NULL
		)`); err != nil {
		t.Fatalf("create index_meta: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			project TEXT,
			directory TEXT,
			started_at TEXT,
			status TEXT,
			title TEXT,
			summary TEXT
		)`); err != nil {
		t.Fatalf("create sessions: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, project, directory, started_at, status) VALUES ('s1','presquash','/tmp','2026-01-01T00:00:00Z','active')`); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('migration:019_session_relay.sql', 1)`); err != nil {
		t.Fatalf("record old migration row: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('schema_version', 41)`); err != nil {
		t.Fatalf("record schema_version: %v", err)
	}
	db.Close()

	st, err := Open(dir, "presquash")
	if err != nil {
		t.Fatalf("open pre-squash (upgrade): %v", err)
	}
	defer st.Close()

	// The shim recorded 001_schema.sql without executing it.
	if countMigration(t, st.DB, "001_schema.sql") != 1 {
		t.Fatalf("expected 001_schema.sql recorded by shim, not executed")
	}
	// The old individual row is left in place (harmless bookkeeping).
	if countMigration(t, st.DB, "019_session_relay.sql") != 1 {
		t.Fatalf("expected old migration row to remain")
	}
	// 040 still ran: session_events + the new sessions columns exist.
	if !tableExists(t, st.DB, "session_events") {
		t.Fatalf("session_events missing after upgrade (040 should run)")
	}
	if !columnExists(t, st.DB, "sessions", "agent_session_id") {
		t.Fatalf("sessions.agent_session_id missing after upgrade (040 should run)")
	}
	// 041 still ran.
	if tableExists(t, st.DB, "change_snapshots") {
		t.Errorf("change_snapshots should be dropped by 041 on upgrade")
	}
	// Pre-existing data survived.
	var status string
	if err := st.DB.QueryRow(`SELECT status FROM sessions WHERE id='s1'`).Scan(&status); err != nil {
		t.Fatalf("read seeded session: %v", err)
	}
	if status != "active" {
		t.Errorf("seeded session rewritten: status=%q", status)
	}

	// Re-open is idempotent: 001_schema.sql still recorded once, no error.
	st.Close()
	st2, err := Open(dir, "presquash")
	if err != nil {
		t.Fatalf("re-open pre-squash: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "001_schema.sql") != 1 {
		t.Fatalf("001_schema.sql recorded more than once on re-open")
	}
}
