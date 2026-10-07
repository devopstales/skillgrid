package store

import (
	"testing"
)

// TestScanTablesMigrateIdempotent covers migration 050 (additive scan +
// dependency graph tables): after store.Open, all five new table/view names
// are present in sqlite_master, and re-opening the same store file re-runs
// the runner without error (no-op on the already-created tables).
//
// SATISFIES: `happy path scan migration applies idempotently`
func TestScanTablesMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "scanproj050")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	for _, tbl := range []string{"scans", "findings", "findings_fts", "dependencies", "dep_edges"} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT count(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, tbl,
		).Scan(&n); err != nil {
			t.Fatalf("sqlite_master %s: %v", tbl, err)
		}
		if n == 0 {
			t.Fatalf("table/view %s missing after migration", tbl)
		}
	}
	// 050 must be recorded once and not re-applied on re-open.
	if countMigration(t, st.DB, "050_scan_findings.sql") != 1 {
		t.Fatalf("expected 050 migration recorded once, got %d", countMigration(t, st.DB, "050_scan_findings.sql"))
	}
	st.Close()

	// Idempotency: re-opening the same file re-runs the runner without
	// error (no-op on existing tables).
	st2, err := Open(dir, "scanproj050")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	for _, tbl := range []string{"scans", "findings", "findings_fts", "dependencies", "dep_edges"} {
		var n int
		if err := st2.DB.QueryRow(
			`SELECT count(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, tbl,
		).Scan(&n); err != nil {
			t.Fatalf("sqlite_master after re-open %s: %v", tbl, err)
		}
		if n == 0 {
			t.Fatalf("table/view %s missing after re-open", tbl)
		}
	}
	if countMigration(t, st2.DB, "050_scan_findings.sql") != 1 {
		t.Fatalf("050 applied more than once")
	}
}
