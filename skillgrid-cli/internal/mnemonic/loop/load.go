package loop

import (
	"context"
	"database/sql"
	"strings"
)

// LastStep is the resume handle stored on the newest session row.
type LastStep struct {
	Task      string
	StoppedAt string
	OneLiner  string
}

// LoadLastStep reads the newest session for project. When sessionID is set,
// that row wins. A missing row is an empty step, not an error.
func LoadLastStep(ctx context.Context, db *sql.DB, project, sessionID string) (LastStep, error) {
	var step LastStep
	if db == nil {
		return step, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	row := db.QueryRowContext(ctx, `
		SELECT COALESCE(title, ''), COALESCE(summary, ''), COALESCE(ended_at, started_at, '')
		FROM sessions
		WHERE project = ? AND (? = '' OR id = ?)
		ORDER BY CASE WHEN id = ? THEN 0 ELSE 1 END,
		         COALESCE(ended_at, started_at) DESC
		LIMIT 1`, project, sessionID, sessionID, sessionID)
	var title, summary, stopped string
	if err := row.Scan(&title, &summary, &stopped); err != nil {
		if err == sql.ErrNoRows {
			return step, nil
		}
		return step, err
	}
	step.Task = strings.TrimSpace(title)
	if step.Task == "" {
		step.Task = OneLinerFromSummary(summary)
	}
	step.StoppedAt = strings.TrimSpace(stopped)
	step.OneLiner = OneLinerFromSummary(summary)
	return step, nil
}
