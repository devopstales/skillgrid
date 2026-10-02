### Step 3: Goal-Backward Verification

For each truth in the briefing, walk the four levels in [goal-backward.md](goal-backward.md):

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

Run the audit in [verification-gap.md](verification-gap.md).

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

**Self-check before persisting `report.md` (the QA half):**
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

Scan the tests that cover this change against the contracts in [test-rigor.md](test-rigor.md):

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

Run the audit per [test-strategy.md](test-strategy.md).

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

### Step 9.5: Deterministic Gate Checks

Run `node .agents/skills/verification/qa/scripts/qa-gate.mjs .` from the project root. It
runs the three sub-checks (state drift, structure drift, size budget), captures
their exit codes + SCOPE lines, and prints a JSON summary. Read the JSON.

**Reading the JSON:**

- **`checks.state_drift`** — exit 1 (drift) → WARNING in the report's
  `## State Drift` section with the drift table from `lines[]`. Never
  CRITICAL, never affects the four-state verdict. Exit 2 (parse error) →
  WARNING naming the error. If drifted, patch `state.yaml` to match the
  derived values (spec zone is source of truth). Commit the fix.
- **`checks.ship_drift`** — exit 1 (drift) → WARNING in the report's
  `## Structure Drift` section with the table from `lines[]`. Advisory.
  Exit 0 with `lines[0]` starting "skipped" → no base ref; note it.
- **`checks.size_budget`** — exit 1 (overage) → WARNING per over-budget
  skill. Exit 0 → no action. Advisory.
- **Review evidence (advisory, not a script check)** — at the point QA routes
  to review (PASS / CONCERNS / WAIVED), a later change's `review.md`
  (written by `skillgrid:requesting-code-review`) is the durable audit record of
  that review. QA does not read or gate on a not-yet-existing `review.md` — but
  if `review.md` is **absent** in the spec folder when QA re-runs in
  re-verification mode (a code-zone change after a prior review), note a
  WARNING: "review evidence stale — `review.md` predates the latest code-zone
  change; re-run `skillgrid:requesting-code-review`." Advisory, never CRITICAL,
  never affects the four-state verdict.

**Scope:** read `scopes.composite`. This is the worst scope across all
checks (worst-scope-wins, per `verification-scope.md`). Step 9.7 uses this
value directly — do not re-derive it from the individual `SCOPE:` lines.

### Step 9.7: Verification Scope + Staleness Check

Per `_shared/conventions/verification-scope.md`: a zero count is never a bare
zero — it carries its **scope**. Two checks.

**Scope of the derivations.** For each count or enumeration this QA half
produced (the traceability matrix, the verification-gap audit, the drift
tables), name the scope of the input it saw, on the report's `## Verification
Scope` section:

| Derivation | Scope when |
|---|---|
| Traceability matrix | `COMPLETE` if every scenario in `acceptance.feature` was read; `TRUNCATED` if the file was read partially; `UNREADABLE` if the file was missing/unparseable |
| Verification-gap audit | `COMPLETE` if all changed behaviors were enumerated from the diff; `TRUNCATED` if the diff was bounded (e.g. `--since` window); `UNREADABLE` if the diff could not be read |
| State Drift (9.5) | `scopes.state_drift` from the `qa-gate.mjs` JSON |
| Structure Drift (9.5) | `scopes.ship_drift` from the `qa-gate.mjs` JSON |
| Composite (all) | `scopes.composite` from the `qa-gate.mjs` JSON (worst-scope-wins) |

Record each as `SCOPE: <atom>`. If a derivation's scope is not `COMPLETE`, that
is a named non-answer, not a clean bill — it is reflected in the gate per the
fail-closed rule below.

**Stale-verification check.** A verification is **stale** when the evidence it
rests on is older than the code it is meant to cover. Concretely: if any
code-zone file changed **after** the QA-half verdict in `report.md` was written
(re-verification mode after a prior gate, or a fix landed after a prior PASS),
the prior verification is stale and this run is re-verifying it. Record in the
report's `## Verification Scope` section:

- `STALE: none` — no code-zone change since the last verification; the prior
  evidence is current.
- `STALE: <paths>` — name the changed paths; the prior verification does not
  cover them. This run's evidence (named tests, run commands) is what the gate
  rests on, **not** the prior verdict.

A stale check is **advisory** in itself (it never flips the verdict by itself)
but it is what makes the fail-closed rule below meaningful: a gate that rests
on stale evidence while its scope is not `COMPLETE` routes away from PASS.

### Step 9.8: Floor Computation

Per `_shared/conventions/floor.md`: the gate is the **weakest dimension**, never
an average. Reduce every audit and gate this run produced to its weakest
dimension, then take the floor across them. Write the result on the report's
`## Floor` section (one row per dimension + a `**FLOOR:**` line naming the
weakest).

**The floor caps the verdict.** The gate can never render higher than its
floor:
- any dimension at its FAIL value (a gate `FAIL`, a truth `UNVERIFIED`, a
  scenario `UNTESTED`/`FAILING`, a ticket `MISSING_RED`, a gap `CRITICAL`, a
  security `CRITICAL`, a scope `UNREADABLE`) → floor is **FAIL**;
- any dimension at its non-answer value (scope `TRUNCATED`/`UNSCOPED`, a
  `PRESENT_BEHAVIOR_UNVERIFIED` truth, a `STALE`/`MISSING_GREEN` ticket, a
  `PARTIAL` scenario, a `WARNING` gap/security) → floor is at most **CONCERNS**;
- only when every dimension is at its best value (all `VERIFIED` / `COMPLIANT` /
  `OK` / `PASS`-or-N/A / scope `COMPLETE`) is the floor **PASS**-eligible.

The mean of the passing dimensions is reported for context only — the gate
branches on the floor, never on the mean. This makes hard rule 7 (fail-closed on
scope) and the four-state table below a single computation rather than a set of
separate checks: the floor *is* the gate.
