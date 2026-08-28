## ADDED Requirements

### Requirement: Go symbol extraction

The system SHALL extract symbols (functions, methods, types) from Go files during code
indexing and represent them as graph nodes with typed edges.

#### Scenario: Extract symbols from a Go file
- **GIVEN** a Go file containing a function, a method, and a type definition
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** one symbol node exists per definition with kind, name, path, and line range properties
- **THEN** a DEFINES edge connects the file node to each symbol node

#### Scenario: Call edges between symbols
- **GIVEN** two Go files where a function in one file calls a function in the other
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** a CALLS edge exists from the caller symbol to the callee symbol

#### Scenario: Import edges
- **GIVEN** two Go packages where one imports the other and uses a symbol from it
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** an IMPORTS edge connects the importing file node to the imported file node

#### Scenario: Extend and embed edges
- **GIVEN** a Go type that embeds or implements another type
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** an EXTENDS edge connects the deriving type to the target type

#### Scenario: Symbol pipeline disabled
- **GIVEN** symbol extraction is disabled in `config.d/indexing.yaml`
- **WHEN** code indexing runs
- **THEN** no symbol nodes or CALLS/IMPORTS/EXTENDS/DEFINES edges are created

### Requirement: Incremental symbol edge regeneration

The system SHALL regenerate symbol edges for a file when that file changes, preserving
edges for unchanged files.

#### Scenario: Modified file re-index
- **GIVEN** an indexed Go file with existing symbol nodes and edges
- **WHEN** the file changes and code indexing runs
- **THEN** the file's symbol nodes and edges are deleted and regenerated based on the new content

#### Scenario: Unchanged files untouched
- **GIVEN** an indexed Go file with symbol nodes and edges
- **WHEN** a different file changes and code indexing runs
- **THEN** the unchanged file's symbol nodes and edges remain unchanged

#### Scenario: Per-file isolation of extraction errors
- **GIVEN** a Go file that fails symbol extraction (e.g., parse error)
- **WHEN** code indexing runs for a directory containing that file
- **THEN** the indexing pass continues for other files and the failure is reported in `graph_status`
