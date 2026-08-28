## ADDED Requirements

### Requirement: MCP server installation

The system SHALL provide `skillgrid mcp install` for installing MCP server configurations.

#### Scenario: MCP server entry added
- **GIVEN** `skillgrid mcp install <server>` runs
- **WHEN** the command completes
- **THEN** the MCP server entry is added to `config.d/mcp.yaml`

#### Scenario: MCP server entry idempotent
- **GIVEN** an MCP server entry already exists in `config.d/mcp.yaml`
- **WHEN** `skillgrid mcp install <server>` runs again
- **THEN** the existing entry is preserved (first-write-wins)

#### Scenario: MCP server validated
- **GIVEN** `skillgrid mcp install <server>` runs
- **WHEN** the server name is unknown
- **THEN** a clear error is returned listing available servers
