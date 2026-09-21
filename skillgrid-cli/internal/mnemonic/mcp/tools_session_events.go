package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// sessionChangesTool is the session_changes MCP tool (session events layer,
// TICKET-04). It answers what changed in a session from its event stream plus
// the net commit range, so a fresh session can resume without the old
// checkpoint file or hub. Read-only over memory.SessionChanges: events return
// in sequence order with from_commit (start) and to_commit (end). An unknown
// session id fails with a session-not-found error.
func sessionChangesTool() mcplib.Tool {
	return mcplib.NewTool("session_changes",
		mcplib.WithDescription("Return a session's event stream in sequence order plus its net commit range (from_commit at start, to_commit at end), so a fresh session can resume without the old checkpoint file or hub. Fails with a session-not-found error for an unknown session id."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("The session id to read the event stream for (sessions.id).")),
	)
}

func handleSessionChanges(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}

	events, from, to, err := h.Memory().SessionChanges(ctx, sessionID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"session_id":  sessionID,
		"from_commit": from,
		"to_commit":   to,
		"events":      events,
	})
}

// HandleSessionChangesForTest exposes the session_changes handler to the CLI
// package's session tests so the "CLI mirrors MCP on the same store" proof can
// drive the MCP path in-process. It is a pure alias of the registered handler
// — no behavior, no new tool surface.
func HandleSessionChangesForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleSessionChanges(ctx, req)
}
