package mcp

import (
	"context"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

func registerFactTools(s *server.MCPServer) {
	s.AddTool(factAddTool(), handleFactAdd)
	s.AddTool(factSearchTool(), handleFactSearch)
	s.AddTool(factForgetTool(), handleFactForget)
	s.AddTool(factDecayTool(), handleFactDecay)
	s.AddTool(factDecayAllTool(), handleFactDecayAll)
}

// factScopeParam and factTargetParam are the two params every fact_* tool
// accepts (change 2026-10-24-global-llm-wiki, TICKET-02): scope routes the
// operation to the project or __global__ store; target maps to the facts
// category (memory/user) on the write ops.
// factScopeParam is the scope param option shared by every fact_* tool.
func factScopeParam() mcplib.ToolOption {
	return mcplib.WithString("scope", mcplib.Description("Store scope: 'project' (default) or 'global' (the cross-project fact store). 'all' is read-only (fused search, coming in a later change)."))
}

// factTargetParam is the target param option shared by the write ops.
func factTargetParam() mcplib.ToolOption {
	return mcplib.WithString("target", mcplib.Description("Fact target: 'memory' (default, agent operational notes) or 'user' (user profile). Maps to the facts category."))
}

func factAddTool() mcplib.Tool {
	return mcplib.NewTool("fact_add",
		mcplib.WithDescription("Add a durable fact to Fact Memory and record a session_events trail (action_type=fact_add). With scope=global the fact lands in the cross-project __global__ store and the trail in the caller's project store. Returns the new fact id."),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Fact text (required)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in the project store; for scope=global the trail lands there, so the session must exist there)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
		factTargetParam(),
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
	scope, err := factScopeFor(req.GetString("scope", ""))
	if err != nil {
		return toolError(err)
	}
	target, err := facts.CategoryForTarget(req.GetString("target", ""))
	if err != nil {
		return toolError(err)
	}

	data, trail, cleanup, err := openFactStores(svc, projectID, scope)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := data.AddWith(ctx, sessionID, content, target, trail)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id":    id,
		"session_id": sessionID,
		"project":    projectID,
		"scope":      string(scope),
		"event":      "fact_add",
	})
}

func factSearchTool() mcplib.Tool {
	return mcplib.NewTool("fact_search",
		mcplib.WithDescription("Lexical FTS search over Fact Memory (soft-deleted facts excluded). With scope=global the search reads the cross-project __global__ store and trails to the caller's project store. Returns matching facts ranked by bm25 and records a session_events trail (action_type=fact_search) with the matched fact ids."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms (any-term recall)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in the project store)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
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
	scope, err := factScopeFor(req.GetString("scope", ""))
	if err != nil {
		return toolError(err)
	}

	data, trail, cleanup, err := openFactStores(svc, projectID, scope)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	f, err := data.SearchWith(ctx, sessionID, query, limit, includeDeleted, trail)
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
		mcplib.WithDescription("Soft-delete a fact (sets deleted_at; the row survives for the audit trail and default search excludes it afterwards). With scope=global the fact is looked up in the cross-project __global__ store and the trail lands in the caller's project store. Records a session_events trail (action_type=fact_forget). Forgetting an already-forgotten fact is a no-op."),
		mcplib.WithNumber("fact_id", mcplib.Required(), mcplib.Description("Fact id to forget (JSON number)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in the project store)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
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
	scope, err := factScopeFor(req.GetString("scope", ""))
	if err != nil {
		return toolError(err)
	}

	data, trail, cleanup, err := openFactStores(svc, projectID, scope)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	if err := data.Forget(ctx, sessionID, int64(factID), trail); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id": int64(factID),
		"event":   "fact_forget",
	})
}

func factDecayTool() mcplib.Tool {
	return mcplib.NewTool("fact_decay",
		mcplib.WithDescription("Apply the 014 AKL importance decay to one fact: importance_score *= exp(-decay_rate * age_days) (the decay rate is the fact's stored recency_decay value). With scope=global the fact is looked up in the cross-project __global__ store and the trail lands in the caller's project store. Returns the new score and records a session_events trail (action_type=fact_decay)."),
		mcplib.WithNumber("fact_id", mcplib.Required(), mcplib.Description("Fact id to decay (JSON number)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in the project store)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
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
	scope, err := factScopeFor(req.GetString("scope", ""))
	if err != nil {
		return toolError(err)
	}

	data, trail, cleanup, err := openFactStores(svc, projectID, scope)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	newScore, err := data.Decay(ctx, sessionID, int64(factID), trail)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"fact_id":   int64(factID),
		"new_score": newScore,
		"event":     "fact_decay",
	})
}

func factDecayAllTool() mcplib.Tool {
	return mcplib.NewTool("fact_decay_all",
		mcplib.WithDescription("Run the batch decay + below-threshold purge over ALL live facts (acceptance scenario 5: 'Decay via 014 AKL, logs event, purge below threshold'). With scope=global the pass covers the cross-project __global__ store (project facts untouched) and the trail lands in the caller's project store. Applies the 014 AKL decay to every non-deleted fact, then soft-deletes facts whose post-decay importance_score drops below threshold. Returns {decayed, purged} and records a single session_events trail (action_type=fact_decay_batch)."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the batch trail event to (must exist in the project store)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
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
	scope, err := factScopeFor(req.GetString("scope", ""))
	if err != nil {
		return toolError(err)
	}

	data, trail, cleanup, err := openFactStores(svc, projectID, scope)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	decayed, purged, err := data.DecayAll(ctx, sessionID, threshold, trail)
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

// factScopeFor normalizes the MCP scope param (blank = project, the
// pre-change default) and validates it.
func factScopeFor(raw string) (facts.Scope, error) {
	s := facts.Scope(strings.TrimSpace(raw))
	if s == "" {
		return facts.ScopeProject, nil
	}
	if !s.Valid() {
		return s, fmt.Errorf("unknown scope %q (want project, global, or all)", raw)
	}
	return s, nil
}

// openFactStores opens the stores a scope routes to and returns a
// (dataStore, trailStore) pair plus a cleanup that releases both handles.
// scope=project: both roles are the caller's project store (unchanged
// behavior). scope=global: data is the __global__ store, trail is the
// caller's project store (C3). scope=all errors for write ops (fused search
// lands in a later change).
func openFactStores(svc *service.Service, projectID string, scope facts.Scope) (data, trail *facts.Store, cleanup func(), err error) {
	switch scope {
	case facts.ScopeProject, "":
		h, clean, err := svc.Open(projectID)
		if err != nil {
			return nil, nil, nil, err
		}
		s := facts.New(h.Store().DB, projectID)
		return s, s, clean, nil
	case facts.ScopeGlobal:
		gh, gclean, err := svc.OpenGlobal()
		if err != nil {
			return nil, nil, nil, err
		}
		ph, pclean, err := svc.Open(projectID)
		if err != nil {
			gclean()
			return nil, nil, nil, err
		}
		return facts.New(gh.Store().DB, service.GlobalProjectID),
			facts.New(ph.Store().DB, projectID),
			func() { pclean(); gclean() }, nil
	case facts.ScopeAll:
		return nil, nil, nil, fmt.Errorf("scope 'all' is read-only (fused search); writes need scope project or global")
	default:
		return nil, nil, nil, fmt.Errorf("unknown scope %q (want project, global, or all)", scope)
	}
}
