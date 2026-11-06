# 008 — Transport (the `code_*` MCP family + CLI)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** memory `00-foundation/004-transport-mcp` §1/§4 (the one-server-three-
families layout + the shared server conventions), `00-foundation/005-transport-cli`
(the CLI delegation), `004-graph-queries` (the reads), `006-search` (the search reads),
`001-files-chunks` (`code_status` reads `index_freshness`).
**Consumed by:** the KB `002-sources`/`006-validate`/`007-lint` (the `code_resolve` +
`code_packages` + `code_search` reads), the opencode/pi plugins (the `code_*` pass-
through).

---

## 1. Capability

The code-index exposes its capability as the **`code_*` family** of the **one MCP
server** (memory `004-transport-mcp` §1: "one MCP server, three tool families: `mem_*`,
`code_*`, `wiki_*`"). This spec defines the `code_*` family. It re-uses the **shared
server conventions** (memory `004-transport-mcp` §4) — it does not re-declare them:

- **Project resolution at tool-call time** (cwd → config → git binding → basename).
- **Write queue** (the shared MCP write queue; `code_index` is a queued write).
- **Degradation in the envelope (INV-3)** — `degraded: true` + reason on every
  `code_*` tool (stale index, no vector, no LSP, no graph, no grammar).
- **`sync_id` where identity matters (INV-6)** — `code_resolve` returns `sync_id`; the
  local `id`/`uid` are internal.
- **HTTP mirror** — every `code_*` tool has an HTTP mirror (POST/GET) for the CLI +
  non-MCP clients. Same contract, same write queue.

## 2. Profiles

Same two profiles as the memory change (memory `004-transport-mcp` §2):
- **`agent`** — the tools an agent may call during work (the default for plugins).
- **`all`** — `agent` + admin tools (ops/CLI).

## 3. The `code_*` tool surface

### agent profile (16 tools)

| Tool | Purpose | Spec basis |
|---|---|---|
| `code_status` | Index health: stale flag, fair coverage, unresolved counts, **capabilities** (graph/fts/vector/lsp/pdg/communities/process) | `001` §4 |
| `code_index` | Run incremental index (cwd git root); `--full`/`--lsp`/`--pdg`/`--communities`/`--process` flags | `003` §7, `005` §2 |
| `code_search` | Hybrid search (FTS + structural + semantic, RRF) — the "prefer over grep" tool | `006` §5 |
| `code_read` | Fetch indexed source for path/line-range (secret redaction) | `006` §7 |
| `code_orient` | **The navigation entry point** — the progressive-disclosure "map" of a symbol/package/file (metadata + TOC + bounded outbound edges, `depth`-re-rootable). The agent's "find a feature" first verb | `004` §8 |
| `code_explain` | Node + degree + connections ranked by neighbor degree | `004` §8 |
| `code_path` | BFS shortest path or "where the graph stops" | `004` §7 |
| `code_impact` | Risk-tiered blast radius (WILL BREAK / LIKELY AFFECTED) | `004` §5 |
| `code_affected` | "which tests do I run after this change?" | `004` §6 |
| `code_grep` | Index-free structural AST grep (by-example) | `006` §3 |
| `code_semantic` | Embedding search (symbol + chunk) | `006` §5 |
| `code_get_callers` / `code_get_callees` / `code_get_dependents` / `code_get_implementors` / `code_get_hierarchy` / `code_get_tests_for` | The neighbor views | `004` §2 |
| **`code_resolve`** | **NEW** — resolve a `symbol:` anchor → `{exists, file, line, called_from[], ambiguous, candidates[]}` (the KB pin-check contract) | `004` §3 |
| **`code_packages`** | **NEW** — top-level Go package map (the KB coverage-gap contract) | `004` §4 |

> **`code_resolve` + `code_packages` are the two load-bearing additions** (the KB's
> forward-refs resolve to them — `000-overview` §5). The other 14 agent tools are v1
> tools kept/renamed.

### admin profile (`all`, not `agent`)

| Tool | Purpose | v2 status |
|---|---|---|
| `code_rename` | Rename plan (graph edits vs text-search edits, dry_run default) | v2 |
| `code_unresolved` | The drop-not-guess backlog (`unresolved_edges` ledger) | v2 |
| `code_circles` | Import cycles (re-derived from `kind='imports'` edges, not a stored table) | v2 |
| `code_deps` / `code_dep_impact` | The SBOM dependency graph (`dependencies`/`dep_edges`) | v2 (thin, `002` §7) |
| `code_embedding_status` | provider/model/dim/embedded count | v2 |
| `code_watch` | Start the live incremental watcher | v2 (`007`) |
| `code_pdg_query` / `code_taint` | PDG + taint reads | **v2.1** (report "run `--pdg`" in v2) |
| `code_communities` / `code_god_nodes` / `code_explain_community` | Leiden partition reads | **v2.1** (report "run `--communities`" in v2) |
| `code_processes` / `code_process` | Execution-flow reads | **v2.1** (report "run `--process`" in v2) |
| `code_route` / `code_navigates` / `code_docs` / `code_configs` / `code_sql_schema` | The knowledge-graph reads | v2 (the knowledge nodes are in the core graph, `003` §5) |

**The v2.1 admin tools** (`code_pdg_query`, `code_taint`, `code_communities`, …) are
**declared in the surface** (so the contract is fixed) but **report "run
`code_index --<flag>`"** in v2 (their tables are absent until the v2.1 pass runs,
`005` §4–6). They are not *missing* — they are *deferred*.

## 4. The CLI (delegated by memory `005-transport-cli`)

`mnemonic index` (memory's CLI) **delegates to the code-index change**. The code-index
defines the full CLI command group (the HTTP-mirrored tools, same contract):

```
mnemonic index [flags]        incremental index (--full/--lsp/--pdg/--communities/--process)
mnemonic status               index health (the code_status read)
mnemonic search <query>       hybrid search (--language, --limit)
mnemonic orient <symbol>      symbol orientation
mnemonic explain <symbol>     the explain read
mnemonic path <from> <to>     the path read
mnemonic impact <symbol>      blast radius (--min-confidence, --max-depth)
mnemonic affected [base]      tests to run (--base, --stdin, --changed)
mnemonic grep <pattern>       structural AST grep (--path, --language)
mnemonic read <path>          fetch a span (--start-line, --end-line)
mnemonic watch                start the live watcher
```

**Conventions** (inherited, memory `005-transport-cli`): project selection explicit for
reads (omitted → current project); `--json` for machine output (matches the MCP
contract); `--datasource` overrides the store dialect (SQLite → PG) for the invocation;
`--lsp`/`--pdg`/`--communities`/`--process` gate the optional passes.

## 5. The `code_status` capabilities report (the KB's "what can the graph answer?")

`code_status` reads `index_freshness` (`001` §4) and reports:
- the **stale flag** (the stat-walk result: fresh / stale),
- the **counts** (files/symbols/edges/chunks/unresolved),
- the **fair coverage** (per language, `004` §9),
- the **`capabilities` JSON** — which features are on/off:
  `{graph: true, fts: true, vector: <bool>, lsp: <bool>, pdg: <bool>,
  communities: <bool>, process: <bool>}`.
- the **extractor tier per language** (tree-sitter-rich / tree-sitter-map / regex — the
  degradation is visible, `003` §3.3).
- the **last index time** + the **watch status** (active/inactive, last re-index).

**The KB reads `code_status`'s `capabilities`** to know: is the graph built? is LSP on
(so the `symbol:` call-site check is tight, or best-effort)? is the vector leg up (so
`code_search` is fused, or FTS-only)? This is how the KB's **no-graph / no-LSP
degradation** (CI-7) is *discovered* at runtime, not assumed.

## 6. The write path (async, queued)

`code_index` is a **write** (it mutates `files`/`symbols`/`edges`/`chunks`). It is
**queued** (the shared MCP write queue) and **async** for the run: a POST that starts an
index returns a `run_id` + `status` immediately; the index completes on the write queue.
A repeat POST for an already-active index returns `409` (do not double-start). The
*optional passes* (LSP/PDG/communities/process) run after the core tx commits (separate
connection, `003` §7, `005` §2). **Read tools** (`code_search`/`code_resolve`/
`code_packages`/`code_status`/…) are **synchronous**.

## 7. Invariants (re-stated)

INV-2 (dual-mode, `--datasource`), INV-3 (degradation envelope on every `code_*` tool),
INV-5 (trails recorded by the query path), INV-6 (`sync_id`), INV-7 (the engine
extracts — the agent does not hand-edit the graph). CI-INV-1 (derived), CI-INV-2
(current-state), CI-INV-3 (drop-not-guess), CI-INV-4 (stale is a signal, not a failure).

## 8. Open items

None. The tool surface (16 agent + the admin set), the profiles, the CLI group, and the
`code_status` capabilities report are fixed. The v2.1 admin tools are declared
(deferred, not missing). The one carried tunable is the `MaxFileSize` + the
debounce window (config defaults, `001`/`007`).
