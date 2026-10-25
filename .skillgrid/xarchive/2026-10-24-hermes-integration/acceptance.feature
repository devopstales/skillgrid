# Hermes Agent Integration

<!--
Source: .skillgrid/specs/2026-10-24-hermes-integration/
Trace: change briefing ## Must-Haves; blueprint tasks SATISFIES lines.
Format rules:
- Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
- No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
- One blank line between the heading and the fence is fine
- Gates: each requirement carries a `#### Gates` block — the runnable shadow of
  its happy-path and failure scenarios. Author it BEFORE implementing.
-->

## Requirements

### Requirement: MemoryProvider plugin activates in Hermes

The system SHALL discover the Mnemonic MemoryProvider plugin in Hermes, report its availability based on the mnemonic binary, and expose the memory tool schemas when active.

#### Scenario: happy path — MemoryProvider is discovered and activates

```gherkin
      Given the MemoryProvider plugin is installed at "~/.hermes/plugins/mnemonic/"
      And the "mnemonic" binary is on PATH
      When I run "hermes memory setup"
      Then "mnemonic" appears in the list of available providers
      And "mnemonic" shows as available
```

#### Scenario: edge — MemoryProvider is unavailable when mnemonic binary is missing

```gherkin
      Given the MemoryProvider plugin is installed at "~/.hermes/plugins/mnemonic/"
      And the "mnemonic" binary is NOT on PATH
      When I run "hermes memory setup"
      Then "mnemonic" appears in the list of available providers
      And "mnemonic" shows as unavailable with reason "mnemonic binary not found"
```

#### Scenario: failure — provider reports unavailable with actionable reason

```gherkin
      Given the MemoryProvider plugin is installed
      And the "mnemonic" binary is NOT on PATH
      When the Hermes runtime probes the provider availability
      Then the provider reports unavailable
      And the unavailable reason contains the install command "skillgrid install"
```

#### Scenario: happy path — MemoryProvider exposes memory tool schemas

```gherkin
      Given the MemoryProvider is active in a Hermes session
      When I inspect the agent tool schemas
      Then "mem_save" is present with required params title, type, and content
      And "mem_search" is present with required param query
      And "mem_context" is present
      And "mem_get" is present with required param id
      And "mem_update" is present with required param id
```

#### Gates

- happy path: provider discovered, available, tool schemas exposed
- failure: provider unavailable, actionable reason

### Requirement: Memory round-trip through the provider

The system SHALL persist a fact via the MemoryProvider's mem_save tool and recall it in a subsequent session via prefetch or mem_search.

#### Scenario: happy path — save and recall a fact

```gherkin
      Given the MemoryProvider is active in a Hermes session
      When I save a fact "the database uses Postgres 16" via mem_save
      And I start a new Hermes session
      And I search for "database" via mem_search
      Then the search results contain "Postgres 16"
```

#### Scenario: edge — prefetch returns empty on the first turn

```gherkin
      Given the MemoryProvider is active in a fresh Hermes session
      When I ask a question that triggers a prefetch
      Then the prefetch context is empty
      And the agent does not hallucinate prior context
```

#### Scenario: edge — prefetch returns cached result on the second turn

```gherkin
      Given the MemoryProvider is active in a Hermes session
      And a prior turn queued a prefetch for "API rate limit"
      When I ask a new question in the next turn
      Then the prefetch context contains the cached search results
```

#### Scenario: failure — sync_turn does not block the agent on mnemonic error

```gherkin
      Given the MemoryProvider is active in a Hermes session
      And the "mnemonic" binary returns an error on "mem save"
      When a turn completes
      Then the agent responds without delay
      And the sync_turn error is logged but not surfaced to the user
```

#### Gates

- happy path: fact saved in one session, recalled in another
- failure: sync_turn error does not block the agent

### Requirement: MCP server registration

The system SHALL register the mnemonic MCP server in Hermes config.yaml with the MNEMONIC_MCP_EXCLUDE env var to suppress mem_* tools (handled natively by the MemoryProvider).

#### Scenario: happy path — MCP server is registered in Hermes config

```gherkin
      Given the skillgrid installer has run with "--agents hermes"
      When I read "~/.hermes/config.yaml"
      Then the "mcp_servers" block contains an entry named "mnemonic"
      And the entry command is "mnemonic" with args "mcp"
      And the entry env includes "MNEMONIC_MCP_EXCLUDE" set to "mem"
```

#### Scenario: happy path — MCP server passes the Hermes test

```gherkin
      Given the MCP server is registered in Hermes config
      When I run "hermes mcp test skillgrid-mnemonic"
      Then the command exits 0
      And the discovered tool list includes "code_explore"
      And the discovered tool list includes "web_search"
      And the discovered tool list does NOT include "mem_save"
```

#### Scenario: edge — all MCP servers from config.d/mcp.yaml are registered

```gherkin
      Given the skillgrid installer has run with "--agents hermes"
      When I read "~/.hermes/config.yaml"
      Then the "mcp_servers" block contains entries for all servers in "config.d/mcp.yaml"
```

#### Scenario: failure — malformed config.yaml triggers backup and error

```gherkin
      Given "~/.hermes/config.yaml" contains a YAML syntax error
      When I run "skillgrid install --agents hermes"
      Then the installer reports the YAML error
      And a backup of the original config.yaml is created
      And the installer does not overwrite the malformed file
```

#### Gates

- happy path: MCP registered, passes hermes mcp test, mem_* excluded
- failure: malformed config triggers backup, no overwrite

### Requirement: Skill deployment

The system SHALL copy the skillgrid skill catalog to Hermes's skill discovery path during install.

#### Scenario: happy path — skillgrid skills are deployed to Hermes

```gherkin
      Given the skillgrid installer has run with "--agents hermes"
      When I list "~/.hermes/skills/"
      Then the directory contains "brainstorming"
      And the directory contains "test-driven-development"
```

#### Scenario: edge — skill deployment is idempotent

```gherkin
      Given the skillgrid installer has already run with "--agents hermes"
      When I run "skillgrid install --agents hermes" again
      Then the skills are not duplicated in "~/.hermes/skills/"
```

#### Scenario: failure — read-only skills directory does not fail the install

```gherkin
      Given "~/.hermes/skills/" is read-only
      When I run "skillgrid install --agents hermes"
      Then the installer reports the permission error for skills
      And the MCP and MemoryProvider setup still complete
```

#### Gates

- happy path: skills deployed to ~/.hermes/skills/

### Requirement: Memory protocol injection

The system SHALL inject the Mnemonic memory protocol into the Hermes system prompt via the MemoryProvider's system_prompt_block, with a SOUL.md marker block as fallback.

#### Scenario: happy path — protocol is injected via system_prompt_block

```gherkin
      Given the MemoryProvider is active in a Hermes session
      When I inspect the system prompt
      Then it contains the Mnemonic Memory Protocol section
      And the protocol references mem_save, mem_search, and mem_context
```

#### Scenario: edge — protocol fallback in SOUL.md when provider is not active

```gherkin
      Given the MemoryProvider is NOT active
      And the skillgrid installer has run with "--agents hermes"
      When I read "~/.hermes/SOUL.md"
      Then it contains the marker "<!-- BEGIN SKILLGRID MNEMONIC -->"
      And it contains the marker "<!-- END SKILLGRID MNEMONIC -->"
      And the protocol content is between the markers
```

#### Scenario: failure — SOUL.md block is replaced on re-install

```gherkin
      Given "~/.hermes/SOUL.md" contains an existing Mnemonic protocol block
      When I run "skillgrid install --agents hermes" again
      Then exactly one Mnemonic protocol block remains in SOUL.md
      And the block is between the BEGIN and END markers
```

#### Gates

- happy path: protocol in system prompt via system_prompt_block
- failure: SOUL.md block replaced, not duplicated

### Requirement: Kanban ticketing backend

The system SHALL support Hermes Kanban as a ticketing backend, publishing skillgrid tickets to a Kanban board with dependency links and assignee profiles.

#### Scenario: happy path — ticket is published to Hermes Kanban

```gherkin
      Given the ticketing config has type "hermes-kanban"
      And a Kanban board exists for the project
      When the ticketing skill publishes a ticket titled "Implement feature X"
      Then "hermes kanban list" shows a task titled "Implement feature X"
      And the task assignee matches the configured profile
```

#### Scenario: happy path — ticket with dependencies creates a Kanban link

```gherkin
      Given the ticketing config has type "hermes-kanban"
      And ticket "Task A" has already been published to Kanban
      When the ticketing skill publishes "Task B" with parent "Task A"
      Then "hermes kanban show" for Task B lists Task A as a parent
      And Task B is in "todo" status
```

#### Scenario: edge — Kanban board is created if it does not exist

```gherkin
      Given the ticketing config has type "hermes-kanban" with board "myproject"
      And no Kanban board named "myproject" exists
      When I run "skillgrid install --agents hermes"
      Then a Kanban board named "myproject" exists
      And "hermes kanban boards list" shows "myproject"
```

#### Scenario: failure — Kanban board creation failure is non-fatal

```gherkin
      Given the "hermes kanban boards create" command fails
      When I run "skillgrid install --agents hermes"
      Then the installer reports the Kanban board creation failure as a warning
      And the MCP and MemoryProvider setup completes
```

#### Gates

- happy path: ticket published, appears in kanban list
- failure: board creation failure is non-fatal

### Requirement: Install flow

The system SHALL configure all integration layers (MemoryProvider, MCP, skills, protocol, Kanban) in a single `skillgrid install --agents hermes` command, with dry-run support and non-fatal prerequisite checks.

#### Scenario: happy path — full install configures all layers

```gherkin
      Given a fresh machine with "hermes" and "mnemonic" on PATH
      When I run "skillgrid install --agents hermes"
      Then the MemoryProvider plugin is at "~/.hermes/plugins/mnemonic/"
      And the MCP server is registered in "~/.hermes/config.yaml"
      And the skills are at "~/.hermes/skills/"
      And the SOUL.md contains the protocol block
      And the ticketing config has type "hermes-kanban"
      And a Kanban board exists for the project
```

#### Scenario: edge — dry-run prints planned changes without writing

```gherkin
      Given a fresh machine with "hermes" and "mnemonic" on PATH
      When I run "skillgrid install --dry-run --agents hermes"
      Then the output lists all planned changes
      And no files are written to "~/.hermes/"
      And no Kanban board is created
```

#### Scenario: failure — install continues when hermes binary is missing

```gherkin
      Given the "hermes" binary is NOT on PATH
      And the "mnemonic" binary IS on PATH
      When I run "skillgrid install --agents hermes"
      Then the installer prints the install command for hermes
      And the installation continues
      And the mnemonic setup completes
```

#### Gates

- happy path: all layers configured in one command
- edge: dry-run writes nothing
- failure: missing hermes is non-fatal

### Requirement: No regression to existing agents

The system SHALL not break existing agent installations (opencode, kilocode, cursor, pi) or the existing Backlog.md ticketing path.

#### Scenario: happy path — existing agents still install correctly

```gherkin
      Given a machine with "opencode" on PATH
      When I run "skillgrid install --agents opencode"
      Then the OpenCode MCP config is written correctly
      And "cd mnemonic && go test ./..." passes
      And "cd skillgrid-cli && go test ./..." passes
```

#### Scenario: edge — existing Backlog.md ticketing still works

```gherkin
      Given a project with ticketing type "backlogmd"
      When the ticketing skill publishes a ticket
      Then the ticket is written to the ".backlog/" directory
      And no Kanban board is touched
```

#### Scenario: failure — hermes install does not affect existing agent configs

```gherkin
      Given a machine with an existing OpenCode installation
      When I run "skillgrid install --agents hermes"
      Then the OpenCode config is unchanged
      And the Hermes config is written
```

#### Gates

- happy path: existing agents install correctly, tests pass
- failure: hermes install does not touch OpenCode config
