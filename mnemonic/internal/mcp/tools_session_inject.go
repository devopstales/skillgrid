package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
)

func registerSessionInjectTools(s *server.MCPServer) {
	s.AddTool(memInjectSessionTool(), handleMemInjectSession)
}

func memInjectSessionTool() mcplib.Tool {
	return mcplib.NewTool("mem_inject_session",
		mcplib.WithDescription("On-demand session context injection: hybrid (BM25 FTS + optional semantic vector, RRF-fused) retrieval of injectable observations rendered into a token-cost-annotated context block. Degrades to BM25-only (degraded=true) when no embedder is configured; never a hard fail."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms for the injectable observations")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithBoolean("all_projects", mcplib.Description("Widen retrieval to every project bucket in the data directory (default false)")),
		mcplib.WithNumber("max_tokens", mcplib.Description("Token budget for the context block (default 2000)")),
	)
}

func handleMemInjectSession(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	allProjects := req.GetBool("all_projects", false)
	maxTokens := int(req.GetFloat("max_tokens", 2000))

	res, err := context_harness.HybridRetrieve(ctx, h.Memory(), projectID, query, allProjects, maxTokens)
	if err != nil {
		return toolError(err)
	}
	block := context_harness.RenderContextBlock(res, projectID)
	return JSONResult(map[string]any{
		"block":        block,
		"items":        res.Items,
		"total_tokens": res.TotalTokens,
		"degraded":     res.Degraded,
	})
}

// HandleMemInjectSessionForTest exposes the mem_inject_session handler to
// tests so the registered-handler path can be driven in-process. It is a pure
// alias of the registered handler — no behavior, no new tool surface.
func HandleMemInjectSessionForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleMemInjectSession(ctx, req)
}
