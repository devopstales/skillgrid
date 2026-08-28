## ADDED Requirements

### Requirement: brainstorming checks prior work
The brainstorming skill SHALL instruct agents to search memory for prior related work.

#### Scenario: Prior work check in Explore project context
- **GIVEN** the brainstorming skill is active
- **WHEN** executing the "Explore project context" step
- **THEN** the agent is instructed to call `mem_search` for prior related decisions and discoveries

### Requirement: openspec-apply-change uses code search
The openspec-apply-change skill SHALL instruct agents to use code search for codebase exploration.

#### Scenario: Code search in read context files
- **GIVEN** the openspec-apply-change skill is active
- **WHEN** reading context files before implementation
- **THEN** the agent is instructed to use `code_search` to locate relevant implementation files

### Requirement: openspec-explore checks memory
The openspec-explore skill SHALL instruct agents to search memory for prior related exploration.

#### Scenario: Memory check in explore
- **GIVEN** the openspec-explore skill is active
- **WHEN** investigating a problem space
- **THEN** the agent is instructed to call `mem_search` for prior related work

### Requirement: project-context preserves mnemonic markers
The project-context skill SHALL recognize and preserve mnemonic protocol markers in AGENTS.md.

#### Scenario: Mnemonic markers preserved during refresh
- **GIVEN** AGENTS.md contains mnemonic protocol markers
- **WHEN** project-context refresh runs
- **THEN** the markers and their content are preserved byte-identical

#### Scenario: Both engram and mnemonic markers preserved
- **GIVEN** AGENTS.md contains both engram and mnemonic protocol markers
- **WHEN** project-context refresh runs
- **THEN** both marker blocks are preserved byte-identical

### Requirement: spec-as-source references memory
The spec-as-source skill SHALL instruct agents to search memory for related specs and decisions.

#### Scenario: Memory search during spec drafting
- **GIVEN** the spec-as-source skill is active
- **WHEN** drafting or modifying a spec
- **THEN** the agent is instructed to call `mem_search` for prior related specs and decisions
