# 01-facts-schema acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/004-hermes-memory/)

Feature: Fact Memory schema on store open
  As an operator
  I want Fact Memory tables created on open
  So that facts persist without rewriting observations

  @happy @p0
  Scenario: Store open creates Fact Memory tables
    Given a project store after Tiered Storage migrations
    When the store is opened
    Then Fact Memory tables exist
    And prior observations and Tiered Storage rows are unchanged

  @edge
  Scenario: Re-open is idempotent
    Given a store that already applied the Fact Memory migration
    When the store is opened again
    Then no duplicate Fact Memory tables are created

  @failure @p1
  Scenario: Missing vector extension fails closed on vector ops
    Given a store without the vector extension loaded
    When a Fact Memory vector operation is requested
    Then the operation fails closed
    And non-vector Fact Memory open and search still work
