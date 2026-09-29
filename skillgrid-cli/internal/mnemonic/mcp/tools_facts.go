package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
)

func registerFactTools(s *server.MCPServer) {
	s.AddTool(factAddTool(), handleFactAdd)
}

func factAddTool() mcplib.Tool {
	return mcplib.NewTool("fact_add",
		mcplib.WithDescription("Add a durable fact to Fact Memory and record a session_events trail (action_type=fact_add). Returns the new fact id."),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Fact text (required)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleFactAdd(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := facts.New(h.Store().DB, projectID).Add(ctx, sessionID, content)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id":    id,
		"session_id": sessionID,
		"project":    projectID,
		"event":      "fact_add",
	})
}
