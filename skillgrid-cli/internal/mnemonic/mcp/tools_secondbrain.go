package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/secondbrain"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

var (
	errNoQuery      = errors.New("query is required")
	errNoAction     = errors.New("action is required")
	errNoSubaction  = errors.New("subaction is required for archive")
	errNoObsIDs     = errors.New("obs_ids is required")
	errNoReason     = errors.New("reason is required for archive subaction")
	errNoCanonical  = errors.New("canonical obs_id is required for dedup_merge")
	errNoNewTitle   = errors.New("new_title is required for consolidate")
	errBadAction    = errors.New("unknown action")
)

func cwd() (string, error) {
	return os.Getwd()
}

// registerSecondBrainTools wires the second-brain tools (change
// 2026-09-30-mnemonic-second-brain). mem_ask is LIVE (the deterministic
// cited floor over BlendedSearch); mem_lifecycle is a not-yet-implemented
// dispatch stub — the action dispatch lands in TICKET-03 — returned as a
// value error, never thrown.
func registerSecondBrainTools(s *server.MCPServer) {
	s.AddTool(memAskTool(), handleMemAsk)
	s.AddTool(memLifecycleTool(), handleMemLifecycle)
}

func memAskTool() mcplib.Tool {
	return mcplib.NewTool("mem_ask",
		mcplib.WithDescription("Ask the brain a question and get a cited answer. Deterministic no-LLM floor: gathers relevant observations via blended search (FTS5 + vector RRF; degrades to keyword-only when no embedder) and returns token-bounded, citation-bearing results. matched_via is keyword or hybrid; degraded is true when the vector leg was skipped. Pass project to scope, or all_projects=true to span every store."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("The question or keywords to ask the brain")),
		mcplib.WithString("mode", mcplib.Description("Answer mode: cited (default, deterministic no-LLM floor) or llm (prose with [obs:<id>] citations, failing open to the cited floor on LLM error/timeout).")),
		mcplib.WithString("project", mcplib.Description("Optional project name to scope the answer under (defaults to the CWD-resolved project).")),
		mcplib.WithBoolean("all_projects", mcplib.Description("Span every project store (default false).")),
		mcplib.WithNumber("max_tokens", mcplib.Description("Token budget for the citations (default 2000).")),
	)
}

func memLifecycleTool() mcplib.Tool {
	return mcplib.NewTool("mem_lifecycle",
		mcplib.WithDescription("Lifecycle actions over the memory store: health report, dedup scan/merge, consolidate, archive/restore. Every mutating op is audited. TICKET-01 registers the tool; the action dispatch lands in TICKET-03, so calls currently return a not-implemented value error."),
		mcplib.WithString("action", mcplib.Required(), mcplib.Description("Action: health | dedup_scan | dedup_merge | consolidate | archive")),
		mcplib.WithString("subaction", mcplib.Description("Archive subaction: archive | restore | list | stale")),
		mcplib.WithString("project", mcplib.Description("Optional project name to scope the action under (defaults to the CWD-resolved project).")),
		mcplib.WithArray("obs_ids", mcplib.Description("Observation IDs the action applies to (dedup_merge, consolidate, archive).")),
		mcplib.WithString("reason", mcplib.Description("Archive reason (archive subaction).")),
		mcplib.WithNumber("stale_days", mcplib.Description("Stale threshold in days (archive stale subaction; default 90).")),
		mcplib.WithBoolean("dry_run", mcplib.Description("For dedup_scan: return clusters without writing (default true).")),
		mcplib.WithString("new_title", mcplib.Description("Title for the consolidated observation (consolidate action).")),
	)
}

// handleMemAsk resolves the project (explicit name or CWD), routes through
// secondbrain.Ask (mode "cited" is the deterministic floor, "llm" adds prose
// with [obs:<id>] citations, failing open to the cited floor), and returns the
// result as raw JSON.
func handleMemAsk(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	query := strings.TrimSpace(req.GetString("query", ""))
	if query == "" {
		return toolError(errNoQuery)
	}
	mode := strings.ToLower(strings.TrimSpace(req.GetString("mode", "cited")))
	if mode != "cited" && mode != "llm" {
		return toolError(fmt.Errorf("mode must be 'cited' or 'llm', got %q", mode))
	}

	explicitProject := strings.TrimSpace(req.GetString("project", ""))
	var projectID string
	if explicitProject != "" {
		projectID = project.NormalizeID(explicitProject)
	} else {
		cwd, err := cwd()
		if err != nil {
			return toolError(err)
		}
		projectID, err = svc.ResolveProject(cwd)
		if err != nil {
			return toolError(err)
		}
	}

	allProjects := req.GetBool("all_projects", false)
	maxTokens := int(req.GetFloat("max_tokens", 0))

	res, err := secondbrain.Ask(ctx, svc, mode, query, projectID, allProjects, maxTokens)
	if err != nil {
		return toolError(err)
	}
	if res == nil {
		res = &secondbrain.AskResult{}
	}
	out := map[string]any{
		"project":      projectID,
		"all_projects": allProjects,
	}
	// _health_warnings (TICKET-06): the inline, non-breaking lifecycle
	// warnings for this project. Only when the handler knows a concrete
	// project — the all-projects path does not, so it stays [].
	if !allProjects && projectID != "" {
		out["_health_warnings"] = secondbrain.ForProject(svc, projectID)
	} else {
		out["_health_warnings"] = []secondbrain.HealthWarning{}
	}
	applyAskResult(out, res)
	return JSONResult(out)
}

// applyAskResult projects an AskResult into a response map: the citation
// fields plus the underscore-prefixed total-token meta (OCBI: meta fields are
// underscore-prefixed; errors are returned as values, never thrown).
func applyAskResult(out map[string]any, res *secondbrain.AskResult) {
	out["answer"] = res.Answer
	out["matched_via"] = res.MatchedVia
	out["degraded"] = res.Degraded
	if res.Sources != nil {
		out["sources"] = res.Sources
	}
	citations := make([]map[string]any, len(res.Citations))
	for i, c := range res.Citations {
		citations[i] = map[string]any{
			"id":      c.ID,
			"title":   c.Title,
			"type":    c.Type,
			"snippet": c.Snippet,
			"project": c.Project,
		}
	}
	out["citations"] = citations
	out["_total_tokens"] = res.TotalTokens
}

// intArray extracts a []int64 from the request arguments for the given key.
// Returns nil when the key is absent or not an array.
func intArray(req mcplib.CallToolRequest, key string) []int64 {
	raw, ok := req.GetArguments()[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(arr))
	for _, v := range arr {
		switch n := v.(type) {
		case float64:
			out = append(out, int64(n))
		case int64:
			out = append(out, n)
		case int:
			out = append(out, int64(n))
		}
	}
	return out
}

// resolveProject extracts the project from the request (explicit name or
// CWD-resolved) and returns a normalized project ID.
func resolveProject(svc *service.Service, req mcplib.CallToolRequest) (string, error) {
	explicitProject := strings.TrimSpace(req.GetString("project", ""))
	if explicitProject != "" {
		return project.NormalizeID(explicitProject), nil
	}
	cwd, err := cwd()
	if err != nil {
		return "", err
	}
	return svc.ResolveProject(cwd)
}

// handleMemLifecycle dispatches the mem_lifecycle action to the secondbrain
// lifecycle functions. Every mutating op is audited in the 043 lifecycle_log
// table. Errors are returned as values, never thrown.
func handleMemLifecycle(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	action := strings.TrimSpace(req.GetString("action", ""))
	if action == "" {
		return toolError(errNoAction)
	}

	projectID, err := resolveProject(svc, req)
	if err != nil {
		return toolError(err)
	}

	switch action {
	case "health":
		rep, err := secondbrain.Health(ctx, svc, projectID)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"project": projectID,
			"health":  rep,
		})

	case "dedup_scan":
		dryRun := req.GetBool("dry_run", true)
		clusters, density, err := secondbrain.DedupScan(ctx, svc, projectID, dryRun)
		if err != nil {
			return toolError(err)
		}
		degraded := false
		for i := range clusters {
			if clusters[i].Degraded {
				degraded = true
				break
			}
		}
		return JSONResult(map[string]any{
			"project":  projectID,
			"clusters": clusters,
			"density":  density,
			"degraded": degraded,
		})

	case "dedup_merge":
		obsIDs := intArray(req, "obs_ids")
		if len(obsIDs) == 0 {
			return toolError(errNoCanonical)
		}
		canonical := obsIDs[0]
		if err := secondbrain.DedupMerge(ctx, svc, projectID, canonical); err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"project":  projectID,
			"merged":   true,
			"canonical": canonical,
		})

	case "consolidate":
		obsIDs := intArray(req, "obs_ids")
		if len(obsIDs) == 0 {
			return toolError(errNoObsIDs)
		}
		newTitle := strings.TrimSpace(req.GetString("new_title", ""))
		if newTitle == "" {
			return toolError(errNoNewTitle)
		}
		newID, err := secondbrain.Consolidate(ctx, svc, projectID, obsIDs, newTitle)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"project":   projectID,
			"new_id":    newID,
			"consolidated": obsIDs,
		})

	case "archive":
		subaction := strings.TrimSpace(req.GetString("subaction", ""))
		if subaction == "" {
			return toolError(errNoSubaction)
		}
		obsIDs := intArray(req, "obs_ids")
		reason := strings.TrimSpace(req.GetString("reason", ""))
		staleDays := int(req.GetFloat("stale_days", 90))
		if subaction == "archive" && reason == "" {
			return toolError(errNoReason)
		}
		result, err := secondbrain.Archive(ctx, svc, projectID, subaction, obsIDs, reason, staleDays)
		if err != nil {
			return toolError(err)
		}
		out := map[string]any{
			"project":   projectID,
			"subaction": subaction,
		}
		if result != nil {
			out["affected"] = result.Affected
			if result.Stale != nil {
				out["stale"] = result.Stale
			}
			if result.Listed != nil {
				out["listed"] = result.Listed
			}
		}
		return JSONResult(out)

	default:
		return toolError(fmt.Errorf("%w: %q", errBadAction, action))
	}
}
