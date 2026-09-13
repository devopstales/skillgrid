package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// defaultSnapshotRetention is the auto-prune retention for store snapshots
// (014 step 20.3): after each Snapshot(), all but the keep-most-recent
// snapshots of the project are deleted. Operators tune it via the
// mnemonic.snapshot.retention config key (SetSnapshotRetention).
const defaultSnapshotRetention = 10

// SnapshotRow is one serialized observations row inside a snapshot BLOB
// (014 step 20.1). It mirrors the snapshot read column list so a restore
// round-trips every column the store persists — including soft-deleted rows
// (deleted_at) so a restore brings a deleted row back exactly. The pointer
// fields store SQL NULLs (a NULL round-trips as a NULL on restore).
type SnapshotRow struct {
	ID             int64
	SessionID      string
	Type           string
	Title          string
	Content        string
	Project        string
	Scope          string
	TopicKey       *string
	Source         *string
	NormalizedHash string
	RevisionCount  int
	PromptID       *int64
	CreatedAt      string
	UpdatedAt      string
	Pinned         int
	DuplicateCount int
	LastSeenAt     *string
	ExpiresAt      *string
	ToolName       *string
	Owner          *string
	Visibility     *string
	Status         *string
	RetrievalUsage int
	ImportanceScore *float64
	RecencyDecay   *float64
	MaturityTier   *string
	Provenance     *string
	MemoryType     *string
	GraphRef       *int64
	DeletedAt      *string
}

// SnapshotInfo is one row of the ListSnapshots output (014 step 20.4): the
// snapshot id, the project it captured, its integrity hash, and its timestamp.
type SnapshotInfo struct {
	ID        int64  `json:"id"`
	ProjectID string `json:"project_id"`
	StateHash string `json:"state_hash"`
	CreatedAt string `json:"created_at"`
}

// snapshotRetention is the effective auto-prune retention for the service:
// the configured value, or defaultSnapshotRetention when unset (a zero or
// negative value means "use the default").
func (s *Service) snapshotRetention() int {
	if s == nil || s.snapshotRetentionCfg <= 0 {
		return defaultSnapshotRetention
	}
	return s.snapshotRetentionCfg
}

// SetSnapshotRetention configures the auto-prune retention (014 step 20.3):
// after each Snapshot(), all but the keep-most-recent snapshots are deleted.
// The service layer calls it from the mnemonic.snapshot.retention config key;
// a non-positive value falls back to defaultSnapshotRetention.
func (s *Service) SetSnapshotRetention(keep int) {
	if s == nil {
		return
	}
	s.snapshotRetentionCfg = keep
}

// snapshotCols is the column list the snapshot read (Snapshot) and write
// (restore) share, so the read and the restore cannot drift apart. It matches
// the observations schema at migration 030.
const snapshotCols = `
	id, session_id, type, title, content, project, scope, topic_key,
	source, normalized_hash, revision_count, prompt_id, created_at, updated_at,
	pinned, duplicate_count, last_seen_at, expires_at, tool_name,
	owner, visibility, status, retrieval_usage,
	importance_score, recency_decay, maturity_tier, provenance, memory_type,
	graph_ref, deleted_at`

// snapshotSelectCols is the snapshot read: snapshotCols (including soft-
// deleted rows — a snapshot is a faithful point-in-time view) ordered by id so
// the serialized BLOB is deterministic.
const snapshotSelectCols = `
	SELECT ` + snapshotCols + `
	FROM observations
	WHERE project = ?
	ORDER BY id`

// Snapshot captures a point-in-time view of the project's observations state
// (014 step 20.1). It serializes every observation row (including soft-deleted
// ones) into a JSON BLOB, stores it with a SHA-256 integrity hash in the
// snapshots table (migration 030), and auto-prunes old snapshots down to the
// configured retention. It returns the new snapshot's id.
//
// The capture is a single SELECT (read-only; the store's single connection
// makes it consistent). It is multi-version: every call adds a new row, so a
// project can roll back to any previous capture via RestoreSnapshot.
func (s *Service) Snapshot(ctx context.Context) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errSnapshotUninit
	}
	rows, err := s.readSnapshotRows(ctx)
	if err != nil {
		return 0, err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(rows); err != nil {
		return 0, fmt.Errorf("serialize snapshot: %w", err)
	}
	data := buf.Bytes()
	hash := sha256Hex(data)
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO snapshots (project_id, state_hash, data, created_at)
		VALUES (?, ?, ?, ?)`,
		s.projectID, hash, data, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("snapshot last insert id: %w", err)
	}
	// Auto-prune (014 step 20.3): keep only the configured number of most
	// recent snapshots. Best-effort — a prune failure must never fail the
	// capture (the snapshot is already durable).
	if perr := s.pruneSnapshots(ctx, s.snapshotRetention()); perr != nil {
		// best-effort: the snapshot row is committed; log and move on
		_ = perr
	}
	return id, nil
}

// readSnapshotRows reads every observation row of the project (including
// soft-deleted rows) into SnapshotRow values, in id order.
func (s *Service) readSnapshotRows(ctx context.Context) ([]SnapshotRow, error) {
	db := s.store.DB
	rows, err := db.QueryContext(ctx, snapshotSelectCols, s.projectID)
	if err != nil {
		return nil, fmt.Errorf("snapshot query: %w", err)
	}
	defer rows.Close()

	var out []SnapshotRow
	for rows.Next() {
		var r SnapshotRow
		if err := rows.Scan(
			&r.ID, &r.SessionID, &r.Type, &r.Title, &r.Content, &r.Project, &r.Scope,
			&r.TopicKey, &r.Source, &r.NormalizedHash, &r.RevisionCount, &r.PromptID,
			&r.CreatedAt, &r.UpdatedAt, &r.Pinned, &r.DuplicateCount, &r.LastSeenAt,
			&r.ExpiresAt, &r.ToolName, &r.Owner, &r.Visibility, &r.Status, &r.RetrievalUsage,
			&r.ImportanceScore, &r.RecencyDecay, &r.MaturityTier, &r.Provenance, &r.MemoryType,
			&r.GraphRef, &r.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("snapshot scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("snapshot iterate: %w", err)
	}
	return out, nil
}

// sha256Hex returns the lowercase hex SHA-256 of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// restoreRowFn runs one snapshot row's upsert on the restore transaction. It
// is the seam TestSnapshotRestoreAtomic swaps in a failing executor through
// (mirroring the distillRollbackExec pattern): a restore is only safe if every
// row's write commits or rolls back TOGETHER. The default (restoreRowDefault)
// runs the statement on the tx, so the whole restore is one atomic commit.
type restoreRowFn func(ctx context.Context, tx *sql.Tx, row *SnapshotRow) error

// restoreRowDefault is the default restore-row executor: it upserts one row on
// the restore transaction (a soft-deleted row is brought back by the explicit
// deleted_at value, an existing row is overwritten exactly).
func restoreRowDefault(ctx context.Context, tx *sql.Tx, row *SnapshotRow) error {
	if _, err := tx.ExecContext(ctx, restoreUpsertSQL,
		row.ID, row.SessionID, row.Type, row.Title, row.Content, row.Project, row.Scope,
		row.TopicKey, row.Source, row.NormalizedHash, row.RevisionCount, row.PromptID,
		row.CreatedAt, row.UpdatedAt, row.Pinned, row.DuplicateCount, row.LastSeenAt,
		row.ExpiresAt, row.ToolName, row.Owner, row.Visibility, row.Status, row.RetrievalUsage,
		row.ImportanceScore, row.RecencyDecay, row.MaturityTier, row.Provenance, row.MemoryType,
		row.GraphRef, row.DeletedAt,
	); err != nil {
		return err
	}
	return nil
}

// restoreUpsertSQL is the per-row restore statement: an upsert by id that
// writes every captured column (including deleted_at), so a restore is an
// exact round-trip of the point-in-time view.
const restoreUpsertSQL = `
	INSERT INTO observations (
		id, session_id, type, title, content, project, scope, topic_key,
		source, normalized_hash, revision_count, prompt_id, created_at, updated_at,
		pinned, duplicate_count, last_seen_at, expires_at, tool_name,
		owner, visibility, status, retrieval_usage,
		importance_score, recency_decay, maturity_tier, provenance, memory_type,
		graph_ref, deleted_at
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?
	)
	ON CONFLICT(id) DO UPDATE SET
		session_id      = excluded.session_id,
		type            = excluded.type,
		title           = excluded.title,
		content         = excluded.content,
		project         = excluded.project,
		scope           = excluded.scope,
		topic_key       = excluded.topic_key,
		source          = excluded.source,
		normalized_hash = excluded.normalized_hash,
		revision_count  = excluded.revision_count,
		prompt_id       = excluded.prompt_id,
		created_at      = excluded.created_at,
		updated_at      = excluded.updated_at,
		pinned          = excluded.pinned,
		duplicate_count = excluded.duplicate_count,
		last_seen_at    = excluded.last_seen_at,
		expires_at      = excluded.expires_at,
		tool_name       = excluded.tool_name,
		owner           = excluded.owner,
		visibility      = excluded.visibility,
		status          = excluded.status,
		retrieval_usage = excluded.retrieval_usage,
		importance_score = excluded.importance_score,
		recency_decay  = excluded.recency_decay,
		maturity_tier  = excluded.maturity_tier,
		provenance     = excluded.provenance,
		memory_type    = excluded.memory_type,
		graph_ref      = excluded.graph_ref,
		deleted_at      = excluded.deleted_at`

// RestoreSnapshot deserializes snapshotID and replaces the project's current
// observations state with it (014 step 20.1). The whole restore is ONE
// transaction: every captured row is upserted (exact column round-trip), any
// row created after the capture is hard-deleted (orphan sweep), and the FTS
// index is rebuilt — committed atomically. A failure anywhere rolls the whole
// restore back, so the store is either exactly pre-restore or exactly the
// captured state, never half-restored.
func (s *Service) RestoreSnapshot(ctx context.Context, snapshotID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errSnapshotUninit
	}
	if snapshotID <= 0 {
		return fmt.Errorf("snapshot id %d is invalid", snapshotID)
	}
	var stateHash string
	var data []byte
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT state_hash, data FROM snapshots WHERE id = ? AND project_id = ?`,
		snapshotID, s.projectID,
	).Scan(&stateHash, &data)
	if err == sql.ErrNoRows {
		return fmt.Errorf("snapshot %d not found for project %q", snapshotID, s.projectID)
	}
	if err != nil {
		return fmt.Errorf("read snapshot %d: %w", snapshotID, err)
	}
	// Integrity check: the stored hash must match the BLOB's actual SHA-256.
	// A mismatch means the row was tampered with or corrupted — restoring it
	// would write garbage, so fail loud before touching observations.
	if sha256Hex(data) != stateHash {
		return fmt.Errorf("snapshot %d integrity check failed (state_hash mismatch)", snapshotID)
	}
	var rows []SnapshotRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("deserialize snapshot %d: %w", snapshotID, err)
	}

	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin restore tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// 1. Restore the captured rows (exact column round-trip) inside the tx.
	fn := s.restoreRowFnFn()
	keep := make([]int64, 0, len(rows))
	for i := range rows {
		if err := fn(ctx, tx, &rows[i]); err != nil {
			return fmt.Errorf("restore observation %d: %w", rows[i].ID, err)
		}
		keep = append(keep, rows[i].ID)
	}

	// 2. Remove any observation created after the capture (its id is not in the
	//    snapshot) — the orphan sweep, inside the same tx. An empty snapshot
	//    deletes every project row (a faithful empty restore).
	if err := sweepOrphanRows(ctx, tx, s.projectID, keep); err != nil {
		return err
	}

	// 3. Rebuild the FTS index: the upserts and the sweep leave observations_fts
	//    out of sync (the triggers only fire on the statements they observe, and
	//    an ON CONFLICT upsert that changes no FTS columns fires no trigger).
	//    A full rebuild is idempotent and cheap relative to the restore.
	if _, err := tx.ExecContext(ctx, `INSERT INTO observations_fts(observations_fts) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("rebuild fts after restore: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit restore: %w", err)
	}
	committed = true
	return nil
}

// restoreRowFnFn returns the active restore-row executor, reading the seam
// under lock so a test's swap and the restore loop never race.
func (s *Service) restoreRowFnFn() restoreRowFn {
	s.restoreMu.Lock()
	fn := s.restoreRowFn
	s.restoreMu.Unlock()
	if fn == nil {
		return restoreRowDefault
	}
	return fn
}

// sweepOrphanRows hard-deletes the project's observations whose id is not in
// keep (the captured snapshot's ids). An empty keep set deletes every project
// row (a faithful empty restore).
func sweepOrphanRows(ctx context.Context, tx *sql.Tx, projectID string, keep []int64) error {
	q := `DELETE FROM observations WHERE project = ?`
	args := []any{projectID}
	if len(keep) > 0 {
		placeholders := make([]string, len(keep))
		for i, id := range keep {
			placeholders[i] = "?"
			args = append(args, id)
		}
		q += ` AND id NOT IN (` + strings.Join(placeholders, ", ") + `)`
	}
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("restore sweep: %w", err)
	}
	return nil
}

// PruneSnapshots deletes all but the keep most recent snapshots of the
// project (014 step 20.3). A non-positive keep keeps everything (no-op). The
// delete is by id: the keep most recent snapshots are the ones with the
// highest ids (AUTOINCREMENT is monotonic).
func (s *Service) PruneSnapshots(ctx context.Context, keep int) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errSnapshotUninit
	}
	if keep <= 0 {
		return nil
	}
	if _, err := s.store.DB.ExecContext(ctx, `
		DELETE FROM snapshots
		WHERE project_id = ?
		  AND id NOT IN (
			SELECT id FROM snapshots
			WHERE project_id = ?
			ORDER BY id DESC
			LIMIT ?
		  )`,
		s.projectID, s.projectID, keep,
	); err != nil {
		return fmt.Errorf("prune snapshots: %w", err)
	}
	return nil
}

// pruneSnapshots is the internal auto-prune call (best-effort) used by
// Snapshot after each capture. It swallows nothing — the error is returned so
// the caller can decide (Snapshot treats it as best-effort).
func (s *Service) pruneSnapshots(ctx context.Context, keep int) error {
	return s.PruneSnapshots(ctx, keep)
}

// ListSnapshots returns the project's snapshots newest-first (014 step 20.4),
// each with its id, integrity hash, and capture timestamp. It is the read path
// behind `mem snapshot list`.
func (s *Service) ListSnapshots(ctx context.Context) ([]SnapshotInfo, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errSnapshotUninit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, project_id, state_hash, created_at
		FROM snapshots
		WHERE project_id = ?
		ORDER BY id DESC`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	out := []SnapshotInfo{}
	for rows.Next() {
		var info SnapshotInfo
		if err := rows.Scan(&info.ID, &info.ProjectID, &info.StateHash, &info.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		out = append(out, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return out, nil
}

// ── Row-level locking (014 step 20.2) ────────────────────────────────────────

// rowLockTimeout is the busy-timeout applied to the row-lock connection used by
// UpdateRow: 5 seconds, per the step 20.2 requirement. A concurrent writer
// holding the WAL write lock is waited on up to this long before the write
// fails with SQLITE_BUSY.
const rowLockTimeout = 5000 // milliseconds

// rowLockDsnSuffix appends the row-lock pragmas to a sqlite DSN: _txlock=
// immediate so db.BeginTx runs BEGIN IMMEDIATE (acquiring the write lock
// immediately, serializing concurrent writes on the same row). busy_timeout is
// applied via PRAGMA (the DSN only carries the begin-mode; the timeout is set
// explicitly on open). The %26 is the URL-encoded '&' — the modernc.org driver
// parses the DSN as a query string, so a literal '&' would be read as a
// filename separator.
const rowLockDsnSuffix = `?_txlock=immediate`

// UpdateRow updates one observation's content under row-level locking (014
// step 20.2): it wraps the UPDATE in a BEGIN IMMEDIATE transaction, so SQLite
// acquires the write lock at BEGIN and concurrent writes to the same row are
// serialized by the WAL write lock (the second writer blocks, up to the
// 5-second busy_timeout, until the first commits). It returns the rows
// affected (0 when the observation is absent or soft-deleted).
//
// It is a method on *Service (rather than a free function) so the concurrency
// test can drive BOTH writers through the same code path: the first via the
// store handle, the second via the row-lock handle — both are *sql.DB on the
// same file, and both run BEGIN IMMEDIATE.
func (s *Service) UpdateRow(ctx context.Context, db *sql.DB, id int64, content string) (int64, error) {
	if db == nil {
		return 0, errSnapshotUninit
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin row-lock tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		UPDATE observations
		SET content = ?, updated_at = ?,
		    revision_count = COALESCE(revision_count, 0) + 1
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		content, now, id, s.projectID,
	)
	if err != nil {
		return 0, fmt.Errorf("row-lock update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("row-lock rows affected: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit row-lock tx: %w", err)
	}
	committed = true
	return n, nil
}

// errSnapshotUninit is the sentinel for a not-initialized snapshot service.
var errSnapshotUninit = errors.New("snapshot service not initialized")
