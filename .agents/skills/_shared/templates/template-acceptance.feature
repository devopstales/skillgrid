# Source: docs/skillgrid/changes/<NNN-slug>/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: <one-line capability for step 01>
  As a <role>
  I want <capability>
  So that <value>

  @happy @p0
  Scenario: <happy path from per-step WHAT — name referenced by tasks.md Run line>
    Given <precondition>
    When  <action>
    Then  <observable outcome>

  @edge
  Scenario: <edge case>
    Given <precondition>
    When  <action>
    Then  <expected fallback or boundary behavior>

  @failure @p1
  Scenario: <failure state — align with change.md Error handling row when applicable>
    Given <precondition>
    When  <action>
    Then  <expected error or recovery>

@step-02
Feature: <one-line capability for step 02>
  As a <role>
  I want <capability>
  So that <value>

  @happy @p0
  Scenario: <happy path>
    Given <precondition>
    When  <action>
    Then  <observable outcome>

  @edge
  Scenario: <edge case>
    Given <precondition>
    When  <action>
    Then  <expected fallback or boundary behavior>

  @failure @p1
  Scenario: <failure state>
    Given <precondition>
    When  <action>
    Then  <expected error or recovery>

# Rules:
# - ≥1 @happy + @edge + @failure per step
# - Every change.md per-step WHAT bullet → a Scenario
# - Every applicable threat-matrix row → a Scenario in its owning @step-NN
# - Scenario names unique and referenceable from tasks.md `Run:` / Expected lines
# - @p0 must cover change.md Definition of Done user-visible criteria
