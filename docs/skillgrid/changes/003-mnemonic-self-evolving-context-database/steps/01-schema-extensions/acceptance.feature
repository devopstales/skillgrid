# 01-schema-extensions acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/)

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
