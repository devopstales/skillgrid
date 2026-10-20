# 02-service-facade acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/001-hybrid-teams-architecture/)

Feature: Teams service facade lifecycle
  As an orchestrator
  I want spawn, pull, read, submit, review, and done
  So that agents can claim and complete team tasks

  @happy @p0
  Scenario: Spawn returns pending task id
    Given an open project store
    When a team task with brief is spawned
    Then a pending task id is returned

  @edge
  Scenario: Pull claims top priority with brief
    Given several pending tasks at different priorities
    When next task is pulled and read
    Then the top-priority unassigned task is claimed with brief markdown

  @edge
  Scenario: Output review done advance status
    Given a claimed team task
    When output, a passed review, then mark done run
    Then status is review_spec then done with passed review and results

  @failure @p1
  Scenario: Empty pull fails clearly
    Given no pending tasks
    When next task is pulled
    Then a clear error returns and nothing is claimed
