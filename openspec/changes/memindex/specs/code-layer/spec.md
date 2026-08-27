## ADDED Requirements

### Requirement: Incremental code indexing

The system SHALL provide `skillgrid index` for incremental file/chunk indexing with FTS5 search.

#### Scenario: Cold index
- **GIVEN** a project with unindexed `.go` files
- **WHEN** `skillgrid index` runs
- **THEN** files are chunked and indexed in FTS

#### Scenario: Warm no-op
- **GIVEN** a project where no files changed since last index
- **WHEN** `skillgrid index` runs
- **THEN** no re-indexing occurs (mtime + hash check)

#### Scenario: Incremental update
- **GIVEN** a project where some files changed
- **WHEN** `skillgrid index` runs
- **THEN** only changed files are re-changed and re-indexed

#### Scenario: Exclude paths
- **GIVEN** `indexing.yaml` excludes `node_modules/**` and `**/.git/**`
- **WHEN** indexing runs
- **THEN** excluded paths are skipped

### Requirement: Code search

The system SHALL provide `code_search` for FTS5 search over indexed chunks.

#### Scenario: Symbol search
- **GIVEN** indexed source files
- **WHEN** an agent calls `code_search` with a symbol name
- **THEN** matching file paths, line ranges, and snippets are returned

#### Scenario: Code read
- **GIVEN** a path and optional line range
- **WHEN** `code_read` is called
- **THEN** the full chunk or file slice is returned

#### Scenario: Index status
- **GIVEN** `code_status` is called
- **WHEN** the tool executes
- **THEN** file count, chunk count, last indexed time, and staleness hint are returned
