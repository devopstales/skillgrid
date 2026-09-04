# 05-commit-hooks-cli acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/004-hermes-memory/)

Feature: Commit fact extract and memory skill CLI
  As an agent and operator
  I want commit-time facts and CLI parity
  So that Fact Memory and Agent Skills stay in sync with MCP

  @happy @p0
  Scenario: Commit extracts facts and optional auto-skill
    Given a session ready for mnemonic commit
    When mnemonic commit runs
    Then prior compaction behaviour is preserved
    And facts are extracted and an Agent Skill may be auto-written

  @edge
  Scenario: Skip auto-skill when no reusable pattern
    Given a commit with no reusable Agent Skill pattern
    When mnemonic commit runs
    Then no auto-skill is written
    And the trail CLI remains unchanged

  @failure @p1
  Scenario: CLI memory and skill match MCP or fail cleanly
    Given Fact Memory and Agent Skill modules are available
    When the operator runs memory or skill CLI including execute
    Then outcomes match the MCP tools
    And invalid actions fail without corrupting Fact Memory
