# 03-skills-registry acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/004-hermes-memory/)

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
