package codeindex

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAutoReopen covers 03.3 (Scenario: Reader auto-reopens the new index):
// after a publication, a running process opens the newly-published index on its
// next tool call (within the injected ~5s window) with no restart. The reopen
// window is injectable so the test does not actually wait 5 seconds.
func TestAutoReopen(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "index.sqlite")
	if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
		t.Fatalf("seed live: %v", err)
	}

	var reopened int
	// A very short reopen window (injected) so the test is fast: the process
	// reopens once the published index is newer than its open AND the window
	// (10ms) has elapsed.
	tr := NewReopenTracker(live, 10*time.Millisecond)
	tr.OnReopen = func(_ string) { reopened++ }

	// Immediately after opening, the index is not newer → no reopen.
	if tr.Check() {
		t.Fatalf("Check should not reopen before a new index is published")
	}

	// Simulate the agent's own edit being published: the live index is now
	// newer than the tracker's open time.
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(live, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	// Let the reopen window elapse (injected 10ms).
	time.Sleep(15 * time.Millisecond)

	if !tr.Check() {
		t.Fatalf("expected the running process to auto-reopen the new index")
	}
	if reopened != 1 {
		t.Errorf("expected exactly 1 reopen, got %d", reopened)
	}

	// A second Check right after the reopen must NOT reopen again (the open
	// time was re-recorded; no newer index since).
	if tr.Check() {
		t.Fatalf("Check reopened again immediately after a reopen (no new publish)")
	}
}
