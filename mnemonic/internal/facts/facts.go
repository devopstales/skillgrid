// Package facts is the Fact Memory module: durable, retrievable facts kept
// beside observations (change 2026-09-04-hermes-memory). Add, Search, Forget,
// Decay, and DecayAll live in scope.go with the scope routing (project /
// global / all) and the two-target category (memory / user, migration 072,
// change 2026-10-24-global-llm-wiki); this file holds the Store handle and
// the read-only listing (List / ListWith).
package facts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// DB returns the wrapped *sql.DB (tests use it to assert on the routed
// store's rows; production code works through the store's methods).
func (s *Store) DB() *sql.DB { return s.db }

// Fact is one durable fact row (TICKET-02 read model).
type Fact struct {
	ID              int64   `json:"id"`
	Content         string  `json:"content"`
	ImportanceScore float64 `json:"importance_score"`
	RecencyDecay    float64 `json:"recency_decay"`
	MaturityTier    string  `json:"maturity_tier"`
	RetrievalUsage  int     `json:"retrieval_usage"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// defaultSearchLimit caps a Search call when the caller passes limit <= 0.
const defaultSearchLimit = 20

// List returns every live (non-soft-deleted) fact, newest first (the CLI
// `memory list` read path; it writes no session_events trail, mirroring the
// no-trail convention of the other read-only registry listing). Soft-deleted
// rows are omitted by default.
func (s *Store) List(ctx context.Context, limit int) ([]Fact, error) {
	return s.ListWith(ctx, limit, false)
}

// ListWith is List with an explicit soft-delete filter: includeDeleted lifts
// the deleted_at IS NULL clause (the audit/escape-hatch read).
func (s *Store) ListWith(ctx context.Context, limit int, includeDeleted bool) ([]Fact, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("facts store not initialized")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	deletedClause := ""
	if !includeDeleted {
		deletedClause = " AND deleted_at IS NULL"
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content, importance_score, recency_decay, maturity_tier,
		       retrieval_usage, created_at, updated_at
		FROM facts
		WHERE 1 = 1`+deletedClause+`
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list facts: %w", err)
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
		return nil, fmt.Errorf("iterate fact list: %w", err)
	}
	return out, nil
}

// DefaultPurgeThreshold is the below-threshold purge cutoff for DecayAll:
// facts whose post-decay importance_score drops below this are soft-deleted.
// It mirrors the 0.5 prune threshold the Dream Executor uses for observations
// (memory.dream.go, 014 step 12.3) — the same "low-importance" bar, so a fact
// and an observation both have to clear 0.5 to survive decay+purge.
const DefaultPurgeThreshold = 0.5
