package store

import (
	"testing"
)

// TestStoreOpenAddsUnresolvedRefsAndSymbolSegments covers 034: opening a store
// creates the unresolved_refs and symbol_segments tables, records migration
// 034 exactly once, and the unresolved_refs identity index exists (it carries
// the target-state upsert).
func TestStoreOpenAddsUnresolvedRefsAndSymbolSegments(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "proj034")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"unresolved_refs", "symbol_segments"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "034_unresolved_refs.sql") != 1 {
		t.Fatalf("expected 034 migration recorded once")
	}
	var idx int
	if err := st.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_unresolved_identity'`,
	).Scan(&idx); err != nil {
		t.Fatalf("sqlite_master index: %v", err)
	}
	if idx != 1 {
		t.Fatalf("expected the unresolved_refs identity index to exist")
	}
}
