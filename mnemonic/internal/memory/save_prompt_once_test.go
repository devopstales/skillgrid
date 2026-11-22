package memory

// save_prompt_once_test.go — SavePromptOnce dedup window boundary: a
// stored prompt inside the window dedups; an identical prompt older than
// the window stores. The CLI tests cover the in-window re-fire; this pins
// the window edge.

import (
	"context"
	"testing"
)

func TestSavePromptOnceWindowBoundary(t *testing.T) {
	st := openTestStore(t)
	svc := newTestServiceWithStore(t, st)
	ctx := context.Background()
	id, err := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", "")
	if err != nil || id == 0 {
		t.Fatalf("first store: id=%d err=%v", id, err)
	}
	if id, _ := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", ""); id != 0 {
		t.Fatalf("in-window re-fire must dedup, got id %d", id)
	}
	// age the row past the window (the test-only raw seam).
	raw, ok := st.(trailSeam)
	if !ok {
		t.Fatal("store lacks the RawExec test seam")
	}
	if err := raw.RawExec(`UPDATE user_prompts SET created_at = datetime('now', ?)`,
		"-11 minutes"); err != nil {
		t.Fatalf("age row: %v", err)
	}
	if id, _ := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", ""); id == 0 {
		t.Fatalf("an identical prompt past the window is a new capture")
	}
}
