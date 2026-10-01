package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Pin marks an observation so it sorts ahead of everything else in
// mem_context and boosts mem_search ordering. Pinned state is local to this
// device store (not synced), matching Engram's local pin semantics.
func (s *Service) Pin(ctx context.Context, id int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET pinned = 1
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("pin: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// Unpin clears the pin flag so the observation returns to normal recency
// ordering.
func (s *Service) Unpin(ctx context.Context, id int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET pinned = 0
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("unpin: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// Pinned returns the observation's pinned flag.
func (s *Service) Pinned(ctx context.Context, id int64) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return false, errors.New("memory service not initialized")
	}
	var pinned int
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT pinned FROM observations WHERE id = ? AND project = ?`,
		id, s.projectID,
	).Scan(&pinned)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("observation %d not found", id)
	}
	if err != nil {
		return false, fmt.Errorf("pinned flag: %w", err)
	}
	return pinned == 1, nil
}

// SetExpiresAt sets the observation's expires_at timestamp. value must be
// RFC3339 (or empty to clear). Malformed timestamps are rejected so callers
// never persist an unparseable TTL that breaks soft-exclude filters.
func (s *Service) SetExpiresAt(ctx context.Context, id int64, value string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	value = strings.TrimSpace(value)
	if value != "" {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("invalid expires_at: %w", err)
		}
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET expires_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		nullString(value), id, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("set expires_at: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// BumpDuplicate records that a save matched an existing observation's
// (title+content+type) hash within the 24h window. It increments
// duplicate_count and refreshes last_seen_at so recency weighting and the
// "seen N times" signal stay current without creating a new row. Best-effort:
// callers may ignore the error (it must never block the save response).
func (s *Service) BumpDuplicate(ctx context.Context, id int64) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET duplicate_count = COALESCE(duplicate_count, 0) + 1,
		    last_seen_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		now, id, s.projectID,
	)
}

// MarkSuperseded stamps observation `oldID` as superseded by `newID` (the
// AUDN "delete" arm, ADR-0011). In one transaction it sets invalid_at=now,
// superseded_by=newID, status='superseded' on the old row and records a
// `supersedes` edge (src=oldID, dst=newID) in memory_relations so the chain
// is walkable. The old row stays in the table (soft — never deleted); the
// bi-temporal filter now excludes it from live reads.
func (s *Service) MarkSuperseded(ctx context.Context, oldID, newID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mark_superseded: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, `
		UPDATE observations
		SET invalid_at = ?, superseded_by = ?, status = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		nullString(now), newID, StatusSuperseded, oldID, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("mark_superseded: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("mark_superseded: observation %d not found", oldID)
	}
	// Upsert the supersedes edge (src=old → dst=new) within the same
	// transaction for atomicity. modernc.org/sqlite does not support
	// ON CONFLICT against a partial unique index, so look up a live row,
	// UPDATE in place if present, INSERT if not (same idiom as
	// RecordRelation in relations.go).
	var existingID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM memory_relations
		WHERE src_obs_id = ? AND dst_obs_id = ? AND relation = ? AND deleted_at IS NULL
		LIMIT 1`,
		oldID, newID, "supersedes",
	).Scan(&existingID)
	if err == nil {
		if _, err := tx.ExecContext(ctx, `
			UPDATE memory_relations SET created_at = ? WHERE id = ?`,
			now, existingID,
		); err != nil {
			return fmt.Errorf("mark_superseded: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("mark_superseded: %w", err)
	} else {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO memory_relations (src_obs_id, dst_obs_id, relation, project, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			oldID, newID, "supersedes", s.projectID, now,
		); err != nil {
			return fmt.Errorf("mark_superseded: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mark_superseded: %w", err)
	}
	committed = true
	return nil
}

// TTLSoftExpiry returns the count of non-deleted observations past their
// expires_at timestamp as of now — a read-only diagnostic that surfaces how
// many memories a periodic sweep could retire. Callers typically fold this
// into mem_doctor output.
func (s *Service) TTLSoftExpiry(ctx context.Context) (int, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("memory service not initialized")
	}
	var n int
	// Compare via strftime('%s', ...) so RFC3339 ("T"/"Z") and "YYYY-MM-DD HH:MM:SS"
	// timestamps both compare as integer seconds, not as raw strings (where 'T' > ' '
	// would mis-order same-day values).
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL
		  AND expires_at IS NOT NULL AND expires_at != ''
		  AND strftime('%s', expires_at) <= strftime('%s', 'now')`,
		s.projectID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("ttl sweep: %w", err)
	}
	return n, nil
}

// TTLRetire soft-deletes all expired rows for the project (idempotent).
// Returns the number retired.
func (s *Service) TTLRetire(ctx context.Context) (int, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("memory service not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET deleted_at = ?
		WHERE project = ? AND (deleted_at IS NULL OR deleted_at = '')
		  AND expires_at IS NOT NULL AND expires_at != ''
		  AND strftime('%s', expires_at) <= strftime('%s', 'now')`,
		now, s.projectID,
	)
	if err != nil {
		return 0, fmt.Errorf("ttl retire: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
