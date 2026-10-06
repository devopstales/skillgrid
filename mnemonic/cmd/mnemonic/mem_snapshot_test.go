package main

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestMemSnapshotCLI covers 20.4: `mem snapshot create` captures a point-in-
// time snapshot (returns an id); `mem snapshot restore <id>` rolls the project
// back to that state; `mem snapshot list` lists the snapshots newest-first
// with their state hash and timestamp.
func TestMemSnapshotCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-snapshot"

	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title)
		VALUES ('sess-snap', ?, '/tmp', '2026-01-01T00:00:00Z', 'active', 'snapshot session')`,
		project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-snap",
		Type:      "decision",
		Title:     "snapshot baseline",
		Content:   "baseline content for the snapshot",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	st.Close()

	// mem snapshot create → a snapshot id is returned.
	out := runMemCLI(t, dataDir, "snapshot", "create", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"created": true`) {
		t.Fatalf("mem snapshot create should report created:true, got:\n%s", out)
	}
	if !strings.Contains(out, `"snapshot":`) {
		t.Fatalf("mem snapshot create should return a snapshot id, got:\n%s", out)
	}

	// mem snapshot list → the snapshot is listed with state_hash + created_at.
	out = runMemCLI(t, dataDir, "snapshot", "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"count": 1`) {
		t.Fatalf("mem snapshot list should show count 1, got:\n%s", out)
	}
	if !strings.Contains(out, `"state_hash"`) || !strings.Contains(out, `"created_at"`) {
		t.Fatalf("mem snapshot list should show state_hash + created_at, got:\n%s", out)
	}

	// Modify the store (a second observation is added).
	st2, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	mem2 := memory.New(st2, project)
	if _, err := mem2.Save(ctx, memory.SaveInput{
		SessionID: "sess-snap",
		Type:      "decision",
		Title:     "added after snapshot",
		Content:   "this did not exist at capture time",
	}); err != nil {
		t.Fatalf("save after snapshot: %v", err)
	}
	st2.Close()

	// mem snapshot restore <id> → the project rolls back to the captured state.
	snapID := extractSnapshotID(t, runMemCLI(t, dataDir, "snapshot", "list", "--project", project, "--dir", dataDir))
	out = runMemCLI(t, dataDir, "snapshot", "restore", snapID, "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"restored": true`) {
		t.Fatalf("mem snapshot restore should report restored:true, got:\n%s", out)
	}

	// Verify the restore: only the baseline observation remains (the one added
	// after the snapshot is gone).
	st3, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("reopen 2: %v", err)
	}
	var n int
	if err := st3.DB.QueryRow(`
		SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`,
		project).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	st3.Close()
	if n != 1 {
		t.Fatalf("after restore, live observation count=%d want 1 (the baseline)", n)
	}
}

// extractSnapshotID pulls the highest snapshot id out of a `mem snapshot list`
// JSON output (the newest snapshot = the one the test just created).
func extractSnapshotID(t *testing.T, out string) string {
	t.Helper()
	// The list output is newest-first; the first "id" value is the snapshot we
	// created. Extract the integer after the first `"id":`.
	i := strings.Index(out, `"id":`)
	if i < 0 {
		t.Fatalf("no snapshot id in list output:\n%s", out)
	}
	rest := out[i+len(`"id":`):]
	j := 0
	for j < len(rest) && (rest[j] == ' ' || (rest[j] >= '0' && rest[j] <= '9')) {
		j++
	}
	id := strings.TrimSpace(rest[:j])
	if id == "" {
		t.Fatalf("empty snapshot id in list output:\n%s", out)
	}
	return id
}

// TestMemSnapshotCLIBadArgs proves the CLI rejects bad snapshot args clearly.
func TestMemSnapshotCLIBadArgs(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-snapshot-badargs"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()

	// mem snapshot restore with no id → clear error (exit 2).
	out, err := runMemCLIExpectError(t, dataDir, "snapshot", "restore", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem snapshot restore with no id should fail, got: %s", out)
	}
	if !strings.Contains(out, "requires a snapshot id") {
		t.Fatalf("bad restore args should be rejected clearly, got: %s", out)
	}

	// mem snapshot restore with a non-numeric id → clear error.
	out, err = runMemCLIExpectError(t, dataDir, "snapshot", "restore", "notanumber", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem snapshot restore with a bad id should fail, got: %s", out)
	}
	if !strings.Contains(out, "invalid snapshot id") {
		t.Fatalf("bad restore id should be rejected clearly, got: %s", out)
	}
}
