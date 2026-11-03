package store

import (
	"testing"

	_ "modernc.org/sqlite/vec"
)

// The blank import registers the vec0 vtab + vec_distance_cosine + vec_f32
// with the driver so the 042 migration's CREATE VIRTUAL TABLE succeeds in
// this test's *sql.DB. The production blank import lives in the vectorstore
// package; the migration only needs the vtab registered to create the tables.

// TestVec0TablesCreatedAfterOpen pins that the 042 migration creates the two
// vec0 virtual tables on a freshly opened store.
func TestVec0TablesCreatedAfterOpen(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "proj-vec0")
	if err != nil {
		t.Fatalf("Open (runs migrate): %v", err)
	}
	defer func() { _ = s.Close() }()
	for _, table := range []string{"vec_symbols", "vec_chunks"} {
		var n int
		err := s.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n)
		if err != nil {
			t.Fatalf("query sqlite_master for %s: %v", table, err)
		}
		if n == 0 {
			t.Fatalf("vec0 table %q not created by migration 042", table)
		}
	}
}

// TestVec0MigrationIdempotent pins that opening the same store twice does not
// error and does not recreate the tables (the migration runner must skip
// already-applied migrations via index_meta).
func TestVec0MigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	s1, err := Open(dir, "proj-vec0-idem")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	s2, err := Open(dir, "proj-vec0-idem")
	if err != nil {
		t.Fatalf("second Open (re-runs migrate): %v", err)
	}
	defer func() { _ = s2.Close() }()
	for _, table := range []string{"vec_symbols", "vec_chunks"} {
		var n int
		if err := s2.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %q count=%d err=%v (expected exactly 1)", table, n, err)
		}
	}
}
