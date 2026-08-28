## ADDED Requirements

### Requirement: Program-dependence-graph extraction

The system SHALL extract a per-function program-dependence graph (basic blocks with
control-flow and data-dependence edges) during code indexing, when PDG extraction is
enabled, for function symbols in the enabled language set.

#### Scenario: Extract basic blocks and CFG
- **GIVEN** a function with a conditional branch and a loop
- **WHEN** code indexing runs with `pdg.enabled=true` and the function's language is enabled
- **THEN** one `basic_block` node exists per basic block with `symbol_id`, `start_line`, `end_line`, `stmt_kind` properties
- **THEN** `CFG` edges connect each block to the block(s) it can transfer control to

#### Scenario: Extract data dependence (CDG)
- **GIVEN** a function where one statement's result is used by a later statement
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a `CDG` edge connects the defining block to the using block

#### Scenario: Reaching definitions
- **GIVEN** a variable defined and later used in a function
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a `REACHING_DEF` edge records the definition's flow to the use

#### Scenario: PDG disabled
- **GIVEN** `pdg.enabled=false` in config
- **WHEN** code indexing runs
- **THEN** no `basic_block` nodes or `CFG`/`CDG`/`REACHING_DEF`/`SOURCE`/`SINK`/`SANITIZES`/`TAINTED`/`TAINT_PATH` edges are created

#### Scenario: Function of a non-enabled language
- **GIVEN** a function whose language is not in `pdg.languages`
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** no PDG nodes or edges are created for that function

### Requirement: Taint analysis and precomputed flows

The system SHALL mark taint sources and sinks, record sanitizers, and precompute
source-to-sink taint paths (intra- then interprocedural) at index time, so the `explain`
and `pdg_query` tools are reads over persisted data.

#### Scenario: Source and sink markers
- **GIVEN** a function whose input parameter or external read is used in an external write or execution
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a `SOURCE` marker node/edge and a `SINK` marker node/edge exist per the configured source/sink classes

#### Scenario: Sanitizer on the path
- **GIVEN** a sanitizer function called between a tainted source and a sink
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a `SANITIZES` edge records the sanitizer, and the finding reflects it

#### Scenario: Precomputed taint path (intra-procedural)
- **GIVEN** a source that flows to a sink within one function
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a `TAINT_PATH` edge (source→sink) persists the flow with a `confidence` and `via_symbols` property

#### Scenario: Interprocedural taint
- **GIVEN** a callee that returns a tainted value and a caller that uses that return as a sink source
- **WHEN** code indexing runs with `pdg.enabled=true`
- **THEN** a cross-function `TAINT_PATH` is recorded spanning the CALLS edge

#### Scenario: Findings are candidates
- **WHEN** any taint finding is returned
- **THEN** it is labeled as a candidate with `confidence`, the full path, and sanitizer list, and is not asserted as a proof

#### Scenario: Incremental taint regeneration
- **GIVEN** a function with persisted PDG and taint data
- **WHEN** the function's body changes and code indexing runs
- **THEN** the function's pass-scoped (`pass='pdg'`) nodes and edges are regenerated; other functions are unchanged
