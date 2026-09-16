package mcp

import (
	"context"
	"errors"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func registerRetrievalTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{semanticSearchTool(), handleSemanticSearch},
		{loadFullDetailsTool(), handleLoadFullDetails},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func semanticSearchTool() mcplib.Tool {
	return mcplib.NewTool("semantic_search",
		mcplib.WithDescription("Ranked L1 overviews (with abstracts) over tiered / long-term memory. Never returns full L2 bodies — use load_full_details for those. Default corpus is long-term memories only."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search query")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithString("corpus", mcplib.Description("ltm (default) or all")),
		mcplib.WithNumber("limit", mcplib.Description("Max results (default 20)")),
	)
}

func loadFullDetailsTool() mcplib.Tool {
	return mcplib.NewTool("load_full_details",
		mcplib.WithDescription("Load full L2 markdown for a path returned by semantic_search."),
		mcplib.WithString("path", mcplib.Required(), mcplib.Description("Full L2 filesystem path")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleSemanticSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	corpus := req.GetString("corpus", "")
	limit := int(req.GetFloat("limit", 20))
	out, err := svc.SemanticSearch(ctx, projectID, query, corpus, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleLoadFullDetails(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	path, err := req.RequireString("path")
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	content, err := svc.LoadFullDetails(ctx, projectID, path)
	if err != nil {
		if errors.Is(err, service.ErrPathNotFound) || strings.Contains(err.Error(), "not found") {
			return toolError(err)
		}
		return toolError(err)
	}
	return JSONResult(map[string]any{"content": content, "path": path})
}
