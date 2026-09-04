# Acceptance Format (Gherkin / BDD — skillgrid v3)

One **change-level** `acceptance.feature` holds all steps. Each step is one `Feature` tagged `@step-NN`. It is the **WHAT** contract — observable behavior, not HOW (`change.md` holds HOW; `tasks.md` holds execution). `sdd-verify` maps every scenario to a passing run in that step's `### Verification` block inside `tasks.md`. `sdd-apply` names a specific scenario in its test task — never "cover the acceptance".

Canonical blank: [`../_shared/templates/template-acceptance.feature`](../../_shared/templates/template-acceptance.feature).

## File shape

```gherkin
# Source: docs/skillgrid/changes/<NNN-slug>/change.md

@step-01
Feature: <one-line capability this step delivers>
  As a <role>
  I want <capability>
  So that <value to user / system>

  @happy @p0
  Scenario: <happy path>
    Given <precondition>
    When  <action>
    Then  <observable outcome>

  @edge
  Scenario: <edge case>
    Given <precondition>
    When  <action>
    Then  <expected behavior at the boundary>

  @failure @p1
  Scenario: <failure / rejection / rollback>
    Given <precondition>
    When  <action>
    Then  <expected error / status / fallback>
```

## Rules

1. **One change-level file** — all Features in `docs/skillgrid/changes/<NNN-slug>/acceptance.feature`. No `steps/` tree.
2. **One `Feature` per step**, tagged `@step-NN` matching `tasks.md` section `## NN-<name>`.
3. **Every step has ≥ 3 scenarios**: `@happy`, `@edge`, `@failure`. Omit only with a `#` comment reason.
4. **Scenarios are user-observable.** No function names, file paths, or line numbers in Gherkin.
5. **Tags are the selection contract** for verify (`@step-NN`, `@p0`, `@happy`, …).
6. **Scenario names unique** across the file; tasks.md `Run:` lines reference them literally.
7. **`@p0` maps to change.md Definition of Done** user-visible criteria.

## Threat rows

`change.md` marks an applicability-driven threat matrix. Applicable rows → ≥1 scenario in the owning `@step-NN` Feature. Gaps → envelope `risks`.

## Traceability

```
change.md     ──► Goal / DoD / per-step WHAT / threat rows
                      │
tasks.md      ──► ## NN sections + Run/Expected ──► sdd-apply
                      │
acceptance.feature ──► @step-NN Features/Scenarios ──► sdd-verify → ### Verification
```
