package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// EnsureSession registers a harness session (Cursor conversation id, OpenCode
// session id, …) under projectID when no row exists, and records agent and
// title on an existing row only where they are blank. Unlike
// SessionStartByClientID it never re-resolves the project from directory: the
// caller already opened the store for projectID, and the capture hook and the
// session-start hook must land on the same row.
func (s *Service) EnsureSession(ctx context.Context, sessionID, projectID, directory, title, agent string) (created bool, err error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return false, errors.New("memory service not initialized")
	}
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" {
		return false, errors.New("session id is required")
	}
	if projectID == "" {
		return false, errors.New("project is required")
	}
	agent = strings.TrimSpace(agent)
	title = strings.TrimSpace(title)

	_, _, lerr := lookupSessionRow(ctx, s.store.DB, sessionID)
	if lerr == nil {
		_, _ = s.store.DB.ExecContext(ctx, `
			UPDATE sessions
			SET agent = COALESCE(NULLIF(TRIM(agent), ''), NULLIF(?, '')),
			    title = COALESCE(NULLIF(TRIM(title), ''), NULLIF(?, ''))
			WHERE id = ?`,
			agent, title, sessionID)
		return false, nil
	}
	if !errors.Is(lerr, sql.ErrNoRows) {
		return false, fmt.Errorf("ensure session look-up: %w", lerr)
	}

	absDir := strings.TrimSpace(directory)
	if absDir != "" {
		if a, aerr := filepath.Abs(absDir); aerr == nil {
			absDir = a
		}
	}
	head := gitHead(absDir)
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("ensure session begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO sessions (id, project, directory, title, started_at, status, agent_session_id, from_commit, agent)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, 'active', ?, ?, NULLIF(?, ''))`,
		sessionID, projectID, absDir, title, now, sessionID, head, agent,
	); err != nil {
		return false, fmt.Errorf("insert session: %w", err)
	}
	if _, err = appendSessionEvent(ctx, tx, projectID, sessionID, "session_start", head, now); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("ensure session commit: %w", err)
	}
	committed = true
	return true, nil
}
