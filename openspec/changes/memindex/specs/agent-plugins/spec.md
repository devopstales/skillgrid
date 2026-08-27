## ADDED Requirements

### Requirement: OpenCode plugin

The system SHALL provide an OpenCode plugin (`memindex.ts`) for session lifecycle and Memory Protocol injection.

#### Scenario: Memory Protocol injected
- **GIVEN** the OpenCode plugin is loaded
- **WHEN** a session starts
- **THEN** the Memory Protocol is injected via `chat.system.transform`

#### Scenario: Session auto-start
- **GIVEN** the OpenCode plugin is loaded
- **WHEN** a new session begins
- **THEN** `POST /sessions` is called automatically

#### Scenario: Index nudge
- **GIVEN** the OpenCode plugin is loaded and the index is stale
- **WHEN** a session starts
- **THEN** the agent receives a warning (not an auto-run full index)

### Requirement: Kilo plugin

The system SHALL support Kilo via copying the OpenCode plugin (same HTTP + MCP split).

#### Scenario: Kilo plugin copied when missing
- **GIVEN** `~/.config/opencode/plugins/memindex.ts` exists and `~/.config/kilo/plugins/memindex.ts` does not
- **WHEN** `skillgrid setup kilocode` runs
- **THEN** the file is copied to Kilo's plugins directory

#### Scenario: Existing Kilo plugin not overwritten
- **GIVEN** `~/.config/kilo/plugins/memindex.ts` already exists
- **WHEN** `skillgrid setup kilocode` runs
- **THEN** the existing file is preserved (first-write-wins)

### Requirement: Cursor rule

The system SHALL provide a Cursor `.mdc` rule for MCP + protocol guidance.

#### Scenario: Cursor rule installed
- **GIVEN** `skillgrid setup cursor` runs
- **WHEN** the setup completes
- **THEN** `~/.cursor/rules/memindex.mdc` contains the Memory Protocol and `code_*` usage rules

#### Scenario: Cursor MCP entry installed
- **GIVEN** `skillgrid setup cursor` runs
- **WHEN** the setup completes
- **THEN** `~/.cursor/mcp.json` contains the `skillgrid-memindex` MCP server entry
