# 03-status-compact acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/006-structured-session-handoff/)
# Threat: Mnemonic tool surface — status/compact present; mem_save works; compact needs no facts

Feature: Session status and thin knowledge compact
  As an agent
  I want status and thin knowledge refresh
  So that continuity works without Fact Memory

  @happy @p0
  Scenario: Status and compact without facts
    Given a session with one Session Handoff
    When session_status runs
    Then it reports handoff count and last cost or context
    And knowledge_compact refreshes knowledge without facts

  @edge
  Scenario: No handoffs yet
    Given no Session Handoffs
    When status and knowledge_compact run
    Then handoff count is zero
    And knowledge is empty without a crash

  @failure @p1 @security
  Scenario: New tools leave mem_save intact
    Given MCP tools after status and compact land
    When tools are listed and mem_save is invoked
    Then session_status and knowledge_compact are registered
    And mem_save still succeeds
    And compact needs no Fact Memory
