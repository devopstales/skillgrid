package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

func registerQueryEventTools(s *server.MCPServer) {
	s.AddTool(queryEventsTool(), handleQueryEvents)
	s.AddTool(exportEventsTool(), handleExportEvents)
}

func queryEventsTool() mcplib.Tool {
	return mcplib.NewTool("mem_query_events",
		mcplib.WithDescription("Query session tool-call events with filters. Sensitive events excluded by default."),
		mcplib.WithString("action", mcplib.Description("Filter by action_type (file_read, file_write, command_exec, tool_use, commit)")),
		mcplib.WithString("file", mcplib.Description("Filter by path LIKE pattern (e.g. 'src/%')")),
		mcplib.WithString("since", mcplib.Description("RFC3339 lower bound (inclusive)")),
		mcplib.WithString("until", mcplib.Description("RFC3339 upper bound (inclusive)")),
		mcplib.WithString("session", mcplib.Description("Filter by session_id")),
		mcplib.WithString("agent", mcplib.Description("Filter by owning harness (cursor, opencode, kilo)")),
		mcplib.WithString("tool", mcplib.Description("Filter by exact tool name (case-insensitive)")),
		mcplib.WithString("command", mcplib.Description("Filter by command LIKE pattern (e.g. 'npm %')")),
		mcplib.WithBoolean("count", mcplib.Description("Return count only, no events")),
		mcplib.WithBoolean("sensitive", mcplib.Description("Include sensitive events")),
	)
}

func exportEventsTool() mcplib.Tool {
	return mcplib.NewTool("mem_export_events",
		mcplib.WithDescription("Export session tool-call events as JSONL (one JSON object per line). Sensitive events excluded by default."),
		mcplib.WithString("action", mcplib.Description("Filter by action_type (file_read, file_write, command_exec, tool_use, commit)")),
		mcplib.WithString("file", mcplib.Description("Filter by path LIKE pattern (e.g. 'src/%')")),
		mcplib.WithString("since", mcplib.Description("RFC3339 lower bound (inclusive)")),
		mcplib.WithString("until", mcplib.Description("RFC3339 upper bound (inclusive)")),
		mcplib.WithString("session", mcplib.Description("Filter by session_id")),
		mcplib.WithString("agent", mcplib.Description("Filter by owning harness (cursor, opencode, kilo)")),
		mcplib.WithString("tool", mcplib.Description("Filter by exact tool name (case-insensitive)")),
		mcplib.WithString("command", mcplib.Description("Filter by command LIKE pattern (e.g. 'npm %')")),
		mcplib.WithBoolean("sensitive", mcplib.Description("Include sensitive events")),
	)
}

func eventOptsFromReq(req mcplib.CallToolRequest) memory.QueryEventOpts {
	return memory.QueryEventOpts{
		Action:           req.GetString("action", ""),
		File:             req.GetString("file", ""),
		Since:            req.GetString("since", ""),
		Until:            req.GetString("until", ""),
		Session:          req.GetString("session", ""),
		Agent:            req.GetString("agent", ""),
		Tool:             req.GetString("tool", ""),
		Command:          req.GetString("command", ""),
		IncludeSensitive: req.GetBool("sensitive", false),
		CountOnly:        req.GetBool("count", false),
	}
}

func handleQueryEvents(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	opts := eventOptsFromReq(req)
	events, count, err := h.Memory().QueryEvents(ctx, opts)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"events":             events,
		"count":              count,
		"sensitive_included": opts.IncludeSensitive,
	})
}

func handleExportEvents(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	opts := eventOptsFromReq(req)
	jsonl, err := h.Memory().ExportEvents(ctx, opts)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"jsonl": jsonl})
}
