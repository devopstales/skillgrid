# Plugin Install Specification

## Purpose
TBD — describe editor plugin installation behavior.

## Requirements

### Requirement: Plugin installation

The system SHALL provide `skillgrid plugin install` for installing editor plugins.

#### Scenario: Plugin installed for editor
- **GIVEN** `skillgrid plugin install --target=opencode` runs
- **WHEN** the command completes
- **THEN** the OpenCode plugin is copied to `~/.config/opencode/plugins/`

#### Scenario: Plugin first-write-wins
- **GIVEN** a plugin already exists at the target path
- **WHEN** `skillgrid plugin install` runs
- **THEN** the existing plugin is preserved (first-write-wins)

#### Scenario: Plugin target validated
- **GIVEN** `skillgrid plugin install --target=unknown` runs
- **WHEN** the command validates the target
- **THEN** a clear error is returned listing supported editors (opencode, kilo, cursor, vscode)
