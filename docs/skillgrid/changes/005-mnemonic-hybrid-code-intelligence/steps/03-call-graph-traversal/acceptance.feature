# 03-call-graph-traversal acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/)

Feature: Call-graph traversal with confidence labels
  As a coding agent
  I want callers callees and dependents
  So that I can judge blast radius before edits

  @happy @p0
  Scenario: Known symbol returns graph views with confidence
    Given an indexed known symbol with resolved edges
    When an agent requests callers callees dependents implementors hierarchy and tests-for
    Then each view returns edges for that symbol
    And every edge carries a confidence label

  @edge
  Scenario: Ambiguous resolution is labeled not dropped
    Given a symbol with ambiguous edge resolution
    When an agent requests graph neighbors for it
    Then edges return with AMBIGUOUS confidence
    And those edges are not silently omitted

  @failure @p1
  Scenario: Unknown symbol graph query invents no edges
    Given an indexed project
    When an agent requests callers for a missing symbol
    Then the response is empty or not-found
    And no invented edges are returned
