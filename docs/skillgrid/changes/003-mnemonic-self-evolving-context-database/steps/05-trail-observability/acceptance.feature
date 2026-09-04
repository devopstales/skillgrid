# 05-trail-observability acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/)

Feature: Retrieval trail inspection
  As an operator
  I want to inspect recent retrieval trails
  So that I can debug opaque recall

  @happy @p0
  Scenario: Trail recent and show expose query paths
    Given recorded retrieval trails
    When the operator lists recent trails or shows one by id
    Then each entry shows query, directories, files, and result path

  @edge
  Scenario: Empty store lists nothing without error
    Given a store with no retrieval trails
    When the operator lists recent trails
    Then an empty list is returned without error

  @failure @p1
  Scenario: Unknown trail id is not found
    Given no trail exists for the requested id
    When the operator shows that trail id
    Then a not-found result is returned
