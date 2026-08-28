## ADDED Requirements

### Requirement: Mnemonic protocol block in AGENTS.md
The system SHALL inject the Mnemonic Memory Protocol into AGENTS.md between managed markers, matching the engram protocol pattern.

#### Scenario: Protocol injected on setup
- **GIVEN** a repository with AGENTS.md
- **WHEN** `skillgrid setup` runs with mnemonic enabled
- **THEN** the mnemonic protocol block is injected between `<!-- BEGIN MNEMONIC MEMORY PROTOCOL — managed by mnemonic setup -->` and `<!-- END MNEMONIC MEMORY PROTOCOL -->` markers

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
The injected protocol SHALL include code search ladder and web research cache rules.

#### Scenario: Code search ladder present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains the `code_status` → `code_search` → `code_read` ladder

#### Scenario: Web cache workflow present
- **GIVEN** the protocol block is present
- **WHEN** an agent reads it
- **THEN** it contains `web_cache_lookup` → remote fetch → `web_cache_save` workflow
