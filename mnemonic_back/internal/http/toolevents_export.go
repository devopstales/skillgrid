package http

import (
	"context"
	"database/sql"
	"time"
)

// ToolEvent, ToolEventFilter, and EventStats are the shapes `skillgrid logs`,
// `skillgrid sessions`, and `skillgrid stats` read straight from the store, so
// the CLI and the HTTP API can never disagree on filtering or MCP detection.
type (
	ToolEvent       = toolEvent
	ToolEventFilter = toolEventFilter
	EventStats      = eventStats
	NameCount       = nameCount
)

// QueryToolEvents is queryToolEvents for callers outside the HTTP server.
func QueryToolEvents(ctx context.Context, db *sql.DB, projectID string, f ToolEventFilter) ([]ToolEvent, error) {
	return queryToolEvents(ctx, db, projectID, f)
}

// ComputeEventStats is computeEventStats with `since` in the shared relative
// or RFC3339 syntax.
func ComputeEventStats(ctx context.Context, db *sql.DB, projectID, since, agent string) (EventStats, error) {
	lower, err := parseSince(since, time.Now())
	if err != nil {
		return EventStats{}, err
	}
	return computeEventStats(ctx, db, projectID, lower, agent)
}

// IsMCPTool reports whether a recorded tool name is an MCP server tool.
func IsMCPTool(name string) bool { return isMCPTool(name) }
