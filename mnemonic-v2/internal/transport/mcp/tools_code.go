package mcp

// tools_code.go — the code_* tool definitions (D1-pinned, spec 008-transport
// §3) + their dispatch fns. Each tool is: parse args → CodeHandler call →
// the dispatch fn returns the result (the server wraps it in the envelope).
// The profile (Agent/All) filters which tools are registered (the
// two-profile mechanism, D-PROFILE — not v1's SKILLGRID_MCP_CODE_TOOLS env
// filter).
//
// This file is the MCP adapter's tool surface — the CLI
// (cmd/mnemonic/index.go) has its own adapter over the same CodeHandler.
// The tool definitions here are the D1 contract: the plugins' only
// interface into the code-index.

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/graph"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// ── tool definitions (the D1 contract — the plugins' only interface) ──────

func codeStatusTool() mcplib.Tool {
	return mcplib.NewTool("code_status",
		mcplib.WithDescription("Index health (4-state freshness, NOT a boolean; per-language extractor tier; schema version; fair coverage; capabilities). Call when the index may be stale (after clone, branch switch, or large refactors)."),
		mcplib.WithString("project", mcplib.Description("Optional project id. Wins over cwd resolution.")),
	)
}

func codeIndexTool() mcplib.Tool {
	return mcplib.NewTool("code_index",
		mcplib.WithDescription("Run incremental code index for the cwd git root. Returns a run_id + status immediately (async); a repeat while active returns 409 operation_conflict. The --full/--lsp/--pdg/--communities/--process flags gate the optional passes (v2: the core run)."),
		mcplib.WithString("project", mcplib.Description("Optional project id. Wins over cwd resolution.")),
		mcplib.WithBoolean("full", mcplib.Description("Full re-index (default: incremental)")),
		mcplib.WithBoolean("lsp", mcplib.Description("Run the LSP pass (v2.1)")),
		mcplib.WithBoolean("pdg", mcplib.Description("Run the PDG pass (v2.1)")),
		mcplib.WithBoolean("communities", mcplib.Description("Run the communities pass (v2.1)")),
		mcplib.WithBoolean("process", mcplib.Description("Run the process pass (v2.1)")),
	)
}

func codeSearchTool() mcplib.Tool {
	return mcplib.NewTool("code_search",
		mcplib.WithDescription("Hybrid search (FTS + structural + semantic, RRF K=60). Prefer this over grep/rg when exploring unknown areas of a large repo. Use code_read only after search narrows path and line range."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 10)")),
		mcplib.WithString("language", mcplib.Description("Filter by language")),
		mcplib.WithString("under", mcplib.Description("Package prefix scope (the files.path prefix)")),
		mcplib.WithString("project", mcplib.Description("Optional project id. Wins over cwd resolution.")),
	)
}

func codeReadTool() mcplib.Tool {
	return mcplib.NewTool("code_read",
		mcplib.WithDescription("Fetch indexed source for a path (and optional line range) after code_search narrows the location. mode: full (the live source span, secret redaction) or content (the drill handle: signature + neighbor ids, no bodies; a file → the TOC)."),
		mcplib.WithString("path", mcplib.Required(), mcplib.Description("Repo-relative file path from code_search")),
		mcplib.WithNumber("start_line", mcplib.Description("Start line (1-based)")),
		mcplib.WithNumber("end_line", mcplib.Description("End line (1-based)")),
		mcplib.WithString("mode", mcplib.Description("full (default) or content (the drill handle)")),
		mcplib.WithString("project", mcplib.Description("Optional project id. Wins over cwd resolution.")),
	)
}

func codeOrientTool() mcplib.Tool {
	return mcplib.NewTool("code_orient",
		mcplib.WithDescription("The navigation entry point — the progressive-disclosure 'map' of a symbol/package/file (metadata + TOC + bounded outbound edges, depth-re-rootable). The agent's 'find a feature' first verb."),
		mcplib.WithString("symbol", mcplib.Description("The symbol name to orient on")),
		mcplib.WithString("package", mcplib.Description("The package path to orient on")),
		mcplib.WithString("file", mcplib.Description("The file path to orient on")),
		mcplib.WithNumber("depth", mcplib.Description("The orientation depth (default 1)")),
	)
}

func codeExplainTool() mcplib.Tool {
	return mcplib.NewTool("code_explain",
		mcplib.WithDescription("Node + degree + connections ranked by the neighbor's degree (hubs first)."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("The symbol to explain")),
	)
}

func codePathTool() mcplib.Tool {
	return mcplib.NewTool("code_path",
		mcplib.WithDescription("BFS shortest path between two symbols, or 'where the graph stops' (the graph's boundary for the from symbol)."),
		mcplib.WithString("from", mcplib.Required(), mcplib.Description("The source symbol")),
		mcplib.WithString("to", mcplib.Description("The target symbol (omit for the graph-stop view)")),
	)
}

func codeImpactTool() mcplib.Tool {
	return mcplib.NewTool("code_impact",
		mcplib.WithDescription("Risk-tiered blast radius (WILL BREAK / LIKELY AFFECTED) — the dependents of a symbol, capped by depth + min-confidence."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("The symbol to impact-analyze")),
		mcplib.WithString("min_confidence", mcplib.Description("Minimum edge confidence (EXTRACTED/INFERRED/AMBIGUOUS)")),
		mcplib.WithNumber("max_depth", mcplib.Description("Maximum traversal depth (default 5)")),
	)
}

func codeAffectedTool() mcplib.Tool {
	return mcplib.NewTool("code_affected",
		mcplib.WithDescription("'Which tests do I run after this change?' — the tests affected by a set of changed files (changed) or a git ref (base). Exactly one of changed/base is set. The MCP transport has no git: base reports the CLI command (mnemonic affected --base <ref>) — the CLI does the git-diff."),
		mcplib.WithString("changed", mcplib.Description("JSON array of changed file paths (repo-relative)")),
		mcplib.WithString("base", mcplib.Description("The git ref to diff against (the CLI does the git-diff; the MCP surface reports the CLI command)")),
		mcplib.WithNumber("depth", mcplib.Description("The traversal depth (default 5)")),
	)
}

func codeGrepTool() mcplib.Tool {
	return mcplib.NewTool("code_grep",
		mcplib.WithDescription("Index-free structural AST grep (by-example) — the structural leg of the hybrid search."),
		mcplib.WithString("pattern", mcplib.Required(), mcplib.Description("The structural pattern")),
		mcplib.WithString("language", mcplib.Description("Filter by language")),
	)
}

func codeSemanticTool() mcplib.Tool {
	return mcplib.NewTool("code_semantic",
		mcplib.WithDescription("Embedding search (symbol + chunk) — the semantic leg of the hybrid search (degraded: no_vector when the embedder is absent)."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("The semantic query")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 10)")),
	)
}

// codeNeighborTool builds one of the six neighbor-view tools (code_get_
// callers/callees/dependents/implementors/hierarchy/tests_for — spec 004 §2).
func codeNeighborTool(name, view string) mcplib.Tool {
	desc := map[string]string{
		"callers":      "Symbols that call/import the given symbol (reverse edges). Every edge carries a confidence label; ambiguous name-only resolution returns AMBIGUOUS edges, never a silent drop.",
		"callees":      "Symbols the given symbol calls/imports (forward edges). Every edge carries a confidence label.",
		"dependents":   "Symbols that depend on the given symbol (calls/imports in either direction). Every edge carries a confidence label.",
		"implementors": "Symbols implementing the given interface/type.",
		"hierarchy":    "Structural hierarchy (extends/implements) around the given symbol.",
		"tests_for":    "Test relationships for the given symbol.",
	}[view]
	return mcplib.NewTool(name,
		mcplib.WithDescription(desc),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("The symbol to query")),
	)
}

func codeResolveTool() mcplib.Tool {
	return mcplib.NewTool("code_resolve",
		mcplib.WithDescription("Resolve a symbol: anchor → {exists, file, line, called_from[], ambiguous, candidates[]} (the KB pin-check contract). 0 matches → exists:false; 1 → exists:true; >1 → ambiguous:true + candidates[]."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("The symbol name to resolve")),
		mcplib.WithString("called_from", mcplib.Description("The caller's file (the disambiguation hint)")),
	)
}

func codePackagesTool() mcplib.Tool {
	return mcplib.NewTool("code_packages",
		mcplib.WithDescription("Top-level Go package map (path, files, symbols) — the KB coverage-gap contract."),
	)
}

func codeRenameTool() mcplib.Tool {
	return mcplib.NewTool("code_rename",
		mcplib.WithDescription("Rename plan (graph edits vs text-search edits, dry_run default). v2: the plan is the symbol's references (the graph edges) + the text-search hits."),
		mcplib.WithString("old", mcplib.Required(), mcplib.Description("The current symbol name")),
		mcplib.WithString("new", mcplib.Required(), mcplib.Description("The new symbol name")),
		mcplib.WithBoolean("dry_run", mcplib.Description("Plan only, do not apply (default true)")),
	)
}

func codeUnresolvedTool() mcplib.Tool {
	return mcplib.NewTool("code_unresolved",
		mcplib.WithDescription("The drop-not-guess backlog (the unresolved_edges ledger — references the index saw but could not resolve)."),
		mcplib.WithNumber("limit", mcplib.Description("Maximum items (default 20)")),
	)
}

func codeCirclesTool() mcplib.Tool {
	return mcplib.NewTool("code_circles",
		mcplib.WithDescription("Import cycles (re-derived from kind='imports' edges, not a stored table)."),
	)
}

func codeDepsTool() mcplib.Tool {
	return mcplib.NewTool("code_deps",
		mcplib.WithDescription("The SBOM dependency graph (dependencies/dep_edges)."),
	)
}

func codeDepImpactTool() mcplib.Tool {
	return mcplib.NewTool("code_dep_impact",
		mcplib.WithDescription("The SBOM dependency impact — which dependents are affected by a dependency change."),
		mcplib.WithString("purl", mcplib.Required(), mcplib.Description("The dependency purl")),
	)
}

func codeEmbeddingStatusTool() mcplib.Tool {
	return mcplib.NewTool("code_embedding_status",
		mcplib.WithDescription("Embedding status (provider/model/dim/embedded count)."),
	)
}

func codeWatchTool() mcplib.Tool {
	return mcplib.NewTool("code_watch",
		mcplib.WithDescription("Start the live incremental watcher (spec 007 — the fsnotify watch loop: debounce, burst-collapse, re-check, M10 delete)."),
		mcplib.WithString("project", mcplib.Description("Optional project id. Wins over cwd resolution.")),
	)
}

// codeDeferredTool builds a v2.1-deferred tool (spec 008 §3 — declared in the
// surface, reporting "run code_index --flag" in v2; deferred, not missing).
func codeDeferredTool(name, flag string) mcplib.Tool {
	return mcplib.NewTool(name,
		mcplib.WithDescription(fmt.Sprintf("%s is a v2.1-deferred tool — run code_index --%s to build its data (declared in the surface, not missing).", name, flag)),
	)
}

func codeRouteTool() mcplib.Tool {
	return mcplib.NewTool("code_route",
		mcplib.WithDescription("Query framework routes: list route nodes (URL pattern + method + framework) and their references edges to handler symbols. A handler name returns the routes that serve it; a path pattern returns the matching route and its handler."),
		mcplib.WithString("handler", mcplib.Description("Handler symbol name: return the routes that reference it")),
		mcplib.WithString("path", mcplib.Description("URL path pattern: return the matching route node and its handler")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum routes to return (default 20)")),
	)
}

func codeNavigatesTool() mcplib.Tool {
	return mcplib.NewTool("code_navigates",
		mcplib.WithDescription("Query framework navigation (screen → screen) edges. A from-screen returns the screens it navigates to; a to-screen returns the senders."),
		mcplib.WithString("from", mcplib.Description("Source screen path (e.g. /home)")),
		mcplib.WithString("to", mcplib.Description("Destination screen path (e.g. /about)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum navigations to return (default 20)")),
	)
}

func codeDocsTool() mcplib.Tool {
	return mcplib.NewTool("code_docs",
		mcplib.WithDescription("Query doc knowledge nodes (kind='doc' symbols — the .md files + their links)."),
		mcplib.WithString("path", mcplib.Description("The doc path to query")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum docs to return (default 20)")),
	)
}

func codeConfigsTool() mcplib.Tool {
	return mcplib.NewTool("code_configs",
		mcplib.WithDescription("Query config knowledge nodes (kind='config' symbols — the .yaml/.toml/.json files + their references)."),
		mcplib.WithString("path", mcplib.Description("The config path to query")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum configs to return (default 20)")),
	)
}

func codeSQLSchemaTool() mcplib.Tool {
	return mcplib.NewTool("code_sql_schema",
		mcplib.WithDescription("Query SQL schema knowledge nodes (kind='table' symbols — the .sql files' CREATE TABLE statements)."),
		mcplib.WithString("table", mcplib.Description("The table name to query")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum tables to return (default 20)")),
	)
}

// ── dispatch fns (the args → CodeHandler call mapping) ─────────────────────

func dispatchCodeStatus(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Status(ctx, argString(args, "project"))
	}
}

func dispatchCodeIndex(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Index(ctx, argString(args, "project"),
			argBool(args, "full"), argBool(args, "lsp"), argBool(args, "pdg"),
			argBool(args, "communities"), argBool(args, "process"))
	}
}

func dispatchCodeSearch(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		q := searchQueryFromArgs(args)
		return h.Search(ctx, argString(args, "query"), q)
	}
}

func dispatchCodeRead(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Read(ctx, argString(args, "path"),
			int(argNumber(args, "start_line")), int(argNumber(args, "end_line")),
			argString(args, "mode"))
	}
}

func dispatchCodeOrient(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		depth := int(argNumber(args, "depth"))
		if sym := argString(args, "symbol"); sym != "" {
			// Resolve the symbol name to its id (the orient's unit).
			syms, err := h.Store().SymbolByName(ctx, sym)
			if err != nil || len(syms) == 0 {
				return nil, fmt.Errorf("code_orient: symbol %q not found", sym)
			}
			return h.Orient(ctx, graph.OrientUnit{Kind: graph.UnitSymbol, ID: syms[0].ID}, depth)
		}
		if pkg := argString(args, "package"); pkg != "" {
			return h.Orient(ctx, graph.OrientUnit{Kind: graph.UnitPackage, Path: pkg}, depth)
		}
		if f := argString(args, "file"); f != "" {
			return h.Orient(ctx, graph.OrientUnit{Kind: graph.UnitFile, Path: f}, depth)
		}
		return nil, fmt.Errorf("code_orient: provide 'symbol', 'package', or 'file'")
	}
}

func dispatchCodeExplain(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		sym := argString(args, "symbol")
		syms, err := h.Store().SymbolByName(ctx, sym)
		if err != nil || len(syms) == 0 {
			return nil, fmt.Errorf("code_explain: symbol %q not found", sym)
		}
		return h.Explain(ctx, syms[0].ID)
	}
}

func dispatchCodePath(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		from := argString(args, "from")
		fromSyms, err := h.Store().SymbolByName(ctx, from)
		if err != nil || len(fromSyms) == 0 {
			return nil, fmt.Errorf("code_path: from symbol %q not found", from)
		}
		to := argString(args, "to")
		if to == "" {
			// The graph-stop view (toID = 0).
			return h.Path(ctx, fromSyms[0].ID, 0)
		}
		toSyms, err := h.Store().SymbolByName(ctx, to)
		if err != nil || len(toSyms) == 0 {
			return nil, fmt.Errorf("code_path: to symbol %q not found", to)
		}
		return h.Path(ctx, fromSyms[0].ID, toSyms[0].ID)
	}
}

func dispatchCodeImpact(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		sym := argString(args, "symbol")
		syms, err := h.Store().SymbolByName(ctx, sym)
		if err != nil || len(syms) == 0 {
			return nil, fmt.Errorf("code_impact: symbol %q not found", sym)
		}
		return h.Impact(ctx, syms[0].ID, argString(args, "min_confidence"), int(argNumber(args, "max_depth")))
	}
}

func dispatchCodeAffected(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		// F3 — the base arg (spec 008: {"changed": [...]} / {"base": "<ref>"},
		// exactly one set). The MCP transport has no git: when base is set
		// (and changed is empty), report the CLI command (the CLI --base does
		// the git-diff; the MCP surface declares the arg for the contract).
		base := argString(args, "base")
		changed := argStringSlice(args, "changed")
		if base != "" && len(changed) == 0 {
			return nil, fmt.Errorf("code_affected: the MCP transport has no git — run `mnemonic affected --base %s` (the CLI does the git-diff)", base)
		}
		return h.Affected(ctx, changed, int(argNumber(args, "depth")))
	}
}

func dispatchCodeGrep(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Grep(ctx, argString(args, "pattern"), argString(args, "language"))
	}
}

func dispatchCodeSemantic(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Semantic(ctx, argString(args, "query"), int(argNumber(args, "limit")))
	}
}

func dispatchCodeNeighbors(h *handler.CodeHandler, view string) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		sym := argString(args, "symbol")
		syms, err := h.Store().SymbolByName(ctx, sym)
		if err != nil || len(syms) == 0 {
			return nil, fmt.Errorf("code neighbor view: symbol %q not found", sym)
		}
		return h.Neighbors(ctx, syms[0].ID, view)
	}
}

func dispatchCodeResolve(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		var calledFrom *string
		if cf := argString(args, "called_from"); cf != "" {
			calledFrom = &cf
		}
		return h.Resolve(ctx, argString(args, "symbol"), calledFrom)
	}
}

func dispatchCodePackages(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		pkgs, err := h.Packages(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"packages": pkgs}, nil
	}
}

func dispatchCodeRename(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Rename(ctx, argString(args, "old"), argString(args, "new"), argBool(args, "dry_run"))
	}
}

func dispatchCodeUnresolved(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Unresolved(ctx, int(argNumber(args, "limit")))
	}
}

func dispatchCodeCircles(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Circles(ctx)
	}
}

func dispatchCodeDeps(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Deps(ctx)
	}
}

func dispatchCodeDepImpact(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.DepImpact(ctx, argString(args, "purl"))
	}
}

func dispatchCodeEmbeddingStatus(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.EmbeddingStatus(ctx)
	}
}

func dispatchCodeWatch(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Watch(ctx, argString(args, "project"))
	}
}

func dispatchCodeDeferred(h *handler.CodeHandler, tool, flag string) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		// The v2.1-deferred contract: report "run code_index --flag" (deferred,
		// not missing). The error is mapped to the envelope by the server.
		return nil, h.DeferredTool(ctx, tool, flag)
	}
}

func dispatchCodeRoute(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Route(ctx, argString(args, "handler"), argString(args, "path"), int(argNumber(args, "limit")))
	}
}

func dispatchCodeNavigates(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Navigates(ctx, argString(args, "from"), argString(args, "to"), int(argNumber(args, "limit")))
	}
}

func dispatchCodeDocs(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Docs(ctx, argString(args, "path"), int(argNumber(args, "limit")))
	}
}

func dispatchCodeConfigs(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Configs(ctx, argString(args, "path"), int(argNumber(args, "limit")))
	}
}

func dispatchCodeSQLSchema(h *handler.CodeHandler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.SQLSchema(ctx, argString(args, "table"), int(argNumber(args, "limit")))
	}
}

// searchQueryFromArgs maps the code_search args onto the search.SearchQuery.
func searchQueryFromArgs(args map[string]any) search.SearchQuery {
	q := search.SearchQuery{}
	q.Query = argString(args, "query")
	q.Project = argString(args, "project")
	if v := argNumber(args, "limit"); v > 0 {
		q.Limit = int(v)
	}
	q.Language = argString(args, "language")
	q.Under = argString(args, "under")
	return q
}
