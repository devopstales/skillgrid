# 04-embedding-recall acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/002-mnemonic-identity-and-parity/)

Feature: Optional embedding recall
  As an agent
  I want vector recall behind the flag
  So that semantic recall is available alongside keyword search

  @happy @p0
  Scenario: Vector recall is available behind the flag
    Given the flag is enabled
    When searched
    Then vector recall is returned fused with keyword results

  @edge
  Scenario: Keyword-only fallback when vectors are absent
    Given no vector data present
    When searched
    Then only keyword results are returned

  @failure @p1
  Scenario: Disabled flag yields no vector recall
    Given the flag is disabled
    When searched
    Then no vector recall is returned
