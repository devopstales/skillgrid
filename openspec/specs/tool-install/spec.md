# Tool Install Specification

## Purpose
TBD — describe global tool installation and agents directory copy behavior.

## Requirements

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

#### Scenario: Cucumber tool installed
- **GIVEN** the install pipeline runs
- **WHEN** the tool install step runs
- **THEN** `npm install -g @cucumber/cucumber` is executed

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

### Requirement: Global tool failures are non-fatal

A single global-tool npm failure SHALL NOT abort the rest of the install pipeline.
The step logs the failing package, continues with the remaining packages, and
reports a summary at the end.

#### Scenario: One tool fails, others still install
- **GIVEN** `skillgrid install` runs and the network blocks `npm install -g @cucumber/cucumber`
- **WHEN** the tool install step reaches that package
- **THEN** the error is logged with the package name, the remaining tools in the list are still attempted, and the exit code reflects the partial failure

#### Scenario: Dry-run lists all tools
- **GIVEN** the user runs `skillgrid --dry-run`
- **WHEN** the pipeline reaches the global-tools step
- **THEN** all package names from `GlobalTools()` are printed (prefixed with `[dry-run]`) in deterministic order

### Requirement: Install order is deterministic

Tool install steps SHALL be executed in a stable, documented order so logs are
reproducible and a flaky package at position N never blocks a package at
position N+1.

#### Scenario: Order: agents → MCP packages → MCP servers → global tools
- **GIVEN** the install pipeline runs with at least one agent selected
- **WHEN** the install order is observed
- **THEN** agents are installed first, then MCP packages, then MCP server
  configuration, then global npm tools — in that exact sequence
