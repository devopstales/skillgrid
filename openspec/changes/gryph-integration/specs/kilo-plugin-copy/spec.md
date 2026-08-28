## ADDED Requirements

### Requirement: Generic Kilo plugin copy

The system SHALL provide a generic mechanism to copy any plugin into Kilo's plugins directory.

#### Scenario: Plugin copied to Kilo
- **GIVEN** a plugin file exists at a source path
- **WHEN** the copy command targets Kilo
- **THEN** the plugin is copied to `~/.config/kilo/plugins/`

#### Scenario: Existing plugin preserved
- **GIVEN** a plugin already exists at the Kilo target path
- **WHEN** the copy runs
- **THEN** the existing file is preserved (first-write-wins)

#### Scenario: Multiple plugins copied
- **GIVEN** multiple plugins need Kilo installation
- **WHEN** the copy command runs
- **THEN** each plugin is copied independently with separate first-write-wins checks
