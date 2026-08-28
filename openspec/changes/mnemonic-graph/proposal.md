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

## What Changes

- New `graph` module (`skillgrid-cli/internal/mnemonic/graph/`) over the existing
  `store` — a property-graph layer on SQLite: `graph_nodes` + `graph_edges` tables and
  FTS over node names.
- Migration `002_graph.sql` — additive; existing tables untouched.
- Service methods on `service.Service` (facade pattern unchanged): node/edge upserts,
  traversal (neighbors, impact, shortest path via recursive CTEs), node search, subgraph
  extraction, stats.
- New `graph_*` MCP tools: `graph_slink`, `graph_nodes`, `graph_context`, `graph_impact`,
  `graph_trace`, `graph_status`.
- New HTTP routes on `skillgrid serve`: `GET/POST /graph/...` (subgraph, node, neighbors,
  impact, trace, link, status) + a `/graph` visualization page (Sigma.js + graphology,
  embedded, no CDN, ForceAtlas2 layout — same stack as gitnexus-web).
- Symbol pipeline (phase 2): Go-file symbol extraction (functions/methods/types) feeding
  `symbol` nodes + `DEFINES`/`CALLS`/`IMPORTS`/`EXTENDS` edges, running inside the
  existing incremental `codeindex` flow with a config gate.
- Phase 1 auto-synthesis: deterministic edges from existing rows
  (`CONTAINS` session→observation, `MENTIONS` observation→file, `CITES`
  observation→web_entry, `FOLLOWS_FROM` session→session) plus agent-authored links.
- Documentation: `docs/13-mnemonic.md` gains a graph section; OpenAPI spec adds `/graph/*`
  routes.

No breaking changes: no existing tool, route, table, or config key is renamed or removed.

## Capabilities

### New Capabilities

- `graph-layer`: Property-graph store (nodes + typed edges in separate tables) with
  idempotent upserts and recursive-CTE traversal (neighbors, impact, shortest path)
- `graph-symbols`: Go symbol extraction (function/method/type nodes, CALLS/IMPORTS/
  DEFINES/EXTENDS edges) integrated into incremental code indexing
- `graph-transport`: `graph_*` MCP tools and `/graph/*` HTTP routes over the graph service
- `graph-viz`: Browser graph visualizer (Sigma.js + graphology) served at `/graph`

### Modified Capabilities

- `code-layer`: Indexing gain an opt-in symbol extraction pass (config-gated,
  off-by-default until first release of `graph-symbols` ships Go support)
- `http-api`: Serve routes extended with `/graph/*` and the `/graph` static page

## Impact

- **Code**: `skillgrid-cli/{internal/mnemonic/{store,migrations,graph,service,mcp,http,ui},cmd/skillgrid}`
- **Deps**: `github.com/smacker/go-tree-sitter` (+go module, cgo — build-only gcc
  requirement), `@sigma.*`/`graphology` vendored as embedded static assets (same pattern as
  swagger-ui-dist)
- **Data**: Additive schema only; old `.sqlite` files work unchanged (migrations run)
- **Docs**: `docs/13-mnemonic.md`, `openspec/specs` (delta specs in this change)
- **Build**: `task build`/`task all` unaffected; binary size +~1.5 MB (sigma assets) +
  tree-sitter cgo for the go grammar
