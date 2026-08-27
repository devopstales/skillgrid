## ADDED Requirements

### Requirement: OpenCode plugin registers skills directory

The OpenCode plugin SHALL register the Skillgrid skills directory in OpenCode's skill discovery path so that skills appear in the editor's skill list.

#### Scenario: Skills discoverable after plugin load
- **GIVEN** the Skillgrid OpenCode plugin is installed and OpenCode is restarted
- **WHEN** a new session starts
- **THEN** Skillgrid skills (brainstorming, writing-plans, etc.) appear in the skill discovery list

#### Scenario: Skills directory path configured
- **GIVEN** the plugin's `config` hook fires at startup
- **WHEN** the hook executes
- **THEN** the skills directory path is added to `config.skills.paths`

### Requirement: OpenCode plugin injects bootstrap meta-skill

The OpenCode plugin SHALL inject the Skillgrid bootstrap meta-skill at the start of each session to enforce the workflow.

#### Scenario: Bootstrap prepended to first user message
- **GIVEN** a new OpenCode session with the Skillgrid plugin active
- **WHEN** the user sends their first message
- **THEN** the bootstrap meta-skill content is prepended to the message parts

#### Scenario: Bootstrap contains workflow enforcement
- **GIVEN** the bootstrap meta-skill content
- **WHEN** the agent reads it
- **THEN** it includes the 1% rule, brainstorming-before-coding enforcement, and OpenSpec pipeline reference

### Requirement: OpenCode plugin merges MCP server config

The OpenCode plugin SHALL read `config.d/*.yaml` and merge Skillgrid-managed MCP servers into the editor's runtime MCP config.

#### Scenario: MCP servers available after plugin load
- **GIVEN** `config.d/mcp.yaml` declares MCP servers with `skillgrid-` prefix
- **WHEN** the plugin initializes
- **THEN** those MCP servers are available in the session without manual configuration

#### Scenario: Existing MCP entries preserved
- **GIVEN** the user has non-Skillgrid MCP servers configured
- **WHEN** the plugin merges its config
- **THEN** existing entries are preserved and only `skillgrid-` prefixed entries are added

### Requirement: OpenCode plugin installs slash commands

The OpenCode plugin SHALL install Skillgrid slash commands into the editor's command directory.

#### Scenario: Slash commands available
- **GIVEN** the plugin is installed
- **WHEN** the user types `/` in the editor
- **THEN** `/skillgrid-status`, `/skillgrid-sync`, and `/skillgrid-update` commands are available
