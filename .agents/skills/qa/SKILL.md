---
name: qa
description: "Use when a change is ready for the quality gate, after all implementation tasks and before the final review or merge — runs the test suite, verification-gap and TDD-evidence audits, and renders the four-state gate (PASS / CONCERNS / FAIL / WAIVED)."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: BMAD:bmad-testarch-trace + BMAD:test-levels-framework + gsd-core:gsd-verifier + gsd-core:TESTING-STANDARDS + gstack:cso + superpowers:verification-before-completion
---

# QA

Own the quality gate. Produce a test plan with layer selection, verify the goal was actually achieved (not just the tasks), audit verification coverage, test quality, security, and code quality, and render a four-state gate decision with configurable thresholds.

**Announce at start:** "I'm using the skillgrid:qa skill to verify this change meets its quality bar."

**Core principle:** Task completion ≠ goal achievement. Passing tests ≠ verified behavior. The gate is the decision, not the test suite.

**Config:** Read `.skillgrid/config.yaml` before starting.
- `testing.runner` — test command
- `testing.layers` — available test layers (e.g. `[unit, integration, e2e]`)
- `testing.coverage` — coverage command
- `testing.mutation` — mutation testing command (empty string = disabled)
- `quality.coverage_min` — minimum coverage % (0 = disabled, default 80)
- `quality.mutation_min` — minimum mutation score % (0 = disabled, default 80)
- `quality.p0_pass_rate` — minimum P0 pass rate % (default 100)
- `quality.p1_pass_rate` — minimum P1 pass rate % (default 95)
- `conventions.specs_root` (default `.skillgrid/specs/`) — the change's artifacts
- `conventions.glossary` (default `.skillgrid/glossary/`) — vocabulary for findings
- `ticketing.enabled` — update ticket status at gate transitions
- `security.trivy.command` — Trivy scan command ("" = disabled, use manual fallback)
- `security.trivy.severities` — minimum Trivy severity to report (default "CRITICAL")
- `security.trivy.scan_types` — Trivy scanners (default "vuln,secret,misconfig")
- `security.trivy.fail_on` — minimum Trivy severity that FAILs the gate ("" = report only)
- If `quality:` is absent from config: use the defaults above and note in the report "quality: not configured — using defaults. Run `skillgrid:onboarding` to set thresholds."

## When to Use

**Mandatory:**
- After all implementation tasks in a change are complete, before the final code review or merge
- After a re-verification run (when a prior QA gate returned CONCERNS or FAIL)

**Valuable:**
- Before declaring an epic or feature "done"
- After a significant refactor that touches multiple tickets
- When a human asks "is this actually done?"

**When NOT to use:**
- The change has a single small ticket with a single acceptance scenario — the spec reviewer in `skillgrid:requesting-code-review` already covers it
- You are mid-implementation (QA is a gate at the end, not a running check)

## The Process

### Step 1: Read the Change

Read, in order:
1. `briefing.md` — the falsifiable requirements (truths) and the stated goal
2. `acceptance.feature` — every scenario (the traceability oracle)
3. `blueprint.md` — the task breakdown and `SATISFIES` lines
4. `tasks.md` — the ticket list and dependency graph
5. `findings.md` (if present) — spike/sketch results that constrain the verification

Identify: every scenario name, every ticket, every `SATISFIES` mapping, the stated goal from the briefing, and each requirement's `#### Gates` block (the runnable shadow of its scenarios — see `skillgrid:test-driven-verification`).

### Step 2: Write the Test Plan

Produce `.skillgrid/specs/<topic>/test-plan.md` from [templates/test-plan.md](templates/test-plan.md).

**Process:**
1. List every scenario from `acceptance.feature`.
2. For each, check: does a test exist that exercises it? Name it. If not, status = `pending`.
3. **Select a layer** for each behavior per [references/test-strategy.md](references/test-strategy.md): highest available layer that fits (from `testing.layers`), degrade if unavailable. Apply the Duplicate Coverage Guard: if a lower layer already covers this, use the lower layer.
4. Add risk-ranked tests for behaviors the acceptance scenarios do NOT cover (internal invariants, error paths, edge cases from `findings.md`). Select layers for these too.
5. Fill the Edge-Case Matrix from the blueprint's task descriptions and the acceptance scenarios' edge/failure variants.
6. Assign a cadence to each test: `pr` (runs on every PR, blocks merge) or `nightly` (scheduled CI, regression net only).
7. Rank: P0 = would break production or lose data. P1 = would degrade the experience.

**Rules:**
- Every row names a concrete test or scenario — "covered by existing tests" is not a status.
- A `covered` row whose test did not actually run (unregistered, filtered, skipped) counts as `pending`.
- Out of Scope is explicit: name what is NOT tested and why.

**Commit the test plan** (spec zone rule: commit `.skillgrid/specs/` changes before code — the `pre-commit` zone guard enforces it).

### Step 3: Goal-Backward Verification

For each truth in the briefing, walk the four levels in [references/goal-backward.md](references/goal-backward.md):

| Level | What You Check |
|-------|---------------|
| Truth | The falsifiable claim is actually true — a named test proves it |
| Artifact | The named file/component exists, is substantive, and is wired |
| Key Link | The wiring between components is exercised end-to-end |
| Data Flow | Real data moves through the system as claimed |

**Force stance:** assume the goal was NOT achieved. Falsify the implementation narrative.

For each level, run the **named test** that covers it (not the full suite). Record the status:
- `VERIFIED` — test passed, exercises behavior
- `PRESENT_BEHAVIOR_UNVERIFIED` — code exists but no test exercises the transition → routes to human
- `UNVERIFIED` — no evidence found

**Rule:** presence is not behavior. A grep that confirms a function exists is not verification. A test that calls it and asserts the output is.

### Step 4: Verification-Gap Audit

Run the audit in [references/verification-gap.md](references/verification-gap.md).

**Ask one question for each changed behavior:** *"If this behavior broke where it's actually used, would verification fail?"*

Classify each gap:
- **Regression gap** — no test covering the use would fail
- **Missing-adoption gap** — a place that should use the new behavior doesn't
- **Broken-verification gap** — a test appears to cover it but wouldn't catch a regression
- **Missing-oracle gap** — a requirement's happy-path scenario has no `G<n>` entry (no runnable `CHECK`+`EXPECT`, no `manual`, no `ABANDON`). A scenario with no oracle is a traceability gap, not a completed behavior.

**Evidence rule:** read the test before claiming what it covers. Search the whole repo by symbol before claiming no test exists. Every finding must stand on its own evidence.

### Step 5: Traceability Check

Build the matrix: every scenario in `acceptance.feature` → the test that covers it → did it run? → pass/fail?

**Compliance statuses** (every scenario lands in exactly one):

| Status | Meaning | Severity |
|---|---|---|
| `COMPLIANT` | A covering test exists and passed at runtime | — |
| `PARTIAL` | A test passes but only partially covers the scenario | WARNING |
| `FAILING` | A covering test exists and failed | CRITICAL |
| `UNTESTED` | No covering test exists | CRITICAL (for required scenarios) |

**Counting rules:**
- Count the **actual** requirements and scenarios from `acceptance.feature` — never invent totals.
- The report's `scenarios: N/N` must equal the count you retrieved, not a guessed or rounded number.
- A scenario with **no covering test** is `UNTESTED`. A scenario whose covering test **failed** is `FAILING`.
- A test that passes but only partially covers the scenario is `PARTIAL`.

**Rules:**
- A scenario with no test = traceability gap (report in the matrix, not in the verification-gap audit).
- A test that ran and failed = CRITICAL finding.
- A test that exists but was skipped or filtered = treat as `UNTESTED`.
- **Never edit the expectation to match the code.** If a test disagrees with the matrix, fix the code.

**Self-check before persisting the QA report:**
1. Every scenario lands in exactly one compliance status.
2. Scenario totals match the actual count in `acceptance.feature`.
3. Every CRITICAL finding names a file / test / command / exit code.
4. Verdict is consistent with blockers: `FAIL` iff ≥1 CRITICAL, ≥1 unchecked ticket, or a test/build command exited non-zero.

### Step 6: TDD Evidence Audit

For each ticket in `tasks.md`, check the TDD Evidence:

| Check | Pass | Fail |
|-------|------|------|
| RED evidence | Commit or dry-run output showing the test failed before implementation | Missing |
| GREEN evidence | Commit or suite output showing the test passed after implementation | Missing |
| Scenario name match | The `SATISFIES` scenario name in the evidence matches the one in `acceptance.feature` | Stale |

**Verdicts:** `OK` / `MISSING_RED` / `MISSING_GREEN` / `STALE`

A `MISSING_RED` is a CRITICAL finding — it means the test may have been written after the code, in which case it proves nothing about what the code should do.

### Step 7: Test Quality Audit

Scan the tests that cover this change against the contracts in [references/test-rigor.md](references/test-rigor.md):

| Contract | Check |
|----------|-------|
| Real code | Is the test asserting on production behavior, not on the mock? |
| No vacuous truths | Can the assertion actually fail for a real input? |
| No pass-always | Would the test fail if the feature it describes were removed? |
| Test claimed path | Does the test exercise the code path its name claims? |
| Complete mocks | Does the mock return a realistic shape? |
| Counter-test | For "does not X" behavior, is there an explicit negative assertion? |

Also check for banned assertion patterns (tautologies, snapshot-only, source-text assertions, circular mocks).

**P0 triangulation check:** for every P0 behavior, is there a second test with different inputs? If not, file it.

### Step 8: Security Audit

Run the audit per [references/test-strategy.md](references/test-strategy.md).

**Mode A — Trivy (when `security.trivy.command` is set):**

Run the configured Trivy command. Parse the findings and classify per the reference:

| Trivy scanner | Severity | Classification |
|---------------|----------|----------------|
| vuln | CRITICAL | CRITICAL |
| vuln | HIGH | WARNING (or CRITICAL if `fail_on` includes HIGH) |
| secret | any | CRITICAL |
| misconfig | CRITICAL/HIGH | WARNING |
| misconfig | MEDIUM/LOW | SUGGESTION |

If `security.trivy.fail_on` is set and any finding meets or exceeds that severity, the Security gate **FAILs**. If `fail_on` is empty, findings are reported but never block.

**Mode B — Manual fallback (when Trivy is empty or not found at scan time):**

- **Secrets Archaeology:** scan git history for secret prefixes, check tracked files for `.env`/`.pem`/`credentials.json`, grep source for hardcoded keys.
- **Dependency Audit:** run the stack's CVE scanner (`npm audit` / `go list -m -u all` / `pip-audit` / `cargo audit`). Check for install scripts in production deps.
- **OWASP Spot-Check:** for each changed file that handles user input, check against OWASP Top 10. Flag suspicious patterns for a human to confirm.

**Classification (Mode B):**
- **CRITICAL** — a secret committed to git, a HIGH/CRITICAL CVE with a known fix in a production dependency
- **WARNING** — a secret inline in code but not committed, a CVE with no fix, an install script running arbitrary code
- **SUGGESTION** — an OWASP pattern that is suspicious but needs human confirmation

### Step 9: Code Quality Gate

Run the commands from `config.yaml` and compare against the `quality:` thresholds:

| Gate | Command | Threshold | FAIL when |
|------|---------|-----------|-----------|
| Coverage | `testing.coverage` | `quality.coverage_min` (default 80) | Coverage < threshold (if threshold > 0) |
| Mutation | `testing.mutation` | `quality.mutation_min` (default 80) | Mutation score < threshold (if command set and threshold > 0) |
| Lint | `commands.lint` | errors only | Any error (warnings = SUGGESTION) |
| Typecheck | `commands.typecheck` | errors only | Any error |
| P0 pass rate | run P0 tests from test plan | `quality.p0_pass_rate` (default 100) | Any P0 test fails |
| P1 pass rate | run P1 tests from test plan | `quality.p1_pass_rate` (default 95) | P1 pass rate < threshold |
| Trivy security | `security.trivy.command` | `security.trivy.fail_on` ("" = report only) | Any finding ≥ `fail_on` severity (if set) |

**Mutation testing:** if `testing.mutation` is empty, this gate is N/A (not a FAIL). If set, run it and report the mutation score. A surviving mutant is a concrete specification of missing coverage — treat it as a failing test.

**Dead code:** if a dead-code tool is configured, run it and report findings as SUGGESTION. Not a hard gate.

**Trivy security gate:** if `security.trivy.command` is set and `security.trivy.fail_on` is non-empty, any Trivy finding at or above the `fail_on` severity FAILs the gate. If `fail_on` is empty, Trivy findings are reported but never block. If `security.trivy.command` is empty, this gate is N/A (manual mode handles security separately).

**Threshold = 0 means disabled.** A gate with threshold 0 is N/A — it cannot FAIL.

### Step 10: Render the Gate

Produce `.skillgrid/specs/<topic>/qa-report.md` from [templates/qa-report.md](templates/qa-report.md).

**Four-state gate (HARD — thresholds from config):**

| Verdict | Criteria |
|---------|----------|
| **PASS** | ALL of: (1) all truths VERIFIED, (2) all scenarios covered by a test that ran and passed, (3) no CRITICAL findings from any audit, (4) no MISSING_RED, (5) all code-quality gates PASS or N/A, (6) P0 pass rate ≥ `quality.p0_pass_rate`, (7) P1 pass rate ≥ `quality.p1_pass_rate`, (8) coverage ≥ `quality.coverage_min` (if > 0), (9) mutation ≥ `quality.mutation_min` (if > 0 and command set), (10) no Trivy finding at or above `security.trivy.fail_on` severity (if `fail_on` is set). |
| **CONCERNS** | No CRITICAL findings and all hard gates PASS, BUT at least one of: a PRESENT_BEHAVIOR_UNVERIFIED level, a WARNING finding, a P0 without triangulation, a missing adoption that is low-risk, or a lint warning. Open items are named with a path to resolution. |
| **FAIL** | ANY of: a truth UNVERIFIED, a scenario with no covering test, a test that ran and failed, a MISSING_RED, a regression gap with no covering test, a missing-oracle gap (a happy-path scenario with no `G<n>`), a CRITICAL security finding, a code-quality gate FAIL (coverage/mutation/lint/typecheck/P0/P1 below threshold), or a Trivy finding at or above `security.trivy.fail_on` severity (if `fail_on` is set). |
| **WAIVED** | The human explicitly waived a specific gate criterion. The waiver, the criterion, and the accepted risk are recorded in the report. WAIVED is never a machine decision. |

**Hard rules:**
1. A human decision always overrides the machine verdict.
2. A change that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.
3. A non-empty `pending` list in the traceability matrix makes the gate **FAIL**, including in headless mode.
4. Passing tests do not substitute for running the system. The goal-backward check is the evidence, not the suite count.
5. A code-quality gate at threshold 0 is N/A — it cannot FAIL. Do not render FAIL for a disabled gate.
6. An `ABANDON`-ed gate is a handoff, never a pass: it keeps the gate unmet, so the gate is not **PASS** (it routes to **FAIL** or **WAIVED** on the human's explicit decision), and it is surfaced with the met / unmet / abandoned counts.

**Commit the QA report** (spec zone rule).

### Step 11: Report and Route

Present the gate verdict and the finding counts. Then route:

| Gate | Next Action |
|------|-------------|
| PASS | Proceed to `skillgrid:requesting-code-review` (or `skillgrid:parallel-code-review` for 50+ lines / high-risk). Update ticket status → `review`. |
| CONCERNS | List open items. For each: fix now / defer (to ticket) / human look. Fix the in-scope set with tests, re-run Steps 3-9 for the fixed items (re-verification mode: full check on failed, regression-only on passed). Re-render the gate. |
| FAIL | List CRITICAL findings and failed gates. Fix each with a failing test first (TDD). Re-run Steps 3-9. Re-render the gate. Cap: 3 rounds, then escalate to human. |
| WAIVED | Record the waiver. Proceed to review. |

**Fix loop cap:** 3 rounds. If the gate is still FAIL after 3 rounds of fixes, stop and escalate. The finding is likely not a test gap — it's an architectural or scope problem that needs a human decision.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "The test suite is green, that's enough" | Green suite = no known regressions. It does not prove the goal was achieved. A placeholder file passes the suite. Goal-backward verification is the evidence. |
| "All tasks are checked off" | Task completion ≠ goal achievement. The task "create chat component" is done when the file exists. The goal "working chat interface" is done when a test proves a message sends and renders. |
| "The spec reviewer already checked the scenarios" | The spec reviewer checks that the diff *makes* the scenario pass. QA checks that verification *would catch* a regression. Different questions. A scenario can be green now and unverified for the future. |
| "I don't have time for a test plan" | A test plan is not a test suite — it's a 15-minute check that names what's covered and what isn't. The gate it produces is the difference between "I think it's done" and "here's the evidence." |
| "The verification-gap audit is overkill for this change" | If the change touches a public API, a data path, or a state machine, the gap audit is 10 minutes. If it's a one-line config change, skip the skill entirely. |
| "PRESENT_BEHAVIOR_UNVERIFIED is fine, the code is there" | Code that is present but not exercised by a test is a future bug. The state transition, the cancellation, the ordering guarantee — none of them are proven until a test walks through them. |
| "I'll waive the gate" | WAIVED is a human decision, not a machine one. The machine renders the verdict; the human overrides it. The waiver is recorded, not implied. |
| "Coverage is 79%, that's close enough to 80%" | The threshold is a hard gate. 79 < 80 = FAIL. Adjust the threshold in config.yaml if 80 is wrong for this project — don't negotiate at the gate. |
| "The CVE is in a dev dependency, it's fine" | A CVE in a production dependency is CRITICAL. A CVE in a dev dependency is SUGGESTION. Check `package.json` `dependencies` vs `devDependencies` before classifying. |
| "I'll skip the security audit, it's a backend change" | Backend changes handle user input. The OWASP spot-check is 5 minutes. If the change touches no user input, note "no user-facing input in this change" and skip — but say so explicitly. |
| "Mutation testing is slow, skip it this time" | If `testing.mutation` is set and `quality.mutation_min` > 0, it's a hard gate. If it's too slow, run it on the changed files only (`--since` flag) or set the threshold to 0 in config. Don't skip silently. |
| "Trivy is already in CI, no need to run it here" | CI runs Trivy on push/PR. The QA gate runs it on the **working tree** — uncommitted changes, unpushed commits. A finding that's in your branch but not yet pushed is invisible to CI. Run it locally. |
| "The Trivy finding is in a dependency we don't control" | That's what the fix version column is for. If there's a fix, it's CRITICAL. If there's no fix, it's WARNING with a note. "We don't control it" is not a classification. |

## Red Flags

- Claiming PASS without running a named test for at least one truth
- Marking a truth VERIFIED based on a grep, not a test
- A traceability matrix with "covered by existing tests" as a status
- A `MISSING_RED` that is logged but not treated as CRITICAL
- A fix loop that exceeds 3 rounds without escalating
- Editing a test expectation to make the matrix green
- Rendering WAIVED without a named human and a recorded risk
- Skipping the verification-gap audit because "the tests look fine"
- Rendering PASS when a code-quality gate is below threshold (coverage, mutation, P0, P1)
- Treating a disabled gate (threshold 0) as FAIL
- Skipping the security audit without an explicit "no user-facing input" note
- Rendering PASS when a Trivy finding meets or exceeds `fail_on` severity
- Running Trivy with `--severity LOW` and calling the gate "clean" when `config.yaml` says CRITICAL
- Assigning every test to E2E because "it's more thorough" (Duplicate Coverage Guard violation)
- A test at the wrong layer: unit-testing what integration covers, or e2e-testing what unit covers

## Verification

- [ ] The full test suite (`testing.runner`) ran and exited 0; every P0/P1 test named in the test plan actually ran (not skipped or filtered)
- [ ] Lint and typecheck (`commands.lint` / `commands.typecheck`) ran with exit 0 and zero errors
- [ ] Coverage, mutation, and every code-quality gate PASS or N/A against `quality:` thresholds (no gate below threshold rendered as PASS)
- [ ] Goal-backward verification ran at all four levels; every truth in the briefing is `VERIFIED` by a named test, not a grep
- [ ] The traceability matrix is complete: every scenario in `acceptance.feature` is in exactly one compliance status, and totals match the actual count (not guessed)
- [ ] Every CRITICAL finding (verification, TDD, test quality, security) is resolved or explicitly WAIVED with a named human and a recorded risk
- [ ] The QA report (`qa-report.md`) is committed with concrete output — the four-state verdict, finding counts, and named evidence (file / test / command / exit code), never "looks good"

## Final Rule

```
Goal verified (named tests, 4 levels)
+ all scenarios traceable
+ no CRITICAL findings (verification, TDD, test quality, security)
+ all code-quality gates PASS or N/A (coverage, mutation, lint, typecheck, P0, P1)
+ layer selection justified (no duplicate coverage, no wrong-layer tests)
→ PASS

Otherwise → CONCERNS or FAIL, with named findings and a path to resolution.
Thresholds come from config.yaml. WAIVED is always a human decision, always recorded.
```
