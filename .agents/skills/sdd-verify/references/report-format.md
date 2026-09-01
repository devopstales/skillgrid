# SDD Verify Report Format

## Compliance Statuses

- ✅ `COMPLIANT`: covering test exists and passed.
- ❌ `FAILING`: covering test exists but failed.
- ❌ `UNTESTED`: no covering test found.
- ⚠️ `PARTIAL`: test passes but covers only part of the scenario.

## Report Template

~~~markdown
```yaml
schema: skillgrid.verify-result/v1
evidence_revision: sha256:{current-evidence-digest}
verdict: pass
blockers: 0
critical_findings: 0
requirements: {complete}/{actual-total}
scenarios: {complete}/{actual-total}
test_command: {exact command}
test_exit_code: 0
test_output_hash: sha256:{exact-output-digest}
build_command: {exact command}
build_exit_code: 0
build_output_hash: sha256:{exact-output-digest}
```

## Verification Report

**Change**: {change-name}
**Version**: {spec version or N/A}
**Mode**: {Strict TDD | Standard}

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | {N} |
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

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| {REQ-01} | {Scenario} | `{file} > {test}` | ✅ COMPLIANT |
| {REQ-02} | {Scenario} | (none found) | ❌ UNTESTED |

**Compliance summary**: {N}/{total} scenarios compliant

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| {Req name} | ✅ Implemented | {brief note} |

### Coherence (Design)
| Decision | Followed? | Notes |
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

The YAML envelope MUST be the first non-empty content and contains every field exactly once. Counts come from the actual retrieved specs — never invent envelope totals. A valid `fail` is a legitimate, persistable verdict: it records what was checked and what failed. Human prose after the envelope never controls routing. Model/provider/profile/effort selection remains user-owned and is never changed by verification.

Before persisting, build the complete report as exact candidate bytes and self-check it (the Skillgrid replacement for an external admission validator): every envelope field is present and non-contradictory, the `requirements` / `scenarios` totals match the actual counts you retrieved from the specs, and every `CRITICAL` finding is backed by a test file / command / exit code. If the self-check fails, fix the report before persisting rather than writing an inconsistent artifact; if you cannot fix it, return `partial` and leave the prior report untouched.

## Blocked Preflight

When preflight denies entry before any command runs (no test runner detectable, the change folder missing its tasks artifact, or the edit/verify scope unsafe), emit the normal `fail` envelope plus these recovery fields and do NOT run the declared test/build commands:

```yaml
blocked_preflight: true
substantive_failure: false
command_failed: false
block_reason: {no-running-runner | missing-tasks-artifact | unsafe-scope}
```

Use `test_exit_code: 125` and `build_exit_code: 125` for the commands that did not run (the `125` convention means "not executed / preflight denial"), and note in the report which commands were declared but never run. Never emit this recovery shape for a substantive verification failure or an executed command failure.

When Strict TDD is active, insert the TDD compliance, test-layer distribution, changed-file coverage, and quality-metrics sections from [strict-tdd-verify.md](strict-tdd-verify.md).
