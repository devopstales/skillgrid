# vscode-plugin Specification

## Purpose

The VSCode extension that gives VSCode the same skillgrid integration as other
hosts: registers the `skillgrid-mcp` MCP server, exposes skillgrid skills as
palette commands, and (for graph-capable changes) renders the code-graph
visualisation. This release ships the contract; the extension is a reserved
target (see plugin-install "Supported plugin set") until a `vscode` change
lands.

## Requirements

### Requirement: VSCode plugin contract

When the VSCode capability lands, the extension SHALL provide skillgrid
integration: register the Mnemonic MCP server, expose skills as palette
commands, and render the code-graph visualisation.

#### Scenario: Extension installs the Mnemonic MCP server
- **GIVEN** the installed VSCode extension is active and a workspace opens
- **WHEN** the extension reads `config.d/mcp.yaml` (or the equivalent user config)
- **THEN** the `skillgrid-mcp` server entry is added to VSCode's MCP configuration so the agent can call `mem_*` / `code_*` / `web_*` tools

#### Scenario: Skills are available via the command palette
- **GIVEN** the installed VSCode extension is active
- **WHEN** the user opens the command palette and types a skillgrid skill name
- **THEN** the matching skill runs against the workspace (same effect as running the same skill in a terminal host)

#### Scenario: Same Memory Protocol as other hosts
- **GIVEN** the installed VSCode extension is active
- **WHEN** the extension injects the Memory Protocol into the agent system prompt
- **THEN** the text is byte-identical to the OpenCode/Kilo protocol (no per-host drift)

### Requirement: Reserved target in this release

`vscode` is a reserved future target. Until a dedicated `vscode` change lands,
the CLI SHALL NOT offer `skillgrid setup vscode` (or `--agents vscode`) in the
supported set, and the install pipeline SHALL NOT attempt to write into
`~/.vscode/` or install a VSCode extension.

#### Scenario: setup reports vscode as unsupported (for now)
- **GIVEN** the user runs `skillgrid setup vscode`
- **WHEN** the target is validated
- **THEN** the command reports that the `vscode` target is reserved for a
  future release and lists the currently-supported targets (`opencode`,
  `kilocode` / `kilo`, `cursor`)

#### Scenario: VSCode is not in the supported agent set
- **GIVEN** the user runs `skillgrid install --agents opencode,kilo,cursor,vscode`
- **WHEN** the agent selection step runs
- **THEN** the command reports the `vscode` key as reserved (not an unknown
  typo) so the user understands it is a known-land, not a mistake

#### Scenario: No VSCode paths are touched
- **GIVEN** the user runs `skillgrid install` (with or without `--yes`)
- **WHEN** the install pipeline completes
- **THEN** no files are written to `~/.vscode/` and no VSCode extension is installed
