# 03-lifecycle-parity acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/002-mnemonic-identity-and-parity/)

Feature: Observation lifecycle parity
  As an agent
  I want pinning, expiry, duplicate count, and recency honoured
  So that recall quality matches the reference

  @happy @p0
  Scenario: Lifecycle columns are honoured
    Given observations with lifecycle state
    When evaluated
    Then pinning, expiry, duplicate count, and recency are honoured

  @edge
  Scenario: Expired entries are dropped
    Given entries past their expiry
    When evaluated
    Then they are no longer returned

  @failure @p1
  Scenario: Invalid state is rejected
    Given an invalid lifecycle state
    When evaluated
    Then it is rejected
