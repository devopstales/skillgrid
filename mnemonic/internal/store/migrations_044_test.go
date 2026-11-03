package store

import (
	"testing"
)

// TestMigration044Columns covers 044 (bitemporal observations, ADR-0011):
// (a) observations gains valid_at + invalid_at + superseded_by; (b) the
// partial index idx_obs_invalid_at exists; (c) the migration is recorded once
// and re-opening is idempotent.
func TestMigration044Columns(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "bitemp044")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, col := range []string{"valid_at", "invalid_at", "superseded_by"} {
		if !columnExists(t, st.DB, "observations", col) {
			t.Errorf("observations missing column %s", col)
		}
	}
	if countMigration(t, st.DB, "044_bitemporal.sql") != 1 {
		t.Fatalf("expected 044 migration recorded once")
	}

	// The partial index over (project, invalid_at) exists.
	var n int
	if err := st.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_obs_invalid_at'`).Scan(&n); err != nil {
		t.Fatalf("index: %v", err)
	}
	if n != 1 {
		t.Error("expected idx_obs_invalid_at to exist")
	}
}

// TestMigration044Idempotent verifies re-running the migration set does not
// error: the runner tracks applied migrations in index_meta, so the 044
// ALTERs must not re-fire on re-open.
func TestMigration044Idempotent(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "bitemp044idem")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, col := range []string{"valid_at", "invalid_at", "superseded_by"} {
		if !columnExists(t, st.DB, "observations", col) {
			t.Errorf("observations missing column %s", col)
		}
	}
	st.Close()

	// Re-open: migrations re-run over the set but 044 must be skipped.
	st2, err := Open(dir, "bitemp044idem")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "044_bitemporal.sql") != 1 {
		t.Fatalf("044 applied more than once")
	}
}
