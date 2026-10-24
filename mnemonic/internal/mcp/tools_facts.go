package mcp

import (
	"context"
	"encoding/json"
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
// 'all' is the fused-search read path (TICKET-03): it searches both stores
// and returns hits interleaved by each store's rank, each labeled with its
// scope and category.
func factScopeParam() mcplib.ToolOption {
	return mcplib.WithString("scope", mcplib.Description("Store scope: 'project' (default) or 'global' (the cross-project fact store). 'all' searches BOTH stores (fused search) and returns hits interleaved by each store's rank, each labeled with scope + category. Read-only."))
}

// factTargetParam is the target param option shared by the write ops.
func factTargetParam() mcplib.ToolOption {
	return mcplib.WithString("target", mcplib.Description("Fact target: 'memory' (default, agent operational notes) or 'user' (user profile). Maps to the facts category."))
}

func factAddTool() mcplib.Tool {
	return mcplib.NewTool("fact_add",
		mcplib.WithDescription("Add a durable fact to Fact Memory and record a session_events trail (action_type=fact_add). With scope=global the fact lands in the cross-project __global__ store and the trail in the caller's project store. Returns the new fact id. Alternatively, pass an `operations` array (add/patch/remove) to apply a batch as a unit — the global store's char budget (memory 2200 / user 1375) is checked on the batch's FINAL result only, overflow returns a structured error with current_entries + usage, and exact duplicates are rejected as no-duplicate-added."),
		mcplib.WithString("content", mcplib.Description("Fact text (required unless `operations` is given)")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session id to attribute the trail event to (must exist in the project store; for scope=global the trail lands there, so the session must exist there)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
		factScopeParam(),
		factTargetParam(),
		mcplib.WithArray("operations", mcplib.Description("Batch operations array (add/patch/remove applied as a unit; content is not required when given). Each op: {\"action\": \"add\"|\"patch\"|\"remove\", \"content\": \"new text\" (add/patch), \"old_text\": \"substring to match\" (patch/remove). The char budget is checked on the FINAL result only; overflow rolls back the whole batch and returns current_entries + usage.")),
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

	// Batch path: an `operations` array is applied as a unit (the char
	// budget is checked on the FINAL result only; overflow rolls back the
	// whole batch and returns the structured current_entries + usage error).
	// content is not required in this path.
	if ops, err := getFactOperations(req); err != nil {
		return toolError(err)
	} else if len(ops) > 0 {
		return handleFactBatch(ctx, data, trail, sessionID, projectID, scope, target, ops)
	}

	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}
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

// getFactOperations returns the `operations` argument as []facts.Operation
// (nil when absent). The mcp-go CallToolRequest has no typed array accessor,
// so the argument is pulled from GetArguments and re-marshaled through JSON
// (the same round-trip the library uses for its own accessors). A PRESENT but
// malformed array is a clean error (the batch boundary validates input); an
// absent argument stays nil (the single-fact content path).
func getFactOperations(req mcplib.CallToolRequest) ([]facts.Operation, error) {
	raw, ok := req.GetArguments()["operations"]
	if !ok {
		return nil, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("operations argument: %w", err)
	}
	var ops []facts.Operation
	if err := json.Unmarshal(b, &ops); err != nil {
		return nil, fmt.Errorf("operations argument must be an array of {action, content, old_text} objects: %w", err)
	}
	return ops, nil
}

// handleFactBatch applies the fact_add `operations` array as a unit through
// facts.Batch and returns the structured BatchResult (success + usage +
// current_entries). A failed batch (overflow / ambiguous / no-match /
// validation) returns a tool error carrying the structured payload (the
// agent reads current_entries / closest_entries to decide the retry).
func handleFactBatch(ctx context.Context, data, trail *facts.Store, sessionID, projectID string, scope facts.Scope, target string, ops []facts.Operation) (*mcplib.CallToolResult, error) {
	res, err := data.Batch(ctx, target, ops)
	if err != nil {
		return toolError(err)
	}
	// Trail row for the audit trail (T04 review finding): every write op
	// records a session_events entry in the caller's project store — for
	// scope=global the trail is cross-store, so a failed insert would leave
	// the applied batch un-audited and is a clean error (mirrors the
	// AddWith cross-store contract). action_type stays "fact_add" (a batch
	// is a unit of fact_add operations); the payload summarizes the unit.
	// A FAILED batch (err above) writes nothing — no ops applied, nothing
	// to audit.
	if err := trail.InsertEvent(ctx, sessionID, "fact_add", map[string]any{
		"batch": true,
		"ops":   len(ops),
	}); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"success":            res.Success,
		"no_duplicate_added": res.NoDuplicateAdded,
		"usage":              res.Usage,
		"current_entries":    res.CurrentEntries,
		"session_id":         sessionID,
		"project":            projectID,
		"scope":              string(scope),
		"event":              "fact_add_batch",
	})
}

func factSearchTool() mcplib.Tool {
	return mcplib.NewTool("fact_search",
		mcplib.WithDescription("Lexical FTS search over Fact Memory (soft-deleted facts excluded). With scope=global the search reads the cross-project __global__ store and trails to the caller's project store. With scope=all the search covers BOTH stores (fused search): hits are interleaved by each store's own bm25 rank and every hit is labeled with its scope (global/project) and category. Returns matching facts and records a session_events trail (action_type=fact_search, mode=fused) with the matched fact ids."),
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

	if scope == facts.ScopeAll {
		// Fused search (TICKET-03): both stores, hits interleaved by each
		// store's rank, each labeled scope + category; the trail lands in
		// the project store only.
		global, project, cleanup, err := openFusedStores(svc, projectID)
		if err != nil {
			return toolError(err)
		}
		defer cleanup()
		f, err := facts.FusedSearch(ctx, sessionID, query, limit, includeDeleted, global, project)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"facts": f,
			"count": len(f),
		})
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
// has its own path: openFusedStores).
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

// openFusedStores opens BOTH stores for the scope=all read path (TICKET-03):
// the __global__ store plus the caller's project store, plus a cleanup that
// releases both handles.
func openFusedStores(svc *service.Service, projectID string) (global, project *facts.Store, cleanup func(), err error) {
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
}
