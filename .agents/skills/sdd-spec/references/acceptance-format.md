# Acceptance Format (Gherkin / BDD — skillgrid)

Each step has exactly one `acceptance.feature` file. It is the **WHAT** contract for that step — observable end-to-end behavior, not the internal HOW (HOW lives in `plan.md`, execution in `tasks.md`). `sdd-verify` maps every scenario in this file to a passing test run; a scenario without a green run is a FAIL for the step. `sdd-apply` names a specific scenario in its test task — never "cover the acceptance".

## File shape

```gherkin
# <NN>-<step-name> acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/<NNN-slug>/)

Feature: <one-line capability this step delivers>
  As a <role>
  I want <capability>
  So that <value to user / system>

  @happy @p0
  Scenario: <happy path — happy case is always present>
    Given <precondition the tester can set up>
    When  <action the tester can take>
    Then  <observable outcome the tester can assert>
    And   <additional observable outcome, if any>

  @edge
  Scenario: <edge case — empty state / boundary / concurrent>
    Given <precondition>
    When  <action>
    Then  <expected behavior at the boundary>

  @failure @p1
  Scenario: <failure / rejection / rollback>
    Given <precondition>
    When  <action>
    Then  <expected error / status / fallback>
    And   <no partial state remains, if rollback is part of the step>
```

## Rules

1. **One `Feature` per file.** Do not split a step's acceptance across multiple files — the per-step `verification.md` is the unit of the gate.
2. **Every step has ≥ 3 scenarios**: one `@happy`, one `@edge`, one `@failure`. If a step genuinely has no edge/failure case (e.g. a pure config-flag flip), document the reason in a `#` comment — do not silently omit.
3. **Scenarios are user-observable.** No function names, file paths, or internal line numbers. "When  `validateToken()` is called" is HOW — "When  a request arrives without a `Authorization` header" is WHAT.
4. **Tags are the selection contract.** `sdd-verify`'s test command uses these to select which scenarios to run (`--tags` for cucumber, `-name` for jest, `-run` for Go test, or the project's equivalent). `@p0` marks the blocking scenario for the step.
5. **Scenario names are unique within a step file.** `sdd-apply`'s test task references them literally; duplicate names break the trace.
6. **RFC 2119 in `THEN` lines is optional.** Gherkin `Then` already implies assertion-strength; `Must`-style strength is expressed by tags (`@p0` = blocking, `@p1` = should, `@p2` = may).

## Carrying the design's threat rows

`sdd-plan` marks an applicability-driven threat matrix, and **applicable rows are spec inputs.** For each row marked `Applicable`, ensure at least one scenario in the relevant step's `acceptance.feature` covers its planned RED test. A design-applicable row with no covering scenario in any step is a handoff gap — add the scenario to the correct step, or flag it in the envelope `risks` (do not silently drop it). `N/A` rows need no scenario.

The step is chosen by which part of the system the threat row touches — a VCS/PR-boundary row lands in the step that owns that boundary, not in a generic "infrastructure" step.

## Traceability

```
intent.md  ──►  capability names       │
                                           ├─► steps/<NN>/<name>/acceptance.feature
plan.md    ──►  threat rows + WHAT     │            │
                                           │            ▼
tasks.md   ──►  execution sequence   ──► sdd-apply references scenario names
```

- A `tasks.md` testing line must name the scenario it covers — e.g. `2.3 Write RED test for scenario "rejects token with expired signature"`.
- A `verification.md` compliance table must have one row per scenario in that step's `acceptance.feature`.
- If a scenario in `acceptance.feature` has no matching test run in `verification.md`, the step is `FAIL`.
