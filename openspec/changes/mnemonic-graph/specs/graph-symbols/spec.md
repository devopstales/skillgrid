## ADDED Requirements

### Requirement: Multi-language symbol extraction

The system SHALL extract symbols (functions, methods, types) from Go, TypeScript, and
Python files during code indexing and represent them as graph nodes with typed edges.
Extraction is per-language, with per-file failure isolation.

#### Scenario: Extract symbols from a Go file
- **GIVEN** a Go file containing a function, a method, and a type definition
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** one symbol node exists per definition with kind, name, path, line range, and language properties
- **THEN** a DEFINES edge connects the file node to each symbol node

#### Scenario: Extract symbols from a TypeScript file
- **GIVEN** a TypeScript file containing a function, an exported method, and an interface
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** one symbol node exists per definition with a `language` property set to typescript
- **THEN** a DEFINES edge connects the file node to each symbol node

#### Scenario: Extract symbols from a Python file
- **GIVEN** a Python file containing a function and a class method
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** one symbol node exists per definition with a `language` property set to python
- **THEN** a DEFINES edge connects the file node to each symbol node

#### Scenario: Call edges between symbols (same language)
- **GIVEN** two files of the same language where a function in one calls a function in the other
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** a CALLS edge exists from the caller symbol to the callee symbol

#### Scenario: Import edges
- **GIVEN** a file that imports another file and uses a symbol from it
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** an IMPORTS edge connects the importing file node to the imported file node

#### Scenario: Extend and embed edges
- **GIVEN** a type that inherits, embeds, or implements another type
- **WHEN** code indexing runs with symbol extraction enabled
- **THEN** an EXTENDS edge connects the deriving type to the target type

#### Scenario: Symbol pipeline disabled
- **GIVEN** symbol extraction is disabled in `config.d/indexing.yaml`
- **WHEN** code indexing runs
- **THEN** no symbol nodes or CALLS/IMPORTS/EXTENDS/DEFINES edges are created

#### Scenario: Language not in enabled set
- **GIVEN** `symbols.languages` does not include a file's language
- **WHEN** code indexing runs over that file
- **THEN** no symbol nodes are created for that file and the file is otherwise indexed normally

### Requirement: Incremental symbol edge regeneration

The system SHALL regenerate symbol edges for a file when that file changes, preserving
edges for unchanged files.

#### Scenario: Modified file re-index
- **GIVEN** an indexed file with existing symbol nodes and edges
- **WHEN** the file changes and code indexing runs
- **THEN** the file's symbol nodes and pass-scoped edges are deleted and regenerated based on the new content

#### Scenario: Unchanged files untouched
- **GIVEN** an indexed file with symbol nodes and edges
- **WHEN** a different file changes and code indexing runs
- **THEN** the unchanged file's symbol nodes and edges remain unchanged

#### Scenario: Per-file isolation of extraction errors
- **GIVEN** a file that fails symbol extraction (e.g., parse error)
- **WHEN** code indexing runs for a directory containing that file
- **THEN** the indexing pass continues for other files and the failure is reported in `graph_status`

#### Scenario: Pass-scoped regeneration
- **GIVEN** a file with symbol edges and PDG edges
- **WHEN** symbol extraction is re-run for that file
- **THEN** only edges owned by the symbol pass (`pass='symbols'`) are deleted and regenerated
- **THEN** PDG and contract edges are preserved
