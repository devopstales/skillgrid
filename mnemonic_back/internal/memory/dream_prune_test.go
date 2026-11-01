package memory

import (
	"context"
	"testing"
	"time"
)

// TestDreamPrune (014 step 12.3): 10 observations with varying importance —
// only the low-importance ones are soft-deleted; high and medium survive.
// Maturity tier is expressed through the durable-type weight (no stored column).
func TestDreamPrune(t *testing.T) {
	_, svc := newTestStore(t, "prune")
	sid := newSession(t, svc)
	ctx := context.Background()
	de := NewDreamExecutor(svc)
	// Pin the clock so recency is deterministic (all created "now" → recency 1).
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	de.now = func() time.Time { return fixed }

	typeSpec := []struct {
		usage  int
		typ    string
		expect bool // expect pruned (true) or kept (false)
	}{
		{0, "learning", true},   // 0.38 low
		{0, "lesson", true},     // 0.38 low
		{0, "bugfix", true},     // 0.38 low
		{10, "learning", false}, // 0.58 medium
		{25, "decision", false}, // 0.85 high
		{50, "decision", false}, // 1.0  high
		{0, "architecture", false}, // 0.54 medium (durable)
		{15, "config", false},      // 0.72  (durable)
		{30, "architecture", false}, // 0.92
		{5, "lesson", false},        // 0.48 (just above threshold)
	}
	ids := make([]int64, len(typeSpec))
	for i, sp := range typeSpec {
		id, err := svc.Save(ctx, SaveInput{
			Title:     "obs " + sp.typ,
			Type:      sp.typ,
			Content:   "content " + sp.typ,
			Scope:     "project",
			SessionID: sid,
		})
		if err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
		// Backdate created_at to the pinned clock so recency is deterministic
		// (all rows are "now" → recency 1). retrieval_usage is set explicitly.
		created := fixed.UTC().Format(time.RFC3339)
		svc.DB().Exec(`UPDATE observations SET retrieval_usage = ?, created_at = ? WHERE id = ?`, sp.usage, created, id)
		ids[i] = id
	}

	res, err := de.prune(ctx, 0.5)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	prunedSet := make(map[int64]bool, len(res.IDs))
	for _, id := range res.IDs {
		prunedSet[id] = true
	}
	for i, sp := range typeSpec {
		got := prunedSet[ids[i]]
		if got != sp.expect {
			t.Errorf("obs %d (usage=%d type=%s): pruned=%v, want %v", i, sp.usage, sp.typ, got, sp.expect)
		}
	}
	if res.Pruned != 3 {
		t.Fatalf("pruned = %d, want 3", res.Pruned)
	}
	for i, sp := range typeSpec {
		if _, err := svc.Get(ctx, ids[i]); sp.expect {
			if err == nil {
				t.Errorf("obs %d should be soft-deleted (Get succeeded)", i)
			}
		} else if err != nil {
			t.Errorf("obs %d should be kept (Get failed: %v)", i, err)
		}
	}
}

// TestDreamImportanceMonotonic guards the prune scoring: retrieval usage and
// durability raise the score, age lowers it. (Supports 12.3.)
func TestDreamImportanceMonotonic(t *testing.T) {
	de := NewDreamExecutor(nil)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	de.now = func() time.Time { return now }
	fresh := now.UTC().Format(time.RFC3339)
	old := now.Add(-60 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// A fresh, never-retrieved, non-durable observation scores 0.38 (below the
	// 0.5 prune threshold). A hot durable decision scores 1.0. An old,
	// never-retrieved lesson scores ~0.08 (age decays recency to 0).
	if got := de.importance("learning", fresh, 0); got >= 0.5 {
		t.Fatalf("fresh learning should be below threshold, got %f", got)
	}
	if got := de.importance("decision", fresh, 50); got < 0.9 {
		t.Fatalf("hot decision should be high, got %f", got)
	}
	if got := de.importance("lesson", old, 0); got > 0.2 {
		t.Fatalf("old lesson should be low, got %f", got)
	}
	// Retrieval usage must be monotonic non-decreasing.
	if de.importance("learning", fresh, 30) < de.importance("learning", fresh, 10) {
		t.Fatal("importance not monotonic in usage")
	}
	// Durability must raise the score at equal usage/age.
	if de.importance("decision", fresh, 0) <= de.importance("learning", fresh, 0) {
		t.Fatal("durable type should score >= non-durable at equal usage/age")
	}
}
