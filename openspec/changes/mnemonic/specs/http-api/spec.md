## ADDED Requirements

### Requirement: HTTP API surfaces

The system SHALL expose REST endpoints for memory, code, and web cache operations.

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

### Requirement: Plugin usage pattern

OpenCode/Kilo plugins SHALL use the HTTP API for session lifecycle and compaction.

#### Scenario: Plugin auto-starts HTTP server
- **GIVEN** the plugin loads and `GET /health` fails
- **WHEN** the plugin initializes
- **THEN** it spawns `skillgrid serve` in the background

#### Scenario: Session created on start
- **GIVEN** the plugin is loaded
- **WHEN** a new session starts
- **THEN** `POST /sessions` is called with the workspace directory

#### Scenario: Compaction recovery
- **GIVEN** the plugin is loaded and a compaction occurs
- **WHEN** the compaction hook fires
- **THEN** `GET /context` is called and the result injected before the agent continues
