package handoff

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HandoffRefInput links a handoff artifact to the commit range + spec dir it
// covers.
type HandoffRefInput struct {
	HandoffID   string // cleave handoff id, task id, or "checkpoint:<name>"
	HandoffType string // 'session' | 'team' | 'checkpoint'
	FromCommit  string
	ToCommit    string
	SpecDir     string
	TeamID      string
	TaskID      string
}

// RecordHandoffRef upserts a handoff ref (idempotent on
// (handoff_id, handoff_type, project)). Blank handoff id / type are rejected
// before any write.
func (h *Hub) RecordHandoffRef(ctx context.Context, in HandoffRefInput) error {
	if h == nil || h.DB == nil {
		return errors.New("handoff hub not initialized")
	}
	if strings.TrimSpace(in.HandoffID) == "" {
		return errors.New("handoff_id is required")
	}
	if strings.TrimSpace(in.HandoffType) == "" {
		return errors.New("handoff_type is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := h.DB.ExecContext(ctx, `
		INSERT INTO handoff_refs
			(handoff_id, handoff_type, project, from_commit, to_commit,
			 spec_dir, team_id, task_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(handoff_id, handoff_type, project) DO UPDATE SET
			from_commit = COALESCE(handoff_refs.from_commit, excluded.from_commit),
			to_commit = COALESCE(handoff_refs.to_commit, excluded.to_commit),
			spec_dir = COALESCE(handoff_refs.spec_dir, excluded.spec_dir),
			team_id = COALESCE(handoff_refs.team_id, excluded.team_id),
			task_id = COALESCE(handoff_refs.task_id, excluded.task_id)`,
		in.HandoffID, in.HandoffType, h.Project,
		nullString(in.FromCommit), nullString(in.ToCommit),
		nullString(in.SpecDir), nullString(in.TeamID), nullString(in.TaskID), now)
	if err != nil {
		return fmt.Errorf("upsert handoff ref: %w", err)
	}
	return nil
}

// DeleteHandoffRef removes a ref (used to roll back when the owning handoff
// row fails, keeping the ref table free of orphans).
func (h *Hub) DeleteHandoffRef(ctx context.Context, handoffID, handoffType string) error {
	if h == nil || h.DB == nil {
		return errors.New("handoff hub not initialized")
	}
	_, err := h.DB.ExecContext(ctx,
		`DELETE FROM handoff_refs WHERE project = ? AND handoff_id = ? AND handoff_type = ?`,
		h.Project, handoffID, handoffType)
	return err
}

// GetHandoffRef returns one ref or a nil error with ok=false when absent.
func (h *Hub) GetHandoffRef(ctx context.Context, handoffID, handoffType string) (*HandoffRef, bool, error) {
	if h == nil || h.DB == nil {
		return nil, false, errors.New("handoff hub not initialized")
	}
	row := h.DB.QueryRowContext(ctx, `
		SELECT handoff_id, handoff_type, COALESCE(from_commit,''), COALESCE(to_commit,''),
		       COALESCE(spec_dir,''), COALESCE(team_id,''), COALESCE(task_id,''), created_at
		FROM handoff_refs
		WHERE project = ? AND handoff_id = ? AND handoff_type = ?`,
		h.Project, handoffID, handoffType)
	var r HandoffRef
	if err := row.Scan(&r.HandoffID, &r.HandoffType, &r.FromCommit, &r.ToCommit,
		&r.SpecDir, &r.TeamID, &r.TaskID, &r.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &r, true, nil
}

// ListHandoffRefs returns the project's refs, newest-first (limit clamped to
// 1..200).
func (h *Hub) ListHandoffRefs(ctx context.Context, limit int) ([]HandoffRef, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := h.DB.QueryContext(ctx, `
		SELECT handoff_id, handoff_type, COALESCE(from_commit,''), COALESCE(to_commit,''),
		       COALESCE(spec_dir,''), COALESCE(team_id,''), COALESCE(task_id,''), created_at
		FROM handoff_refs WHERE project = ?
		ORDER BY created_at DESC
		LIMIT ?`, h.Project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HandoffRef{}
	for rows.Next() {
		var r HandoffRef
		if err := rows.Scan(&r.HandoffID, &r.HandoffType, &r.FromCommit, &r.ToCommit,
			&r.SpecDir, &r.TeamID, &r.TaskID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
