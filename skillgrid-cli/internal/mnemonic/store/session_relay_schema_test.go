package store

import (
	"testing"
)

// TestSessionRelayTablesRemoved covers the post-041 world on the fresh-open
// path: session_handoffs / session_archives (019) are GONE — 041 drops them.
// The 019 migration row stays recorded (history is not rewritten), the relay
// indexes are gone, and sessions/observations/session_events are untouched.
// (The Hub-era upgrade path with data-bearing rows is covered by
// TestMigration041DropsHandoffTables in migrations_041_test.go; the old
// insert/read assertions against the dropped tables are deleted.)
func TestSessionRelayTablesRemoved(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "relayproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"session_handoffs", "session_archives"} {
		if tableExists(t, st.DB, name) {
			t.Fatalf("expected relay table %s dropped by 041", name)
		}
	}
	if countMigration(t, st.DB, "019_session_relay.sql") != 1 {
		t.Fatalf("expected 019 migration still recorded once (history kept)")
	}
	if countMigration(t, st.DB, "041_drop_handoff_tables.sql") != 1 {
		t.Fatalf("expected 041 migration recorded once")
	}
	for _, idx := range []string{"idx_handoffs_source", "idx_handoffs_status", "idx_archives_session"} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil {
			t.Fatalf("index %s: %v", idx, err)
		}
		if n != 0 {
			t.Fatalf("expected relay index %s dropped by 041", idx)
		}
	}
	// The surviving tables are untouched.
	for _, name := range []string{"observations", "sessions", "session_events"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("%s missing after 041 (drop only)", name)
		}
	}
}
