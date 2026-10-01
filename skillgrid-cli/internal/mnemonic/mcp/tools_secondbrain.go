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
)

var (
	errNoQuery                 = errors.New("query is required")
	errNoAction                = errors.New("action is required")
	errLifecycleNotImplemented = errors.New("mem_lifecycle: action dispatch not yet implemented (lands in TICKET-03)")
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

// handleMemLifecycle is the TICKET-01 stub: the tool is registered and the
// action is parsed, but the dispatch (health/dedup/consolidate/archive) lands
// in TICKET-03. It returns a not-implemented value error so a call never
// throws and never claims success.
func handleMemLifecycle(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	action := strings.TrimSpace(req.GetString("action", ""))
	if action == "" {
		return toolError(errNoAction)
	}
	return toolError(errLifecycleNotImplemented)
}
