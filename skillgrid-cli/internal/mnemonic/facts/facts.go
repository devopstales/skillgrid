// Package facts is the Fact Memory module: durable, retrievable facts kept
// beside observations (change 2026-09-04-hermes-memory). Step 02 (TICKET-01)
// ships Add only; Search/Forget/Decay arrive with TICKET-02.
package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
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
