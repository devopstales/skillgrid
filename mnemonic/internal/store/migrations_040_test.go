package store

import (
	"strings"
	"testing"
	"time"
)

// TestMigration040SessionEvents covers 040 (session events layer storage):
// (a) sessions gains agent_session_id, from_commit, to_commit + 6 counters;
// (b) session_events exists with UNIQUE(session_id, sequence) + 2 indexes;
// (c) counters default to 0; (d) migration recorded once, re-open idempotent.
func TestMigration040SessionEvents(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "eventsproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, col := range []string{"agent_session_id", "from_commit", "to_commit",
		"files_read", "files_written", "commands_exec", "errors",
		"sensitive_actions", "blocked_actions"} {
		if !columnExists(t, st.DB, "sessions", col) {
			t.Errorf("sessions missing column %s", col)
		}
	}
	if !tableExists(t, st.DB, "session_events") {
		t.Fatalf("missing table session_events")
	}
	if countMigration(t, st.DB, "040_session_events.sql") != 1 {
		t.Fatalf("expected 040 migration recorded once")
	}

	// Counters default to 0 when a session is inserted without them.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'eventsproj', '/tmp', ?, 'active')`, now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	var counters [6]int
	if err := st.DB.QueryRow(`
		SELECT files_read, files_written, commands_exec, errors, sensitive_actions, blocked_actions
		FROM sessions WHERE id = 's1'`).Scan(
		&counters[0], &counters[1], &counters[2], &counters[3], &counters[4], &counters[5]); err != nil {
		t.Fatalf("read counters: %v", err)
	}
	for i, v := range counters {
		if v != 0 {
			t.Errorf("counter %d = %d, want 0", i, v)
		}
	}

	// UNIQUE(session_id, sequence): the second insert with the same pair fails.
	if _, err := st.DB.Exec(`
		INSERT INTO session_events (session_id, project, sequence, action_type, timestamp)
		VALUES ('s1', 'eventsproj', 0, 'session_start', ?)`, now); err != nil {
		t.Fatalf("insert event seq 0: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO session_events (session_id, project, sequence, action_type, timestamp)
		VALUES ('s1', 'eventsproj', 0, 'session_start', ?)`, now); err == nil {
		t.Errorf("expected UNIQUE(session_id, sequence) violation on duplicate seq 0")
	} else if !strings.Contains(err.Error(), "UNIQUE") {
		t.Errorf("expected UNIQUE violation, got: %v", err)
	}
	// A different sequence for the same session is fine.
	if _, err := st.DB.Exec(`
		INSERT INTO session_events (session_id, project, sequence, action_type, timestamp)
		VALUES ('s1', 'eventsproj', 1, 'session_end', ?)`, now); err != nil {
		t.Fatalf("insert event seq 1: %v", err)
	}

	// Both indexes exist.
	for _, idx := range []string{"idx_events_session_seq", "idx_events_project_time"} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil {
			t.Fatalf("index %s: %v", idx, err)
		}
		if n != 1 {
			t.Errorf("expected index %s to exist", idx)
		}
	}

	// Re-open is idempotent: migration still recorded once, rows intact.
	st.Close()
	st2, err := Open(dir, "eventsproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "040_session_events.sql") != 1 {
		t.Fatalf("040 applied more than once")
	}
	var evts int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM session_events WHERE session_id='s1'`).Scan(&evts); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if evts != 2 {
		t.Fatalf("events lost after re-open: got %d, want 2", evts)
	}
}
