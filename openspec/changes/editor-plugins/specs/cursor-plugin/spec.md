## ADDED Requirements

### Requirement: Cursor plugin registers skills directory

The Cursor plugin SHALL declare its skills directory in the plugin manifest so that skills are discoverable by the editor.

#### Scenario: Skills discoverable after plugin install
- **GIVEN** the Skillgrid Cursor plugin is installed via `/add-plugin skillgrid` and Cursor is restarted
- **WHEN** a new session starts
- **THEN** Skillgrid skills appear in the skill discovery list

#### Scenario: Skills path declared in manifest
- **GIVEN** the plugin's `plugin.json` manifest
- **WHEN** Cursor loads the plugin
- **THEN** the `skills` field points to the skills directory

### Requirement: Cursor plugin injects bootstrap meta-skill

The Cursor plugin SHALL inject the Skillgrid bootstrap meta-skill at session start via lifecycle hooks.

#### Scenario: Bootstrap injected on session start
- **GIVEN** a new Cursor session with the Skillgrid plugin active
- **WHEN** the session starts
- **THEN** the bootstrap meta-skill content is injected into the agent context

#### Scenario: Bootstrap contains workflow enforcement
- **GIVEN** the bootstrap meta-skill content
- **WHEN** the agent reads it
- **THEN** it includes the 1% rule, brainstorming-before-coding enforcement, and OpenSpec pipeline reference

### Requirement: Cursor plugin merges MCP server config

The Cursor plugin SHALL read `config.d/*.yaml` and merge Skillgrid-managed MCP servers into Cursor's MCP config.

#### Scenario: MCP servers available after plugin load
- **GIVEN** `config.d/mcp.yaml` declares MCP servers with `skillgrid-` prefix
- **WHEN** the plugin initializes
- **THEN** those MCP servers are available in the session

#### Scenario: Existing MCP entries preserved
- **GIVEN** the user has non-Skillgrid MCP servers configured
- **WHEN** the plugin merges its config
- **THEN** existing entries are preserved

### Requirement: Cursor plugin installs slash commands

The Cursor plugin SHALL install Skillgrid slash commands into Cursor's command system.

#### Scenario: Slash commands available
- **GIVEN** the plugin is installed
- **WHEN** the user invokes `/skillgrid-status`
- **THEN** the command executes and shows plugin status
