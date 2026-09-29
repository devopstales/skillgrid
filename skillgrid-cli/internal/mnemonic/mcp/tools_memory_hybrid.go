package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills"
)

func registerMemoryHybridTools(s *server.MCPServer) {
	s.AddTool(memoryHybridSearchTool(), handleMemoryHybridSearch)
}

func memoryHybridSearchTool() mcplib.Tool {
	return mcplib.NewTool("hybrid_search",
		mcplib.WithDescription("Hybrid (BM25 FTS5 + optional vector RRF) search over Fact Memory and the Agent Skill registry. The vector leg uses the project embedder when configured; without one it degrades to BM25-only (never a hard fail). Returns facts and skills ranked side by side with per-leg provenance."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms (any-term recall)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the fact_search trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithNumber("limit", mcplib.Description("Max results per scope (default 20)")),
		mcplib.WithBoolean("include_deleted", mcplib.Description("Include soft-deleted facts (default false)")),
	)
}

func handleMemoryHybridSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}
	limit := int(req.GetFloat("limit", 0))
	includeDeleted := req.GetBool("include_deleted", false)

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	db := h.Store().DB
	res, err := hybrid.SearchMemory(ctx,
		facts.New(db, projectID),
		skills.New(db, h.Root()),
		query, limit,
		hybrid.MemoryOptions{
			Embedder:       embedder.Default(),
			IncludeDeleted: includeDeleted,
			ReadingSession: sessionID,
		})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"query":    res.Query,
		"legs":     res.Legs,
		"facts":    res.Facts,
		"skills":   res.Skills,
		"warnings": res.Warnings,
	})
}
