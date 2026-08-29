## ADDED Requirements

### Requirement: MCP configuration during install

The system SHALL configure MCP server entries for selected agents during the normal install pipeline.

#### Scenario: MCP configured for selected agents
- **GIVEN** `skillgrid install` runs with agents selected
- **WHEN** the install pipeline reaches the MCP configuration step
- **THEN** MCP server entries are registered in each selected agent's configuration file

#### Scenario: MCP configuration is idempotent
- **GIVEN** MCP server entries already exist in agent configuration files
- **WHEN** `skillgrid install` runs again
- **THEN** existing entries are preserved (first-write-wins)

#### Scenario: Unknown agent during MCP configuration
- **GIVEN** `skillgrid install` runs with an unknown agent key
- **WHEN** the MCP configuration step runs
- **THEN** a clear error is returned listing available agents
