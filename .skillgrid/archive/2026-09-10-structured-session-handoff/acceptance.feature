# Source: docs/skillgrid/changes/006-structured-session-handoff/change.md
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.

@step-01
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

@step-02
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

  @happy @p0
  Scenario: Resume with optional archive
    Given a recorded Session Handoff
    When session_resume requests an archive
    Then a next-session prompt is returned
    And an archive record is available when requested

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

@step-03
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

@step-04
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

@step-05
Feature: Optional Session Handoff watchdog
  As an operator
  I want a flag-gated context-limit watchdog
  So that handoff can trigger past a threshold

  @happy @p0
  Scenario: Enabled watchdog past threshold
    Given the handoff watchdog is enabled
    And usage is past the configured threshold
    When the watchdog checks
    Then a Session Handoff runs via the same path as explicit handoff

  @edge
  Scenario: Disabled or below threshold is no-op
    Given the watchdog is disabled by default or usage is below the threshold
    When the watchdog checks
    Then no automatic Session Handoff runs

  @failure @p1
  Scenario: Invalid watchdog config fails closed
    Given an invalid watchdog threshold or flag value
    When the watchdog checks
    Then no automatic Session Handoff runs
    And a clear configuration error is surfaced
