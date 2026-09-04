# 01-schema-extractors acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/)

Feature: Additive graph schema and language extractors
  As a coding agent
  I want symbols and edges after index
  So that navigation starts from structured code intelligence

  @happy @p0
  Scenario: Indexed Go and TS yield symbols and edges
    Given a project with Go and TypeScript or TSX sources
    When the code index runs
    Then symbols and edges for those files are queryable
    And graph tables exist without rewriting files or chunks

  @edge
  Scenario: Malformed file falls back and index continues
    Given one file that fails primary extraction
    When the code index runs
    Then fallback is used for that file
    And the index completes for remaining files

  @failure @p1
  Scenario: Store open preserves existing chunk index
    Given a store that already has files and chunks
    When the store opens with the new graph schema
    Then files and chunks remain intact
    And chunk search still returns prior hits
