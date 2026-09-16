package relay

import (
	"context"
	"fmt"
	"strings"
)

// Snapshot is the aggregated session status for a project (change 006, step 03).
// It is the backing type for the session_status MCP tool.
//
// HandoffCount is the number of session_handoffs rows for the project (0 when
// the store is empty — a fresh project reports zero counts, not a crash).
// ContextUsagePercent and CostUSD are the LAST KNOWN cost/context for the
// session, and are ONLY populated when the caller supplies them (the optional
// context_usage_percent? / cost_usd? fields). The relay never invents cost or
// context it does not have: when the caller omits them the pointers are nil.
type Snapshot struct {
	HandoffCount int
	// ContextUsagePercent is the caller-supplied context usage percent (0–100).
	// Nil when the caller did not supply it.
	ContextUsagePercent *float64
	// CostUSD is the caller-supplied last known cost in USD. Nil when the
	// caller did not supply it.
	CostUSD *float64
}

// Stats carries the optional caller-supplied cost/context for Status. Both
// fields are pointers so a nil value means "the caller did not supply this"
// (the relay must not invent cost or context it does not have).
type Stats struct {
	ContextUsagePercent *float64
	CostUSD             *float64
}

// Status aggregates the session handoff count (from session_handoffs) plus the
// last known cost/context when the caller supplies it via stats. It never
// invents cost or context: when stats is nil (or has nil fields) the output
// omits those fields. A project with no handoffs yet yields HandoffCount 0 and
// no error (warn + continue, not a crash).
func Status(ctx context.Context, db Store, projectID string, stats *Stats) (Snapshot, error) {
	if db == nil {
		return Snapshot{}, fmt.Errorf("relay: store is required")
	}
	if strings.TrimSpace(projectID) == "" {
		return Snapshot{}, fmt.Errorf("relay: project id is required")
	}

	n, err := HandoffCount(ctx, db, projectID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("relay: count handoffs: %w", err)
	}

	out := Snapshot{HandoffCount: n}
	if stats != nil {
		out.ContextUsagePercent = stats.ContextUsagePercent
		out.CostUSD = stats.CostUSD
	}
	return out, nil
}
