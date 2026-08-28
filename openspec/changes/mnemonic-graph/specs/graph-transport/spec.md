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
