# 04-skill-execute-hybrid acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/004-hermes-memory/)

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
