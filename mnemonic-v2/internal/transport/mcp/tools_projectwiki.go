// tools_projectwiki.go — the projectwiki_* tool family (008-transport §3, the 011 openwiki
// port): the 12 tool definitions (the D1 contract), the dispatch fns (the
// MCP adapter's half of the K6 seam — the CLI has its own over the same
// ProjectWikiHandler), and the profile-filtered tool set (agent = 8 working-set
// tools, all = +4 admin tools).
//
// Profiles (008 §2): agent — query, read, search, generate, validate,
// lint, status, cite_source; all — agent + reindex, index, bootstrap,
// rebuild_meta. projectwiki_status is agent-profile (a read-only view of derived
// state); projectwiki_generate is agent-profile (the project wiki's "save" equivalent).
package mcp

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ── tool definitions (the D1 contract) ─────────────────────────────────────

func projectWikiQueryTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_query",
		mcplib.WithDescription("Stage-1 deep retrieval (wiki-first): ranked links + 1-line snippets over the project wiki. Full hydration is stage 2 (projectwiki_read). A thin result hints falling back to mem_lookup."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("type", mcplib.Description("Filter by page type (decision, invariant, workflow, context)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 10)")),
		mcplib.WithString("match_mode", mcplib.Description("Term matching: any (default) or all")),
		mcplib.WithBoolean("grounded_only", mcplib.Description("Restrict to grounded pages (decision, invariant, workflow)")),
		// thin is parked: the engine's thin-threshold flag is not yet wired;
		// the arg returns from the schema until it exists (the
		// mem_get_observation history_cursor pattern).
	)
}

func projectWikiReadTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_read",
		mcplib.WithDescription("Stage-2 hydration: the full page (or one ## section) + backlinks. Grounded pages carry their pin-check state (fresh/stale/unresolved)."),
		mcplib.WithString("path", mcplib.Required(), mcplib.Description("The page rel_path (e.g. decisions/auth-model.md)")),
		mcplib.WithString("section", mcplib.Description("A single ## section name (empty = the full body)")),
		mcplib.WithBoolean("include_backlinks", mcplib.Description("List the pages that link here (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiSearchTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_search",
		mcplib.WithDescription("The raw FTS power path over the ProjectWiki index: more rows than projectwiki_query's default, longer snippets. For power searches, not the default retrieval."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 25)")),
	)
}

func projectWikiGenerateTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_generate",
		mcplib.WithDescription("Run ProjectWiki generation (the project wiki's save equivalent — the agent triggers it proactively). Default incremental (entity-targeted); full re-derives every page. Returns changed: N. Degraded runs (LLM absent) carry degraded:true."),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("Mandatory ownership assertion: the resolved project must match this")),
		mcplib.WithBoolean("full", mcplib.Description("Full re-derive (ignore the processed set, default false)")),
	)
}

func projectWikiValidateTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_validate",
		mcplib.WithDescription("The pin-check: validate grounded pages against the live code. Empty paths = every grounded page. Returns per-page status + checks; strict carries the CI verdict (ok = no drift)."),
		mcplib.WithString("paths", mcplib.Description("JSON array of rel_paths to scope the check (empty = all grounded pages)")),
		mcplib.WithBoolean("strict", mcplib.Description("Carry the strict verdict in the result (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiLintTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_lint",
		mcplib.WithDescription("Deterministic health over the ProjectWiki index: missing titles, empty bodies, stale index rows. fix auto-repairs the safe findings (missing-title backfill) and re-lints."),
		mcplib.WithBoolean("fix", mcplib.Description("Auto-repair the safe findings (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiStatusTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_status",
		mcplib.WithDescription("Vault health (read-only): indexed page count, the run phase/id, baseline + event counts. The agent uses it to decide when to save / re-verify."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiCiteSourceTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_cite_source",
		mcplib.WithDescription("Record an external citation on a page's sources (external:<name> + version + source_url; the build_fingerprint is computed). Does not fetch/store the body. Idempotent per (name, version)."),
		mcplib.WithString("page", mcplib.Required(), mcplib.Description("The page rel_path to cite on")),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("The external source name (stored as external:<name>)")),
		mcplib.WithString("version", mcplib.Description("The cited version/tag")),
		mcplib.WithString("source_url", mcplib.Description("The source URL")),
		mcplib.WithString("source_commit", mcplib.Description("The source commit (when known)")),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("Mandatory ownership assertion: the resolved project must match this")),
	)
}

func projectWikiReindexTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_reindex",
		mcplib.WithDescription("FULL rebuild of the ProjectWiki index (projectwiki_pages/projectwiki_pages_fts/projectwiki_claims) from the markdown. Idempotent. Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiIndexTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_index",
		mcplib.WithDescription("Incremental reindex: changed pages only (advances the checkpoint). Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func projectWikiBootstrapTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_bootstrap",
		mcplib.WithDescription("Scaffold a new project wiki vault (the sentinel config.json, PROJECTWIKI_SCHEMA.md, templates, the page folders). Idempotent no-op on an existing vault. Admin-only."),
		mcplib.WithString("project", mcplib.Description("The project name to scaffold (defaults to the CWD-resolved project)")),
	)
}

func projectWikiRebuildMetaTool() mcplib.Tool {
	return mcplib.NewTool("projectwiki_rebuild_meta",
		mcplib.WithDescription("Rebuild the derived metadata that is NOT the FTS index: the projectwiki_run_state checkpoint + the backlink in-degree cache. Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

// ── dispatch fns (the MCP adapter's half of the K6 seam) ───────────────────

func dispatchProjectWikiQuery(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		req := handler.QueryRequest{
			Query:        argString(args, "query"),
			Type:         argString(args, "type"),
			MatchMode:    argString(args, "match_mode"),
			GroundedOnly: argBool(args, "grounded_only"),
		}
		if v := argNumber(args, "limit"); v > 0 {
			req.Limit = int(v)
		}
		return kh.Query(ctx, handler.DefaultCWD, argString(args, "project"), req)
	}
}

func dispatchProjectWikiRead(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Read(ctx, handler.DefaultCWD, argString(args, "path"), argString(args, "section"), argString(args, "project"), argBool(args, "include_backlinks"))
	}
}

func dispatchProjectWikiSearch(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		limit := int(argNumber(args, "limit"))
		return kh.Search(ctx, handler.DefaultCWD, argString(args, "query"), argString(args, "project"), limit)
	}
}

func dispatchProjectWikiGenerate(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Generate(ctx, handler.DefaultCWD, "", argString(args, "expected_project"), argBool(args, "full"))
	}
}

func dispatchProjectWikiValidate(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Validate(ctx, handler.DefaultCWD, argString(args, "project"), argStringSlice(args, "paths"), argBool(args, "strict"))
	}
}

func dispatchProjectWikiLint(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Lint(ctx, handler.DefaultCWD, argString(args, "project"), argBool(args, "fix"))
	}
}

func dispatchProjectWikiStatus(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Status(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchProjectWikiCiteSource(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.CiteSource(ctx, handler.DefaultCWD, "",
			argString(args, "expected_project"),
			argString(args, "page"),
			argString(args, "name"),
			argString(args, "version"),
			argString(args, "source_url"),
			argString(args, "source_commit"),
		)
	}
}

func dispatchProjectWikiReindex(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Reindex(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchProjectWikiIndex(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Index(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchProjectWikiBootstrap(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Bootstrap(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchProjectWikiRebuildMeta(kh *handler.ProjectWikiHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.RebuildMeta(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

// ── the server's dispatch + tool set ───────────────────────────────────────

// projectWikiToolDispatch returns the projectwiki_* tool name → dispatch fn map (the profile
// filters the set — the agent profile drops the admin tools).
func (s *Server) projectWikiToolDispatch() map[string]func(ctx context.Context, args map[string]any) (any, error) {
	kh := s.projectWikiTools
	if kh == nil {
		return nil
	}
	return map[string]func(ctx context.Context, args map[string]any) (any, error){
		"projectwiki_query":       dispatchProjectWikiQuery(kh),
		"projectwiki_read":        dispatchProjectWikiRead(kh),
		"projectwiki_search":      dispatchProjectWikiSearch(kh),
		"projectwiki_generate":    dispatchProjectWikiGenerate(kh),
		"projectwiki_validate":    dispatchProjectWikiValidate(kh),
		"projectwiki_lint":        dispatchProjectWikiLint(kh),
		"projectwiki_status":      dispatchProjectWikiStatus(kh),
		"projectwiki_cite_source": dispatchProjectWikiCiteSource(kh),
		// admin (All profile only)
		"projectwiki_reindex":      dispatchProjectWikiReindex(kh),
		"projectwiki_index":        dispatchProjectWikiIndex(kh),
		"projectwiki_bootstrap":    dispatchProjectWikiBootstrap(kh),
		"projectwiki_rebuild_meta": dispatchProjectWikiRebuildMeta(kh),
	}
}

// projectWikiToolSet returns the profile-filtered projectwiki_* tool definitions (the agent
// profile drops the admin tools — the two-profile mechanism, D-PROFILE).
func (s *Server) projectWikiToolSet() []toolEntry {
	kh := s.projectWikiTools
	if kh == nil {
		return nil
	}
	entries := []toolEntry{
		// agent profile (8)
		{name: "projectwiki_query", def: projectWikiQueryTool(), isWrite: false},
		{name: "projectwiki_read", def: projectWikiReadTool(), isWrite: false},
		{name: "projectwiki_search", def: projectWikiSearchTool(), isWrite: false},
		{name: "projectwiki_generate", def: projectWikiGenerateTool(), isWrite: true},
		{name: "projectwiki_validate", def: projectWikiValidateTool(), isWrite: false},
		{name: "projectwiki_lint", def: projectWikiLintTool(), isWrite: false},
		{name: "projectwiki_status", def: projectWikiStatusTool(), isWrite: false},
		{name: "projectwiki_cite_source", def: projectWikiCiteSourceTool(), isWrite: true},
		// admin (All profile only, 4)
		{name: "projectwiki_reindex", def: projectWikiReindexTool(), isWrite: false},
		{name: "projectwiki_index", def: projectWikiIndexTool(), isWrite: false},
		{name: "projectwiki_bootstrap", def: projectWikiBootstrapTool(), isWrite: false},
		{name: "projectwiki_rebuild_meta", def: projectWikiRebuildMetaTool(), isWrite: false},
	}
	if s.profile == Agent {
		admin := map[string]bool{}
		for _, n := range projectWikiAdminOnlyTools {
			admin[n] = true
		}
		out := make([]toolEntry, 0, len(entries))
		for _, e := range entries {
			if !admin[e.name] {
				out = append(out, e)
			}
		}
		return out
	}
	return entries
}

// projectWikiAdminOnlyTools is the admin-only projectwiki_* tool names (the agent profile
// drops them — 008-transport §2).
var projectWikiAdminOnlyTools = []string{
	"projectwiki_reindex", "projectwiki_index", "projectwiki_bootstrap", "projectwiki_rebuild_meta",
}
