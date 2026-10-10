package memory

import (
	"context"
	"database/sql"
	"fmt"
)

// TICKET-03: hot-path reads + all writers off the compat VIEW.
//
// Migration 054 (TICKET-02) split `observations` into observations_core +
// five side tables behind a VIEW with INSTEAD OF triggers. View-mediated
// writes LAND correctly, but four driver semantics are engine-level broken
// through a view (proven in ticket-02-review.md on stock SQLite):
// LastInsertId stale, RowsAffected always 0, INSERT...RETURNING NULL, and
// INSERT...ON CONFLICT rejected ("cannot UPSERT a view").
//
// This file is the seam the hot path uses instead of the view:
//
//   - obsSplitSelectCols / obsSplitFrom: the mem_context/mem_search read
//     shape (core + state/provenance/stats/temporal, LEFT JOINed). The
//     observation_vectors table is deliberately ABSENT: hot reads never
//     select vector columns, so the query plan cannot scan blobs
//     (SATISFIES hot-path-avoids-blobs). Vectors stay lazy: read only via
//     observation_vectors when the caller actually needs an embedding
//     (SearchByVector, obsEmbedding).
//   - obsSplitSelectCols is positionally identical to obsSelectCols (same
//     31 columns, same order), so scanObservations serves both unchanged.
//   - obsExecer + fts*Obs: FTS maintenance for direct-table writes. The
//     legacy observations_fts_* triggers ride observations_legacy after the
//     054 RENAME and the view triggers only fire on view-mediated writes,
//     so a direct core/side write must maintain observations_fts itself,
//     mirroring the legacy trigger bodies (insert on row insert,
//     delete+insert on FTS-column change, delete with old values on hard
//     delete). Writes that touch no FTS column (title/content/type/project)
//     need no FTS work — same observable index state.
//   - The INSTEAD OF triggers on the view STAY as a safety net for ad-hoc
//     (test/vacuum) view writes; production no longer routes through them.

// obsSplitSelectCols is the hot-read column list: the same 31 columns in
// the same order as obsSelectCols, qualified to the owning split table.
const obsSplitSelectCols = `
	o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
	o.topic_key, pr.source, o.normalized_hash, o.revision_count, pr.prompt_id, o.created_at, o.updated_at,
	COALESCE(st.pinned, 0), COALESCE(sa.duplicate_count, 0), sa.last_seen_at, st.expires_at, pr.tool_name,
	pr.owner, COALESCE(st.visibility, 'private'), COALESCE(st.status, 'active'), COALESCE(sa.retrieval_usage, 0),
	sa.importance_score, sa.recency_decay, sa.maturity_tier, pr.provenance, pr.memory_type,
	tm.valid_at, tm.invalid_at, tm.superseded_by`

// obsSplitFrom is the hot-read FROM shape: core plus every side table the
// hot columns live in, LEFT JOINed exactly like the compat view. No
// observation_vectors: any EXPLAIN QUERY PLAN over this shape cannot name
// that table (verified by TestHotPathAvoidsVectors).
const obsSplitFrom = `observations_core o
	LEFT JOIN observation_state st ON st.observation_id = o.id
	LEFT JOIN observation_provenance pr ON pr.observation_id = o.id
	LEFT JOIN observation_stats sa ON sa.observation_id = o.id
	LEFT JOIN observation_temporal tm ON tm.observation_id = o.id`

// obsExecer is the narrow "a write target" for split-table statements:
// both *sql.DB and *sql.Tx satisfy it, so single writes and tx batches
// share the same helpers.
type obsExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// ftsInsertObs mirrors the legacy observations_fts_insert trigger for one
// directly-written row.
func ftsInsertObs(ctx context.Context, x obsExecer, id int64, title, content, typ, project string) error {
	_, err := x.ExecContext(ctx, `
		INSERT INTO observations_fts(rowid, title, content, type, project)
		VALUES (?, ?, ?, ?, ?)`,
		id, title, content, typ, project,
	)
	if err != nil {
		return fmt.Errorf("fts insert %d: %w", id, err)
	}
	return nil
}

// ftsDeleteObs mirrors the legacy observations_fts_delete trigger: the old
// column values must be passed (FTS5 deletes by indexed terms).
func ftsDeleteObs(ctx context.Context, x obsExecer, id int64, title, content, typ, project string) error {
	_, err := x.ExecContext(ctx, `
		INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
		VALUES ('delete', ?, ?, ?, ?, ?)`,
		id, title, content, typ, project,
	)
	if err != nil {
		return fmt.Errorf("fts delete %d: %w", id, err)
	}
	return nil
}

// ftsReplaceObs mirrors the legacy observations_fts_update trigger.
func ftsReplaceObs(ctx context.Context, x obsExecer, id int64, oldTitle, oldContent, oldType, newTitle, newContent, newType, project string) error {
	if err := ftsDeleteObs(ctx, x, id, oldTitle, oldContent, oldType, project); err != nil {
		return err
	}
	return ftsInsertObs(ctx, x, id, newTitle, newContent, newType, project)
}

// sideScopeWhere returns the WHERE fragment scoping a side-table point
// write to one observation of this project (the project/deleted_at scoping
// the pre-split `UPDATE observations ... WHERE id = ? AND project = ? AND
// deleted_at IS NULL` carried). table is the side table being updated
// (needed to qualify observation_id inside the EXISTS subquery).
func sideScopeWhere(table string) string {
	return fmt.Sprintf(`observation_id = ? AND EXISTS (
		SELECT 1 FROM observations_core c
		WHERE c.id = %s.observation_id AND c.project = ? AND c.deleted_at IS NULL)`, table)
}

// splitInsertArgs carries one full observation row, already split by owning
// table. Nullable columns use driver-ready values (nil for NULL).
type splitInsertArgs struct {
	SessionID, Type, Title, Content, Project, Scope string
	TopicKey                                       any
	Hash, Now, Source                              string
	PromptID                                       any
	ToolName                                       any
	Owner                                          string
	ExpiresAt                                      any
	Provenance                                     any
	MemoryType                                     any
}

// txInsertSplitObservation writes core + all five side rows + the FTS entry
// in ONE transaction and returns the new core id. The core INSERT runs
// first on a real table, so LastInsertId is exact (the view trigger path
// left it stale). Column defaults mirror the 054 INSTEAD OF trigger
// COALESCEs (status 'active', visibility 'private', source 'agent',
// pinned/duplicate/retrieval/revision 0, importance 0 / decay 1 / tier
// 'fresh'); every other unlisted side column stays NULL.
func txInsertSplitObservation(ctx context.Context, db *sql.DB, a splitInsertArgs) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("insert observation: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO observations_core (
			session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, NULL)`,
		a.SessionID, a.Type, a.Title, a.Content, a.Project, a.Scope, a.TopicKey,
		a.Hash, a.Now, a.Now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert observation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO observation_state (
			observation_id, status, visibility, archived_at, archive_reason,
			review_after, expires_at, pinned
		) VALUES (?, 'active', 'private', NULL, NULL, NULL, ?, 0)`,
		id, a.ExpiresAt,
	); err != nil {
		return 0, fmt.Errorf("insert observation state: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO observation_provenance (
			observation_id, source, prompt_id, tool_name, provenance,
			owner, memory_type, graph_ref
		) VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`,
		id, a.Source, a.PromptID, a.ToolName, a.Provenance, a.Owner, a.MemoryType,
	); err != nil {
		return 0, fmt.Errorf("insert observation provenance: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO observation_stats (
			observation_id, duplicate_count, last_seen_at, retrieval_usage,
			importance_score, recency_decay, maturity_tier, risk_score
		) VALUES (?, 0, NULL, 0, 0, 1, 'fresh', NULL)`,
		id,
	); err != nil {
		return 0, fmt.Errorf("insert observation stats: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO observation_temporal (
			observation_id, valid_at, invalid_at, superseded_by
		) VALUES (?, ?, NULL, NULL)`,
		id, a.Now,
	); err != nil {
		return 0, fmt.Errorf("insert observation temporal: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO observation_vectors (
			observation_id, embedding, embedding_model, embedding_created_at
		) VALUES (?, NULL, NULL, NULL)`,
		id,
	); err != nil {
		return 0, fmt.Errorf("insert observation vectors: %w", err)
	}
	if err := ftsInsertObs(ctx, tx, id, a.Title, a.Content, a.Type, a.Project); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("insert observation: commit: %w", err)
	}
	committed = true
	return id, nil
}

// deleteSplitSides removes every side-table row for observationID on x.
// The core row itself is deleted by the caller (core last, so a concurrent
// reader never sees a core row without its sides).
func deleteSplitSides(ctx context.Context, x obsExecer, observationID int64) error {
	for _, table := range []string{
		"observation_vectors",
		"observation_temporal",
		"observation_stats",
		"observation_provenance",
		"observation_state",
	} {
		if _, err := x.ExecContext(ctx,
			`DELETE FROM `+table+` WHERE observation_id = ?`, observationID); err != nil {
			return fmt.Errorf("delete %s %d: %w", table, observationID, err)
		}
	}
	return nil
}
