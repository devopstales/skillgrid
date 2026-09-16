package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestServiceFacadePathOpensOnce is the step-04 double-open regression guard.
// It exercises a single-project facade entry point that still exists on the
// Service (the public Open seam, the production path used by MCP/HTTP) and
// asserts the project store opens exactly once for that logical op. Before the
// wrapper collapse, any surviving open-delegate-close wrapper that opened the
// project a second time would push the counter to 2 and fail; after the
// collapse it pins the invariant so a future double-open fails.
func TestServiceFacadePathOpensOnce(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	st, err := store.Open(dataDir, "doubleopen-guard")
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	st.Close()

	store.ResetOpenCount()
	_, cleanup, err := svc.Open("doubleopen-guard")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	cleanup()
	if got := store.OpenCount(); got != 1 {
		t.Errorf("svc.Open opened the store %d times, want exactly 1 (double-open)", got)
	}
}

// TestMnemonicCommitOpensOnce proves the MnemonicCommit facade (a KEEP method
// with a live MCP production caller, which internally opens the project) opens
// the store exactly once for the logical op. MnemonicCommit fires an async
// tiering hook that reopens the store in a background goroutine, so the
// counter is snapshotted immediately after the (synchronous) commit returns,
// before the hook goroutine can start, and the wait group is then awaited
// before the test exits (the hook's extra Open is the async tiering path, not
// a double-open of the logical op).
func TestMnemonicCommitOpensOnce(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	st, err := store.Open(dataDir, "commit-open-once")
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	st.Close()

	var wg sync.WaitGroup
	store.ResetOpenCount()
	_, err = svc.MnemonicCommit(context.Background(), "commit-open-once", MnemonicCommitInput{
		Title:          "open once probe",
		LessonsLearned: "MnemonicCommit must open the project store exactly once",
	}, &wg)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	// Snapshot immediately: the commit path itself opened exactly once; the
	// tiering hook goroutine may not have run yet.
	count := store.OpenCount()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tiering hook timed out")
	}

	if count != 1 {
		t.Errorf("MnemonicCommit opened the store %d times, want exactly 1 (double-open)", count)
	}
}
