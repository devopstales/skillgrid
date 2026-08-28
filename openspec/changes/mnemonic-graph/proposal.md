## Why

Mnemonic stores memory (observations), a code index (files/chunks), and a web cache as
relational rows with FTS5 search. There is no way to ask *relationship* questions — "what
is related to this decision?", "which files does this bugfix observation touch?", "what is
the blast radius of this symbol?" — because nothing links rows across the three stores and
GitNexus-style structural depth (callers, callees, impact, trace) is unavailable in the
single binary.

GitNexus proves the shape works: a property graph (typed nodes + typed edges) over code,
queried through high-level tools (`context`, `impact`, `trace`, `query`) and visualized
with Sigma.js. Mnemonic adds the memory and web-research dimension GitNexus lacks, and the
request is to bring GitNexus-like graph functionality into Mnemonic — a cross-entity
knowledge graph — while keeping the existing stores intact and storing graph data in
separate tables.

Scope note: this change is **GitNexus capability parity**, computed offline and stored in
Mnemonic's existing SQLite graph — the same capabilities an agent can invoke (context,
impact, trace, rename, check, detect_changes, taint/PDG, route/tool maps, shape check,
API impact, raw graph query), plus the memory dimension GitNexus lacks. Work is broken
into five phases, each independently shippable and config-gated.

## What Changes

- New `graph` module (`skillgrid-cli/internal/mnemonic/graph/`) over the existing
  `store` — a property-graph layer on SQLite: `graph_nodes` + `graph_edges` tables and
  FTS over node names.
- Migrations `002_graph.sql`, `003_memory_ext.sql`, `004_pdg_contracts.sql` — all
  additive; existing tables untouched.
- Service methods on `service.Service` (facade pattern unchanged): node/edge upserts,
  traversal (neighbors, impact, shortest path via recursive CTEs), node search, subgraph
  extraction, taint/PDG reads, API-contract reads, mini-Cypher evaluation, stats.
- **27 new MCP tools** across five groups (details in design.md):
  - Core graph (6): `graph_slink`, `graph_nodes`, `graph_context`, `graph_impact`,
    `graph_trace`, `graph_status`
  - Exploration/analysis (5): `list_repos`, `check`, `rename` (read-only planner),
    `detect_changes`, `cypher` (mini-Cypher subset)
  - Memory (10): `mem_timeline`, `mem_stats`, `mem_doctor`, `mem_current_project`,
    `mem_update`, `mem_delete`, `mem_save_prompt`, `mem_capture_passive`, `mem_review`,
    `graph_judge`
  - PDG/taint (2, opt-in): `explain`, `pdg_query`
  - API contracts (4, opt-in): `route_map`, `tool_map`, `shape_check`, `api_impact`
- New HTTP routes on `skillgrid serve`: `GET/POST /graph/...` (subgraph, node, neighbors,
  impact, trace, link, status, taint, pdg, routes, tools, shapes, api-impact) +
  `GET/POST /memory/...` (prompts, timeline, stats, doctor, update, delete, review,
  capture-passive, judge) + a `/graph` visualization page (Sigma.js + graphology,
  embedded, no CDN, ForceAtlas2 layout — same stack as gitnexus-web).
- Symbol pipeline (phase 2): **Go + TypeScript + Python** symbol extraction
  (functions/methods/types) feeding `symbol` nodes + `DEFINES`/`CALLS`/`IMPORTS`/
  `EXTENDS` edges, running inside the existing incremental `codeindex` flow with a config
  gate.
- **PDG / taint pass** (phase 3, opt-in): per-function `basic_block` nodes,
  `CFG`/`CDG`/`REACHING_DEF` edges, `SOURCE`/`SINK`/`SANITIZES`/`TAINTED`/`TAINT_PATH`
  precomputed flows (intra- then interprocedural).
- **API-contract pass** (phase 4, opt-in): `route`/`tool`/`shape` nodes +
  `HANDLES_ROUTE`/`FETCHES`/`ENTRY_POINT_OF`/`HANDLES_TOOL`/`DEFINES_ROUTE`/
  `CONSUMES_PROP` edges for HTTP/REST + MCP/RPC frameworks.
- Phase 1 auto-synthesis: deterministic edges from existing rows
  (`CONTAINS` session→observation, `MENTIONS` observation→file, `CITES`
  observation→web_entry, `FOLLOWS_FROM` session→session) plus agent-authored links and
  conflict verdicts (`CONFLICTS_WITH`/`SUPERSEDES` via `graph_judge`).
- Memory extensions: `user_prompts` table (+FTS) for `mem_save_prompt`;
  `observations.review_after` column for `mem_review`.
- Documentation: `docs/13-mnemonic.md` gains a graph + memory-extension section; OpenAPI
  spec adds `/graph/*` and `/memory/*` routes.

No breaking changes: no existing tool, route, table, or config key is renamed or removed.

## Capabilities

### New Capabilities

- `graph-layer`: Property-graph store (nodes + typed edges in separate tables) with
  idempotent upserts and recursive-CTE traversal (neighbors, impact, shortest path)
- `graph-symbols`: Multi-language (Go/TypeScript/Python) symbol extraction
  (function/method/type nodes, CALLS/IMPORTS/DEFINES/EXTENDS edges) integrated into
  incremental code indexing
- `graph-pdg`: Program-dependence-graph + taint flows per function (basic blocks,
  CFG/CDG/reaching-defs, source/sink/sanitizer edges, precomputed taint paths)
- `graph-contracts`: API route/tool/response-shape extraction with consumer mapping
- `graph-transport`: `graph_*` + `mem_*` + exploration MCP tools and
  `/graph/*` + `/memory/*` HTTP routes over the service
- `graph-viz`: Browser graph visualizer (Sigma.js + graphology) served at `/graph`

### Modified Capabilities

- `code-layer`: Indexing gains opt-in symbol, PDG, and API-contract extraction passes
  (all config-gated; off by default in `config.d/indexing.yaml`)
- `memory-layer`: New persistence for user prompts (`mem_save_prompt`) and a per-
  observation review cycle (`mem_review`) + `review_after` column
- `http-api`: Serve routes extended with `/graph/*`, `/memory/*`, and the `/graph`
  static page
- `mcp-transport`: 27 new tools registered alongside existing `mem_*`/`code_*`/`web_*`

## Impact

- **Code**: `skillgrid-cli/{internal/mnemonic/{store,migrations,graph,service,mcp,http,ui,codeindex},cmd/skillgrid}`
- **Deps**: `github.com/smacker/go-tree-sitter` (multi-grammar: go, typescript, python;
  cgo — build-only gcc requirement), `@sigma.*`/`graphology` vendored as embedded static
  assets (same pattern as swagger-ui-dist)
- **Data**: Additive schema only (graph, memory ext, pdg/contracts); old `.sqlite` files
  work unchanged (migrations run)
- **Docs**: `docs/13-mnemonic.md`, `openspec/specs` (delta specs in this change)
- **Build**: `task build`/`task all` unaffected; binary size +~1.5 MB (sigma assets) +
  tree-sitter cgo for three grammars; no new runtime dependencies
