package traffic

import (
	"testing"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// storeTrafficMemory is the test-side shorthand for a row to seed the store.
type storeTrafficMemory struct {
	id   string
	conf float64
	at   int64
}

func toTrafficMemory(m *storeTrafficMemory) *ctxstore.TrafficMemory {
	return &ctxstore.TrafficMemory{
		MemoryID:   m.id,
		SessionID:  "s",
		Category:   "environment",
		Confidence: m.conf,
		CreatedAt:  m.at,
	}
}

func TestEvictToBound_UnderBound(t *testing.T) {
	cs := newTestStore(t, "p-ev-under")
	mems := []*storeTrafficMemory{
		{id: "m1", conf: 0.5, at: 10},
		{id: "m2", conf: 0.6, at: 20},
	}
	for _, m := range mems {
		if err := cs.PutTrafficMemory(toTrafficMemory(m)); err != nil {
			t.Fatalf("PutTrafficMemory %s: %v", m.id, err)
		}
	}
	n, err := EvictToBound(cs, 10)
	if err != nil {
		t.Fatalf("EvictToBound: %v", err)
	}
	if n != 0 {
		t.Errorf("evicted = %d, want 0 (2 rows <= bound 10)", n)
	}
}

func TestEvictToBound_EvictsLowConfidenceFirst(t *testing.T) {
	cs := newTestStore(t, "p-ev-low")
	// 4 rows, bound 2 → 2 evicted. Lowest confidence goes first.
	mems := []*storeTrafficMemory{
		{id: "m-high", conf: 0.9, at: 10},
		{id: "m-mid", conf: 0.5, at: 20},
		{id: "m-low", conf: 0.1, at: 30},
		{id: "m-mid2", conf: 0.5, at: 40},
	}
	for _, m := range mems {
		if err := cs.PutTrafficMemory(toTrafficMemory(m)); err != nil {
			t.Fatalf("PutTrafficMemory %s: %v", m.id, err)
		}
	}
	n, err := EvictToBound(cs, 2)
	if err != nil {
		t.Fatalf("EvictToBound: %v", err)
	}
	if n != 2 {
		t.Fatalf("evicted = %d, want 2", n)
	}
	// Eviction order is confidence ASC, then created_at ASC: m-low (0.1) goes
	// first, then m-mid (0.5, created_at 20) before m-mid2 (0.5, created_at 40).
	// Survivors must be exactly the two highest: m-high and m-mid2.
	rows, err := cs.ListAllTrafficMemories()
	if err != nil {
		t.Fatalf("ListAllTrafficMemories: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range rows {
		ids[r.MemoryID] = true
	}
	if len(rows) != 2 {
		t.Fatalf("survivor count = %d, want 2", len(rows))
	}
	if !ids["m-high"] || !ids["m-mid2"] {
		t.Errorf("survivors = %v, want exactly {m-high, m-mid2}", ids)
	}
	if ids["m-low"] || ids["m-mid"] {
		t.Errorf("low-confidence/older rows survived: %v; m-low (0.1) and m-mid (0.5, older) must be evicted first", ids)
	}
}

func TestEvictToBound_ExactlyAtBound(t *testing.T) {
	cs := newTestStore(t, "p-ev-exact")
	for i, id := range []string{"a", "b", "c"} {
		_ = i
		if err := cs.PutTrafficMemory(toTrafficMemory(&storeTrafficMemory{id: id, conf: 0.5, at: int64(i + 1)})); err != nil {
			t.Fatalf("PutTrafficMemory %s: %v", id, err)
		}
	}
	n, err := EvictToBound(cs, 3)
	if err != nil {
		t.Fatalf("EvictToBound: %v", err)
	}
	if n != 0 {
		t.Errorf("evicted = %d, want 0 (3 rows == bound 3)", n)
	}
}

func TestEvictToBound_NilStore(t *testing.T) {
	var n int
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("EvictToBound(nil) panicked: %v", r)
			}
		}()
		n, err = EvictToBound(nil, 5)
	}()
	if err != nil {
		t.Fatalf("EvictToBound(nil): %v", err)
	}
	if n != 0 {
		t.Errorf("evicted = %d, want 0 (nil store is a fail-open no-op)", n)
	}
}

// TestEvictToBound_StoreErrorFailsOpen: a store fault degrades to (0, nil) —
// eviction is a maintenance path, never a hard failure (ADR-0016). We assert the
// no-panic, no-error contract via the nil-store branch; a live store error hits
// the same early-return path in EvictToBound.
func TestEvictToBound_StoreErrorFailsOpen(t *testing.T) {
	cs := newTestStore(t, "p-ev-err")
	// Close the store first so any underlying fault degrades, not hard-fails.
	if err := cs.Shared().Close(); err != nil {
		// A double-close or already-closed store is acceptable here; the point
		// is that EvictToBound must not panic on a degraded handle.
		t.Logf("pre-close: %v", err)
	}
	var n int
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("EvictToBound on a closed store panicked: %v", r)
			}
		}()
		n, err = EvictToBound(cs, 1)
	}()
	// On a closed db the query fails; EvictToBound must return (0, nil).
	if err != nil {
		t.Fatalf("EvictToBound on closed store: want nil error (fail-open), got %v", err)
	}
	if n != 0 {
		t.Errorf("evicted = %d, want 0 (store fault is a fail-open no-op)", n)
	}
}

func TestEvictToBound_NonPositiveMax(t *testing.T) {
	cs := newTestStore(t, "p-ev-max")
	if err := cs.PutTrafficMemory(toTrafficMemory(&storeTrafficMemory{id: "x", conf: 0.5, at: 1})); err != nil {
		t.Fatalf("PutTrafficMemory: %v", err)
	}
	for _, max := range []int{0, -1} {
		n, err := EvictToBound(cs, max)
		if err != nil {
			t.Fatalf("EvictToBound(max=%d): %v", max, err)
		}
		if n != 0 {
			t.Errorf("max=%d: evicted = %d, want 0 (non-positive max is a no-op)", max, n)
		}
	}
}
