package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
)

// txngate.go — the save-transaction gate the memory pipeline (Task 5)
// writes through. The memory package sees only the SaveTXN interface it
// opens via BeginSaveTxn (the memory's beginSaveTxn seam); this file is
// the single adapter over the SQLite handle, so the Postgres dialect
// (Task 18) plugs into the same seam shape.

// ErrSessionEnded: the session row exists and has ended (terminal, 409).
var ErrSessionEnded = errors.New("store: session ended")

// ErrSessionProjectMismatch: a project_owned session's project differs from
// the resolved project (409).
var ErrSessionProjectMismatch = errors.New("store: session project mismatch")

// SaveTXN is the save-transaction gate seam the memory pipeline writes
// through (the memory package opens it via BeginSaveTxn and sees only this
// interface — INV-2).
type SaveTXN interface {
	// CheckSession enforces the session policy (spec 01-memory/002 §3):
	// a missing row is auto-registered (shared ownership, resolved
	// project); an ended row → ErrSessionEnded; a project_owned row whose
	// project mismatches the resolved project → ErrSessionProjectMismatch.
	// Returns the resolved project (explicit > session's project).
	CheckSession(ctx context.Context, sessionID, project string) (string, error)
	// LookupHash: the idx_obs_dedupe identity lookup over live rows.
	LookupHash(ctx context.Context, project, scope, typ, title, normalizedHash string) (int64, bool, error)
	// LookupTopicKey: the idx_obs_topic identity lookup over live rows.
	LookupTopicKey(ctx context.Context, project, scope, topicKey string) (int64, bool, error)
	// LookupSyncID: a row's sync_id (the pipeline's result shape carries
	// identity by sync_id — INV-6).
	LookupSyncID(ctx context.Context, id int64) (string, error)
	// InsertObservation: a fresh observation (INV-1 provenance, INV-6 sync
	// id) + captured prompt + enrollment-gated sync mutation, same txn.
	InsertObservation(ctx context.Context, p GateInsert) (int64, string, error)
	// ReviseObservation: in-place revision (observation_versions snapshot
	// first), same-txn prompt + journal.
	ReviseObservation(ctx context.Context, id int64, p GateInsert) (int, error)
	// MarkDuplicate: the hash-floor noop (counters only, no new row).
	MarkDuplicate(ctx context.Context, id int64) error
	// EnrollProject: idempotent sync enrollment (Task 17's surface).
	EnrollProject(ctx context.Context, project string) error
	Commit(ctx context.Context) error
	Rollback() error
}

// GateInsert is the gate's insert/revise input.
type GateInsert struct {
	SessionID, Type, Title, Content, ToolName, Project, Scope, TopicKey, OperationID string
	SyncID                string
	NormalizedHash        string
	ReviewAfter           *string
	Importance            float64
	PromptContent         string
}

// saveTXN implements SaveTXN on a *sql.Tx.
type saveTXN struct {
	st *sqliteStore
	tx *sql.Tx
}

func (t *saveTXN) CheckSession(ctx context.Context, sessionID, project string) (string, error) {
	resolved := strings.TrimSpace(project)
	if strings.TrimSpace(sessionID) == "" {
		return resolved, nil
	}
	if resolved == "" {
		var p string
		if err := t.tx.QueryRow(`SELECT ifnull(project, '') FROM sessions WHERE id = ?`, sessionID).Scan(&p); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("store: session lookup: %w", err)
		}
		resolved = p
	}
	var owned, endedAt string
	err := t.tx.QueryRow(`SELECT ifnull(ownership_mode,'shared'), ifnull(ended_at,'') FROM sessions WHERE id = ?`, sessionID).Scan(&owned, &endedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Auto-register: shared ownership for the resolved project (the
		// plugin's first save opens its session implicitly — Engram's
		// create-on-first-use).
		if _, err := t.tx.Exec(`
			INSERT INTO sessions (id, project, directory, ownership_mode, started_at, created_at)
			VALUES (?, ?, '', 'shared', datetime('now'), datetime('now'))
			ON CONFLICT(id) DO NOTHING`,
			sessionID, nullableString(resolved)); err != nil {
			return "", fmt.Errorf("store: auto-register session: %w", err)
		}
		return resolved, nil
	}
	if err != nil {
		return "", fmt.Errorf("store: session lookup: %w", err)
	}
	if endedAt != "" {
		return "", ErrSessionEnded
	}
	if owned == "project_owned" && resolved != "" && strings.TrimSpace(project) != "" {
		var rowProj string
		_ = t.tx.QueryRow(`SELECT ifnull(project,'') FROM sessions WHERE id = ?`, sessionID).Scan(&rowProj)
		if rowProj != resolved {
			return "", ErrSessionProjectMismatch
		}
	}
	return resolved, nil
}

func (t *saveTXN) LookupHash(ctx context.Context, project, scope, typ, title, normalizedHash string) (int64, bool, error) {
	var id int64
	err := t.tx.QueryRow(
		`SELECT id FROM observations
		 WHERE normalized_hash = ? AND ifnull(project,'') = ? AND scope = ? AND type = ? AND title = ?
		   AND deleted_at IS NULL`,
		normalizedHash, project, scope, typ, title).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (t *saveTXN) LookupTopicKey(ctx context.Context, project, scope, topicKey string) (int64, bool, error) {
	var id int64
	err := t.tx.QueryRow(
		`SELECT id FROM observations
		 WHERE ifnull(project,'') = ? AND scope = ? AND topic_key = ? AND deleted_at IS NULL`,
		project, scope, topicKey).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (t *saveTXN) InsertObservation(ctx context.Context, p GateInsert) (int64, string, error) {
	now := nowStamp()
	prov := Provenance{SessionID: p.SessionID, CreatedAt: now}
	provJSON, _ := json.Marshal(prov)
	res, err := t.tx.Exec(`
		INSERT INTO observations (
			sync_id, session_id, type, title, content, tool_name, project, scope,
			topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at,
			importance, retrieval_usage, provenance, created_at, updated_at, review_after
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, datetime('now'), ?, 0, ?, ?, ?, ?)`,
		p.SyncID, p.SessionID, p.Type, p.Title, p.Content,
		nullableString(p.ToolName), nullableString(p.Project), scopeOrDefault(p.Scope),
		nullableString(p.TopicKey), p.NormalizedHash,
		p.Importance, provJSON, now, now, nullableString(deref(p.ReviewAfter)),
	)
	if err != nil {
		return 0, "", err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, "", err
	}
	// replay-safe ledger row (spec 002-schema §3.3)
	if p.OperationID != "" {
		if _, err := t.tx.Exec(`
			INSERT INTO observation_save_operations (operation_id, fingerprint, observation_id)
			VALUES (?, ?, ?)`,
			p.OperationID, "v1:"+p.NormalizedHash, id); err != nil {
			return 0, "", err
		}
	}
	// the captured prompt (same transaction; the prm- sync id is minted
	// here so INV-6 holds inside the gate)
	if p.PromptContent != "" {
		if _, err := t.tx.Exec(`
			INSERT INTO user_prompts (sync_id, session_id, content, project, created_at)
			VALUES (?, ?, ?, ?, datetime('now'))`,
			syncid.NewSyncID("prm"), p.SessionID, p.PromptContent, nullableString(p.Project)); err != nil {
			return 0, "", err
		}
	}
	// step 7: the sync mutation is enrollment-gated (unenrolled → no row)
	if t.st.syncEnabled() && t.enrolled(p.Project) {
		if err := t.enqueueUpsert(p.Project, p.SyncID); err != nil {
			return 0, "", err
		}
	}
	return id, p.SyncID, nil
}

func (t *saveTXN) ReviseObservation(ctx context.Context, id int64, p GateInsert) (int, error) {
	var curTitle, curContent string
	var curRev int
	err := t.tx.QueryRow(`SELECT title, content, revision_count FROM observations WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&curTitle, &curContent, &curRev)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("store: observation %d not found", id)
	}
	if err != nil {
		return 0, err
	}
	newRev := curRev + 1
	// snapshot the pre-state first (spec 002-schema §2.3)
	if _, err := t.tx.Exec(`
		INSERT INTO observation_versions (observation_id, version, title, content, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))`,
		id, newRev, nullableString(curTitle), curContent); err != nil {
		return 0, err
	}
	if _, err := t.tx.Exec(`
		UPDATE observations
		SET title = ?, content = ?, normalized_hash = ?, revision_count = ?,
		    review_after = ?, updated_at = datetime('now')
		WHERE id = ?`,
		p.Title, p.Content, p.NormalizedHash, newRev,
		nullableString(deref(p.ReviewAfter)), id); err != nil {
		return 0, err
	}
	if p.PromptContent != "" {
		if _, err := t.tx.Exec(`
			INSERT INTO user_prompts (sync_id, session_id, content, project, created_at)
			VALUES (?, ?, ?, ?, datetime('now'))`,
			syncid.NewSyncID("prm"), p.SessionID, p.PromptContent, nullableString(p.Project)); err != nil {
			return 0, err
		}
	}
	if t.st.syncEnabled() && t.enrolled(p.Project) {
		if err := t.enqueueUpsert(p.Project, p.SyncID); err != nil {
			return 0, err
		}
	}
	return newRev, nil
}

// LookupSyncID reads a row's sync_id (the pipeline's noop/revise results
// carry the row's identity by sync_id — INV-6).
func (t *saveTXN) LookupSyncID(ctx context.Context, id int64) (string, error) {
	var out string
	err := t.tx.QueryRow(`SELECT sync_id FROM observations WHERE id = ? AND deleted_at IS NULL`, id).Scan(&out)
	return out, err
}

func (t *saveTXN) MarkDuplicate(ctx context.Context, id int64) error {
	_, err := t.tx.Exec(`
		UPDATE observations
		SET duplicate_count = duplicate_count + 1, last_seen_at = datetime('now'), updated_at = datetime('now')
		WHERE id = ? AND deleted_at IS NULL`, id)
	return err
}

func (t *saveTXN) EnrollProject(ctx context.Context, project string) error {
	_, err := t.tx.Exec(`
		INSERT INTO sync_enrolled_projects (project) VALUES (?)
		ON CONFLICT(project) DO NOTHING`, project)
	return err
}

func (t *saveTXN) Commit(ctx context.Context) error {
	err := t.tx.Commit()
	if err == nil && t.st.txnHook != nil {
		t.st.txnHook()
	}
	return err
}

func (t *saveTXN) Rollback() error { return t.tx.Rollback() }

func (t *saveTXN) enrolled(project string) bool {
	var n int
	_ = t.tx.QueryRow(`SELECT COUNT(*) FROM sync_enrolled_projects WHERE project = ?`, project).Scan(&n)
	return n > 0
}

func (t *saveTXN) enqueueUpsert(project, syncID string) error {
	_, err := t.tx.Exec(`
		INSERT INTO sync_mutations (target_key, entity, entity_key, op, payload, source, project)
		VALUES ('local', 'observation', ?, 'upsert', ?, 'local', ?)`,
		syncID, `{"sync_id":"`+syncID+`"}`, project)
	return err
}

// syncEnabled reports whether the store's owning config has sync enabled
// (the memory service injects it at service build time; the SQLiteStore
// defaults off so direct-store callers never journal).
func (s *sqliteStore) syncEnabled() bool { return s.syncOn }

// SetSyncEnabled flips the store's sync journaling flag (the memory service
// sets it from cfg.Sync.Enabled — the store itself owns no config).
func (s *sqliteStore) SetSyncEnabled(on bool) { s.syncOn = on }

// BeginSaveTxn opens the save transaction — the gate's entry (the memory
// package's gateAdapter seam; the handle implements SaveTXN).
func (s *sqliteStore) BeginSaveTxn(ctx context.Context) (SaveTXN, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &saveTXN{st: s, tx: tx}, nil
}
