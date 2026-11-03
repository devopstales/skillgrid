package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// DistillLockService is the per-project distillation lock exposed under its
// distillation name (014 step 17). It is a thin alias over step 12's
// DreamLockService: the same distill_lock table (migration 024), the same
// 5-minute TTL, and the same atomic acquire — so the two names are the same
// lock and a dream and a distill can never run concurrently on one project.
// Wrapping (rather than duplicating) keeps a single source of truth for the
// row-per-project semantics, the TTL boundary, and the auto-release on
// staleness.
type DistillLockService struct {
	db    *sql.DB
	inner *DreamLockService
}

// NewDistillLockService builds a distillation lock bound to the store's
// *sql.DB. The DB must already have migration 024 applied (distill_lock
// exists). It delegates to NewDreamLockService so both constructors share the
// same clock seam default and the same row schema.
func NewDistillLockService(db *sql.DB) *DistillLockService {
	return &DistillLockService{db: db, inner: NewDreamLockService(db)}
}

// Acquire takes the distillation lock for projectID, owned by holder. It
// succeeds when the project is not locked or its lock is stale (auto-released
// after the 5-minute TTL); it fails when a fresh lock is held. One row per
// project is created on first acquire (17.1).
func (l *DistillLockService) Acquire(ctx context.Context, projectID, holder string) error {
	if l == nil || l.inner == nil {
		return errDistillLockUninit
	}
	return l.inner.Acquire(ctx, projectID, holder)
}

// Release removes the distillation lock for projectID. Idempotent: releasing a
// project that is not locked is a no-op (17.1: the row is removed).
func (l *DistillLockService) Release(ctx context.Context, projectID string) error {
	if l == nil || l.inner == nil {
		return errDistillLockUninit
	}
	return l.inner.Release(ctx, projectID)
}

// IsLocked reports whether projectID holds a FRESH (non-stale) distillation
// lock, delegating to the dream lock's freshness check (stale = free).
func (l *DistillLockService) IsLocked(ctx context.Context, projectID string) (bool, error) {
	if l == nil || l.inner == nil {
		return false, errDistillLockUninit
	}
	return l.inner.IsLocked(ctx, projectID)
}

// AsDreamLock exposes the underlying *DreamLockService. Step 12's DreamRollback
// (dream_rollback.go) releases the lock through this type, so a distill rollback
// hands it the same *DreamLockService its Acquire/Release delegate to — the
// DistillLockService and DreamLockService are the same lock (same rows, same
// TTL), so the release lands on the row Acquire created.
func (l *DistillLockService) AsDreamLock() *DreamLockService {
	if l == nil {
		return nil
	}
	return l.inner
}

// DistillLockStatus is one row of the `mem distill status` report: a project's
// lock state as of the read (014 step 17.4). Held is true when a FRESH (non-
// stale) lock is present; LockedAt/LockedBy are the raw row values (present when
// a row exists, whether fresh or stale — a stale lock is still visible, just not
// "held").
type DistillLockStatus struct {
	ProjectID string `json:"project_id"`
	LockedAt  string `json:"locked_at,omitempty"`
	LockedBy  string `json:"locked_by,omitempty"`
	Held      bool   `json:"held"`
}

// DistillLockStatus reads the project's distillation lock row (014 step 17.4).
// It is a read-only diagnostic for `mem distill status`: it reports the lock's
// project_id, locked_at, and locked_by, and whether the lock is currently HELD
// (fresh, not past the 5-minute TTL). It never acquires or releases a lock.
func (l *DistillLockService) DistillLockStatus(ctx context.Context, projectID string) (DistillLockStatus, error) {
	if l == nil || l.db == nil {
		return DistillLockStatus{}, errDistillLockUninit
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return DistillLockStatus{}, fmt.Errorf("project_id is required")
	}
	var lockedAt, lockedBy string
	err := l.db.QueryRowContext(ctx, `
		SELECT locked_at, locked_by FROM distill_lock WHERE project_id = ?`,
		projectID).Scan(&lockedAt, &lockedBy)
	if err == sql.ErrNoRows {
		return DistillLockStatus{ProjectID: projectID, Held: false}, nil
	}
	if err != nil {
		return DistillLockStatus{}, fmt.Errorf("distill lock status %q: %w", projectID, err)
	}
	held, err := l.inner.IsLocked(ctx, projectID)
	if err != nil {
		return DistillLockStatus{}, err
	}
	return DistillLockStatus{ProjectID: projectID, LockedAt: lockedAt, LockedBy: lockedBy, Held: held}, nil
}

var errDistillLockUninit = newDistillLockUninitError()

// newDistillLockUninitError returns the sentinel for a not-initialized distill
// lock service (mirrors the "not initialized" wording of the dream lock).
func newDistillLockUninitError() error {
	return &distillLockUninitError{}
}

type distillLockUninitError struct{}

func (e *distillLockUninitError) Error() string {
	return "distill lock service not initialized"
}
