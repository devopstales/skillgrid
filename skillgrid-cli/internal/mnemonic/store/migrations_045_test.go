package store

import (
	"testing"
)

// TestMigration045 covers 045 (query embedding cache): query_cache exists,
// the migration is recorded once, and re-opening does not apply it again.
func TestMigration045(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "qcache045")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !tableExists(t, st.DB, "query_cache") {
		t.Fatal("query_cache missing")
	}
	if countMigration(t, st.DB, "045_search_aids.sql") != 1 {
		t.Fatal("expected 045 migration recorded once")
	}
	st.Close()

	st2, err := Open(dir, "qcache045")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if !tableExists(t, st2.DB, "query_cache") {
		t.Fatal("query_cache missing after re-open")
	}
	if countMigration(t, st2.DB, "045_search_aids.sql") != 1 {
		t.Fatal("045 applied more than once")
	}
}
