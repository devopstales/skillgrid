package memory

import (
	"context"
	"testing"
)

// TestDistillLockRowPerProject (014 step 17.1): the DistillLockService keeps one
// distill_lock row per project. Acquire(A) creates A's row; Acquire(B) creates
// a SEPARATE row (the lock is per-project, not global); Acquire(A) again
// detects the existing fresh lock and fails; Release(A) removes A's row (and
// only A's).
func TestDistillLockRowPerProject(t *testing.T) {
	_, svc := newTestStore(t, "distilllockproj")
	ctx := context.Background()
	db := svc.DB()
	locks := NewDistillLockService(db)

	countRow := func(projectID string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM distill_lock WHERE project_id = ?`, projectID).Scan(&n); err != nil {
			t.Fatalf("count row %s: %v", projectID, err)
		}
		return n
	}
	countTotal := func() int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM distill_lock`).Scan(&n); err != nil {
			t.Fatalf("count total: %v", err)
		}
		return n
	}

	// 1. Acquire A → creates a row for A.
	if err := locks.Acquire(ctx, "A", "w1"); err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	if got := countRow("A"); got != 1 {
		t.Fatalf("acquire A created %d rows, want 1", got)
	}

	// 2. Acquire B → creates a SEPARATE row for B (A's row is untouched).
	if err := locks.Acquire(ctx, "B", "w2"); err != nil {
		t.Fatalf("acquire B: %v", err)
	}
	if got := countRow("B"); got != 1 {
		t.Fatalf("acquire B created %d rows, want 1", got)
	}
	if got := countRow("A"); got != 1 {
		t.Fatalf("acquire B disturbed A's row (A now has %d)", got)
	}
	if got := countTotal(); got != 2 {
		t.Fatalf("two projects locked, want 2 total rows, got %d", got)
	}

	// 3. Acquire A again → detects the existing fresh lock (fails).
	if err := locks.Acquire(ctx, "A", "w3"); err == nil {
		t.Fatalf("second acquire A should fail while held and fresh")
	}
	if got := countRow("A"); got != 1 {
		t.Fatalf("failed acquire A should leave exactly 1 A row, got %d", got)
	}

	// 4. Release A → removes A's row; B's row is untouched.
	if err := locks.Release(ctx, "A"); err != nil {
		t.Fatalf("release A: %v", err)
	}
	if got := countRow("A"); got != 0 {
		t.Fatalf("release A left %d A rows, want 0", got)
	}
	if got := countRow("B"); got != 1 {
		t.Fatalf("release A disturbed B (B now has %d)", got)
	}
	if got := countTotal(); got != 1 {
		t.Fatalf("after release A, want 1 total row (B), got %d", got)
	}

	// Cleanup: release B so the store is not left locked.
	if err := locks.Release(ctx, "B"); err != nil {
		t.Fatalf("release B (cleanup): %v", err)
	}
}
