## ADDED Requirements

### Requirement: Graph MCP tools

The system SHALL expose graph functionality through the `skillgrid mcp` stdio server as
`graph_*` tools returning raw JSON.

#### Scenario: graph_nodes search
- **GIVEN** symbols and observations exist in the graph
- **WHEN** an agent calls `graph_nodes` with a query and optional kind filter
- **THEN** matching nodes are returned ranked by search relevance and graph degree

#### Scenario: graph_context lookup
- **GIVEN** a node exists in the graph
- **WHEN** `graph_context` is called for that node
- **THEN** the node, its adjacent edges, and neighbor nodes with source-row summaries are returned

#### Scenario: graph_impact analysis
- **GIVEN** a symbol with callers and callees
- **WHEN** `graph_impact` is called with a direction (upstream or downstream)
- **THEN** the reachable set with hop distances and edge types is returned

#### Scenario: graph_trace between nodes
- **GIVEN** two nodes connected by a directed edge path
- **WHEN** `graph_trace` is called with from, to, and edge types
- **THEN** the shortest directed path is returned, or a not-found error if no path exists

#### Scenario: graph_slink manual edge
- **GIVEN** two existing graph nodes
- **WHEN** `graph_slink` is called with a valid edge type between them
- **THEN** the edge is created with `properties_json.source` set to `manual`
- **THEN** re-running automatic edge synthesis does not remove the manual edge

#### Scenario: graph_status report
- **GIVEN** any project state
- **WHEN** `graph_status` is called
- **THEN** node and edge counts by kind and type, the full accepted edge-type vocabulary, symbol freshness, and any extraction errors are returned

#### Scenario: Node not found
- **GIVEN** a node reference that does not exist
- **WHEN** any graph tool referencing it is called
- **THEN** the tool returns a JSON error object naming the missing node

### Requirement: Graph HTTP routes

The system SHALL expose graph endpoints on `skillgrid serve` equivalent to the MCP tools.

#### Scenario: Graph REST equivalence
- **GIVEN** the HTTP server is running
- **WHEN** `GET /graph/subgraph`, `GET /graph/node/{id}`, `GET /graph/node/{id}/neighbors`, `GET /graph/impact/{id}`, `GET /graph/trace`, `GET /graph/status` are called
- **THEN** they behave equivalently to their MCP counterparts

#### Scenario: Manual link creation over HTTP
- **GIVEN** the HTTP server is running with `SKILLGRID_HTTP_TOKEN` set
- **WHEN** `POST /graph/link` is called without a bearer token
- **THEN** the request is rejected with `401 Unauthorized`

#### Scenario: Subgraph payload shape
- **GIVEN** the server is running with graph data
- **WHEN** `GET /graph/subgraph` is called
- **THEN** a single JSON object with `nodes` (id, label, kind, props) and `edges` (source, target, type) arrays is returned

### Requirement: Auto-synthesized edges

The system SHALL derive deterministic edges from existing data rows and rebuild them without
removing manual edges.

#### Scenario: Session contains observations
- **GIVEN** a session with associated observations
- **WHEN** auto-synthesis runs
- **THEN** a CONTAINS edge exists from the session node to each observation node

#### Scenario: Observation mentions files
- **GIVEN** an observation whose content references an indexed file path and that file is indexed
- **WHEN** auto-synthesis runs
- **THEN** a MENTIONS edge exists from the observation node to the file node

#### Scenario: Manual edges survive rebuild
- **GIVEN** a manual edge and auto edges on the same node pair
- **WHEN** auto-synthesis runs again
- **THEN** the auto edges are rebuilt and the manual edge remains

### Requirement: Exploration and analysis MCP tools

The system SHALL expose `list_repos`, `check`, `rename`, `detect_changes`, and `cypher`
tools for repository discovery, structural health, rename planning, change mapping, and
raw graph queries.

#### Scenario: List repositories
- **WHEN** `list_repos` is called
- **THEN** all project stores with per-project graph size and staleness are returned

#### Scenario: Structural check
- **WHEN** `check` is called for a project
- **THEN** orphan nodes, symbols without `DEFINES`, per-pass staleness, and an edge-type distribution are returned

#### Scenario: Rename planner
- **GIVEN** a symbol with in-file and cross-file usages
- **WHEN** `rename` is called for that symbol
- **THEN** all affected locations (call sites, `CALLS` edges, `MENTIONS` files) are returned ranked, without any mutation

#### Scenario: Detect changes
- **GIVEN** a git diff (working tree or since a base ref)
- **WHEN** `detect_changes` is called
- **THEN** the changed files, touched symbols, downstream impact, and observations mentioning the files are returned

#### Scenario: Cypher supported subset
- **GIVEN** a query within the supported mini-Cypher subset
- **WHEN** `cypher` is called
- **THEN** the matching nodes and edges are returned as a JSON array

#### Scenario: Cypher unsupported feature
- **GIVEN** a query using a feature outside the supported subset (e.g., variable-length path, `OPTIONAL MATCH`, aggregation, or `MERGE`)
- **WHEN** `cypher` is called
- **THEN** a structured error names the unsupported feature and lists the supported subset

### Requirement: PDG and taint MCP tools

The system SHALL expose `explain` and `pdg_query` tools that read PDG and taint data
persisted at index time.

#### Scenario: Explain a taint finding
- **GIVEN** a precomputed taint flow for a symbol
- **WHEN** `explain` is called for that flow
- **THEN** the `TAINT_PATH`, block steps, sanitizers, and confidence are returned

#### Scenario: Query the PDG
- **GIVEN** a function with persisted PDG data
- **WHEN** `pdg_query` is called for that function
- **THEN** the CFG/CDG/reaching-def view is returned, optionally scoped to a from/to block range and type

#### Scenario: PDG not available
- **GIVEN** a symbol with no persisted PDG data (`pdg.enabled=false` or the pass was not run)
- **WHEN** `explain` or `pdg_query` is called for that symbol
- **THEN** the tool returns a not-available message naming the symbol and the required config gate

### Requirement: API contract MCP tools

The system SHALL expose `route_map`, `tool_map`, `shape_check`, and `api_impact` tools
that read contract data persisted at index time.

#### Scenario: Route map
- **WHEN** `route_map` is called for a project
- **THEN** routes → handlers → fetchers grouped by framework, with entry points, are returned

#### Scenario: Tool map
- **WHEN** `tool_map` is called for a project
- **THEN** MCP/RPC tools → definitions → handlers → callers are returned

#### Scenario: Shape check
- **WHEN** `shape_check` is called for a route
- **THEN** matched, missing, and extra fields per consumer are returned

#### Scenario: API impact
- **GIVEN** a route handler feeding one or more consumers
- **WHEN** `api_impact` is called for that route
- **THEN** the downstream consumers/tools/routes it feeds are returned with hop depths

#### Scenario: Contracts not available
- **GIVEN** a project with no persisted contract data (`contracts.enabled=false` or the pass was not run)
- **WHEN** any contract tool is called
- **THEN** the tool returns a not-available message naming the required config gate

### Requirement: Memory and verdict tools over transport

The system SHALL expose the memory tools (`mem_timeline`, `mem_stats`, `mem_doctor`,
`mem_current_project`, `mem_update`, `mem_delete`, `mem_save_prompt`,
`mem_capture_passive`, `mem_review`) and `graph_judge` through both the `skillgrid mcp`
stdio server and `skillgrid serve` HTTP routes.

#### Scenario: Memory tools return raw JSON
- **WHEN** any memory tool is called via MCP
- **THEN** it returns raw JSON through the existing `JSONResult` wrapper

#### Scenario: Memory HTTP write routes require auth
- **GIVEN** the HTTP server is running with `SKILLGRID_HTTP_TOKEN` set
- **WHEN** `POST /memory/prompts`, `POST /memory/capture-passive`, `PATCH /memory/observations/{id}`, `DELETE /memory/observations/{id}`, or `POST /graph/judge` is called without a bearer token
- **THEN** the request is rejected with `401 Unauthorized`

#### Scenario: Memory HTTP equivalence
- **GIVEN** the HTTP server is running
- **WHEN** `GET /memory/prompts/recent`, `GET /memory/timeline`, `GET /memory/stats`, `GET /memory/doctor`, `GET /memory/current-project`, or `GET /memory/review` are called
- **THEN** they behave equivalently to their MCP counterparts
