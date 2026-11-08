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
// index_meta and MUST skip already-applied files on a re-run. Skipping is
// proven directly: a re-run of the fully-applied migration set leaves the
// sqlite_master catalog object count unchanged (IF NOT EXISTS is silent, so a
// re-apply-everything runner would still pass on error alone — the count is
// the discriminator). This is the parity pattern Task 18 re-runs against
// migrations_pg/ (there, count pg_class rows the same way).
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
	// catalog object count after the first (fully-applied) open.
	before, err := masterObjectCount(raw)
	if err != nil {
		t.Fatalf("count before: %v", err)
	}
	if before == 0 {
		t.Fatalf("expected a populated sqlite_master catalog after the first open")
	}
	// an explicit Migrate re-run against the fully-applied store must skip
	// every applied file and leave the catalog untouched.
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate re-run: %v", err)
	}
	after, err := masterObjectCount(raw)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if before != after {
		t.Fatalf("Migrate re-run changed the catalog: %d before, %d after — applied files were not skipped", before, after)
	}
	// and the index_meta record must survive the re-run
	found, err = raw.Probe(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = 'migration:001_schema.sql'`)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !found {
		t.Fatalf("index_meta row lost after migrate re-run")
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// masterObjectCount returns the number of tables, indexes, triggers, and views
// in the sqlite_master catalog — the count a re-run-everything migration
// runner would inflate but a skipping runner must leave unchanged.
func masterObjectCount(raw *sqliteStore) (int, error) {
	return raw.ProbeInt(`SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','index','trigger','view')`)
}
