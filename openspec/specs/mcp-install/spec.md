# MCP Install Specification

## Purpose
MCP packages and servers are configured during the normal install pipeline based on `config.d/tools.yaml` and `config.d/mcp.yaml`.

## Requirements

### Requirement: MCP packages installed from tools.yaml

The system SHALL install npm packages listed in `config.d/tools.yaml` (`mcp:` section) before configuring MCP servers.

#### Scenario: MCP packages installed
- **GIVEN** `skillgrid install` runs with agents selected
- **WHEN** the install pipeline reaches the MCP package step
- **THEN** each package in `config.d/tools.yaml` `mcp:` is installed via `npm install -g`

#### Scenario: MCP packages skipped with flag
- **GIVEN** the user runs `skillgrid --skip-tools`
- **WHEN** the MCP package step runs
- **THEN** no MCP packages are installed

### Requirement: MCP servers configured from mcp.yaml

The system SHALL merge MCP server entries from `config.d/mcp.yaml` into each selected agent's configuration file.

#### Scenario: Remote server configured
- **GIVEN** `config.d/mcp.yaml` contains a remote server entry
- **WHEN** the install pipeline configures MCP for an agent
- **THEN** the server is added with `type: remote` and the configured URL

#### Scenario: Local server configured
- **GIVEN** `config.d/mcp.yaml` contains a local server entry
- **WHEN** the install pipeline configures MCP for an agent
- **THEN** the server is added with `type: local` and the configured command array

#### Scenario: MCP configuration is idempotent
- **GIVEN** MCP server entries already exist in agent configuration files
- **WHEN** `skillgrid install` runs again
- **THEN** existing entries are preserved (first-write-wins)

#### Scenario: Unknown agent during MCP configuration
- **GIVEN** `skillgrid install` runs with an unknown agent key
- **WHEN** the MCP configuration step runs
- **THEN** a clear error is returned listing available agents
