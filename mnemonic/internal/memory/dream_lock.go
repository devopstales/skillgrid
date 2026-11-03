package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// dreamLockTTL is the staleness window for a distillation lock (014 step 12.4).
// A lock older than this is treated as abandoned (the holder crashed or the
// process died mid-dream) and is auto-released by the next Acquire. Five
// minutes bounds how long a crashed dream can block its project.
const dreamLockTTL = 5 * time.Minute

// DreamLockService is the per-project distillation lock (014 step 12.4). It is
// backed by the distill_lock table (migration 024): one row per project,
// stamped with the moment it was taken and by whom. Two concurrent dreams on
// the same project would double-merge or double-prune, so the whole dream run
// holds this lock. It is a standalone service (not a *DreamExecutor method) so
// a lock can be acquired and held across the consolidate/synthesize/prune
// phases and released on success or rollback. The clock is a seam (now) so the
// TTL boundary is testable without waiting five real minutes.
type DreamLockService struct {
	db  *sql.DB
	now func() time.Time
}

// NewDreamLockService builds a lock service bound to the store's *sql.DB. The
// DB must already have migration 024 applied (distill_lock exists).
func NewDreamLockService(db *sql.DB) *DreamLockService {
	return &DreamLockService{db: db, now: time.Now}
}

// Acquire takes the distillation lock for projectID, owned by holder. It
// succeeds when the project is not locked, or when the existing lock is stale
// (older than dreamLockTTL — auto-released). It fails (returns an error) when
// the project is held by a fresh lock. A missing/nil DB is an error.
func (l *DreamLockService) Acquire(ctx context.Context, projectID, holder string) error {
	if l == nil || l.db == nil {
		return fmt.Errorf("dream lock service not initialized")
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	now := l.now().UTC()
	staleBefore := now.Add(-dreamLockTTL).UTC().Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)

	// Claim the row when it is absent OR stale. The WHERE clause makes the
	// acquire conditional: a fresh lock leaves the row untouched (rowcount 0),
	// an absent/stale row is written (rowcount 1).
	res, err := l.db.ExecContext(ctx, `
		INSERT INTO distill_lock (project_id, locked_at, locked_by)
		VALUES (?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE
			SET locked_at = excluded.locked_at, locked_by = excluded.locked_by
			WHERE distill_lock.locked_at < ?`,
		projectID, nowStr, holder, staleBefore)
	if err != nil {
		return fmt.Errorf("acquire lock %q: %w", projectID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("acquire lock %q: %w", projectID, err)
	}
	if n == 0 {
		return fmt.Errorf("project %q is already locked (fresh lock held)", projectID)
	}
	return nil
}

// Release removes the distillation lock for projectID. It is idempotent:
// releasing a project that is not locked is a no-op, not an error.
func (l *DreamLockService) Release(ctx context.Context, projectID string) error {
	if l == nil || l.db == nil {
		return fmt.Errorf("dream lock service not initialized")
	}
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	_, err := l.db.ExecContext(ctx, `
		DELETE FROM distill_lock WHERE project_id = ?`,
		projectID)
	if err != nil {
		return fmt.Errorf("release lock %q: %w", projectID, err)
	}
	return nil
}

// IsLocked reports whether projectID currently holds a FRESH (non-stale)
// distillation lock. A stale lock (older than dreamLockTTL) is NOT locked — it
// would be auto-released by the next Acquire, so it reports false.
func (l *DreamLockService) IsLocked(ctx context.Context, projectID string) (bool, error) {
	if l == nil || l.db == nil {
		return false, fmt.Errorf("dream lock service not initialized")
	}
	staleBefore := l.now().Add(-dreamLockTTL).UTC().Format(time.RFC3339)
	var lockedAt string
	err := l.db.QueryRowContext(ctx, `
		SELECT locked_at FROM distill_lock WHERE project_id = ?`,
		projectID).Scan(&lockedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("is locked %q: %w", projectID, err)
	}
	// A stale lock is effectively free.
	return lockedAt >= staleBefore, nil
}
