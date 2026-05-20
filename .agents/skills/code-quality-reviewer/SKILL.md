---
name: code-quality-reviewer
description: >
  Reviews implementation for code health: style, DRY, error handling, test quality,
  security, and maintainability. Produces severity-tagged issues and APPROVED/CHANGES_REQUESTED verdict.
  Used by sdd-review command as quality gate.
license: Apache-2.0
metadata:
  author: aiskillgrid-integration
  version: "1.0"
  mode: reviewer
  triggers:
    - "quality_review"
    - "sdd-review_invoke"
---

# Code Quality Reviewer

## Overview

Systematically assess implementation quality independent of specification compliance. Identify issues by severity, highlight strengths, and produce actionable recommendations.

**Core principle:** Code should be clean, maintainable, and correct — not just functional.

## When to Use

Invoked by:
- `sdd-review` command (primary)
- `sequential-agent-executor` stage 2 review (per-task quality gate)
- `pre-merge-verification` (full-change quality assessment)

Manual usage:
- `/review-quality --files src/ tests/`
- `/review-quality --re-review` (focus on previously flagged issues)

## Input

Expects:
- List of files to review (from task or change diff)
- Full content of each file (read from filesystem)
- Optional: previous review report for re-review mode
- Project style guide if available (`.editorconfig`, `ESLint`, `Prettier`, etc.)

## Review Dimensions

### 1. Readability & Clarity

**Check for:**
- **Descriptive names:** variables, functions, classes self-documenting
- **Avoid magic:** no inline literals without explanation
- **Length:** functions ≤ 50 lines, files ≤ 400 lines
- **Comments that explain why, not what:** avoid redundant comments

**Red flags:**
- `x`, `temp`, `data`, `info` as variable names
- Functions with >3 responsibilities
- Deeply nested conditionals (>3 levels)
- Long parameter lists (>4 parameters)

**Example review comment (MINOR):**
> Line 42: `temp` → rename to `bufferedUserInput` for clarity

---

### 2. DRY & Duplication

**Check for:**
- Repeated code blocks (copy-paste detection)
- Similar logic in multiple files
- Hardcoded values that appear >1 time

**Action:** Flag duplication, suggest helper extraction

**Example (IMPORTANT):**
> Lines 15-23 and 67-75 implement identical error handling. Extract `handleValidationError(err)`.

---

### 3. Error Handling

**Check for:**
- **Explicit errors:** no `catch` that swallows exception
- **User-friendly messages:** no stack traces leaked to users
- **Appropriate error types:** `ValidationError` vs `NotFoundError` vs `InternalError`
- **Logging:** errors logged with context (request ID, user, input)
- **Graceful degradation:** failures handled without crashing

**Red flags:**
- `catch (e) {}` empty handler
- Returning `200` with error in body for exceptional cases
- Uncaught promise rejections

**Example (CRITICAL):**
> Line 89: Database connection failure not caught → process crash on startup

---

### 4. Test Quality

**Assess tests (not just coverage):**

**Good test signs:**
- Arrange-Act-Assert sections clear
- One logical assertion per test (or related group)
- No over-mocking (tests behavior not mocks)
- Edge cases included: null, empty, boundary, invalid

**Bad test signs:**
- Testing implementation details (private methods, internals)
- Excessive stubbing that hides behavior
- Flaky timing (`sleep(100)`) or network calls
- Tests that always pass regardless of code (no assertions)
- Tests that test the mock, not the real code

**Example (IMPORTANT):**
> `test_login_success()` mocks entire auth service but never calls real code → tests mock behavior not implementation

---

### 5. Security

**Check for:**
- **Secrets:** no API keys, passwords, tokens in code
- **Input validation:** user input sanitized before use
- **SQL/command injection:** parameterized queries only
- **XSS prevention:** output encoding where appropriate
- **AuthZ checks:** every endpoint verifies permissions
- **No debug mode in production code:** `if (process.env.NODE_ENV !== 'production')` blocks

**Example (CRITICAL):**
> Line 104: `query("SELECT * FROM users WHERE id=" + userId)` — SQL injection. Use parameterized query.

---

### 6. Performance Patterns

**Check for:**
- **No N+1 queries:** loops shouldn't hit DB per iteration
- **No O(n²) in loops:** nested loops over same collection
- **Appropriate data structures:** map vs array lookup
- **Lazy loading where beneficial:** don't load entire dataset if only need subset
- **Caching:** repeated expensive computation cached

**Example (IMPORTANT):**
> Line 215: For loop inside another for loop over same users array → O(n²). Consider indexing.

---

### 7. Maintainability & Coupling

**Check for:**
- **Single Responsibility:** each function does one thing
- **Low coupling:** changes to one module don't ripple
- **Clear interfaces:** public APIs documented and stable
- **No hidden dependencies:** module imports are explicit
- **Testability:** code structure allows unit testing

**Example (MINOR):**
> `processOrder()` in OrderService handles payment, inventory, AND notification → extract three services

---

## Severity Definitions

| Level | Meaning | Action Required | Timeline |
|-------|---------|----------------|----------|
| **CRITICAL** | Merge blocker — security flaw, data loss, crash, incorrect behavior | Must fix immediately | Before merge |
| **IMPORTANT** | Should fix before merge — maintainability debt, clarity, non-blocking bugs | Fix before or schedule debt | Before merge preferred |
| **MINOR** | Optional polish — naming, formatting, comments | May defer | Any time |

**Severity decision guidelines:**
- CRITICAL: affects correctness, security, or stability in production
- IMPORTANT: makes code harder to maintain but works correctly
- MINOR: subjective style preferences, minor cleanups

## Output Format

```markdown
# Code Quality Review Report

**Change:** <change-id>
**Slice:** <slice-slug> (or full change)
**Reviewer:** <persona or agent-id>
**Timestamp:** <ISO datetime>

## Critical Issues (must fix before merge)

- [ ] SECURITY [src/auth.ts:45-50]: SQL injection via string concatenation
  - **Fix:** Use parameterized query: `db.query("SELECT * FROM users WHERE id = ?", [userId])`
  - **Evidence:** High risk of data breach

## Important Issues (should fix)

- [ ] DRY [src/utils.ts:22-30, src/helpers.ts:10-18]: Duplicate date formatting
  - **Fix:** Extract to `formatDate(date: Date): string` in shared utils
- [ ] READABILITY [src/payment.ts:88]: Function `process` is 80 lines
  - **Fix:** Split into `validate()`, `calculate()`, `apply()`

## Minor Issues (optional)

- [ ] NAMING [src/middleware.ts:12]: `tmp` → `requestId`
- [ ] COMMENT [src/db.ts:45]: Add explanation for timeout choice

## Strengths

- Login flow tests cover all edge cases
- Error messages clear and user-friendly
- Service boundaries well defined

## Summary

Total issues: 9 (1 CRITICAL, 3 IMPORTANT, 5 MINOR)

**Recommendation:** CHANGES_REQUESTED

**Conditions for approval:**
1. Fix SQL injection (CRITICAL)
2. Address DRY duplication (IMPORTANT)
3. Re-run review with fixes

**Re-review requested:** Yes
```

## Review Workflow

1. **Run analysis:** grep-style checks + manual reasoning
2. **Categorize issues:** assign severity and location
3. **Suggest fixes:** provide concrete remediation for each
4. **Highlight strengths:** balanced feedback
5. **Verdict:** APPROVED or CHANGES_REQUESTED
6. **If CHANGES_REQUESTED:** list specific re-review requirements

## Re-review

When reviewer re-runs on same code after fixes:
- Focus only on previously flagged items
- Confirm fixes are correct
- Check for new issues introduced by fixes
- Update verdict accordingly

**Do not re-review unchanged files.**

## Multiple Reviewers

If multiple reviewers required (config flag):
- Each submits independent report
- All must APPROVE (consensus)
- Any CRITICAL from any reviewer → CHANGES_REQUESTED

## Exit Conditions

**APPROVED when:**
- Zero CRITICAL issues
- Zero or resolved IMPORTANT issues
- No showstopper concerns

**CHANGES_REQUESTED when:**
- Any CRITICAL present
- IMPORTANT issues remain after 1 fix cycle
- Pattern of recurring issues

## Integration

This skill is called by:
- `sdd-review` command (primary) — after `trivy-security` / `security-review` when applicable
- `sequential-agent-executor` task review stage
- `pre-merge-verification` final quality check

Output consumed by:
- Implementation subagent (fixes issues)
- `sdd-archive` gate (check approval status)
- `pre-merge-verification` (combines with spec compliance)

## Configuration

`.skillgrid/config.json`:
```json
{
  "quality": {
    "require_tests": true,
    "max_function_length": 50,
    "max_file_length": 400,
    "allow_todos_in_production": false,
    "security_scan": true
  }
}
```

## See Also

- Spec compliance: `spec-compliance-verifier`
- Two-stage review: `sdd-verify` → `sdd-review` → `sdd-archive`
- Source model: Superpowers two-stage code quality review
