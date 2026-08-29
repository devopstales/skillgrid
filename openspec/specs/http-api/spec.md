# http-api Specification

## Purpose
TBD - created by archiving change mnemonic. Update Purpose after archive.

## Requirements

### Requirement: HTTP API surfaces

The system SHALL expose REST endpoints for memory, code, and web cache operations.

#### Scenario: Memory endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `POST /observations`, `GET /search`, `GET /observations/recent` are called
- **THEN** they behave equivalently to their MCP counterparts

#### Scenario: Passive and prompt endpoints
- **GIVEN** the HTTP server is running
- **WHEN** `POST /observations/passive` and `POST /prompts` are called with `session_id` and a project
- **THEN** the server extracts learnings from the passive text and stores the prompt, returning the stored id(s)

#### Scenario: Project migration
- **GIVEN** the HTTP server is running and a project id changed (repo rename, remote base-name change)
- **WHEN** `POST /projects/migrate` is called with `old_project` and `new_project`
- **THEN** rows tagged `old_project` are rolled into `new_project` and the rename is recorded for idempotency

#### Scenario: Nudge state
- **GIVEN** the HTTP server is running
- **WHEN** `GET /memory/last-save-at` is called with a project
- **THEN** the newest observation timestamp is returned (or empty) so the plugin can compute save-staleness

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
- **THEN** `GET /context/compaction` is called (session-scoped) and the result injected before the agent continues, plus a "FIRST ACTION REQUIRED" instruction

#### Scenario: Session create is idempotent
- **GIVEN** the plugin is loaded and the same session id is seen again (reload, reconnect)
- **WHEN** `POST /sessions` is called with the same client id
- **THEN** the existing session is returned without duplication
