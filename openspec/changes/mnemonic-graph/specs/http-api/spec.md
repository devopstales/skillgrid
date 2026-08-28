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
- **WHEN** `GET /graph/subgraph`, `GET /graph/node/{id}`, `GET /graph/node/{id}/neighbors`, `GET /graph/impact/{id}`, `GET /graph/trace`, `GET /graph/status` are called
- **THEN** they behave equivalently to their MCP counterparts
- **WHEN** `POST /graph/link` is called
- **THEN** it upserts a manual edge subject to the same bearer-token auth as other write routes
