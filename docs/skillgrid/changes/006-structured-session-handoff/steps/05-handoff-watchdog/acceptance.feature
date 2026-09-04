# 05-handoff-watchdog acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/006-structured-session-handoff/)

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
