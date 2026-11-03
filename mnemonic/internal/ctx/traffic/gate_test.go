package traffic

import (
	"strings"
	"testing"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// openRawStore opens a real migrated shared project store (the underlying
// *store.Store, not the ctx accessor) so a test can poke the schema directly
// (e.g. to simulate a pre-068 store without the status column).
func openRawStore(t *testing.T, project string) (*store.Store, error) {
	t.Helper()
	return store.Open(t.TempDir(), project)
}

// seedCandidate stores one candidate traffic memory row for the given session
// and returns the stored row (so tests can assert against the live id).
func seedCandidate(t *testing.T, cs *ctxstore.Store, id, session, text string) *ctxstore.TrafficMemory {
	t.Helper()
	m := &ctxstore.TrafficMemory{
		MemoryID:   id,
		SessionID:  session,
		Category:   "preference",
		MemoryText: text,
		Confidence: 0.9,
		CreatedAt:  1000,
		Status:     "candidate",
	}
	if err := cs.PutTrafficMemory(m); err != nil {
		t.Fatalf("PutTrafficMemory %s: %v", id, err)
	}
	return m
}

// TestGate_ApplyPromotesAndIsActive is the KEY proof of the human-gate: a
// candidate is NOT in ActiveMemories before apply; after Gate.Apply it IS, and
// applied_at is stamped.
func TestGate_ApplyPromotesAndIsActive(t *testing.T) {
	cs := newTestStore(t, "p-gate-apply")
	g := NewGate(cs)
	seedCandidate(t, cs, "mem-1", "sess-gate", "prefer compact diffs")

	// Before apply: the candidate is not active.
	active, err := g.ActiveMemories("sess-gate")
	if err != nil {
		t.Fatalf("ActiveMemories (before apply): %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("ActiveMemories before apply = %d, want 0 (candidate is not active)", len(active))
	}

	// Apply promotes it.
	if err := g.Apply("mem-1"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	after, err := g.ActiveMemories("sess-gate")
	if err != nil {
		t.Fatalf("ActiveMemories (after apply): %v", err)
	}
	if len(after) != 1 || after[0].MemoryID != "mem-1" {
		t.Fatalf("ActiveMemories after apply = %+v, want exactly mem-1", after)
	}
	// applied_at must be stamped by Apply.
	if after[0].AppliedAt == nil {
		t.Errorf("applied_at = nil after Apply, want a stamped timestamp")
	}
	if after[0].Status != "applied" {
		t.Errorf("status = %q after Apply, want applied", after[0].Status)
	}
}

// TestGate_RevokeRevertsAndDisappears proves the demotion half: after an apply,
// Gate.Revoke removes the memory from ActiveMemories again (status=revoked,
// applied_at cleared).
func TestGate_RevokeRevertsAndDisappears(t *testing.T) {
	cs := newTestStore(t, "p-gate-revoke")
	g := NewGate(cs)
	seedCandidate(t, cs, "mem-1", "sess-gate", "prefer compact diffs")

	if err := g.Apply("mem-1"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if active, _ := g.ActiveMemories("sess-gate"); len(active) != 1 {
		t.Fatalf("expected 1 active after Apply, got %d", len(active))
	}

	if err := g.Revoke("mem-1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	active, err := g.ActiveMemories("sess-gate")
	if err != nil {
		t.Fatalf("ActiveMemories (after revoke): %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("ActiveMemories after revoke = %d, want 0 (revoked is not active)", len(active))
	}
	// The row still exists but is now revoked (revoked rows are not deleted).
	m, err := cs.GetTrafficMemory("mem-1")
	if err != nil {
		t.Fatalf("GetTrafficMemory: %v", err)
	}
	if m == nil {
		t.Fatal("row disappeared after revoke; revoke must keep the row (status=revoked)")
	}
	if m.Status != "revoked" {
		t.Errorf("status = %q after Revoke, want revoked", m.Status)
	}
	if m.AppliedAt != nil {
		t.Errorf("applied_at = %v after Revoke, want nil (cleared)", m.AppliedAt)
	}
}

// TestGate_NotFoundRejected proves the human-gate: a non-existent id is a clear
// "not found" error for both Apply and Revoke, and no row is created.
func TestGate_NotFoundRejected(t *testing.T) {
	cs := newTestStore(t, "p-gate-notfound")
	g := NewGate(cs)

	if err := g.Apply("nope"); err == nil {
		t.Fatal("Apply(nope) = nil, want a not-found error")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Apply(nope) error = %q, want it to mention not found", err)
	}
	if err := g.Revoke("nope"); err == nil {
		t.Fatal("Revoke(nope) = nil, want a not-found error")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Revoke(nope) error = %q, want it to mention not found", err)
	}
	// No row may have been created by the failed apply/revoke.
	m, err := cs.GetTrafficMemory("nope")
	if err != nil {
		t.Fatalf("GetTrafficMemory(nope): %v", err)
	}
	if m != nil {
		t.Errorf("row %q was created by a not-found apply/revoke", m.MemoryID)
	}
}

// TestGate_Idempotency proves re-applying and re-revoking are safe no-ops:
// Apply twice leaves exactly one active row; Revoke twice does not error.
func TestGate_Idempotency(t *testing.T) {
	cs := newTestStore(t, "p-gate-idem")
	g := NewGate(cs)
	seedCandidate(t, cs, "mem-1", "sess-gate", "prefer compact diffs")

	if err := g.Apply("mem-1"); err != nil {
		t.Fatalf("Apply #1: %v", err)
	}
	if err := g.Apply("mem-1"); err != nil {
		t.Fatalf("Apply #2 (idempotent): %v", err)
	}
	if active, _ := g.ActiveMemories("sess-gate"); len(active) != 1 {
		t.Fatalf("expected 1 active row after double Apply, got %d", len(active))
	}

	if err := g.Revoke("mem-1"); err != nil {
		t.Fatalf("Revoke #1: %v", err)
	}
	if err := g.Revoke("mem-1"); err != nil {
		t.Fatalf("Revoke #2 (idempotent): %v", err)
	}
	if active, _ := g.ActiveMemories("sess-gate"); len(active) != 0 {
		t.Fatalf("expected 0 active rows after double Revoke, got %d", len(active))
	}
}

// TestGate_NilStoreFailOpen proves ADR-0016: a Gate with no store behind it
// degrades every method to a no-op — no error, no panic. Apply/Revoke return
// nil (fail-open), ActiveMemories returns empty.
func TestGate_NilStoreFailOpen(t *testing.T) {
	g := NewGate(nil)
	if err := g.Apply("mem-1"); err != nil {
		t.Errorf("Apply with nil store = %v, want nil (fail-open)", err)
	}
	if err := g.Revoke("mem-1"); err != nil {
		t.Errorf("Revoke with nil store = %v, want nil (fail-open)", err)
	}
	active, err := g.ActiveMemories("sess")
	if err != nil {
		t.Errorf("ActiveMemories with nil store = %v, want nil (fail-open)", err)
	}
	if len(active) != 0 {
		t.Errorf("ActiveMemories with nil store = %d rows, want 0", len(active))
	}
}

// TestGate_Pre068SchemaFailOpen proves ADR-068 fail-open on a schema that
// lacks the status column: a row reads as candidate, so ActiveMemories is
// empty, and Apply is a no-op that does not error (the write is lost, not fatal).
func TestGate_Pre068SchemaFailOpen(t *testing.T) {
	shared, err := openRawStore(t, "p-gate-pre068")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	// Drop the 068 columns to simulate a store migrated before 068 ran.
	if _, err := shared.DB.Exec(`ALTER TABLE ctx_traffic_memories DROP COLUMN applied_at`); err != nil {
		t.Fatalf("drop applied_at: %v", err)
	}
	if _, err := shared.DB.Exec(`ALTER TABLE ctx_traffic_memories DROP COLUMN status`); err != nil {
		t.Fatalf("drop status: %v", err)
	}
	cs := ctxstore.New(shared)
	g := NewGate(cs)
	seedCandidate(t, cs, "mem-1", "sess-gate", "prefer compact diffs")

	// Pre-068: the row reads as candidate, so it is not active.
	active, err := g.ActiveMemories("sess-gate")
	if err != nil {
		t.Fatalf("ActiveMemories (pre-068): %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("ActiveMemories (pre-068) = %d rows, want 0 (candidate default)", len(active))
	}
	// Apply must be a no-op on a schema that cannot persist status (no error).
	if err := g.Apply("mem-1"); err != nil {
		t.Errorf("Apply (pre-068) = %v, want nil (fail-open no-op)", err)
	}
}
