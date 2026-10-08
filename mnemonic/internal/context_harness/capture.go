package context_harness

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// SandboxThreshold is the default byte threshold above which output is gated.
const SandboxThreshold = 4096

// GateDecision is the result of the Output Sandbox Gate decision.
type GateDecision struct {
	Gate    bool
	Reason  string
	Summary string
	Pointer string
}

// SandboxSummary returns at most `limit` chars of output, newline-collapsed.
func SandboxSummary(output string, limit int) string {
	s := strings.Join(strings.Fields(output), " ")
	if len(s) > limit {
		s = s[:limit]
	}
	return s
}

// DecideGate decides whether to gate an output based on actual size + bypass.
func DecideGate(output string, bypass bool, threshold int) GateDecision {
	if bypass {
		return GateDecision{Gate: false, Reason: "bypass"}
	}
	if len(output) <= threshold {
		return GateDecision{Gate: false, Reason: "below threshold"}
	}
	summary := SandboxSummary(output, 200)
	pointer := "ctx_search " + summary
	if len(pointer) > 48 {
		pointer = pointer[:48]
	}
	return GateDecision{
		Gate:    true,
		Reason:  "above threshold",
		Summary: summary,
		Pointer: pointer,
	}
}

// StoreToolOutput inserts a gated output into the sandbox and returns its id.
func StoreToolOutput(ctx context.Context, db *sql.DB, sessionID, projectID, toolName, output string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.ExecContext(ctx,
		`INSERT INTO tool_outputs (session_id, project_id, tool_name, output, size_bytes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, projectID, toolName, output, len(output), now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
