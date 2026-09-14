# Test Plan — <feature>

> Derived from `.skillgrid/specs/YYYY-MM-DD-<topic>/blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the QA gate passes.

## Risk Ranking

<2-3 lines: what is the most likely thing to break in production, and why. This is what the plan tests hardest first.>

## Test Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | <falsifiable behavior> | <public boundary where it's observed> | unit / integration / e2e | <test file::name or scenario name> | P0 / P1 | pr / nightly | pending / planned / covered |
| 2 | | | | | | | |

**Layer** — selected per the layer selection rules in `references/test-strategy.md`. Highest available layer that fits; degrade if the layer is not in `testing.layers`. Duplicate Coverage Guard: if a lower layer already covers this, use the lower layer.

**Cadence** — `pr` (runs on every pull request, blocks merge) or `nightly` (runs in scheduled CI, regression net only).

**Status values:**
- `pending` — no test exists yet
- `planned` — test is being written (RED phase in progress)
- `covered` — test exists, ran, and passed in the verification output

**Rules:**
- Every `SATISFIES` scenario from `tasks.md` must appear here.
- Every row must name a concrete test or scenario — "covered by existing tests" is not a status.
- P0 = would break production or lose data; P1 = would degrade the experience.
- A `covered` row whose test did not actually run (unregistered, filtered, skipped) counts as `pending`.

## Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| <requirement-name> | <boundary condition> | <observable outcome> | <test/scenario> | pending / covered |
| | | | | |

## Out of Scope

- <Explicitly not tested, and why — e.g. "third-party payment gateway behavior — covered by their contract, we test the adapter">
