# 02-handoff-resume acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/006-structured-session-handoff/)
# Threat: Mnemonic tool surface — session_handoff/resume present; mem_save still works

Feature: Session Handoff and resume
  As an agent
  I want MCP handoff and resume
  So that work continues without re-deriving state

  @happy @p0
  Scenario: Handoff writes cleave bundle and row
    Given an active session
    When session_handoff runs
    Then progress, knowledge, and next-prompt files exist
    And a Session Handoff row is recorded
    And session_resume returns a next-session prompt

  @edge
  Scenario: Missing cleave or unknown handoff id
    Given no cleave workspace or an unknown handoff id
    When session_resume is requested
    Then a clear error is returned

  @failure @p1 @security
  Scenario: Fail closed and mem tools remain
    Given a handoff that cannot write cleave files
    When session_handoff is attempted
    Then no orphan handoff row remains
    And session_handoff and session_resume are registered
    And mem_save still succeeds
