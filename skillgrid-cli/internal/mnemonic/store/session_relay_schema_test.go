package store

import (
	"testing"
	"time"
)

// TestSessionRelayMigration covers @step-01 (additive 019 migration):
// (a) store open creates session_handoffs + session_archives without
// rewriting sessions/observations; (b) prior rows survive (count + content
// unchanged, no rebuild); (c) re-open is idempotent — no error, migration
// recorded once, tables and prior rows still intact.
func TestSessionRelayMigration(t *testing.T) {
	dir := t.TempDir()
	db := rawSQLDB(t, dir, "relayproj")
	// Seed a pre-019 store (001-era shape) so we can prove the migration
	// neither rewrites nor rebuilds existing tables.
	if _, err := db.Exec(`
		CREATE TABLE sessions (
		    id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
		    started_at TEXT NOT NULL, ended_at TEXT, summary TEXT,
		    status TEXT NOT NULL DEFAULT 'active');
		CREATE TABLE observations (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    session_id TEXT NOT NULL REFERENCES sessions(id),
		    type TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL,
		    project TEXT, scope TEXT, topic_key TEXT, normalized_hash TEXT,
		    revision_count INTEGER NOT NULL DEFAULT 0,
		    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'relayproj', '/tmp', ?, 'active')`, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, created_at, updated_at)
		VALUES ('s1', 'decision', 'keep-me', 'body', 'relayproj', ?, ?)`, now, now); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
	db.Close()

	st, err := Open(dir, "relayproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	// (a) Relay tables exist after open, migration recorded exactly once.
	for _, name := range []string{"session_handoffs", "session_archives"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected relay table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "019_session_relay.sql") != 1 {
		t.Fatalf("expected 019 migration recorded once")
	}

	// (b) Prior rows survive: counts unchanged and content untouched.
	var sess, obs int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sess); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&obs); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	if sess != 1 || obs != 1 {
		t.Fatalf("prior rows lost: sessions=%d observations=%d", sess, obs)
	}
	var sid, status, title, content string
	if err := st.DB.QueryRow(`SELECT id, status FROM sessions WHERE id='s1'`).Scan(&sid, &status); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT title, content FROM observations WHERE title='keep-me'`).Scan(&title, &content); err != nil {
		t.Fatalf("read observation: %v", err)
	}
	if status != "active" || title != "keep-me" || content != "body" {
		t.Fatalf("row rewritten: status=%q title=%q content=%q", status, title, content)
	}

	// (c) Re-open is idempotent: no error, migration still recorded once,
	// tables and prior rows still intact.
	st.Close()
	st2, err := Open(dir, "relayproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "019_session_relay.sql") != 1 {
		t.Fatalf("019 applied more than once")
	}
	for _, name := range []string{"session_handoffs", "session_archives"} {
		if !tableExists(t, st2.DB, name) {
			t.Fatalf("relay table %s missing after re-open", name)
		}
	}
	var sess2, obs2 int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sess2); err != nil {
		t.Fatalf("re-open count sessions: %v", err)
	}
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE title='keep-me'`).Scan(&obs2); err != nil {
		t.Fatalf("re-open count observations: %v", err)
	}
	if sess2 != 1 || obs2 != 1 {
		t.Fatalf("prior rows lost after re-open: sessions=%d observations=%d", sess2, obs2)
	}

	// The new tables are usable (steps 02–05 will write into them).
	hid := "handoff-1"
	if _, err := st2.DB.Exec(`
		INSERT INTO session_handoffs (project, handoff_id, source_session, status, cleave_path, created_at)
		VALUES ('relayproj', ?, 's1', 'pending', '.skillgrid/.cleave/handoff-1', ?)`, hid, now); err != nil {
		t.Fatalf("insert handoff row: %v", err)
	}
	var gotHandoff, gotPath string
	if err := st2.DB.QueryRow(`SELECT handoff_id, cleave_path FROM session_handoffs WHERE handoff_id=?`, hid).Scan(&gotHandoff, &gotPath); err != nil {
		t.Fatalf("read handoff row: %v", err)
	}
	if gotHandoff != hid || gotPath != ".skillgrid/.cleave/handoff-1" {
		t.Fatalf("handoff row mismatch: %q %q", gotHandoff, gotPath)
	}
	if _, err := st2.DB.Exec(`
		INSERT INTO session_archives (project, session_id, handoff_id, path, created_at)
		VALUES ('relayproj', 's1', ?, '/tmp/archive.tar.gz', ?)`, hid, now); err != nil {
		t.Fatalf("insert archive row: %v", err)
	}
	var arc int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM session_archives WHERE handoff_id=?`, hid).Scan(&arc); err != nil {
		t.Fatalf("count archive row: %v", err)
	}
	if arc != 1 {
		t.Fatalf("archive row missing: %d", arc)
	}
}
