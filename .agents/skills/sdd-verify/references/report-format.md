# SDD Verify Report Format (per step)

Each step gets one `verification.md`. This template fills it.

## Compliance Statuses

- ✅ `COMPLIANT`: covering test exists and passed.
- ❌ `FAILING`: covering test exists but failed.
- ❌ `UNTESTED`: no covering test found.
- ⚠️ `PARTIAL`: test passes but covers only part of the scenario.

## Report Template (per step)

~~~markdown
```yaml
schema: skillgrid.verify-result/v1
change: {NNN-slug}
step: {NN-name}
evidence_revision: sha256:{current-evidence-digest}
verdict: pass
blockers: 0
critical_findings: 0
scenarios: {complete}/{actual-total}
test_command: {exact command}
test_exit_code: 0
test_output_hash: sha256:{exact-output-digest}
build_command: {exact command}
build_exit_code: 0
build_output_hash: sha256:{exact-output-digest}
```

## Verification: {NNN-slug} — Step {NN-name}

**Change**: {NNN-slug}
**Step**: {NN-name}
**Mode**: {Strict TDD | Standard}

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total (this step) | {N} |
| Tasks complete | {N} |
| Tasks incomplete | {N} |

### Build & Tests Execution
**Build**: ✅ Passed / ❌ Failed
```text
{build command and relevant output}
```

**Tests**: ✅ {N} passed / ❌ {N} failed / ⚠️ {N} skipped
```text
{test command and failure details}
```

**Coverage**: {N}% / threshold: {N}% → ✅ Above / ⚠️ Below / ➖ Not available

### Acceptance Compliance Matrix
| Scenario (from acceptance.feature) | Tag | Test | Result |
|-------------|-----|------|--------|
| {Scenario name} | @happy | `{file} > {test}` | ✅ COMPLIANT |
| {Scenario name} | @edge | (none found) | ❌ UNTESTED |

**Compliance summary**: {N}/{total} scenarios compliant

### Correctness (Static Evidence)
| Acceptance scenario | Status | Notes |
|------------|--------|-------|
| {Scenario name} | ✅ Implemented | {brief note} |

### Coherence (Plan)
| Plan decision (this step) | Followed? | Notes |
|----------|-----------|-------|
| {Decision} | ✅ Yes | |

### Issues Found
**CRITICAL**: {list or None}
**WARNING**: {list or None}
**SUGGESTION**: {list or None}

### Verdict
{PASS / PASS WITH WARNINGS / FAIL}
{one-line reason}
~~~

The YAML envelope MUST be the first non-empty content and contains every field exactly once. Counts come from the actual retrieved acceptance scenarios for **this step** — never invent envelope totals. A valid `fail` is a legitimate, persistable verdict: it records what was checked and what failed. Human prose after the envelope never controls routing. Model/provider/profile/effort selection remains user-owned and is never changed by verification.

Before persisting, build the complete report as exact candidate bytes and self-check it (the Skillgrid replacement for an external admission validator): every envelope field is present and non-contradictory, the `scenarios` total matches the actual count of scenarios in this step's `acceptance.feature`, and every `CRITICAL` finding is backed by a test file / command / exit code. If the self-check fails, fix the report before persisting rather than writing an inconsistent artifact; if you cannot fix it, return `partial` and leave the prior step report untouched.

## Blocked Preflight

When preflight denies entry before any command runs (no test runner detectable, the step folder missing its tasks/acceptance artifact, or the edit/verify scope unsafe), emit the normal `fail` envelope plus these recovery fields and do NOT run the declared test/build commands:

```yaml
blocked_preflight: true
substantive_failure: false
command_failed: false
block_reason: {no-running-runner | missing-tasks-artifact | unsafe-scope}
```

Use `test_exit_code: 125` and `build_exit_code: 125` for the commands that did not run (the `125` convention means "not executed / preflight denial"), and note in the report which commands were declared but never run. Never emit this recovery shape for a substantive verification failure or an executed command failure.

When Strict TDD is active, insert the TDD compliance, test-layer distribution, changed-file coverage, and quality-metrics sections from [strict-tdd-verify.md](strict-tdd-verify.md).
