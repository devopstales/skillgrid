package memory

import (
	"context"
	"fmt"
	"strings"
)

// ObservationSnapshot captures the full pre-distill state of a single live
// observation (014 step 12.5). DreamRollback uses these snapshots to restore
// the project to the exact state it had before consolidate/synthesize mutated
// it. It mirrors the obsSelectCols read path so a restore round-trips every
// column the store persists (status, deleted_at, retrieval_usage, timestamps,
// TTL, pinned, …). The pointer columns are stored as *string / *int64 so a
// NULL round-trips as a NULL on restore.
type ObservationSnapshot struct {
	ID             int64
	SessionID      string
	Type           string
	Title          string
	Content        string
	Project        string
	Scope          string
	TopicKey       *string
	NormalizedHash string
	RevisionCount  int
	Owner          *string
	Visibility     string
	Status         string
	RetrievalUsage int
	PromptID       *int64
	CreatedAt      string
	UpdatedAt      string
	Pinned         int
	DuplicateCount int
	LastSeenAt     *string
	ExpiresAt      *string
	ToolName       *string
	GraphRef       *int64
	DeletedAt      *string
}

// Snapshot reads the current live (non-deleted) observations of the project
// into an in-memory snapshot (014 step 12.5). A dream captures this before it
// mutates anything; DreamRollback restores to it on failure. Reading every
// column keeps the restore exact.
func (de *DreamExecutor) Snapshot(ctx context.Context) ([]ObservationSnapshot, error) {
	if de == nil || de.svc == nil || de.svc.DB() == nil {
		return nil, fmt.Errorf("dream executor not initialized")
	}
	rows, err := de.svc.DB().QueryContext(ctx, `
		SELECT id, session_id, type, title, content, project, scope, topic_key,
		       normalized_hash, revision_count, owner, visibility, status,
		       retrieval_usage, prompt_id, created_at, updated_at, pinned,
		       duplicate_count, last_seen_at, expires_at, tool_name, graph_ref, deleted_at
		FROM observations
		WHERE project = ? AND deleted_at IS NULL`,
		de.svc.ProjectID())
	if err != nil {
		return nil, fmt.Errorf("snapshot query: %w", err)
	}
	defer rows.Close()

	var out []ObservationSnapshot
	for rows.Next() {
		var s ObservationSnapshot
		var pinned int
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.Type, &s.Title, &s.Content, &s.Project, &s.Scope, &s.TopicKey,
			&s.NormalizedHash, &s.RevisionCount, &s.Owner, &s.Visibility, &s.Status,
			&s.RetrievalUsage, &s.PromptID, &s.CreatedAt, &s.UpdatedAt, &pinned,
			&s.DuplicateCount, &s.LastSeenAt, &s.ExpiresAt, &s.ToolName, &s.GraphRef, &s.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("snapshot scan: %w", err)
		}
		s.Pinned = pinned
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("snapshot iterate: %w", err)
	}
	return out, nil
}

// DreamRollback reverts the project's observations to the pre-distill state
// captured in pre (014 step 12.5) and releases the distillation lock. It is the
// recovery path for a dream that failed mid-synthesize (or mid-consolidate):
//
//  1. Restore every snapshot row to its exact pre-distill column values (a row
//     that is back but was not in the snapshot — e.g. a consolidated record
//     created during the failed run — is hard-deleted, so only the pre-distill
//     rows remain).
//  2. Release the distillation lock for the project so it can be re-dreamed.
//
// The lock release is best-effort-last: rollback restores data first, then
// frees the lock, so a restore that fails leaves the lock held (fail loud, do
// not free a project whose data may be half-restored).
func (de *DreamExecutor) DreamRollback(ctx context.Context, projectID string, pre []ObservationSnapshot, locks *DreamLockService) error {
	if de == nil || de.svc == nil || de.svc.DB() == nil {
		return fmt.Errorf("dream executor not initialized")
	}
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}

	// 1. Restore the captured rows (exact column round-trip).
	for i := range pre {
		if err := de.restoreSnapshot(ctx, &pre[i]); err != nil {
			return fmt.Errorf("restore observation %d: %w", pre[i].ID, err)
		}
	}

	// 2. Remove any observation created during the failed dream (its id is not
	//    in the pre-distill snapshot) — e.g. the consolidated record or a
	//    partial synthesis. A hard DELETE is safe here: the row is brand-new
	//    (absent from the pre-distill snapshot), so there is no prior state to
	//    preserve. SQLite has no IN-with-Go-slice, so the id list is bound
	//    positionally.
	if err := de.deleteOrphans(ctx, projectID, idsOf(pre)); err != nil {
		return err
	}

	// 3. Release the lock (best-effort, after data is restored).
	if locks != nil {
		if err := locks.Release(ctx, projectID); err != nil {
			return fmt.Errorf("release lock after rollback: %w", err)
		}
	}
	return nil
}

// idsOf returns the observation ids in pre as a fresh []int64 (for the orphan
// sweep's NOT IN clause).
func idsOf(pre []ObservationSnapshot) []int64 {
	out := make([]int64, len(pre))
	for i := range pre {
		out[i] = pre[i].ID
	}
	return out
}

// deleteOrphans hard-deletes the project's observations whose id is not in
// keep (the pre-distill snapshot ids) — i.e. rows created by the failed dream.
// An empty keep set leaves nothing to keep, so nothing is deleted (the sweep
// is a no-op when the dream created no new rows).
func (de *DreamExecutor) deleteOrphans(ctx context.Context, projectID string, keep []int64) error {
	if len(keep) == 0 {
		return nil
	}
	placeholders := make([]string, len(keep))
	args := make([]any, 0, len(keep)+1)
	args = append(args, projectID)
	for i, id := range keep {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		DELETE FROM observations
		WHERE project = ? AND id NOT IN (%s)`, strings.Join(placeholders, ", "))
	if _, err := de.svc.DB().ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("rollback sweep: %w", err)
	}
	return nil
}

// restoreSnapshot writes one observation back to its captured pre-distill
// column values, upserting by id (the row may have been soft-deleted or
// rewritten by the failed dream; restoring must bring it back regardless).
func (de *DreamExecutor) restoreSnapshot(ctx context.Context, s *ObservationSnapshot) error {
	_, err := de.svc.DB().ExecContext(ctx, `
		INSERT INTO observations (
			id, session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, owner, visibility, status,
			retrieval_usage, prompt_id, created_at, updated_at, pinned,
			duplicate_count, last_seen_at, expires_at, tool_name, graph_ref, deleted_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT(id) DO UPDATE SET
			session_id      = excluded.session_id,
			type            = excluded.type,
			title           = excluded.title,
			content         = excluded.content,
			project         = excluded.project,
			scope           = excluded.scope,
			topic_key       = excluded.topic_key,
			normalized_hash = excluded.normalized_hash,
			revision_count  = excluded.revision_count,
			owner           = excluded.owner,
			visibility      = excluded.visibility,
			status          = excluded.status,
			retrieval_usage = excluded.retrieval_usage,
			prompt_id       = excluded.prompt_id,
			created_at      = excluded.created_at,
			updated_at      = excluded.updated_at,
			pinned          = excluded.pinned,
			duplicate_count = excluded.duplicate_count,
			last_seen_at    = excluded.last_seen_at,
			expires_at      = excluded.expires_at,
			tool_name       = excluded.tool_name,
			graph_ref       = excluded.graph_ref,
			deleted_at      = excluded.deleted_at`,
		s.ID, s.SessionID, s.Type, s.Title, s.Content, s.Project, s.Scope, s.TopicKey,
		s.NormalizedHash, s.RevisionCount, s.Owner, s.Visibility, s.Status,
		s.RetrievalUsage, s.PromptID, s.CreatedAt, s.UpdatedAt, s.Pinned,
		s.DuplicateCount, s.LastSeenAt, s.ExpiresAt, s.ToolName, s.GraphRef, s.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("restore %d: %w", s.ID, err)
	}
	return nil
}
