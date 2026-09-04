# 02-cross-store-recall acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/002-mnemonic-identity-and-parity/)

Feature: Cross-store recall and alias unification
  As an agent
  I want recall across every store
  So that fragmented stores become one logical index

  @happy @p0
  Scenario: Recall spans every store
    Given data stored in multiple stores
    When recalled
    Then the results are merged and re-ranked across every store

  @edge
  Scenario: Fragmented stores are one logical index
    Given stores that were previously fragmented
    When queried
    Then they are treated as one logical index

  @failure @p1
  Scenario: Missing data yields no result
    Given no data present
    When recalled
    Then no result is returned
