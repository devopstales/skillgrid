package store

import (
	"testing"
	"time"
)

// applyMigrationsThrough simulates a pre-squash (Hub-era) database on a raw
// *sql.DB: it creates index_meta, executes the squashed 001_schema.sql and
// 040_session_events.sql, and records each (plus the schema_version marker)
// exactly like migrate() does. throughFile is accepted for signature
// compatibility; only "040_session_events.sql" is supported.
func applyMigrationsThrough(t *testing.T, dir, project, throughFile string) {
	t.Helper()
	if throughFile != "040_session_events.sql" {
		t.Fatalf("applyMigrationsThrough: unsupported throughFile %q (only 040_session_events.sql is supported)", throughFile)
	}
	db := rawSQLDB(t, dir, project)
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS index_meta (
			key TEXT PRIMARY KEY,
			schema_version INTEGER NOT NULL
		)
	`); err != nil {
		t.Fatalf("create index_meta: %v", err)
	}
	for _, name := range []string{"001_schema.sql", "040_session_events.sql"} {
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("exec %s: %v", name, err)
		}
		if _, err := db.Exec(
			`INSERT INTO index_meta (key, schema_version) VALUES (?, 1)`, "migration:"+name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET schema_version = excluded.schema_version`,
		2,
	); err != nil {
		t.Fatalf("record schema_version: %v", err)
	}
}

// TestMigration041DropsHandoffTables covers 041 (ONE-WAY Hub/relay removal):
// a Hub-era DB (migrated 001→040, all five tables present with rows, plus a
// live sessions/session_events pair) migrates cleanly — the five tables and
// six indexes are gone, sessions/session_events rows are intact, the 040
// indexes survive, and re-open is idempotent.
func TestMigration041DropsHandoffTables(t *testing.T) {
	dir := t.TempDir()
	applyMigrationsThrough(t, dir, "dropproj", "040_session_events.sql")

	db := rawSQLDB(t, dir, "dropproj")
	now := time.Now().UTC().Format(time.RFC3339)

	// Seed the durable side first: one session + one event (040 shape).
	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'dropproj', '/tmp', ?, 'active')`, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO session_events (session_id, project, sequence, action_type, timestamp)
		VALUES ('s1', 'dropproj', 0, 'session_start', ?)`, now); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	// Seed one row per Hub/relay table to prove data-bearing tables drop cleanly.
	seeds := []string{
		`INSERT INTO change_snapshots (project, branch, "commit", committed_at, created_at)
			VALUES ('dropproj', 'main', 'abc123', '` + now + `', '` + now + `')`,
		`INSERT INTO checkpoints (project, name, created_at)
			VALUES ('dropproj', 'before-apply-x', '` + now + `')`,
		`INSERT INTO handoff_refs (handoff_id, handoff_type, project, created_at)
			VALUES ('h1', 'session', 'dropproj', '` + now + `')`,
		`INSERT INTO session_handoffs (project, handoff_id, source_session, cleave_path, created_at)
			VALUES ('dropproj', 'h1', 's1', '.skillgrid/.cleave/h1', '` + now + `')`,
		`INSERT INTO session_archives (project, session_id, handoff_id, path, created_at)
			VALUES ('dropproj', 's1', 'h1', '/tmp/archive.tar.gz', '` + now + `')`,
	}
	for i, stmt := range seeds {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed hub/relay row %d: %v", i, err)
		}
	}
	// Precondition: all five tables are present with rows before the upgrade.
	for _, name := range []string{
		"change_snapshots", "checkpoints", "handoff_refs", "session_handoffs", "session_archives",
	} {
		if !tableExists(t, db, name) {
			t.Fatalf("precondition: %s missing before 041", name)
		}
	}
	db.Close()

	st, err := Open(dir, "dropproj")
	if err != nil {
		t.Fatalf("open (041 upgrade): %v", err)
	}
	defer st.Close()

	if countMigration(t, st.DB, "041_drop_handoff_tables.sql") != 1 {
		t.Fatalf("expected 041 migration recorded once")
	}
	// Earlier migrations are still recorded exactly once (no re-run).
	for _, m := range []string{"001_schema.sql", "040_session_events.sql"} {
		if countMigration(t, st.DB, m) != 1 {
			t.Fatalf("expected %s recorded once, history rewritten", m)
		}
	}

	// The five tables are absent.
	for _, name := range []string{
		"change_snapshots", "checkpoints", "handoff_refs", "session_handoffs", "session_archives",
	} {
		if tableExists(t, st.DB, name) {
			t.Errorf("expected %s dropped by 041", name)
		}
	}
	// The six Hub/relay indexes are absent.
	for _, idx := range []string{
		"idx_snapshots_project_time", "idx_checkpoints_project_status",
		"idx_handoff_refs_project", "idx_handoffs_source",
		"idx_handoffs_status", "idx_archives_session",
	} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil {
			t.Fatalf("index %s: %v", idx, err)
		}
		if n != 0 {
			t.Errorf("expected index %s dropped by 041", idx)
		}
	}

	// sessions/session_events survive with data.
	if !tableExists(t, st.DB, "sessions") {
		t.Fatalf("sessions missing after 041")
	}
	if !tableExists(t, st.DB, "session_events") {
		t.Fatalf("session_events missing after 041")
	}
	var status string
	if err := st.DB.QueryRow(`SELECT status FROM sessions WHERE id='s1'`).Scan(&status); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if status != "active" {
		t.Errorf("session row rewritten: status=%q", status)
	}
	var evts int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_events WHERE session_id='s1'`).Scan(&evts); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if evts != 1 {
		t.Errorf("events lost in 041 upgrade: got %d, want 1", evts)
	}
	// The 040 indexes survive (no over-drop).
	for _, idx := range []string{"idx_events_session_seq", "idx_events_project_time"} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil {
			t.Fatalf("index %s: %v", idx, err)
		}
		if n != 1 {
			t.Errorf("expected 040 index %s to survive 041", idx)
		}
	}

	// Re-open is idempotent: 041 still recorded once, data intact.
	st.Close()
	st2, err := Open(dir, "dropproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "041_drop_handoff_tables.sql") != 1 {
		t.Fatalf("041 applied more than once")
	}
	var evts2 int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM session_events WHERE session_id='s1'`).Scan(&evts2); err != nil {
		t.Fatalf("re-open count events: %v", err)
	}
	if evts2 != 1 {
		t.Fatalf("events lost after re-open: got %d, want 1", evts2)
	}
}
