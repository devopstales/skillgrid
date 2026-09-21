package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Event is one row of the session_events stream: the append-only,
// per-session log ordered by (session_id, sequence). session_start is always
// sequence 0; session_end is always last.
type Event struct {
	ID           int64  `json:"id"`
	SessionID    string `json:"session_id"`
	Project      string `json:"project"`
	Sequence     int    `json:"sequence"`
	ActionType   string `json:"action_type"`
	ResultStatus string `json:"result_status"`
	IsSensitive  bool   `json:"is_sensitive"`
	ToolName     string `json:"tool_name,omitempty"`
	Path         string `json:"path,omitempty"`
	Command      string `json:"command,omitempty"`
	Commit       string `json:"commit,omitempty"`
	Payload      string `json:"payload,omitempty"`
	Timestamp    string `json:"timestamp"`
}

// gitHead returns `git rev-parse HEAD` in dir. Best-effort by contract: it
// returns "" outside a git repository (or when git is missing) and never an
// error, so session lifecycle never fails on commit capture.
func gitHead(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// eventNow returns the UTC timestamp stamped on session_events rows.
func eventNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// nextSequenceTx returns the next per-session sequence
// (SELECT COALESCE(MAX(sequence),-1)+1 WHERE session_id=?) on tx. It must run
// inside the writer's transaction so the event insert and any counter bump
// commit atomically.
func nextSequenceTx(ctx context.Context, tx *sql.Tx, sessionID string) (int, error) {
	var seq int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return 0, fmt.Errorf("next sequence: %w", err)
	}
	return seq, nil
}

// appendSessionEvent inserts one session_events row on tx at the next sequence.
// commit carries the git HEAD the event was captured at ("" when unknown).
func appendSessionEvent(ctx context.Context, tx *sql.Tx, project, sessionID, actionType, commit, now string) error {
	seq, err := nextSequenceTx(ctx, tx, sessionID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, "commit", timestamp)
		VALUES (?, ?, ?, ?, 'success', ?, ?)`,
		sessionID, project, seq, actionType, commit, now,
	); err != nil {
		return fmt.Errorf("insert %s event: %w", actionType, err)
	}
	return nil
}

// stampToCommit best-effort sets sessions.to_commit to the git HEAD in dir,
// but only when the row has no end commit yet — it never clobbers a recorded
// range. Runs on tx so it commits with the writer's event insert.
func stampToCommit(ctx context.Context, tx *sql.Tx, sessionID, rowProject, dir string) {
	head := gitHead(dir)
	if head == "" {
		return
	}
	_, _ = tx.ExecContext(ctx, `
		UPDATE sessions SET to_commit = ?
		WHERE id = ? AND project = ? AND (to_commit IS NULL OR to_commit = '')`,
		head, sessionID, rowProject,
	)
}

// lookupSessionRow returns the sessions row's project and directory for id,
// scoped to any project. It reports sql.ErrNoRows when the session is unknown.
func lookupSessionRow(ctx context.Context, db *sql.DB, sessionID string) (rowProject, dir string, err error) {
	err = db.QueryRowContext(ctx,
		`SELECT project, directory FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&rowProject, &dir)
	return rowProject, dir, err
}

// SessionChanges returns the session's event stream in sequence order plus the
// net commit range (from_commit at start, to_commit at end) from the sessions
// row. It is the resume read path: a fresh session recovers position from the
// ordered events and the range alone. An unknown session id is a
// session-not-found error.
func (s *Service) SessionChanges(ctx context.Context, sessionID string) (events []Event, from, to string, err error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, "", "", errors.New("memory service not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, "", "", errors.New("session_id is required")
	}
	var fromNull, toNull sql.NullString
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT from_commit, to_commit FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&fromNull, &toNull); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", "", fmt.Errorf("session %s not found", sessionID)
		}
		return nil, "", "", fmt.Errorf("session changes lookup: %w", err)
	}
	if fromNull.Valid {
		from = fromNull.String
	}
	if toNull.Valid {
		to = toNull.String
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, session_id, project, sequence, action_type,
		       COALESCE(result_status, 'success'), COALESCE(is_sensitive, 0),
		       COALESCE(tool_name, ''), COALESCE(path, ''), COALESCE(command, ''),
		       COALESCE("commit", ''), COALESCE(payload, ''), timestamp
		FROM session_events
		WHERE session_id = ?
		ORDER BY sequence`,
		sessionID,
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("session changes events: %w", err)
	}
	defer rows.Close()
	events = []Event{}
	for rows.Next() {
		var e Event
		var sensitive int
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.Project, &e.Sequence, &e.ActionType,
			&e.ResultStatus, &sensitive,
			&e.ToolName, &e.Path, &e.Command, &e.Commit, &e.Payload, &e.Timestamp,
		); err != nil {
			return nil, "", "", fmt.Errorf("scan session event: %w", err)
		}
		e.IsSensitive = sensitive != 0
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", "", fmt.Errorf("iterate session events: %w", err)
	}
	return events, from, to, nil
}
