## MODIFIED Requirements

### Requirement: HTTP API surfaces

The system SHALL expose REST endpoints for memory, code, web cache, and graph operations.

#### Scenario: Memory endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `POST /observations`, `GET /search`, `GET /observations/recent` are called
- **THEN** they behave equivalently to their MCP counterparts

#### Scenario: Code endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `GET /code/status`, `POST /code/index`, `GET /code/search`, `GET /code/read` are called
- **THEN** they behave equivalently to their MCP counterparts

#### Scenario: Web cache endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `GET /web/lookup`, `POST /web/cache`, `GET /web/search` are called
- **THEN** they behave equivalently to their MCP counterparts

#### Scenario: Graph endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `GET /graph/subgraph`, `GET /graph/node/{id}`, `GET /graph/node/{id}/neighbors`, `GET /graph/impact/{id}`, `GET /graph/trace`, `GET /graph/status`, `GET /graph/check`, `GET /graph/rename`, or `GET /graph/detect-changes` are called
- **THEN** they behave equivalently to their MCP counterparts
- **WHEN** `POST /graph/link` or `POST /graph/cypher` is called
- **THEN** it upserts a manual edge or evaluates a mini-Cypher query, respectively, subject to the same bearer-token auth as other write routes

#### Scenario: PDG and contract endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `GET /graph/taint/{symbol_id}`, `GET /graph/pdg/{symbol_id}`, `GET /graph/routes`, `GET /graph/tools`, `GET /graph/shapes/{route_id}`, or `GET /graph/api-impact/{route_id}` are called
- **THEN** they behave equivalently to their MCP counterparts (`explain`, `pdg_query`, `route_map`, `tool_map`, `shape_check`, `api_impact`)

#### Scenario: Memory-extension endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `GET /memory/prompts/recent`, `GET /memory/timeline`, `GET /memory/stats`, `GET /memory/doctor`, `GET /memory/current-project`, or `GET /memory/review` are called
- **THEN** they behave equivalently to their MCP counterparts
- **WHEN** `POST /memory/prompts`, `POST /memory/capture-passive`, `PATCH /memory/observations/{id}`, `DELETE /memory/observations/{id}`, or `POST /graph/judge` are called
- **THEN** they behave equivalently to their MCP counterparts and are subject to the same bearer-token auth as other write routes when a token is configured

#### Scenario: Visualization page
- **GIVEN** the HTTP server is running
- **WHEN** `GET /graph` and its embedded static assets are requested
- **THEN** the visualization page and assets are served
