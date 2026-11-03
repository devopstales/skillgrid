# Source: briefing.md (revised 2026-09-29)
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: briefing.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ briefing.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: Fact Memory + Skills schema on store open
  As an operator
  I want Fact Memory and Skills tables created on open
  So that facts and skills persist without rewriting observations

  @happy @p0
  Scenario: Store open creates Fact Memory tables
    Given a project store after session-events and vector-db migrations
    When the store is opened
    Then Fact Memory and Skills tables exist
    And prior observations and Tiered Storage rows are unchanged
    And 014 importance columns are present on facts

  @edge
  Scenario: Re-open is idempotent
    Given a store that already applied the Fact Memory migration
    When the store is opened again
    Then no duplicate Fact Memory tables are created

  @failure @p1
  Scenario: Missing vector extension degrades gracefully
    Given a store without vec0 tables available
    When a Fact Memory vector operation is requested
    Then the vector leg degrades to in-memory or FTS-only
    And non-vector Fact Memory open and search still work

@step-02
Feature: Fact Memory MCP tools and session events
  As an agent
  I want to add search forget and decay facts
  So that ranked Fact Memory is usable beside observations

  @happy @p0
  Scenario: Fact tools add search and record a session event
    Given Fact Memory tools are registered and observation save still works
    When an agent adds a fact and searches for it
    Then matching facts are returned
    And a session_events row records action_type and fact ids

  @edge
  Scenario: Decay lowers importance and logs events
    Given facts with 014 AKL importance scores
    When decay runs
    Then importance is lowered via 014 AKL and a session_event is logged
    And purge removes only facts below the threshold via Dream Executor

  @failure @p1 @security
  Scenario: Soft-deleted fact absent from default search
    Given a fact that has been forgotten
    When default fact search runs
    Then that fact is absent from results
    And observation tools remain unchanged

@step-03
Feature: Agent Skill registry execute and sandbox
  As an agent
  I want to write list search and execute Agent Skills
  So that reusable scripts are discoverable and run safely

  @happy @p0
  Scenario: Write list search and execute Agent Skills
    Given Agent Skill registry tools are registered and observation save still works
    When an agent writes an Agent Skill then lists searches and executes it
    Then the skill appears in list and search with stored metadata
    And captured output is returned from sandboxed execution
    And skill usage is logged to skill_usage and session_events

  @edge
  Scenario: Overwrite false rejects name collision
    Given an Agent Skill already exists under a name
    When write is requested for that name with overwrite disabled
    Then the write is rejected with a clear error

  @failure @p1 @security
  Scenario: Path escape or unknown language rejects without exec
    Given a path escape or an unknown language
    When use_skill is requested
    Then the request errors without host-wide execution
    And soft-deleted skills stay absent from search

@step-04
Feature: Commit hooks hybrid search and CLI
  As an agent and operator
  I want commit-time facts hybrid search and CLI parity
  So that Fact Memory and Agent Skills stay in sync with MCP

  @happy @p0
  Scenario: Commit extracts facts and hybrid search works
    Given a session ready for mnemonic commit
    When mnemonic commit runs
    Then prior compaction behaviour is preserved
    And facts are extracted and an Agent Skill may be auto-written
    And hybrid search combines lexical and vector ranking via RRF

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
