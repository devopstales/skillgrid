# 04-session-cli acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/006-structured-session-handoff/)

Feature: Session Relay CLI
  As an operator
  I want session handoff, resume, and status
  So that I drive Session Relay without MCP

  @happy @p0
  Scenario: CLI mirrors MCP on the same store
    Given a project store used by MCP Session Relay
    When CLI session handoff, resume, and status run
    Then outcomes match the MCP path on that store

  @edge
  Scenario: Bad flags or missing id
    Given invalid flags or a missing handoff id
    When a session subcommand runs
    Then the process exits non-zero
    And stderr carries a clear message

  @failure @p1
  Scenario: CLI fails closed without a store
    Given no usable project store
    When CLI session handoff is invoked
    Then the command fails with a clear error
    And no partial cleave bundle remains
