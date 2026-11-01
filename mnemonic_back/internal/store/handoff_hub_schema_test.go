package store

import (
	"testing"
)

// TestHandoffHubTablesRemoved covers the post-041 world: the 039 tables
// (change_snapshots / checkpoints / handoff_refs) are GONE after open — 041
// drops them. The 039 migration row stays recorded (history is not
// rewritten), additive tables (observations / sessions / session_events)
// are untouched, and the Hub-era unique-key tests are deleted (their tables
// no longer exist; 041's drop test seeds data-bearing rows pre-drop).
func TestHandoffHubTablesRemoved(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "hubproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"change_snapshots", "checkpoints", "handoff_refs"} {
		if tableExists(t, st.DB, name) {
			t.Fatalf("expected 039 table %s dropped by 041", name)
		}
	}
	if countMigration(t, st.DB, "001_schema.sql") != 1 {
		t.Fatalf("expected 039 migration still recorded once (history kept)")
	}
	if countMigration(t, st.DB, "041_drop_handoff_tables.sql") != 1 {
		t.Fatalf("expected 041 migration recorded once")
	}
	// Additive: the surviving tables are untouched.
	for _, name := range []string{"observations", "sessions", "session_events"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("%s missing after 041 (drop only)", name)
		}
	}
}
