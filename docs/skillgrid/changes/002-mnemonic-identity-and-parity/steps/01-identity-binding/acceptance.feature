# 01-identity-binding acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/002-mnemonic-identity-and-parity/)

Feature: Clone-private identity binding
  As an agent
  I want a stable, clone-private project identity
  So that memories are not scattered across invisible stores

  @happy @p0
  Scenario: Project binds to its clone
    Given a git repository
    When the project is resolved
    Then the project id is bound to that clone and never re-derived from mutable git state

  @edge
  Scenario: Ambiguity returns the candidate list
    Given more than one child repository
    When the project is resolved
    Then the result is ambiguous with the candidate list

  @failure @p1
  Scenario: Config walk is bounded and aliases are seeded
    Given a path outside the enclosing repository root
    When the config is looked up
    Then the walk stops at the enclosing repository root and prior keys route to the new canonical id
