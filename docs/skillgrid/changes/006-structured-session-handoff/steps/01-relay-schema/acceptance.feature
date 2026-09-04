# 01-relay-schema acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/006-structured-session-handoff/)

Feature: Session Relay schema migration
  As an operator
  I want handoff and archive tables on store open
  So that Session Relay persists without rewriting prior data

  @happy @p0
  Scenario: Store open creates handoff tables
    Given a project store
    When the store is opened
    Then handoff and archive tables exist
    And existing sessions and observations are unchanged

  @edge
  Scenario: Re-open is idempotent
    Given a store that already applied the Session Relay migration
    When the store is opened again
    Then no duplicate tables are created

  @failure @p1
  Scenario: Prior rows survive migration
    Given a store with data from earlier migrations
    When the Session Relay migration runs
    Then prior rows remain intact
    And sessions and observations are not rewritten
