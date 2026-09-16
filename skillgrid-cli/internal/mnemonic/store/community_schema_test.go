package store

import (
	"testing"
)

// TestStoreOpenAppliesCommunitySchema covers @step-01 (additive 012 migration):
// a store that opens cleanly has the 012 community tables, the 012 migration
// is recorded once, and the 005 symbols/edges tables remain intact (additive,
// not a rewrite).
func TestStoreOpenAppliesCommunitySchema(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "communityproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"communities", "community_meta"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected 012 table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "012_community_knowledge_graph.sql") != 1 {
		t.Fatalf("expected 012 migration recorded once")
	}
	// 005 tables intact (additive — never rewritten).
	for _, name := range []string{"symbols", "edges"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("005 table %s missing after 012", name)
		}
	}
}
