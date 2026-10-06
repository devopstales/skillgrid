package mcp

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
)

func registerFactTools(s *server.MCPServer) {
	s.AddTool(factAddTool(), handleFactAdd)
	s.AddTool(factSearchTool(), handleFactSearch)
	s.AddTool(factForgetTool(), handleFactForget)
	s.AddTool(factDecayTool(), handleFactDecay)
	s.AddTool(factDecayAllTool(), handleFactDecayAll)
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

func factSearchTool() mcplib.Tool {
	return mcplib.NewTool("fact_search",
		mcplib.WithDescription("Lexical FTS search over Fact Memory (soft-deleted facts excluded). Returns matching facts ranked by bm25 and records a session_events trail (action_type=fact_search) with the matched fact ids."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms (any-term recall)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithNumber("limit", mcplib.Description("Max results (default 20)")),
		mcplib.WithBoolean("include_deleted", mcplib.Description("Include soft-deleted facts (default false)")),
	)
}

func handleFactSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
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

	f, err := facts.New(h.Store().DB, projectID).SearchWith(ctx, sessionID, query, limit, includeDeleted)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"facts": f,
		"count": len(f),
	})
}

func factForgetTool() mcplib.Tool {
	return mcplib.NewTool("fact_forget",
		mcplib.WithDescription("Soft-delete a fact (sets deleted_at; the row survives for the audit trail and default search excludes it afterwards). Records a session_events trail (action_type=fact_forget). Forgetting an already-forgotten fact is a no-op."),
		mcplib.WithNumber("fact_id", mcplib.Required(), mcplib.Description("Fact id to forget (JSON number)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleFactForget(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	factID, err := req.RequireInt("fact_id")
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

	if err := facts.New(h.Store().DB, projectID).Forget(ctx, sessionID, int64(factID)); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id": int64(factID),
		"event":   "fact_forget",
	})
}

func factDecayTool() mcplib.Tool {
	return mcplib.NewTool("fact_decay",
		mcplib.WithDescription("Apply the 014 AKL importance decay to one fact: importance_score *= exp(-decay_rate * age_days) (the decay rate is the fact's stored recency_decay value). Returns the new score and records a session_events trail (action_type=fact_decay)."),
		mcplib.WithNumber("fact_id", mcplib.Required(), mcplib.Description("Fact id to decay (JSON number)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleFactDecay(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	factID, err := req.RequireInt("fact_id")
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

	newScore, err := facts.New(h.Store().DB, projectID).Decay(ctx, sessionID, int64(factID))
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id":    int64(factID),
		"new_score":  newScore,
		"event":      "fact_decay",
	})
}

func factDecayAllTool() mcplib.Tool {
	return mcplib.NewTool("fact_decay_all",
		mcplib.WithDescription("Run the batch decay + below-threshold purge over ALL live facts (acceptance scenario 5: 'Decay via 014 AKL, logs event, purge below threshold'). Applies the 014 AKL decay to every non-deleted fact, then soft-deletes facts whose post-decay importance_score drops below threshold. Returns {decayed, purged} and records a single session_events trail (action_type=fact_decay_batch)."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the batch trail event to (must exist in this project)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		mcplib.WithNumber("threshold", mcplib.Description("Purge cutoff for importance_score (default 0.5, the 014 dream prune threshold). Facts decaying below this are soft-deleted.")),
	)
}

func handleFactDecayAll(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}
	threshold := req.GetFloat("threshold", facts.DefaultPurgeThreshold)
	if threshold < 0 {
		return toolError(fmt.Errorf("threshold must be >= 0, got %v", threshold))
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	decayed, purged, err := facts.New(h.Store().DB, projectID).DecayAll(ctx, sessionID, threshold)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"decayed":   decayed,
		"purged":    purged,
		"threshold": threshold,
		"event":     "fact_decay_batch",
	})
}
