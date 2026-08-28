## MODIFIED Requirements

### Requirement: MCP stdio transport

The system SHALL provide `skillgrid mcp` as an MCP stdio server for agent tool calls,
advertising the existing `mem_*`, `code_*`, and `web_*` tools plus the new `graph_*`,
exploration (`list_repos`, `check`, `rename`, `detect_changes`, `cypher`), memory-extension
(`mem_timeline`, `mem_stats`, `mem_doctor`, `mem_current_project`, `mem_update`,
`mem_delete`, `mem_save_prompt`, `mem_capture_passive`, `mem_review`), PDG/taint
(`explain`, `pdg_query`), and API-contract (`route_map`, `tool_map`, `shape_check`,
`api_impact`) tools, and the `graph_judge` verdict tool (27 new tools in total).

#### Scenario: MCP server starts
- **GIVEN** `skillgrid mcp` is invoked
- **WHEN** the command runs
- **THEN** an MCP stdio server starts and serves tool calls

#### Scenario: All tools registered
- **GIVEN** the MCP server is running
- **WHEN** an agent requests the tool list
- **THEN** `mem_*`, `code_*`, `web_*`, `graph_*`, exploration, memory-extension, PDG, and contract tools are all advertised (existing 17 + new 27)

#### Scenario: New tool dispatch
- **GIVEN** an agent calls a newly added tool via MCP (e.g., `graph_impact`, `rename`, `cypher`, `explain`, `route_map`, `mem_review`)
- **WHEN** the tool is dispatched
- **THEN** the service layer executes the corresponding read or write and returns a raw-JSON result through the OCBI convention

#### Scenario: Pass-gated tool when pass disabled
- **GIVEN** a pass-gated tool (`explain`, `pdg_query`, `route_map`, `tool_map`, `shape_check`, `api_impact`) is called on a project where the underlying pass is disabled
- **WHEN** the tool is dispatched
- **THEN** it returns a not-available message naming the required config gate, not an empty success

### Requirement: HTTP REST transport

The system SHALL provide `skillgraph serve` as an HTTP server, exposing the existing
memory/code/web routes plus the new `/graph/*` and `/memory/*` routes and the
`/graph` visualization page.

#### Scenario: HTTP server starts
- **GIVEN** `skillgrid serve` is invoked
- **WHEN** the command runs
- **THEN** an HTTP server starts on `127.0.0.1:7438`

#### Scenario: Health check
- **GIVEN** the HTTP server is running
- **WHEN** `GET /health` is called
- **THEN** `{"status":"ok","service":"skillgrid-memindex","version":"..."}` is returned

#### Scenario: Graph and memory routes
- **GIVEN** the HTTP server is running
- **WHEN** `/graph/*` and `/memory/*` routes are called
- **THEN** they behave equivalently to their MCP counterparts and write routes apply the existing bearer-token auth when configured
