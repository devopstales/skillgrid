# Source: docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.
# Threat: Mnemonic tool surface — L1-default search (03); mem_save remains registered (04).

@step-01
Feature: Additive tier schema
  As an operator
  I want tier, memory, trail, and embedding tables on store open
  So that existing data is never rewritten

  @happy @p0
  Scenario: Store open adds tables without rewriting rows
    Given an existing store with observations
    When the store is opened after schema upgrade
    Then tier, memory, trail, and embedding tables exist
    And existing observation rows are unchanged

  @edge
  Scenario: Upgrade from schema 008 is idempotent
    Given a store at schema 008
    When migration to 010 runs twice and the store reopens
    Then the schema is at 010 once
    And observations, full-text search, and the code index stay intact

  @failure @p1
  Scenario: Failed migration leaves prior data intact
    Given an existing store with observations and a code index
    When migration fails before completion
    Then prior observations and indexes remain readable

@step-02
Feature: Tiered L0 L1 L2 storage
  As a content writer
  I want automatic abstract and overview sidecars
  So that full-content writes stay fast

  @happy @p0
  Scenario: Content write yields sidecars without blocking
    Given a tier-eligible full-detail content write
    When the write completes
    Then abstract and overview sidecars appear with path columns
    And the full-detail write is not blocked waiting for summarization

  @edge
  Scenario: Tier migrate backfills without changing full detail
    Given existing full-detail files without sidecars
    When the operator runs tier migration
    Then abstract and overview sidecars are backfilled
    And full-detail bytes are unchanged

  @failure @p1
  Scenario: Summarizer failure preserves full detail
    Given a tier-eligible write whose summarizer fails
    When tier generation runs
    Then the full-detail content remains intact
    And the failure is logged

@step-03
Feature: Overview-first semantic retrieval
  As an agent
  I want ranked overviews by default
  So that I load full detail only when needed

  @happy @p0 @security
  Scenario: Semantic search returns ranked overviews only
    Given tiered content with abstracts and overviews
    When an agent runs semantic search
    Then results are ranked overviews with abstracts
    And full-detail bodies are absent

  @happy @p0 @security
  Scenario: Explicit load returns full markdown
    Given a path from a semantic search result
    When the agent loads full details for that path
    Then the full markdown is returned

  @edge
  Scenario: Embeddings off falls back with trail
    Given embeddings are disabled or empty
    When an agent runs semantic search
    Then ranking uses title or abstract fallback
    And a retrieval trail is recorded

  @failure @p1
  Scenario: Unknown path rejects full-detail load
    Given an unregistered path
    When the agent loads full details for it
    Then a clear not-found error is returned

@step-04
Feature: Explicit long-term memory commit
  As an agent
  I want an explicit commit into long-term memory
  So that sessions do not silently compact

  @happy @p0 @security
  Scenario: Explicit commit persists tiered long-term memory
    Given sources ready with an optional source link
    When the agent calls mnemonic commit
    Then long-term memory stores abstract, overview, and full detail
    And the optional source link is preserved

  @edge
  Scenario: Session end does not auto-commit
    Given an active session with unsaved lessons
    When the session ends without mnemonic commit
    Then no new long-term memory is written

  @failure @p1
  Scenario: Missing sources reject without partial write
    Given required sources are missing
    When the agent calls mnemonic commit
    Then clear error returns with no partial row

  @happy @p0 @security
  Scenario: Existing memory save remains registered
    Given retrieval and compaction tools are registered
    When an agent invokes the memory-save tool
    Then that tool remains registered and callable

@step-05
Feature: Retrieval trail inspection
  As an operator
  I want to inspect recent retrieval trails
  So that I can debug opaque recall

  @happy @p0
  Scenario: Trail recent and show expose query paths
    Given recorded retrieval trails
    When the operator lists recent trails or shows one by id
    Then each entry shows query, directories, files, and result path

  @edge
  Scenario: Empty store lists nothing without error
    Given a store with no retrieval trails
    When the operator lists recent trails
    Then an empty list is returned without error

  @failure @p1
  Scenario: Unknown trail id is not found
    Given no trail exists for the requested id
    When the operator shows that trail id
    Then a not-found result is returned
