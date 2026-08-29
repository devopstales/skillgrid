# agents-rules Specification

## Purpose

The block of Mnemonic memory-protocol instructions that is injected into
AGENTS.md (between managed markers) so any agent that reads the repo
instructions knows to call `mem_save` proactively, how to search memory, and
how to recover context after a compaction — without any per-agent setup hooks.

## Requirements

### Requirement: Mnemonic protocol block in AGENTS.md
The system SHALL inject the Mnemonic Memory Protocol into AGENTS.md between managed markers, matching the engram protocol pattern.

#### Scenario: Protocol injected on setup
- **GIVEN** a repository with AGENTS.md
- **WHEN** `skillgrid setup` runs with mnemonic enabled
- **THEN** the mnemonic protocol block is injected between `<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->` and `<!-- END SKILLGRID MNEMONIC -->` markers

#### Scenario: Existing block preserved
- **GIVEN** AGENTS.md already contains the mnemonic protocol block
- **WHEN** `skillgrid setup` runs
- **THEN** the existing block is preserved (first-write-wins)

#### Scenario: Protocol references all tool families
- **GIVEN** the mnemonic protocol block is present
- **WHEN** an agent reads AGENTS.md
- **THEN** the block references memory (`mem_*`), code (`code_*`), and web cache (`web_*`) tools

### Requirement: Protocol covers memory lifecycle
The injected protocol SHALL cover save, search, session lifecycle, and compaction recovery.

#### Scenario: Save rules present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains WHEN TO SAVE rules with type + scope taxonomy

#### Scenario: Search rules present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains WHEN TO SEARCH MEMORY rules

#### Scenario: Session close protocol present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains SESSION CLOSE PROTOCOL with Goal/Instructions/Discoveries/Accomplished/Next Steps/Relevant Files

#### Scenario: Compaction recovery present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains AFTER COMPACTION recovery steps

### Requirement: Protocol includes code and web cache rules

The injected protocol SHALL include the code-search ladder and the web-research
cache workflow.

#### Scenario: Code search ladder present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains the `code_status` → `code_search` → `code_read` ladder

#### Scenario: Web cache workflow present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains the `web_cache_lookup` → remote fetch → `web_cache_save` workflow

### Requirement: Protocol documents capture, privacy, and passivity

The injected protocol SHALL document the passive-capture and privacy contract
so agents do not need to hand-write every observation and know how to keep
secrets out of the store.

#### Scenario: Passive capture is documented
- **GIVEN** the protocol block is present
- **WHEN** an agent reads the "Passive Capture" section
- **THEN** it states that finishing a Task/sub-agent does NOT require a manual
  `mem_save` and that the agent SHOULD include a `## Key Learnings:` section so
  the server can extract it

#### Scenario: Privacy tagging is documented
- **GIVEN** the protocol block is present
- **WHEN** an agent reads the "Privacy" section
- **THEN** it states that sensitive spans should be wrapped in `<private>…</private>` and are stored as `[REDACTED]`

### Requirement: Protocol markers are managed by setup

The injected block SHALL sit between explicit managed markers so `skillgrid
setup` can re-sync it and so `project-context` refreshes can preserve it byte-identical.

#### Scenario: Markers delimit the block
- **GIVEN** the protocol block is present
- **WHEN** an open-code-style harness scans AGENTS.md
- **THEN** the block is delimited by `<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->` and `<!-- END SKILLGRID MNEMONIC -->` and the harness can replace only the text between the markers

#### Scenario: Content of the markers never leaks
- **GIVEN** a file contains the protocol markers
- **WHEN** the file is read by any harness
- **THEN** the marker comment text itself is not interpreted as memory instructions and is not captured into `mem_save` content
