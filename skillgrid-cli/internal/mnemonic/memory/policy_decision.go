package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ActionForTool maps a harness tool name (and optional action type) to the
// stored action: file_read, file_write, command_exec, or tool_use.
func ActionForTool(toolName, actionType string) string { return actionForTool(toolName, actionType) }

// PolicyDecision is one non-allow policy answer to record (ADR-0021).
type PolicyDecision struct {
	SessionID string
	Action    string
	Tool      string
	Path      string
	Command   string
	Result    string // blocked | warned | guided
	Rule      string
	Message   string
}

// RecordPolicyDecision appends the decision to the session's event stream with
// result_status blocked, warned, or guided. It does not bump the read, write,
// or command counters (the call itself is recorded separately if it runs); a
// block bumps blocked_actions.
func (s *Service) RecordPolicyDecision(ctx context.Context, d PolicyDecision) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	switch d.Result {
	case "blocked", "warned", "guided":
	default:
		return fmt.Errorf("policy decision result %q (want blocked, warned, guided)", d.Result)
	}
	sessionID := strings.TrimSpace(d.SessionID)
	rowProject, _, err := lookupSessionRow(ctx, s.store.DB, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("session %s not found", sessionID)
		}
		return err
	}
	action := canonicalAction(d.Action)
	if action == "" {
		action = actionForTool(d.Tool, "")
	}
	path := strings.TrimSpace(d.Path)
	sensitive := IsSensitivePath(path)
	payload, _ := json.Marshal(map[string]string{"policy_rule": d.Rule, "message": d.Message})
	now := eventNow()

	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	seq, err := appendSessionEvent(ctx, tx, rowProject, sessionID, action, "", now)
	if err != nil {
		return err
	}
	sensFlag := 0
	if sensitive {
		sensFlag = 1
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE session_events
		SET result_status = ?, is_sensitive = ?, tool_name = ?, path = ?, command = ?, payload = ?
		WHERE session_id = ? AND sequence = ?`,
		d.Result, sensFlag, strings.TrimSpace(d.Tool), path, strings.TrimSpace(d.Command),
		string(payload), sessionID, seq,
	); err != nil {
		return fmt.Errorf("policy decision event: %w", err)
	}
	if d.Result == "blocked" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE sessions SET blocked_actions = blocked_actions + 1 WHERE id = ?`, sessionID); err != nil {
			return fmt.Errorf("policy decision counter: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// SessionCounters returns the totals a policy rule may threshold on. An
// unknown session returns zeros.
func (s *Service) SessionCounters(ctx context.Context, sessionID string) map[string]int {
	out := map[string]int{
		"tool_calls": 0, "files_read": 0, "files_written": 0,
		"commands_exec": 0, "errors": 0, "blocked_actions": 0,
	}
	if s == nil || s.store == nil || strings.TrimSpace(sessionID) == "" {
		return out
	}
	var calls, reads, writes, cmds, errs, blocked int
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM session_events e WHERE e.session_id = s.id
		     AND e.action_type NOT IN ('session_start','session_end','commit')
		     AND COALESCE(e.result_status,'success') NOT IN ('blocked','warned','guided')),
		  COALESCE(s.files_read,0), COALESCE(s.files_written,0), COALESCE(s.commands_exec,0),
		  COALESCE(s.errors,0), COALESCE(s.blocked_actions,0)
		FROM sessions s WHERE s.id = ?`, sessionID).Scan(&calls, &reads, &writes, &cmds, &errs, &blocked)
	if err != nil {
		return out
	}
	out["tool_calls"], out["files_read"], out["files_written"] = calls, reads, writes
	out["commands_exec"], out["errors"], out["blocked_actions"] = cmds, errs, blocked
	return out
}
