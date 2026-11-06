# Source: docs/skillgrid/changes/001-hybrid-teams-architecture/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# Threat: Mnemonic tool surface — six tools; bad spawn structured error; mem/code unchanged.

@step-01
Feature: Teams schema and content-plane writes
  As an operator
  I want additive teams tables and filesystem content storage
  So that SQL holds metadata while task content lives as markdown

  @happy @p0
  Scenario: Store open adds teams tables safely
    Given a store with observations
    When it opens with the additive teams migration
    Then teams, members, tasks, messages, results, and reviews tables exist
    And observations stay unchanged

  @edge
  Scenario: Markdown on disk SQL paths only
    Given a team task write with a no-op post-write seam
    When content is stored
    Then markdown is under the project files tree
    And SQL stores path and status only without tiered layers

  @edge
  Scenario: Migration remains idempotent on reopen
    Given a store that already applied the teams migration
    When the store opens again
    Then the teams tables still exist
    And observations stay unchanged

  @failure @p1
  Scenario: SQL fail after FS write rolls back
    Given a successful filesystem content write
    When the SQL insert fails
    Then the content file is removed with no corrupt row

@step-02
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

@step-03
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

  @edge
  Scenario: Memory and code tools remain registered
    Given a running MCP server with team tools
    When tools are listed
    Then existing memory and code tools remain present

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

@step-04
Feature: Teams HTTP routes under teams path
  As an HTTP client
  I want teams CRUD under the teams path prefix
  So that writes are auth-gated without colliding with memory reviews

  @happy @p0
  Scenario: Authenticated write under teams path succeeds
    Given a write token is configured
    When an authenticated client posts a teams resource
    Then the write succeeds

  @edge
  Scenario: Gets stay open and teams paths stay distinct
    Given a write token is configured
    When a client gets a teams resource without a bearer
    Then the read succeeds
    And teams routes do not collide with memory review routes

  @failure @p1
  Scenario: Unauthenticated write returns 401
    Given a write token is configured
    When a client posts a teams resource without a bearer
    Then the response status is 401

@step-05
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
