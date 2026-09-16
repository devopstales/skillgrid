# Source: docs/skillgrid/changes/004-hermes-memory/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: Fact Memory schema on store open
  As an operator
  I want Fact Memory tables created on open
  So that facts persist without rewriting observations

  @happy @p0
  Scenario: Store open creates Fact Memory tables
    Given a project store after Tiered Storage migrations
    When the store is opened
    Then Fact Memory tables exist
    And prior observations and Tiered Storage rows are unchanged

  @edge
  Scenario: Re-open is idempotent
    Given a store that already applied the Fact Memory migration
    When the store is opened again
    Then no duplicate Fact Memory tables are created

  @failure @p1
  Scenario: Missing vector extension fails closed on vector ops
    Given a store without the vector extension loaded
    When a Fact Memory vector operation is requested
    Then the operation fails closed
    And non-vector Fact Memory open and search still work

@step-02
Feature: Fact Memory MCP tools and trails
  As an agent
  I want to add search forget and decay facts
  So that ranked Fact Memory is usable beside observations

  @happy @p0
  Scenario: Fact tools add search and record a Retrieval Trail
    Given Fact Memory tools are registered and observation save still works
    When an agent adds a fact and searches for it
    Then matching facts are returned
    And a Retrieval Trail records the mode and fact ids

  @edge
  Scenario: Decay lowers importance and logs events
    Given facts above the purge threshold
    When decay runs
    Then importance is lowered and forgetting events are logged
    And purge removes only facts below the threshold

  @failure @p1 @security
  Scenario: Soft-deleted fact absent from default search
    Given a fact that has been forgotten
    When default fact search runs
    Then that fact is absent from results
    And observation tools remain unchanged

@step-03
Feature: Agent Skill registry write list search
  As an agent
  I want to write list and search Agent Skills
  So that reusable scripts are discoverable beside observations

  @happy @p0
  Scenario: Write list and lexical search Agent Skills
    Given Agent Skill registry tools are registered and observation save still works
    When an agent writes an Agent Skill then lists and searches lexically
    Then the skill appears in list and search with stored metadata

  @edge
  Scenario: Overwrite false rejects name collision
    Given an Agent Skill already exists under a name
    When write is requested for that name with overwrite disabled
    Then the write is rejected with a clear error

  @failure @p1 @security
  Scenario: Soft-deleted skills omitted from default list and search
    Given an Agent Skill that has been soft-deleted
    When default list and search run
    Then that skill is omitted from results
    And observation tools remain unchanged

@step-04
Feature: Sandboxed skill execute and hybrid search
  As an agent
  I want sandboxed use_skill and hybrid ranking
  So that Agent Skills run safely and retrieve by meaning

  @happy @p0
  Scenario: Sandboxed use_skill returns captured IO and logs usage
    Given use_skill is registered and observation save still works
    When an agent runs an allowlisted Agent Skill in the sandbox
    Then captured output is returned
    And skill usage is logged

  @edge
  Scenario: Hybrid search ranks skills and facts
    Given skills and facts with embeddings available
    When hybrid search runs for skills or for facts in hybrid mode
    Then results combine lexical and vector ranking

  @failure @p1 @security
  Scenario: Path escape or unknown language rejects without exec
    Given a path escape or an unknown language
    When use_skill is requested
    Then the request errors without host-wide execution
    And soft-deleted facts stay absent from hybrid fact search

@step-05
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
