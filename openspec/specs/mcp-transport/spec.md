# mcp-transport Specification

## Purpose
TBD - created by archiving change mnemonic. Update Purpose after archive.

## Requirements

### Requirement: MCP stdio transport

The system SHALL provide `skillgrid mcp` as an MCP stdio server for agent tool calls.

#### Scenario: MCP server starts
- **GIVEN** `skillgrid mcp` is invoked
- **WHEN** the command runs
- **THEN** an MCP stdio server starts and serves tool calls

#### Scenario: All tools registered
- **GIVEN** the MCP server is running
- **WHEN** an agent requests the tool list
- **THEN** `mem_*`, `code_*`, and `web_*` tools are advertised, including `mem_capture_passive`

#### Scenario: Passive capture tool
- **GIVEN** the MCP server is running
- **WHEN** an agent calls `mem_capture_passive` with a block of text and a `session_id`
- **THEN** the server extracts learnings and returns the count of saved/skipped observations with stored ids

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
- **WHEN** `GET /context/compaction?project=&session_id=` is called
- **THEN** the session's title/summary and the newest N observations are returned for compaction injection

### Requirement: Result encoding

MCP tool results SHALL be encoded as standard MCP `CallToolResult` responses
(`isError: false` on success, `isError: true` with a parseable error message
on failure) and every success path SHALL return JSON (not plain text), so
host agents can branch on `result.isError` uniformly.

#### Scenario: Success path returns JSON
- **GIVEN** a tool call succeeds (e.g. `mem_search` returns matches)
- **WHEN** the dispatch encodes the result
- **THEN** `isError` is false and the content payload contains a JSON document (not a free-form string) that the host agent can parse

#### Scenario: Missing required argument reports a clear error
- **GIVEN** the agent calls `mem_capture_passive` without `content`
- **WHEN** the dispatch validates arguments
- **THEN** `isError` is true and the error text names the missing argument and the tool name so the agent can self-correct

#### Scenario: Unknown tool name reports "not found"
- **GIVEN** the MCP client calls a tool name that is not registered
- **WHEN** the dispatch runs
- **THEN** the MCP server responds with a `CallToolResult` indicating the unknown tool (host-visible error), not a silent empty result
