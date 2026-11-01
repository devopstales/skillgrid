package store

import (
	"context"
	"testing"
)

// [migration-runs-from-empty] a fresh data directory is migrated from an empty
// file, and closing/reopening the store must not re-apply migrations already
// recorded in index_meta (additive-only discipline).
func TestMigrationRunsFromEmptyAndReopens(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := Open(ctx, dir, "proj-m", SQLite, "")
	if err != nil {
		t.Fatalf("fresh open: %v", err)
	}
	// the full schema must be present after the first open
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	for _, table := range []string{"observations", "observations_fts", "prompts_fts", "sync_mutations", "observation_save_operations"} {
		found, err := raw.Probe(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, table)
		if err != nil {
			t.Fatalf("probe %s: %v", table, err)
		}
		if !found {
			t.Fatalf("missing table %s after fresh open", table)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// reopen: migrations must be idempotent (additive-only discipline)
	st2, err := Open(ctx, dir, "proj-m", SQLite, "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	// and the store must still be fully usable after the reopen
	res, err := st2.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "learning", Title: "after reopen",
		Content: "the store works after an idempotent reopen",
		Project: "proj-m", Scope: "project",
	})
	if err != nil {
		t.Fatalf("add after reopen: %v", err)
	}
	r, err := st2.Search(ctx, SearchQuery{Query: "idempotent", Project: "proj-m", Limit: 10})
	if err != nil {
		t.Fatalf("search after reopen: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("search after reopen broken: %+v", r)
	}
}

// [migration-additive-reopen] the runner records every applied migration in
// index_meta; a second run must skip applied files (the index_meta rows prove
// it — this is the parity pattern Task 18 re-runs against migrations_pg/).
func TestMigrationReopenSkipsAppliedFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := Open(ctx, dir, "proj-m2", SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	// first open recorded the applied migration + the schema_version sentinel
	found, err := raw.Probe(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = 'migration:001_schema.sql'`)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !found {
		t.Fatalf("index_meta must record migration:001_schema.sql after the first open")
	}
	found, err = raw.Probe(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = 'schema_version' AND schema_version >= 1`)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !found {
		t.Fatalf("index_meta must record schema_version after the first open")
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// explicit Migrate re-run (idempotent) plus a second Open must both succeed
	st3, err := Open(ctx, dir, "proj-m2", SQLite, "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st3.Close()
	if err := st3.Migrate(ctx); err != nil {
		t.Fatalf("migrate re-run: %v", err)
	}
	// raw's pooled handle was released above; probe through the reopen's store
	found, err = st3.(interface {
		Probe(query string, args ...any) (bool, error)
	}).Probe(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = 'migration:001_schema.sql'`)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !found {
		t.Fatalf("index_meta row lost after reopen + migrate re-run")
	}
}
