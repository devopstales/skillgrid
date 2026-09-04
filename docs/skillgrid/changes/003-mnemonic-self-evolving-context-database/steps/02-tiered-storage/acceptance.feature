# 02-tiered-storage acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/)

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
