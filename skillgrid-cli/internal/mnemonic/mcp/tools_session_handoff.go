package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay"
)

// registerSessionTools is the session-relay MCP registrar (change 006, step
// 02). It is the step-03 "registrar hook": later session_* tools (status,
// compact) can extend this list without re-editing server.go's registration
// chain. It is additive — it never touches the mem_* / code_* registration.
func registerSessionTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{sessionHandoffTool(), handleSessionHandoff},
		{sessionResumeTool(), handleSessionResume},
		// Step 03: session status + thin knowledge compact (additive).
		{sessionStatusTool(), handleSessionStatus},
		{knowledgeCompactTool(), handleKnowledgeCompact},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func sessionHandoffTool() mcplib.Tool {
	return mcplib.NewTool("session_handoff",
		mcplib.WithDescription("Write a structured session handoff: three .skillgrid/.cleave/ files (PROGRESS / KNOWLEDGE / NEXT_PROMPT) plus a session_handoffs row. Fails closed — if the files cannot be written, no row is recorded (no orphan). Returns the handoff_id and the cleave file paths."),
		mcplib.WithString("progress", mcplib.Required(), mcplib.Description("What was done this session (PROGRESS.md)")),
		mcplib.WithString("next_prompt", mcplib.Required(), mcplib.Description("The prompt that seeds the next session (NEXT_PROMPT.md) — the resume target")),
		mcplib.WithString("knowledge", mcplib.Description("Decisions / learnings (KNOWLEDGE.md). Optional.")),
		mcplib.WithString("session_id", mcplib.Description("Source session id (sessions.id) this handoff is taken from. Optional.")),
		mcplib.WithString("handoff_id", mcplib.Description("Operator-facing handoff identifier / resume target. Blank generates one from the source session + timestamp.")),
		mcplib.WithString("context_summary", mcplib.Description("Optional short context note recorded on the handoff row.")),
	)
}

func handleSessionHandoff(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	progress, err := req.RequireString("progress")
	if err != nil {
		return toolError(err)
	}
	nextPrompt, err := req.RequireString("next_prompt")
	if err != nil {
		return toolError(err)
	}

	in := relay.Bundle{
		Progress:       progress,
		NextPrompt:     nextPrompt,
		Knowledge:      req.GetString("knowledge", ""),
		SourceSession:  req.GetString("session_id", ""),
		ContextSummary: req.GetString("context_summary", ""),
	}

	handoffID, paths, err := relay.Handoff(ctx, h.Store().DB, h.ProjectID(), req.GetString("handoff_id", ""), h.Root(), in)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"handoff_id": handoffID,
		"paths":      paths,
	})
}

func sessionResumeTool() mcplib.Tool {
	return mcplib.NewTool("session_resume",
		mcplib.WithDescription("Resume a prior session by handoff_id: reads the .skillgrid/.cleave/ bundle and returns the stored NEXT_PROMPT as the resume prompt. Fails closed on an unknown handoff id or a missing/incomplete .cleave/ bundle (no invented prompt). Pass archive=true to also record a session_archives row and mark the handoff archived; the archive_id is returned when present."),
		mcplib.WithString("handoff_id", mcplib.Required(), mcplib.Description("The handoff id returned by session_handoff")),
		mcplib.WithBoolean("archive", mcplib.Description("Also archive this handoff (record a session_archives row + flip status to archived). Default false.")),
	)
}

func handleSessionResume(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	handoffID, err := req.RequireString("handoff_id")
	if err != nil {
		return toolError(err)
	}
	archive := req.GetBool("archive", false)

	prompt, id, archiveID, err := relay.Resume(ctx, h.Store().DB, h.ProjectID(), handoffID, h.Root(), archive)
	if err != nil {
		return toolError(err)
	}
	out := map[string]any{
		"prompt":     prompt,
		"handoff_id": id,
	}
	if archive {
		out["archive_id"] = archiveID
	}
	return JSONResult(out)
}
