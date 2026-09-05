# 03-mcp-tools acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/001-hybrid-teams-architecture/)
# Threat: Mnemonic tool surface — six tools; bad spawn structured error

Feature: Teams MCP tool surface
  As an agent
  I want six team and agent MCP tools
  So that task work flows without inbox tools here

  @happy @p0
  Scenario: Six team tools are registered
    Given a running MCP server
    When tools are listed
    Then team_spawn_task, agent_pull_next_task, agent_read_task, agent_submit_output, agent_submit_review, and agent_mark_done are present
    And inbox tools are absent

  @edge
  Scenario: Spawn pull read submit stay consistent
    Given the six team tools
    When spawn, pull, read, and submit output run
    Then SQL metadata and filesystem content stay consistent

  @failure @p1 @security
  Scenario: Bad spawn errors without orphan state
    Given the team spawn tool
    When spawn gets invalid arguments
    Then a structured tool error returns without panic or orphan

  @failure @p1
  Scenario: Unknown id or empty queue errors
    Given no pending work or unknown task id
    When pull or read is invoked
    Then a clear tool error returns without panic
