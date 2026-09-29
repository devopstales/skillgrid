// Package facts is the Fact Memory module: durable, retrievable facts kept
// beside observations (change 2026-09-04-hermes-memory). TICKET-01 ships Add;
// TICKET-02 adds Search (lexical FTS + soft-delete filter), Forget (soft
// delete), and Decay (014 AKL importance reuse).
package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Store is the Fact Memory handle over one project store.
type Store struct {
	db      *sql.DB
	project string
}

// New wraps the project store's database handle. project is stamped on
// session_events trail rows.
func New(db *sql.DB, project string) *Store {
	return &Store{db: db, project: project}
}

// Fact is one durable fact row (TICKET-02 read model).
type Fact struct {
	ID               int64   `json:"id"`
	Content          string  `json:"content"`
	ImportanceScore  float64 `json:"importance_score"`
	RecencyDecay     float64 `json:"recency_decay"`
	MaturityTier     string  `json:"maturity_tier"`
	RetrievalUsage   int     `json:"retrieval_usage"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// defaultSearchLimit caps a Search call when the caller passes limit <= 0.
const defaultSearchLimit = 20

// Add inserts one fact and records a session_events trail
// (action_type="fact_add", payload carries the fact id). The fact insert and
// the event insert commit atomically: if the session is unknown (FK on
// session_events.session_id) the fact is rolled back with it.
func (s *Store) Add(ctx context.Context, sessionID, content string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("facts store not initialized")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, errors.New("fact content is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin fact add: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO facts (content, created_at, updated_at)
		VALUES (?, ?, ?)`, content, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert fact: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("fact id: %w", err)
	}

	payload, err := json.Marshal(map[string]any{"fact_id": id})
	if err != nil {
		return 0, fmt.Errorf("marshal fact payload: %w", err)
	}
	var seq int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return 0, fmt.Errorf("next event sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, payload, timestamp)
		VALUES (?, ?, ?, 'fact_add', 'success', ?, ?)`,
		sessionID, s.project, seq, string(payload), now,
	); err != nil {
		return 0, fmt.Errorf("insert fact_add event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit fact add: %w", err)
	}
	return id, nil
}

// Search is the default lexical fact search (TICKET-02): FTS5 over
// facts_fts, bm25-ranked, soft-deleted facts excluded (deleted_at IS NULL).
// Every call appends a session_events trail (action_type="fact_search",
// payload carries the matched fact ids, the query, and mode="fts"). With no
// matches nothing is written (no dangling events, mirroring the no-op
// convention of the observations search path).
func (s *Store) Search(ctx context.Context, sessionID, query string, limit int) ([]Fact, error) {
	return s.SearchWith(ctx, sessionID, query, limit, false)
}

// SearchWith is Search with an explicit soft-delete filter: includeDeleted
// lifts the deleted_at IS NULL clause (the audit/escape-hatch read).
func (s *Store) SearchWith(ctx context.Context, sessionID, query string, limit int, includeDeleted bool) ([]Fact, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("facts store not initialized")
	}
	ftsQuery, err := buildFactFTSQuery(query)
	if err != nil {
		return nil, err
	}
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	deletedClause := ""
	if !includeDeleted {
		deletedClause = " AND f.deleted_at IS NULL"
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.content, f.importance_score, f.recency_decay, f.maturity_tier,
		       f.retrieval_usage, f.created_at, f.updated_at
		FROM facts f
		INNER JOIN facts_fts ON facts_fts.rowid = f.id
		WHERE facts_fts MATCH ?`+deletedClause+`
		ORDER BY bm25(facts_fts)
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search facts: %w", err)
	}
	defer rows.Close()
	var out []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Content, &f.ImportanceScore, &f.RecencyDecay,
			&f.MaturityTier, &f.RetrievalUsage, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fact search: %w", err)
	}
	if len(out) > 0 {
		ids := make([]int64, len(out))
		for i, f := range out {
			ids[i] = f.ID
		}
		if err := s.insertEvent(ctx, sessionID, "fact_search", map[string]any{
			"query":    query,
			"mode":     "fts",
			"fact_ids": ids,
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Forget soft-deletes a fact (TICKET-02): deleted_at = now, updated_at
// refreshed. The row survives for the audit trail; default search excludes
// it afterwards. Forgetting an already-forgotten fact is a no-op; an unknown
// id is an error and logs no event. The success trail row carries the fact
// id (action_type="fact_forget").
func (s *Store) Forget(ctx context.Context, sessionID string, factID int64) error {
	if s == nil || s.db == nil {
		return errors.New("facts store not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE facts SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`, now, now, factID)
	if err != nil {
		return fmt.Errorf("forget fact: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		if err := s.db.QueryRowContext(ctx,
			`SELECT 1 FROM facts WHERE id = ?`, factID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("forget fact: unknown fact id %d", factID)
			}
			return fmt.Errorf("forget fact: %w", err)
		}
		return nil // already soft-deleted: idempotent no-op
	}
	if err := s.insertEvent(ctx, sessionID, "fact_forget", map[string]any{
		"fact_id": factID,
	}); err != nil {
		return err
	}
	return nil
}

// Decay applies the 014 AKL importance decay to one fact (TICKET-02):
//
//	importance_score = importance_score * exp(-decay_rate * age_days)
//
// reusing the memory package's recencyFactor (the same exp(-decay_rate *
// age_days) the 014 observation scoring uses — not reinvented). The
// recency_decay column keeps its 014 contract (the decay RATE, not the
// factor). A zero rate (the fresh-fact default) is a no-op that still logs
// the trail row; an unknown id is an error. The success trail row
// (action_type="fact_decay") carries the fact id, the old score (importance
// before decay), the new score, and the mode ("akl").
func (s *Store) Decay(ctx context.Context, sessionID string, factID int64) (float64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("facts store not initialized")
	}
	var score, rate float64
	var createdAt string
	if err := s.db.QueryRowContext(ctx, `
		SELECT importance_score, recency_decay, created_at
		FROM facts WHERE id = ?`, factID).
		Scan(&score, &rate, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("decay fact: unknown fact id %d", factID)
		}
		return 0, fmt.Errorf("decay fact: %w", err)
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil || created.IsZero() {
		created = time.Now().UTC()
	}
	newScore := score * memory.RecencyFactor(time.Since(created), rate)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE facts SET importance_score = ?, updated_at = ? WHERE id = ?`,
		newScore, now, factID); err != nil {
		return 0, fmt.Errorf("decay fact update: %w", err)
	}
	if err := s.insertEvent(ctx, sessionID, "fact_decay", map[string]any{
		"fact_id":   factID,
		"old_score": score,
		"new_score": newScore,
		"mode":      "akl",
	}); err != nil {
		return 0, err
	}
	return newScore, nil
}

// insertEvent appends one session_events trail row with the next sequence
// for the session (the same monotonic-sequence convention as Add).
func (s *Store) insertEvent(ctx context.Context, sessionID, actionType string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", actionType, err)
	}
	var seq int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return fmt.Errorf("next event sequence: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, payload, timestamp)
		VALUES (?, ?, ?, ?, 'success', ?, ?)`,
		sessionID, s.project, seq, actionType, string(data), now,
	); err != nil {
		return fmt.Errorf("insert %s event: %w", actionType, err)
	}
	return nil
}

// buildFactFTSQuery converts a plain-text query into a safe FTS5 MATCH
// expression: terms are quoted (FTS special characters can't break the
// syntax), OR-joined (any-term recall, the default search mode). Returns ""
// for a blank query.
func buildFactFTSQuery(query string) (string, error) {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return "", nil
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		escaped[i] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(escaped, " OR "), nil
}
