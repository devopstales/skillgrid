package store

import (
	"testing"
)

// TestStoreOpenAppliesProcessSchema covers @step-02 (additive 014 migration):
// a store that opens cleanly has the 014 process tables, the 014 migration is
// recorded once, and the 005 symbols/edges tables + the 012 community tables
// remain intact (additive, not a rewrite).
func TestStoreOpenAppliesProcessSchema(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "processproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"processes", "process_steps", "process_meta_cache"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected 014 table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "014_process_flows.sql") != 1 {
		t.Fatalf("expected 014 migration recorded once")
	}
	// 005 tables intact (additive — never rewritten).
	for _, name := range []string{"symbols", "edges"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("005 table %s missing after 014", name)
		}
	}
	// 012 community tables intact (additive on top).
	for _, name := range []string{"communities", "community_meta"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("012 table %s missing after 014", name)
		}
	}
}
