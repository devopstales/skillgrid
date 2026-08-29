# Agent Selection Specification

## Purpose
TBD — describe agent selection behavior and available agents.

## Requirements

### Requirement: Agent selection

The CLI SHALL allow users to select which agents to install.

#### Scenario: Interactive selection
- **GIVEN** no `--agents` or `--yes` flag is provided
- **WHEN** the agent selection step runs
- **THEN** an interactive multi-select prompt is shown with available agents

#### Scenario: Default selection in yes mode
- **GIVEN** the user runs `skillgrid --yes`
- **WHEN** the agent selection step runs
- **THEN** opencode, kilo, and cursor are all selected (cursor needs no npm install)

#### Scenario: Interactive defaults pre-check
- **GIVEN** the user opens the interactive multi-select prompt
- **WHEN** the prompt renders
- **THEN** opencode and kilo are pre-selected and cursor is not, and the user may toggle any of them and press enter (or enter for the defaults, `a` for all, `q` to cancel)

#### Scenario: Preset agents via flag
- **GIVEN** the user runs `skillgrid --agents opencode,kilo`
- **WHEN** the agent selection step runs
- **THEN** the specified agents are selected without prompting

#### Scenario: No agents selected
- **GIVEN** the user cancels the interactive prompt
- **WHEN** no agents are selected
- **THEN** an error is shown and the program exits

#### Scenario: Agent install skipped when none selected
- **GIVEN** the user selects no agents
- **WHEN** the install pipeline runs
- **THEN** the agent install step is skipped

### Requirement: Available agents

The CLI SHALL support installing the following agents:
- **OpenCode** (`opencode`) — npm package `opencode-ai`
- **Kilo** (`kilo`) — npm package `@kilocode/cli`
- **Cursor** (`cursor`) — app-side only (no npm package)

#### Scenario: OpenCode installed via npm
- **GIVEN** opencode is selected
- **WHEN** the agent install step runs
- **THEN** `npm install -g opencode-ai` is executed

#### Scenario: Kilo installed via npm
- **GIVEN** kilo is selected
- **WHEN** the agent install step runs
- **THEN** `npm install -g @kilocode/cli` is executed

#### Scenario: Cursor skipped (no npm)
- **GIVEN** cursor is selected
- **WHEN** the agent install step runs
- **THEN** no npm install is attempted (cursor has no npm package)

### Requirement: Agent key validation

The `--agents` flag and the interactive prompt SHALL only accept known agent
keys and SHALL report unknown keys explicitly rather than silently skipping them.

#### Scenario: Unknown agent key is reported on the CLI
- **GIVEN** the user runs `skillgrid --agents opencode,zed`
- **WHEN** the flag is parsed
- **THEN** the command names the unknown key (`zed`) and lists the valid keys it accepts, instead of silently installing only `opencode`

#### Scenario: Case-insensitive matching
- **GIVEN** the user runs `skillgrid --agents OpenCode,Kilo`
- **WHEN** the flag is parsed
- **THEN** both are recognised as valid keys and selected

#### Scenario: Whitespace in `--agents` is tolerated
- **GIVEN** the user runs `skillgrid --agents " opencode, kilo "`
- **WHEN** the flag is parsed
- **THEN** the keys are trimmed and both selected
