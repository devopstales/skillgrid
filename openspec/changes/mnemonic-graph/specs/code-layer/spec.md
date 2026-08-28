## MODIFIED Requirements

### Requirement: Incremental code indexing

The system SHALL provide `skillgrid index` for incremental file/chunk indexing with FTS5
search, plus opt-in extraction passes — symbol, PDG/taint, and API-contract — that derive
graph nodes and edges for the enabled languages and frameworks. Each pass is
independently config-gated and pass-scoped so regeneration touches only its own edges.

#### Scenario: Cold index
- **GIVEN** a project with unindexed source files
- **WHEN** `skillgrid index` runs
- **THEN** files are chunked and indexed in FTS

#### Scenario: Symbol extraction pass
- **GIVEN** symbol extraction is enabled in `config.d/indexing.yaml`
- **WHEN** `skillgrid index` runs over enabled-language files
- **THEN** symbol nodes and DEFINES/CALLS/IMPORTS/EXTENDS edges are created in the graph tables with `pass='symbols'`

#### Scenario: PDG extraction pass
- **GIVEN** PDG extraction is enabled in `config.d/indexing.yaml`
- **WHEN** `skillgrid index` runs over enabled-language files
- **THEN** basic-block nodes and CFG/CDG/REACHING_DEF/SOURCE/SINK/SANITIZES/TAINTED/TAINT_PATH edges are created with `pass='pdg'`

#### Scenario: API-contract extraction pass
- **GIVEN** contract extraction is enabled in `config.d/indexing.yaml`
- **WHEN** `skillgrid index` runs over enabled-framework files
- **THEN** route/tool/shape nodes and their contract edges are created with `pass='contracts'`

#### Scenario: Extraction passes disabled
- **GIVEN** an extraction pass is disabled or unset in config
- **WHEN** `skillgrid index` runs
- **THEN** indexing completes with no graph writes from that pass

#### Scenario: Warm no-op
- **GIVEN** a project where no files changed since last index
- **WHEN** `skillgrid index` runs
- **THEN** no re-indexing occurs (mtime + hash check)

#### Scenario: Incremental update
- **GIVEN** a project where some files changed since last index
- **WHEN** `skillgrid index` runs
- **THEN** only changed files are re-chunked and re-indexed and each enabled pass regenerates that file's own pass-scoped edges

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
- **THEN** file count, chunk count, last indexed time, and staleness hint are returned as before
