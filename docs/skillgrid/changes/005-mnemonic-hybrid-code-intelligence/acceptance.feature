# Source: docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# Threat: Mnemonic tool surface — owning steps 02 and 04
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: Additive graph schema and language extractors
  As a coding agent
  I want symbols and edges after index
  So that navigation starts from structured code intelligence

  @happy @p0
  Scenario: Indexed Go and TS yield symbols and edges
    Given a project with Go and TypeScript or TSX sources
    When the code index runs
    Then symbols and edges for those files are queryable
    And graph tables exist without rewriting files or chunks

  @edge
  Scenario: Malformed file falls back and index continues
    Given one file that fails primary extraction
    When the code index runs
    Then fallback is used for that file
    And the index completes for remaining files

  @failure @p1
  Scenario: Store open preserves existing chunk index
    Given a store that already has files and chunks
    When the store opens with the new graph schema
    Then files and chunks remain intact
    And chunk search still returns prior hits

@step-02
Feature: Identifier-aware symbol search and orientation
  As a coding agent
  I want camelCase symbol lookup and orientation
  So that I find symbols chunk search misses

  @happy @p0
  Scenario: Identifier FTS and orientation locate a symbol
    Given indexed camelCase and snake_case symbols
    When an agent searches symbols and requests signature TOC map list metadata
    Then identifier search finds symbols chunk code_search misses
    And orientation returns signature TOC map list metadata

  @edge
  Scenario: Unknown symbol returns empty or not-found
    Given an indexed project
    When an agent looks up a missing symbol
    Then the response is empty or not-found with no fabricated symbol

  @failure @p1 @security
  Scenario: code_search stable and bad orient args fail
    Given the Mnemonic tool surface is registered
    When an agent inspects code_search and calls orient with missing args
    Then code_search name and query schema are unchanged
    And the orient call is rejected clearly

@step-03
Feature: Call-graph traversal with confidence labels
  As a coding agent
  I want callers callees and dependents
  So that I can judge blast radius before edits

  @happy @p0
  Scenario: Known symbol returns graph views with confidence
    Given an indexed known symbol with resolved edges
    When an agent requests callers callees dependents implementors hierarchy and tests-for
    Then each view returns edges for that symbol
    And every edge carries a confidence label

  @edge
  Scenario: Ambiguous resolution is labeled not dropped
    Given a symbol with ambiguous edge resolution
    When an agent requests graph neighbors for it
    Then edges return with AMBIGUOUS confidence
    And those edges are not silently omitted

  @failure @p1
  Scenario: Unknown symbol graph query invents no edges
    Given an indexed project
    When an agent requests callers for a missing symbol
    Then the response is empty or not-found
    And no invented edges are returned

@step-04
Feature: Offline hybrid code search with pluggable embeddings
  As a coding agent
  I want ranked hybrid hits with provenance
  So that search works without embeddings online

  @happy @p0
  Scenario: Hybrid search ranks offline with provenance
    Given an indexed project with embeddings disabled
    When an agent runs hybrid code search and checks semantic and embedding status
    Then ranked hits include per-signal FTS and signal provenance
    And semantic code search and embedding status are available

  @edge
  Scenario: Down embedder degrades to FTS and signals
    Given hybrid search available and embedder down
    When an agent runs hybrid code search
    Then ranking uses FTS and signals without hard-failing for missing embeddings

  @failure @p1 @security
  Scenario: Hybrid tool is distinct and rejects bad args
    Given the Mnemonic tool surface is registered
    When an agent compares hybrid code search to memory semantic search and sends bad args
    Then hybrid code search is distinct, code_search is unchanged, and bad args are rejected
