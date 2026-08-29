# vscode-plugin Specification

## Purpose
TBD - created by archiving change mnemonic. Update Purpose after archive.

## Requirements

### Requirement: VSCode plugin

The system SHALL provide a VSCode extension for skillgrid integration.

#### Scenario: VSCode extension installed
- **GIVEN** `skillgrid setup vscode` runs
- **WHEN** the setup completes
- **THEN** a VSCode extension is installed or configured to use skillgrid skills and MCP servers

#### Scenario: VSCode MCP server registered
- **GIVEN** the VSCode extension is active
- **WHEN** a workspace opens
- **THEN** the `skillgrid-memindex` MCP server is registered in VSCode's MCP configuration

#### Scenario: VSCode skills available
- **GIVEN** the VSCode extension is active
- **WHEN** the user invokes a skillgrid command
- **THEN** skillgrid skills are accessible via VSCode command palette
