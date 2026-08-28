## ADDED Requirements

### Requirement: Generic OpenCode hooks

The system SHALL provide a generic mechanism to install audit hooks into OpenCode.

#### Scenario: Hooks installed
- **GIVEN** a hook configuration is provided
- **WHEN** the OpenCode hook installer runs
- **THEN** the hook is registered in OpenCode's configuration

#### Scenario: Hook execution
- **GIVEN** an audit hook is installed
- **WHEN** an agent makes a tool call
- **THEN** the hook fires and records the event

#### Scenario: Multiple hooks supported
- **GIVEN** multiple hook definitions
- **WHEN** the installer runs
- **THEN** each hook is registered and fires independently
