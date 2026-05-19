---
name: pre-merge-verification
description: >
  Final quality gate before sdd-archive. Combines spec compliance, code review,
  and test health into single PASS/FAIL verdict. Orchestrates branch merge/PR decision.
license: Apache-2.0
metadata:
  author: aiskillgrid-integration
  version: "1.0"
  dependencies:
    - "spec-compliance-verifier"
    - "code-quality-reviewer"
  mode: gate
  triggers:
    - "sdd-archive_precheck"
    - "merge_gate"
---

# Pre-Merge Verification

## Overview

Final quality gate executed before `sdd-archive`. Validates that ALL previous phases passed and the change is safe to merge, PR, or archive.

**Core principle:** Never merge without passing both spec compliance (`sdd-verify`) and code quality review (`sdd-review`).

## When to Use

Invoked automatically:
- At the start of `sdd-archive` (mandatory pre-check)
- Before manual merge of any `sdd/*` branch
- As final CI/CD gate (if configured)

Manual usage (rare):
- `/pre-merge-check --change <id>`
- `/pre-merge-check --auto-merge` — merge if PASS (CI mode)

## Gate Checklist

All must pass:

### 1. Spec Compliance
- Requirement: `sdd-verify` status = `passed`
- Evidence: `.skillgrid/state/verification_status = passed` OR engram entry
- Action if FAIL: STOP, report "Spec compliance not verified — run sdd-verify"
- Action if PARTIAL: STOP, report "Spec incomplete — fix missing requirements"

### 2. Code Quality Approval
- Requirement: `sdd-review` status = `approved`
- Evidence: `openspec/changes/<id>/reviews/<timestamp>-review.md` with `APPROVED`
- Action if FAIL: STOP, report "Code review not approved — run sdd-review and fix issues"

### 3. Tests Passing
- Requirement: Full test suite green
- Command: Project's standard test command (npm test, make test, etc.)
- Evidence: Recent test log or CI status
- Action if FAIL: STOP, "Tests failing — fix before merging"

### 4. Lint/Typecheck Clean
- Requirement: No lint errors, typecheck errors (if project uses them)
- Command: `npm run lint`, `npm run typecheck`, or equivalent
- Action if FAIL: WARNING (may auto-fix) or BLOCK based on severity

### 5. No Uncommitted Changes
- Requirement: `git status --porcelain` returns empty
- Action if dirty: WARN and offer to commit/stash

### 6. Branch State Valid
- Requirement: Branch is up to date with `main` (or mergeable without conflicts)
- Check: `git merge-base --is-ancestor origin/main HEAD`
- If conflicts: BLOCK, "Rebase onto latest main before merging"

### 7. No Debug Code
- Scan for common debug leftovers:
  - `console.log`, `print()`, `debugger`, `TODO:` comments
  - Hardcoded test data in production paths
- Action: Warn or auto-remove (configurable)

### 8. Security Scan (if enabled)
- Run configured security scan (trivy, bandit, etc.)
- Fail on CRITICAL vulnerabilities

## Gate Evaluation

```bash
#!/bin/bash
# .skillgrid/scripts/run-pre-merge-check.sh

FAILED=0

# 1. Spec compliance
if [ "$(cat .skillgrid/state/verification_status)" != "passed" ]; then
  echo "❌ Spec compliance NOT verified"
  FAILED=1
fi

# 2. Code review
if ! grep -q "APPROVED" openspec/changes/$CHANGE_ID/reviews/*.md 2>/dev/null; then
  echo "❌ Code review not approved"
  FAILED=1
fi

# 3. Tests
npm test 2>&1 | tee /tmp/pre-merge-tests.log
if [ ${PIPESTATUS[0]} -ne 0 ]; then
  echo "❌ Tests failing"
  FAILED=1
fi

# 4. Lint
if [ -f package.json ]; then
  npm run lint 2>&1 | tee /tmp/pre-merge-lint.log
  if [ ${PIPESTATUS[0]} -ne 0 ]; then
    echo "⚠️  Lint errors (non-blocking)"
  fi
fi

# 5. Clean tree
if [ -n "$(git status --porcelain)" ]; then
  echo "❌ Uncommitted changes present"
  FAILED=1
fi

exit $FAILED
```

## Output

Gate report saved to:
`openspec/changes/<id>/verification/pre-merge-gate.md`

Format:
```markdown
# Pre-Merge Verification Report

**Change:** <id>
**Timestamp:** <ISO>
**Branch:** sdd/<id>/<slice>

## Gate Results

| Gate | Status | Evidence |
|------|--------|----------|
| Spec compliance | ✅ PASS | verification/<slice>-report.md |
| Code quality review | ✅ PASS | reviews/2025-05-13-review.md |
| Test suite | ✅ PASS | 127 tests passed |
| Lint/typecheck | ⚠️ WARN | 3 style warnings (non-blocking) |
| Clean working tree | ✅ PASS | no uncommitted changes |
| Branch mergeable | ✅ PASS | up to date with main |
| Security scan | ✅ PASS | 0 critical vulnerabilities |

**Overall:** ✅ PASSED

**Recommendation:** Proceed to merge / open PR / keep working

**Next:** Run `sdd-archive` to complete change lifecycle
```

## Branch Disposition

After gate PASS:

User chooses (or config auto-selects):

1. **Merge to main** — `git checkout main && git merge --no-ff sdd/...`
2. **Open PR** — `git push origin sdd/...` then create PR via `gh` or platform API
3. **Keep branch** — leave branch in place, no merge
4. **Discard** — delete branch, keep changes as local experiment

`pre-merge-verification` presents these options and records user choice.

## Integration with sdd-archive

`sdd-archive` begins with:
```
if ! pre-merge-verification --check; then
  status: blocked
  next_recommended: "Fix failing gates, then re-run: sdd-archive"
  exit 1
fi
```

Only after pre-merge verification PASS does `sdd-archive` proceed with:
- Closing change artifacts
- Updating INDEX if PRD-linked

## CI/CD Integration

For automated pipelines, `pre-merge-verification` can run as CI job:
- Checkout PR branch
- Run all gates
- Post status to PR: "pre-merge verified" check
- Auto-merge if policy allows and all gates pass

## Configuration

`.skillgrid/config.json`:
```json
{
  "pre_merge": {
    "require_spec_compliance": true,
    "require_code_review": true,
    "require_tests_green": true,
    "require_lint_clean": false,       // warnings only
    "allow_uncommitted_changes": false,
    "auto_merge_on_pass": false,
    "security_scan": true
  }
}
```

## Exit Conditions

**PASS:** All required gates pass → allow `sdd-archive` to continue
**FAIL:** Any required gate fails → BLOCK archive, require fixes

## See Also

- Next step: `sdd-archive` (uses this skill as gate)
- Prior gates: `sdd-verify` (spec), `sdd-review` (quality)
- Superpowers mapping: verification-before-completion + finishing-a-development-branch
