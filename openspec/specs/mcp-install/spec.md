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

### Requirement: MCP config backup and preservation

MCP configuration SHALL back up each agent config file before mutating it, so
that a failed or partial `install-mcp` run does not lose the user's existing
MCP entries.

#### Scenario: Config backed up before mutation
- **GIVEN** `~/.config/opencode/opencode.json` contains existing MCP entries
- **WHEN** `skillgrid install` reaches the MCP server configuration step
- **THEN** a backup of the pre-mutation config is created before `install-mcp` runs, and the backup exists even if `install-mcp` fails midway

#### Scenario: Existing entries are not lost on re-run
- **GIVEN** `~/.config/opencode/opencode.json` has a user-added MCP entry `foo` with `enabled: false`
- **WHEN** `skillgrid install` re-runs
- **THEN** the `foo` entry survives the re-run (a first-write-wins merge, not a wholesale replace)

#### Scenario: Adding a new server does not remove others
- **GIVEN** `config.d/mcp.yaml` gains a new server entry in a new release
- **WHEN** `skillgrid install` re-runs
- **THEN** the new entry is added and no pre-existing MCP entry is removed
