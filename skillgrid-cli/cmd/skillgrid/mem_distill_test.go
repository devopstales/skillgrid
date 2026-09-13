package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestDistillStatusCLI covers 17.4: `mem distill status` shows the per-project
// distillation lock state. With a lock held it prints the lock with its
// project_id, locked_at timestamp, and locked_by; with no lock it prints
// "no active locks". The status is a read-only diagnostic — it never acquires
// or releases a lock.
func TestDistillStatusCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-distill-status"

	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// No lock yet → `mem distill status` reports no active locks.
	out := runMemCLI(t, dataDir, "distill", "status", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "no active locks") {
		t.Fatalf("mem distill status with no lock should report no active locks, got:\n%s", out)
	}

	// Acquire a distill lock for the project (the state the CLI should show).
	locks := memory.NewDistillLockService(st.DB)
	if err := locks.Acquire(context.Background(), project, "status-worker"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	st.Close()

	// `mem distill status` now shows the lock held: project_id, locked_at, and
	// locked_by are all present.
	out = runMemCLI(t, dataDir, "distill", "status", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, project) {
		t.Fatalf("mem distill status should show the project id, got:\n%s", out)
	}
	if !strings.Contains(out, "status-worker") {
		t.Fatalf("mem distill status should show locked_by, got:\n%s", out)
	}
	for _, label := range []string{"project_id", "locked_at", "locked_by"} {
		if !strings.Contains(out, label) {
			t.Fatalf("mem distill status missing field %q, got:\n%s", label, out)
		}
	}

	// Release the lock → the CLI reports no active locks again.
	st2, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	locks2 := memory.NewDistillLockService(st2.DB)
	if err := locks2.Release(context.Background(), project); err != nil {
		t.Fatalf("release: %v", err)
	}
	st2.Close()

	out = runMemCLI(t, dataDir, "distill", "status", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "no active locks") {
		t.Fatalf("after release, mem distill status should report no active locks, got:\n%s", out)
	}
}

// TestDistillStatusCLITimestamp guards that the shown locked_at is a real
// RFC3339 timestamp (not empty or a placeholder).
func TestDistillStatusCLITimestamp(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-distill-ts"

	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	locks := memory.NewDistillLockService(st.DB)
	if err := locks.Acquire(context.Background(), project, "ts-worker"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	st.Close()

	out := runMemCLI(t, dataDir, "distill", "status", "--project", project, "--dir", dataDir)
	// The locked_at value must be parseable as RFC3339 (a real timestamp).
	var parsed struct {
		Status struct {
			LockedAt string `json:"locked_at"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("mem distill status is not valid JSON: %v\n%s", err, out)
	}
	if parsed.Status.LockedAt == "" {
		t.Fatalf("locked_at is empty, got:\n%s", out)
	}
	if _, err := time.Parse(time.RFC3339, parsed.Status.LockedAt); err != nil {
		t.Fatalf("locked_at %q is not RFC3339: %v", parsed.Status.LockedAt, err)
	}
}
