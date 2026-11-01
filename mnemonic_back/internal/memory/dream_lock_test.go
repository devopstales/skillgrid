package memory

import (
	"context"
	"testing"
	"time"
)

// TestDreamLockPreventsConcurrent (014 step 12.4): the per-project distillation
// lock. A held-and-fresh project cannot be acquired again; a different project
// succeeds; releasing frees the project for re-acquisition; a stale lock (older
// than the 5-minute TTL) is auto-released and taken over.
func TestDreamLockPreventsConcurrent(t *testing.T) {
	_, svc := newTestStore(t, "lockproj")
	ctx := context.Background()
	db := svc.DB()
	locks := NewDreamLockService(db)

	// 1. Acquire project A (worker-1) → succeeds.
	if err := locks.Acquire(ctx, "A", "worker-1"); err != nil {
		t.Fatalf("first acquire A: %v", err)
	}

	// 2. Acquire A again (worker-2, lock fresh) → fails (held, not stale).
	if err := locks.Acquire(ctx, "A", "worker-2"); err == nil {
		t.Fatalf("second acquire A should fail while held and fresh")
	}

	// 3. Acquire project B (worker-3) → succeeds (different project row).
	if err := locks.Acquire(ctx, "B", "worker-3"); err != nil {
		t.Fatalf("acquire B: %v", err)
	}

	// 4. Release A; re-acquire A (worker-4) → succeeds.
	if err := locks.Release(ctx, "A"); err != nil {
		t.Fatalf("release A: %v", err)
	}
	if err := locks.Acquire(ctx, "A", "worker-4"); err != nil {
		t.Fatalf("re-acquire A after release: %v", err)
	}

	// 5. Stale auto-release: backdate A's lock past the 5-minute TTL; a new
	//    holder can take it even though a (stale) row still exists.
	stale := time.Now().UTC().Add(-6 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE distill_lock SET locked_at = ? WHERE project_id = ?`, stale, "A"); err != nil {
		t.Fatalf("backdate lock: %v", err)
	}
	if err := locks.Acquire(ctx, "A", "worker-5"); err != nil {
		t.Fatalf("acquire A over stale lock: %v", err)
	}
	var by string
	if err := db.QueryRow(`SELECT locked_by FROM distill_lock WHERE project_id = ?`, "A").Scan(&by); err != nil {
		t.Fatalf("read locked_by: %v", err)
	}
	if by != "worker-5" {
		t.Fatalf("stale lock not replaced: locked_by=%q want worker-5", by)
	}

	// Cleanup: release both projects so the store is not left locked.
	for _, p := range []string{"A", "B"} {
		if err := locks.Release(ctx, p); err != nil {
			t.Fatalf("release %s (cleanup): %v", p, err)
		}
	}
}

// TestDreamLockTTLBoundary guards the 5-minute auto-release boundary: a lock
// 4 minutes old is still held (acquire fails); a lock 6 minutes old is stale
// (acquire succeeds).
func TestDreamLockTTLBoundary(t *testing.T) {
	_, svc := newTestStore(t, "lockttl")
	ctx := context.Background()
	db := svc.DB()
	locks := NewDreamLockService(db)

	fresh := time.Now().UTC().Add(-4 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO distill_lock (project_id, locked_at, locked_by) VALUES (?, ?, ?)`, "A", fresh, "old"); err != nil {
		t.Fatalf("seed fresh lock: %v", err)
	}
	if err := locks.Acquire(ctx, "A", "new"); err == nil {
		t.Fatalf("4-minute-old lock should still be held")
	}

	stale := time.Now().UTC().Add(-6 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE distill_lock SET locked_at = ? WHERE project_id = ?`, stale, "A"); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if err := locks.Acquire(ctx, "A", "new"); err != nil {
		t.Fatalf("6-minute-old lock should be auto-released: %v", err)
	}
}
