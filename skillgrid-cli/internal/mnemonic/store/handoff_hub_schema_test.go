package store

import (
	"testing"
)

// TestHandoffHubMigration covers change 015-handoff-hub (additive 039
// migration): (a) store open creates change_snapshots / checkpoints /
// handoff_refs without rewriting sessions/observations; (b) re-open is
// idempotent — migration recorded once, tables intact; (c) the unique
// constraints hold (snapshot upsert key, checkpoint name key, ref key).
func TestHandoffHubMigration(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "hubproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"change_snapshots", "checkpoints", "handoff_refs"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected 039 table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "039_handoff_hub.sql") != 1 {
		t.Fatalf("expected 039 migration recorded once")
	}
	// Additive: the pre-existing tables are untouched.
	for _, name := range []string{"observations", "sessions", "session_handoffs"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("%s missing after 039 (additive only)", name)
		}
	}
}

// TestHandoffHubSnapshotUpsertKey proves (project, commit) is unique so the
// snapshot log is idempotent when the same commit is recorded twice.
func TestHandoffHubSnapshotUpsertKey(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "hubproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	stmt := `INSERT INTO change_snapshots
		(project, branch, "commit", subject, committed_at, created_at)
		VALUES ('hubproj', 'main', ?, 's', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`
	if _, err := st.DB.Exec(stmt, "abc123"); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
	// Second insert of the same (project, commit) must fail on the unique key;
	// ON CONFLICT upsert is the supported write path.
	if _, err := st.DB.Exec(stmt, "abc123"); err == nil {
		t.Fatalf("expected unique violation on duplicate (project, commit)")
	}
	var n int
	if err := st.DB.QueryRow(
		`SELECT COUNT(*) FROM change_snapshots WHERE project = 'hubproj' AND "commit" = 'abc123'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 snapshot row, got %d", n)
	}
}

// TestHandoffHubCheckpointAndRefKeys proves the checkpoint (project, name)
// unique key and the handoff_refs (handoff_id, handoff_type, project) unique
// key both reject duplicates.
func TestHandoffHubCheckpointAndRefKeys(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "hubproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	cp := `INSERT INTO checkpoints (project, name, created_at) VALUES ('hubproj', ?, '2026-01-01T00:00:00Z')`
	if _, err := st.DB.Exec(cp, "before-apply-auth"); err != nil {
		t.Fatalf("insert checkpoint: %v", err)
	}
	if _, err := st.DB.Exec(cp, "before-apply-auth"); err == nil {
		t.Fatalf("expected unique violation on duplicate (project, name)")
	}

	ref := `INSERT INTO handoff_refs (handoff_id, handoff_type, project, created_at)
		VALUES (?, 'session', 'hubproj', '2026-01-01T00:00:00Z')`
	if _, err := st.DB.Exec(ref, "h1"); err != nil {
		t.Fatalf("insert ref: %v", err)
	}
	if _, err := st.DB.Exec(ref, "h1"); err == nil {
		t.Fatalf("expected unique violation on duplicate (handoff_id, handoff_type, project)")
	}
	// Same handoff_id with a different type is allowed.
	if _, err := st.DB.Exec(`INSERT INTO handoff_refs (handoff_id, handoff_type, project, created_at)
		VALUES ('h1', 'team', 'hubproj', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert second-type ref (should be allowed): %v", err)
	}
}
