# 02-fact-tools acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/004-hermes-memory/)

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
