# code-layer Specification

## Purpose
TBD - created by archiving change mnemonic. Update Purpose after archive.

## Requirements

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

### Requirement: Indexing is config-driven

The code indexer SHALL honour include/exclude globs, chunk window, and file-size
cap from `config.d/indexing.yaml` (repo-local or `~/.skillgrid/config.d/`
override, first match wins per key).

#### Scenario: Defaults when no config present
- **GIVEN** no `config.d/indexing.yaml` is found
- **WHEN** indexing runs
- **THEN** defaults apply: `chunk_lines` 80, `chunk_overlap` 10, `max_file_size_kb` 512, and the default include set (`.go`, `.ts`, `.tsx`, `.md`)

#### Scenario: File size cap enforced
- **GIVEN** `max_file_size_kb` is set to 512
- **WHEN** the indexer reaches a 600KB file
- **THEN** the file is skipped (counted in `files_skipped`) and not chunked

#### Scenario: Chunk overlap preserved
- **GIVEN** `chunk_lines` is 80 and `chunk_overlap` is 10
- **WHEN** a file is split into chunks
- **THEN** adjacent chunks share the last 10 lines so cross-chunk symbols remain searchable
