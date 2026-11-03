# 01-schema acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/001-hybrid-teams-architecture/)

Feature: Teams schema and content-plane writes
  As an operator
  I want additive teams tables and filesystem content storage
  So that SQL holds metadata while task content lives as markdown

  @happy @p0
  Scenario: Store open adds teams tables safely
    Given a store with observations
    When it opens with the additive teams migration
    Then teams, members, tasks, messages, results, and reviews tables exist
    And observations stay unchanged

  @edge
  Scenario: Markdown on disk SQL paths only
    Given a team task write with a no-op post-write seam
    When content is stored
    Then markdown is under the project files tree
    And SQL stores path and status only without tiered layers

  @failure @p1
  Scenario: SQL fail after FS write rolls back
    Given a successful filesystem content write
    When the SQL insert fails
    Then the content file is removed with no corrupt row
