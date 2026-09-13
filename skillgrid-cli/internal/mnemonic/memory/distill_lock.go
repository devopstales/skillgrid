package memory

import (
	"context"
	"database/sql"
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
	inner *DreamLockService
}

// NewDistillLockService builds a distillation lock bound to the store's
// *sql.DB. The DB must already have migration 024 applied (distill_lock
// exists). It delegates to NewDreamLockService so both constructors share the
// same clock seam default and the same row schema.
func NewDistillLockService(db *sql.DB) *DistillLockService {
	return &DistillLockService{inner: NewDreamLockService(db)}
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
