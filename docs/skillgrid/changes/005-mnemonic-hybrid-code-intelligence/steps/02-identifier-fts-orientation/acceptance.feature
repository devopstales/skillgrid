# 02-identifier-fts-orientation acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/)
# Threat: Mnemonic tool surface — code_search stable; orient tools; bad args rejected

Feature: Identifier-aware symbol search and orientation
  As a coding agent
  I want camelCase symbol lookup and orientation
  So that I find symbols chunk search misses

  @happy @p0
  Scenario: Identifier FTS and orientation locate a symbol
    Given indexed camelCase and snake_case symbols
    When an agent searches symbols and requests signature TOC map list metadata
    Then identifier search finds symbols chunk code_search misses
    And orientation returns signature TOC map list metadata

  @edge
  Scenario: Unknown symbol returns empty or not-found
    Given an indexed project
    When an agent looks up a missing symbol
    Then the response is empty or not-found with no fabricated symbol

  @failure @p1 @security
  Scenario: code_search stable and bad orient args fail
    Given the Mnemonic tool surface is registered
    When an agent inspects code_search and calls orient with missing args
    Then code_search name and query schema are unchanged
    And the orient call is rejected clearly
