## ADDED Requirements

### Requirement: Kilo plugin registers skills directory

The Kilo plugin SHALL register the Skillgrid skills directory in Kilo's skill discovery path so that skills appear in the editor's skill list.

#### Scenario: Skills discoverable after plugin load
- **GIVEN** the Skillgrid Kilo plugin is installed via `kilo plugin skillgrid-kilo-plugin` and Kilo is restarted
- **WHEN** a new session starts
- **THEN** Skillgrid skills appear in the skill discovery list

#### Scenario: Skills path added to config
- **GIVEN** the plugin's `config` hook fires at startup
- **WHEN** the hook executes
- **THEN** the skills directory is added to Kilo's skill paths

### Requirement: Kilo plugin injects bootstrap meta-skill

The Kilo plugin SHALL inject the Skillgrid bootstrap meta-skill into the system prompt array at session start.

#### Scenario: Bootstrap appended to system prompt
- **GIVEN** a new Kilo session with the Skillgrid plugin active
- **WHEN** the session starts
- **THEN** the bootstrap meta-skill content is appended to the system prompt array via `experimental.chat.system.transform`

#### Scenario: Bootstrap contains workflow enforcement
- **GIVEN** the bootstrap meta-skill content
- **WHEN** the agent reads it
- **THEN** it includes the 1% rule, brainstorming-before-coding enforcement, and OpenSpec pipeline reference

### Requirement: Kilo plugin merges MCP server config

The Kilo plugin SHALL read `config.d/*.yaml` and merge Skillgrid-managed MCP servers into Kilo's runtime MCP config.

#### Scenario: MCP servers available after plugin load
- **GIVEN** `config.d/mcp.yaml` declares MCP servers with `skillgrid-` prefix
- **WHEN** the plugin initializes
- **THEN** those MCP servers are available in the session

#### Scenario: Existing MCP entries preserved
- **GIVEN** the user has non-Skillgrid MCP servers configured
- **WHEN** the plugin merges its config
- **THEN** existing entries are preserved

### Requirement: Kilo plugin installs slash commands

The Kilo plugin SHALL install Skillgrid slash commands into `.kilo/command/`.

#### Scenario: Slash commands available
- **GIVEN** the plugin is installed
- **WHEN** the user types `/skillgrid-status` in Kilo
- **THEN** the command executes and shows plugin status

### Requirement: Kilo plugin responds to session events

The Kilo plugin SHALL subscribe to `session.created` events to trigger setup actions.

#### Scenario: Setup runs on session creation
- **GIVEN** the plugin is loaded
- **WHEN** a new session is created
- **THEN** the plugin performs any deferred setup (skill sync check, MCP merge verification)
