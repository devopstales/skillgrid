# 05-tests acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/001-hybrid-teams-architecture/)
# Threat: Mnemonic tool surface — six names; bad spawn; mem/code remain

Feature: Hybrid teams automated test coverage
  As a maintainer
  I want tests for facade, MCP, and HTTP teams behavior
  So that atomicity and tool-surface safety stay verified

  @happy @p0
  Scenario: Suite covers facade MCP and HTTP teams behavior
    Given the teams implementation is present
    When the full Go test suite runs
    Then facade atomicity, MCP registration and dispatch, and HTTP write auth are covered

  @edge
  Scenario: Registry has six team tools and keeps memory and code tools
    Given the MCP tool registry under test
    When registration is asserted
    Then the six team tool names are present
    And existing memory and code tools remain

  @failure @p1 @security
  Scenario: Tests assert rollback and bad spawn structured error
    Given fixtures where filesystem write succeeds and SQL fails
    When atomicity and bad-spawn tests run
    Then content file rollback is observed
    And bad spawn yields a structured tool error without orphan state
