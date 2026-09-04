# 04-session-compaction acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/)
# Threat: Mnemonic tool surface — mnemonic_commit explicit; mem_save still registered

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
