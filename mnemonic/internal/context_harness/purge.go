package context_harness

import (
	"database/sql"
	"errors"
	"fmt"
)

// contextRevisionsTable is the revisions table added by a later migration
// (055, a subsequent PR in the context-harness chain). It does not exist yet in
// PR 2, so PurgeSession probes for it and purges fail-open when absent.
const contextRevisionsTable = "context_revisions"

// PurgeSession clears the session-scoped context-harness sandbox for one
// session: it deletes tool_outputs rows for (project, session) and, if the
// context_revisions table exists, that session's revision rows too. It returns
// the total rows deleted across both tables.
//
// Fail-open by design: a table that is absent is skipped (not an error), so the
// same purge works across the PR-2 store (tool_outputs only) and the later
// store (tool_outputs + context_revisions) without callers branching.
func PurgeSession(db *sql.DB, project, sessionID string) (int, error) {
	if db == nil {
		return 0, errors.New("purge: nil db")
	}
	total := 0

	res, err := db.Exec(
		`DELETE FROM tool_outputs WHERE project_id = ? AND session_id = ?`,
		project, sessionID,
	)
	if err != nil {
		return 0, fmt.Errorf("purge tool_outputs: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		total += int(n)
	}

	// Defensive: context_revisions does not exist yet in PR 2 (migration 055
	// is a later PR). Probe sqlite_master and purge it only when present, so
	// the call is forward-compatible and stays fail-open.
	var present int
	if err := db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		contextRevisionsTable,
	).Scan(&present); err != nil {
		return total, fmt.Errorf("purge probe %s: %w", contextRevisionsTable, err)
	}
	if present > 0 {
		res, err := db.Exec(
			`DELETE FROM context_revisions WHERE project_id = ? AND session_id = ?`,
			project, sessionID,
		)
		if err != nil {
			return total, fmt.Errorf("purge %s: %w", contextRevisionsTable, err)
		}
		if n, err := res.RowsAffected(); err == nil {
			total += int(n)
		}
	}
	return total, nil
}
