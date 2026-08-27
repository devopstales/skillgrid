## ADDED Requirements

### Requirement: MCP stdio transport

The system SHALL provide `skillgrid mcp` as an MCP stdio server for agent tool calls.

#### Scenario: MCP server starts
- **GIVEN** `skillgrid mcp` is invoked
- **WHEN** the command runs
- **THEN** an MCP stdio server starts and serves tool calls

#### Scenario: All tools registered
- **GIVEN** the MCP server is running
- **WHEN** an agent requests the tool list
- **THEN** `mem_*`, `code_*`, and `web_*` tools are advertised

#### Scenario: MCP tool dispatch
- **GIVEN** an agent calls `mem_search` via MCP
- **WHEN** the tool is dispatched
- **THEN** the service layer executes the search and returns results

### Requirement: HTTP REST transport

The system SHALL provide `skillgrid serve` as an HTTP server for plugin hooks.

#### Scenario: HTTP server starts
- **GIVEN** `skillgrid serve` is invoked
- **WHEN** the command runs
- **THEN** an HTTP server starts on `127.0.0.1:7438`

#### Scenario: Health check
- **GIVEN** the HTTP server is running
- **WHEN** `GET /health` is called
- **THEN** `{"status":"ok","service":"skillgrid-memindex","version":"..."}` is returned

#### Scenario: Session create
- **GIVEN** the HTTP server is running
- **WHEN** `POST /sessions` is called with a directory
- **THEN** a session is created and the session ID returned

#### Scenario: Context for compaction
- **GIVEN** the HTTP server is running
- **WHEN** `GET /context` is called
- **THEN** recent session context is returned for compaction injection
