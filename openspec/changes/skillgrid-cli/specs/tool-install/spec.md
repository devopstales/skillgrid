## ADDED Requirements

### Requirement: Global tool installation

The CLI SHALL install shared global tools regardless of agent selection.

#### Scenario: Skills tool installed
- **GIVEN** the install pipeline runs
- **WHEN** the tool install step runs
- **THEN** `npm install -g skills` is executed

#### Scenario: OpenSpec tool installed
- **GIVEN** the install pipeline runs
- **WHEN** the tool install step runs
- **THEN** `npm install -g @fission-ai/openspec@latest` is executed

#### Scenario: Tools skipped with flag
- **GIVEN** the user runs `skillgrid --skip-tools`
- **WHEN** the tool install step runs
- **THEN** no tool install is attempted

### Requirement: Agents directory copy

The CLI SHALL copy the repo's `.agents/` directory to `~/.agents/`.

#### Scenario: Agents copied
- **GIVEN** the repo contains `.agents/`
- **WHEN** the copy step runs
- **THEN** the contents are copied to `~/.agents/`

#### Scenario: Copy skipped when source missing
- **GIVEN** the repo does not contain `.agents/`
- **WHEN** the copy step runs
- **THEN** a verbose message is logged and the step is skipped

#### Scenario: Copy skipped with flag
- **GIVEN** the user runs `skillgrid --skip-agents`
- **WHEN** the copy step runs
- **THEN** the copy is not attempted
