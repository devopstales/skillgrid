package memory

import (
	"context"
	"testing"
	"time"
)

// TestDistillLockTimeoutAutoRelease (014 step 17.3): a distill lock has a
// 5-minute timeout. A lock 6 minutes old is stale — the next Acquire
// auto-releases it and takes over. A lock only 1 minute old is fresh — the next
// Acquire fails. This guards the dreamLockTTL boundary through the distill name.
func TestDistillLockTimeoutAutoRelease(t *testing.T) {
	_, svc := newTestStore(t, "distilltimeoutproj")
	ctx := context.Background()
	db := svc.DB()
	locks := NewDistillLockService(db)

	// Acquire a lock, then backdate it to 6 minutes ago (simulate staleness).
	if err := locks.Acquire(ctx, "A", "holder-1"); err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	stale := time.Now().UTC().Add(-6 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE distill_lock SET locked_at = ? WHERE project_id = ?`, stale, "A"); err != nil {
		t.Fatalf("backdate lock: %v", err)
	}

	// The stale lock is auto-released: a new acquisition succeeds and takes
	// over (locked_by is now holder-2).
	if err := locks.Acquire(ctx, "A", "holder-2"); err != nil {
		t.Fatalf("acquire over stale lock should succeed (auto-release): %v", err)
	}
	var by string
	if err := db.QueryRow(`SELECT locked_by FROM distill_lock WHERE project_id = ?`, "A").Scan(&by); err != nil {
		t.Fatalf("read locked_by: %v", err)
	}
	if by != "holder-2" {
		t.Fatalf("stale lock not auto-released: locked_by=%q want holder-2", by)
	}

	// A lock that is only 1 minute old is NOT released: a new acquisition
	// fails (the lock is still held and fresh).
	fresh := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE distill_lock SET locked_at = ? WHERE project_id = ?`, fresh, "A"); err != nil {
		t.Fatalf("set fresh lock: %v", err)
	}
	if err := locks.Acquire(ctx, "A", "holder-3"); err == nil {
		t.Fatalf("1-minute-old lock should still be held (not auto-released)")
	}
	// The original holder is unchanged (the failed acquire left it alone).
	if err := db.QueryRow(`SELECT locked_by FROM distill_lock WHERE project_id = ?`, "A").Scan(&by); err != nil {
		t.Fatalf("read locked_by after failed acquire: %v", err)
	}
	if by != "holder-2" {
		t.Fatalf("failed acquire disturbed the held lock: locked_by=%q want holder-2", by)
	}
}
