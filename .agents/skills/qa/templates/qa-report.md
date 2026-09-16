# QA Report — <feature>

> Change: `.skillgrid/specs/YYYY-MM-DD-<topic>/`
> Generated: <ISO 8601 timestamp>
> Gate: PASS / CONCERNS / FAIL / WAIVED

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the gate passes.

### Risk Ranking

<2-3 lines: what is the most likely thing to break in production, and why. This is what the plan tests hardest first.>

### Plan

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

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| <requirement-name> | <boundary condition> | <observable outcome> | <test/scenario> | pending / covered |
| | | | | |

### Out of Scope

- <Explicitly not tested, and why — e.g. "third-party payment gateway behavior — covered by their contract, we test the adapter">

## Goal-Backward Verification

**Stated goal** (from briefing.md): <one line>

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | <falsifiable claim from briefing> | <test/scenario that proves it> | VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED |
| Artifact | <file/component that must exist and be wired> | <grep + test that shows it runs> | VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED |
| Key Link | <wiring between components> | <test that exercises the link end-to-end> | VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED |
| Data Flow | <data actually moves through the system> | <test with real input, observable output> | VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED |

**Status definitions:**
- `VERIFIED` — a test exercises this at the behavior level and passed.
- `PRESENT_BEHAVIOR_UNVERIFIED` — code exists and is wired, but no test exercises the state transition. Never counts as VERIFIED. Routes to human.
- `UNVERIFIED` — no evidence found.

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| <scenario-name> | <test file::name or scenario name> | yes/no | pass/fail/pending |
| | | | |

**Coverage:** <N>/<M> scenarios covered by a test that ran and passed.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| <description> | <file:line or test file> | regression / missing-adoption / broken-verification | <what you read that proves the gap> | <name the smallest change a consumer would observe> |

**Gap shapes:**
- **Regression gap** — changed code regresses where it's used, and no test covering that use would fail.
- **Missing-adoption gap** — a place that should use the new behavior doesn't.
- **Broken-verification gap** — a test appears to cover the behavior but would not catch a regression (skipped, flaky, mock-only, snapshot-only, source-text assertion).

## TDD Evidence Audit

| Task (from tasks.md) | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|----------------------|--------------------|--------------|----------------|---------|
| <ticket-id> | <scenario-name> | <commit/dry-run showing RED> | <commit/suite showing GREEN> | OK / MISSING_RED / MISSING_GREEN / STALE |

**Verdicts:**
- `OK` — RED and GREEN evidence present, scenario name matches.
- `MISSING_RED` — no evidence the test failed before implementation.
- `MISSING_GREEN` — no evidence the test passed after implementation.
- `STALE` — evidence references a different scenario name or test file than the one that exists.

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| <test file::name> | <contract name: real-code / no-vacuous / no-pass-always / test-claimed-path / complete-mocks / counter-test> | <the line or pattern that violates it> |

**Contracts (from gsd-core TESTING-STANDARDS):**
- **Real code** — test exercises production code, not a re-statement of the mock.
- **No vacuous truths** — assertion can actually fail for a real input.
- **No pass-always** — test fails if the feature it describes is removed.
- **Test claimed path** — test exercises the code path the test name claims.
- **Complete mocks** — mock returns a realistic shape, not a partial stub that hides the real interface.
- **Counter-test** — for "does not X" behavior, a test asserts the negative case explicitly.

## Test Strategy Audit

**Available layers** (from `testing.layers`): <unit, integration, e2e>

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| <behavior> | unit / integration / e2e | yes / no | yes / no | <why this layer> |

**Layer distribution:** <N> unit, <M> integration, <K> e2e. <Brief note on whether the distribution is reasonable for this change's risk profile.>

## Security Audit

**Mode:** <Trivy / Manual fallback>

### Trivy Findings (Mode A)

**Command:** `<security.trivy.command> --severity <severities> --scanners <scan_types> <target>`

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | <CVE-ID or vulnerability> | CRITICAL / HIGH / MEDIUM | <file:line or package> | <fix version or "none"> | CRITICAL / WARNING / SUGGESTION |
| secret | <secret type> | any | <file:line> | <rotate + remove> | CRITICAL |
| misconfig | <misconfig description> | CRITICAL / HIGH / MEDIUM | <file:line> | <fix> | WARNING / SUGGESTION |

**Trivy gate:** `fail_on: <severity or "">` → <PASS / FAIL / N/A>

### Manual Findings (Mode B — when Trivy is not configured)

#### Secrets Archaeology

| Location | Type | Evidence | Severity |
|----------|------|----------|----------|
| <file:line or git ref> | committed / tracked / inline / ci | <what you found> | CRITICAL / WARNING |

#### Dependency Audit

| Dependency | CVE / Issue | Severity | Fix Available |
|------------|-------------|----------|---------------|
| <package@version> | <CVE-ID or "install script"> | HIGH / CRITICAL / WARNING | yes (to version) / no |

#### OWASP Spot-Check

| File | OWASP Item | Pattern | Evidence |
|------|------------|---------|----------|
| <file:line> | A01-A08 | <what you saw> | <the code that shows it> |

**Security verdict:** <PASS / N findings — CRITICAL: <n>, WARNING: <n>, SUGGESTION: <n>>

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage | <testing.coverage> | <quality.coverage_min> | <actual %> | PASS / FAIL / N/A |
| Mutation | <mutation command or "not configured"> | <quality.mutation_min> | <actual % or "N/A"> | PASS / FAIL / N/A |
| Lint | <commands.lint> | errors only | <n errors, n warnings> | PASS / FAIL |
| Typecheck | <commands.typecheck> | errors only | <n errors> | PASS / FAIL |
| P0 pass rate | <run P0 tests> | <quality.p0_pass_rate> | <actual %> | PASS / FAIL |
| P1 pass rate | <run P1 tests> | <quality.p1_pass_rate> | <actual %> | PASS / FAIL |
| Trivy security | <security.trivy.command or "not configured"> | <security.trivy.fail_on or "report only"> | <n findings ≥ fail_on> | PASS / FAIL / N/A |

**Quality config status:** <configured / not configured — using defaults>

**Dead code (if configured):** <n findings — SUGGESTION> / N/A

## Findings

### CRITICAL (must fix before merge)

- <finding with file:line and evidence>

### WARNING (should fix before archive)

- <finding>

### SUGGESTION (nice to have)

- <finding>

## Gate Decision

**Verdict:** PASS / CONCERNS / FAIL / WAIVED

**Reasoning:** <2-3 sentences: what drove the verdict>

**Open items (if CONCERNS):**
- <item that blocks full PASS, with a path to resolution>

**Waiver (if WAIVED):**
- <what was waived, by whom, and the risk accepted>

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>
