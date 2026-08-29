## ADDED Requirements

### Requirement: OpenCode plugin

The system SHALL provide an OpenCode plugin (`memindex.ts`) for session lifecycle and Memory Protocol injection.

#### Scenario: Memory Protocol injected
- **GIVEN** the OpenCode plugin is loaded
- **WHEN** a session starts
- **THEN** the Memory Protocol is injected via `chat.system.transform`

#### Scenario: Session auto-start
- **GIVEN** the OpenCode plugin is loaded
- **WHEN** a new session begins
- **THEN** `POST /sessions` is called automatically

#### Scenario: Index nudge
- **GIVEN** the OpenCode plugin is loaded and the index is stale
- **WHEN** a session starts
- **THEN** the agent receives a warning (not an auto-run full index)

#### Scenario: Save nudge is debounced
- **GIVEN** 15+ minutes have elapsed since the last `mem_save` and the session is past its 5-minute grace window
- **WHEN** the next system prompt is built
- **THEN** a "MEMORY REMINDER" nudge is appended and the time-stamp cached to prevent immediate repeats

#### Scenario: Privacy tags are stripped
- **GIVEN** the prompt or Task output contains `<private>…</private>` spans
- **WHEN** the content is forwarded to `POST /prompts` or `POST /observations/passive`
- **THEN** the spans are replaced with `[REDACTED]` before the bytes hit the wire

#### Scenario: Prompt is captured
- **GIVEN** the plugin is loaded and a non-trivial user prompt (>10 chars) arrives
- **WHEN** the `chat.message` hook fires for a top-level session
- **THEN** the prompt is POSTed to `/prompts` (truncated to 2000 chars)

#### Scenario: Task output passively captured
- **GIVEN** the plugin is loaded and a `Task`/sub-agent completes with >50 chars of output
- **WHEN** the `tool.execute.after` hook fires
- **THEN** the output is POSTed to `/observations/passive` with `source: "task-complete"` attributed to the authoritative root

### Requirement: Session ownership contract

The plugin SHALL distinguish top-level (root) sessions from sub-agent/child sessions and bind all attributable writes to the root only.

#### Scenario: Child sub-agent sessions never register themselves
- **GIVEN** a session with a non-empty `parentID`
- **WHEN** `session.created` fires
- **THEN** the plugin records the ownership mapping but does not `POST /sessions` for that id

#### Scenario: mem_* writes bind to the root
- **GIVEN** an unobserved child session (parent resolved via the OpenCode SDK)
- **WHEN** `mem_save`, `mem_save_prompt`, `mem_session_summary`, or `mem_capture_passive` fire
- **THEN** the plugin walks the parent chain to the root, registers the root if needed, and sets `output.args.session_id` to the root id

#### Scenario: Write hook is fail-closed on unresolvable ownership
- **GIVEN** the parent chain cannot be resolved (missing ancestor, cycle, cross-project mismatch, tombstoned session, or registration failure)
- **WHEN** a mem_* write hook fires
- **THEN** the hook throws (so the call is retryable) rather than silently forwarding the unresolved child id

#### Scenario: Capture hooks are fail-open
- **GIVEN** a `chat.message` or `Task` passive-capture hook fires for an unresolvable session
- **WHEN** ownership cannot be determined
- **THEN** the hook skips the write silently (capture is best-effort)

#### Scenario: Tombstoned trees do not revive
- **GIVEN** a session tree has been deleted (marked invalid)
- **WHEN** a later event re-creates one of its ids
- **THEN** the plugin keeps the tombstone and no write or registration occurs for that id

### Requirement: Unit test suite

The plugin SHALL ship a `node --test` suite (`mnemonic.test.mjs`) covering the ownership contract end-to-end.

#### Scenario: Test suite is green
- **GIVEN** both `plugins/opencode/mnemonic.test.mjs` and `plugins/kilo/mnemonic.test.mjs` exist
- **WHEN** `node --test plugins/<agent>/mnemonic.test.mjs` runs
- **THEN** all scenarios pass (registration, fail-closed writes, sub-agent suppression, SDK failure retry, prompt capture, private-tag strip, Task attribution, compaction rules)

### Requirement: Kilo plugin

The system SHALL support Kilo via copying the OpenCode plugin (same HTTP + MCP split).

#### Scenario: Kilo plugin copied when missing
- **GIVEN** `~/.config/opencode/plugins/memindex.ts` exists and `~/.config/kilo/plugins/memindex.ts` does not
- **WHEN** `skillgrid setup kilocode` runs
- **THEN** the file is copied to Kilo's plugins directory

#### Scenario: Existing Kilo plugin not overwritten
- **GIVEN** `~/.config/kilo/plugins/memindex.ts` already exists
- **WHEN** `skillgrid setup kilocode` runs
- **THEN** the existing file is preserved (first-write-wins)

### Requirement: Cursor rule

The system SHALL provide a Cursor `.mdc` rule for MCP + protocol guidance.

#### Scenario: Cursor rule installed
- **GIVEN** `skillgrid setup cursor` runs
- **WHEN** the setup completes
- **THEN** `~/.cursor/rules/memindex.mdc` contains the Memory Protocol and `code_*` usage rules

#### Scenario: Cursor MCP entry installed
- **GIVEN** `skillgrid setup cursor` runs
- **WHEN** the setup completes
- **THEN** `~/.cursor/mcp.json` contains the `skillgrid-memindex` MCP server entry
