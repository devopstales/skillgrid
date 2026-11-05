// tools_kb.go — the kb_* tool family (008-transport §3, the 011 openwiki
// port): the 12 tool definitions (the D1 contract), the dispatch fns (the
// MCP adapter's half of the K6 seam — the CLI has its own over the same
// KbHandler), and the profile-filtered tool set (agent = 8 working-set
// tools, all = +4 admin tools).
//
// Profiles (008 §2): agent — query, read, search, generate, validate,
// lint, status, cite_source; all — agent + reindex, index, bootstrap,
// rebuild_meta. kb_status is agent-profile (a read-only view of derived
// state); kb_generate is agent-profile (the KB's "save" equivalent).
package mcp

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ── tool definitions (the D1 contract) ─────────────────────────────────────

func kbQueryTool() mcplib.Tool {
	return mcplib.NewTool("kb_query",
		mcplib.WithDescription("Stage-1 deep retrieval (wiki-first): ranked links + 1-line snippets over the knowledge base. Full hydration is stage 2 (kb_read). A thin result hints falling back to mem_lookup."),
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

func kbReadTool() mcplib.Tool {
	return mcplib.NewTool("kb_read",
		mcplib.WithDescription("Stage-2 hydration: the full page (or one ## section) + backlinks. Grounded pages carry their pin-check state (fresh/stale/unresolved)."),
		mcplib.WithString("path", mcplib.Required(), mcplib.Description("The page rel_path (e.g. decisions/auth-model.md)")),
		mcplib.WithString("section", mcplib.Description("A single ## section name (empty = the full body)")),
		mcplib.WithBoolean("include_backlinks", mcplib.Description("List the pages that link here (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbSearchTool() mcplib.Tool {
	return mcplib.NewTool("kb_search",
		mcplib.WithDescription("The raw FTS power path over the KB index: more rows than kb_query's default, longer snippets. For power searches, not the default retrieval."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 25)")),
	)
}

func kbGenerateTool() mcplib.Tool {
	return mcplib.NewTool("kb_generate",
		mcplib.WithDescription("Run KB generation (the KB's save equivalent — the agent triggers it proactively). Default incremental (entity-targeted); full re-derives every page. Returns changed: N. Degraded runs (LLM absent) carry degraded:true."),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("Mandatory ownership assertion: the resolved project must match this")),
		mcplib.WithBoolean("full", mcplib.Description("Full re-derive (ignore the processed set, default false)")),
	)
}

func kbValidateTool() mcplib.Tool {
	return mcplib.NewTool("kb_validate",
		mcplib.WithDescription("The pin-check: validate grounded pages against the live code. Empty paths = every grounded page. Returns per-page status + checks; strict carries the CI verdict (ok = no drift)."),
		mcplib.WithString("paths", mcplib.Description("JSON array of rel_paths to scope the check (empty = all grounded pages)")),
		mcplib.WithBoolean("strict", mcplib.Description("Carry the strict verdict in the result (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbLintTool() mcplib.Tool {
	return mcplib.NewTool("kb_lint",
		mcplib.WithDescription("Deterministic health over the KB index: missing titles, empty bodies, stale index rows. fix auto-repairs the safe findings (missing-title backfill) and re-lints."),
		mcplib.WithBoolean("fix", mcplib.Description("Auto-repair the safe findings (default false)")),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbStatusTool() mcplib.Tool {
	return mcplib.NewTool("kb_status",
		mcplib.WithDescription("Vault health (read-only): indexed page count, the run phase/id, baseline + event counts. The agent uses it to decide when to save / re-verify."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbCiteSourceTool() mcplib.Tool {
	return mcplib.NewTool("kb_cite_source",
		mcplib.WithDescription("Record an external citation on a page's sources (external:<name> + version + source_url; the build_fingerprint is computed). Does not fetch/store the body. Idempotent per (name, version)."),
		mcplib.WithString("page", mcplib.Required(), mcplib.Description("The page rel_path to cite on")),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("The external source name (stored as external:<name>)")),
		mcplib.WithString("version", mcplib.Description("The cited version/tag")),
		mcplib.WithString("source_url", mcplib.Description("The source URL")),
		mcplib.WithString("source_commit", mcplib.Description("The source commit (when known)")),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("Mandatory ownership assertion: the resolved project must match this")),
	)
}

func kbReindexTool() mcplib.Tool {
	return mcplib.NewTool("kb_reindex",
		mcplib.WithDescription("FULL rebuild of the KB index (kb_pages/kb_pages_fts/kb_claims) from the markdown. Idempotent. Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbIndexTool() mcplib.Tool {
	return mcplib.NewTool("kb_index",
		mcplib.WithDescription("Incremental reindex: changed pages only (advances the checkpoint). Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

func kbBootstrapTool() mcplib.Tool {
	return mcplib.NewTool("kb_bootstrap",
		mcplib.WithDescription("Scaffold a new knowledge vault (the sentinel config.json, KNOWLEDGE_SCHEMA.md, templates, the page folders). Idempotent no-op on an existing vault. Admin-only."),
		mcplib.WithString("project", mcplib.Description("The project name to scaffold (defaults to the CWD-resolved project)")),
	)
}

func kbRebuildMetaTool() mcplib.Tool {
	return mcplib.NewTool("kb_rebuild_meta",
		mcplib.WithDescription("Rebuild the derived metadata that is NOT the FTS index: the kb_run_state checkpoint + the backlink in-degree cache. Admin-only."),
		mcplib.WithString("project", mcplib.Description("Filter by project (defaults to the CWD-resolved project)")),
	)
}

// ── dispatch fns (the MCP adapter's half of the K6 seam) ───────────────────

func dispatchKbQuery(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
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

func dispatchKbRead(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Read(ctx, handler.DefaultCWD, argString(args, "path"), argString(args, "section"), argString(args, "project"), argBool(args, "include_backlinks"))
	}
}

func dispatchKbSearch(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		limit := int(argNumber(args, "limit"))
		return kh.Search(ctx, handler.DefaultCWD, argString(args, "query"), argString(args, "project"), limit)
	}
}

func dispatchKbGenerate(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Generate(ctx, handler.DefaultCWD, "", argString(args, "expected_project"), argBool(args, "full"))
	}
}

func dispatchKbValidate(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Validate(ctx, handler.DefaultCWD, argString(args, "project"), argStringSlice(args, "paths"), argBool(args, "strict"))
	}
}

func dispatchKbLint(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Lint(ctx, handler.DefaultCWD, argString(args, "project"), argBool(args, "fix"))
	}
}

func dispatchKbStatus(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Status(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchKbCiteSource(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
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

func dispatchKbReindex(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Reindex(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchKbIndex(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Index(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchKbBootstrap(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.Bootstrap(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

func dispatchKbRebuildMeta(kh *handler.KbHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return kh.RebuildMeta(ctx, handler.DefaultCWD, argString(args, "project"))
	}
}

// ── the server's dispatch + tool set ───────────────────────────────────────

// kbToolDispatch returns the kb_* tool name → dispatch fn map (the profile
// filters the set — the agent profile drops the admin tools).
func (s *Server) kbToolDispatch() map[string]func(ctx context.Context, args map[string]any) (any, error) {
	kh := s.kbTools
	if kh == nil {
		return nil
	}
	return map[string]func(ctx context.Context, args map[string]any) (any, error){
		"kb_query":       dispatchKbQuery(kh),
		"kb_read":        dispatchKbRead(kh),
		"kb_search":      dispatchKbSearch(kh),
		"kb_generate":    dispatchKbGenerate(kh),
		"kb_validate":    dispatchKbValidate(kh),
		"kb_lint":        dispatchKbLint(kh),
		"kb_status":      dispatchKbStatus(kh),
		"kb_cite_source": dispatchKbCiteSource(kh),
		// admin (All profile only)
		"kb_reindex":      dispatchKbReindex(kh),
		"kb_index":        dispatchKbIndex(kh),
		"kb_bootstrap":    dispatchKbBootstrap(kh),
		"kb_rebuild_meta": dispatchKbRebuildMeta(kh),
	}
}

// kbToolSet returns the profile-filtered kb_* tool definitions (the agent
// profile drops the admin tools — the two-profile mechanism, D-PROFILE).
func (s *Server) kbToolSet() []toolEntry {
	kh := s.kbTools
	if kh == nil {
		return nil
	}
	entries := []toolEntry{
		// agent profile (8)
		{name: "kb_query", def: kbQueryTool(), isWrite: false},
		{name: "kb_read", def: kbReadTool(), isWrite: false},
		{name: "kb_search", def: kbSearchTool(), isWrite: false},
		{name: "kb_generate", def: kbGenerateTool(), isWrite: true},
		{name: "kb_validate", def: kbValidateTool(), isWrite: false},
		{name: "kb_lint", def: kbLintTool(), isWrite: false},
		{name: "kb_status", def: kbStatusTool(), isWrite: false},
		{name: "kb_cite_source", def: kbCiteSourceTool(), isWrite: true},
		// admin (All profile only, 4)
		{name: "kb_reindex", def: kbReindexTool(), isWrite: false},
		{name: "kb_index", def: kbIndexTool(), isWrite: false},
		{name: "kb_bootstrap", def: kbBootstrapTool(), isWrite: false},
		{name: "kb_rebuild_meta", def: kbRebuildMetaTool(), isWrite: false},
	}
	if s.profile == Agent {
		admin := map[string]bool{}
		for _, n := range kbAdminOnlyTools {
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

// kbAdminOnlyTools is the admin-only kb_* tool names (the agent profile
// drops them — 008-transport §2).
var kbAdminOnlyTools = []string{
	"kb_reindex", "kb_index", "kb_bootstrap", "kb_rebuild_meta",
}
