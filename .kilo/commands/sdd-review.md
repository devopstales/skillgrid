---
name: sdd-review
id: sdd-review
category: Workflow
description: Perform code quality review on completed implementation
agent: odin
subtask: true
---

You are an SDD sub-agent managing the code review phase.

**VDD ROAST — run before standard review:**
- Read `.agents/skills/vdd-roast/SKILL.md`
- Invoke the adversarial perspective: zero tolerance for errors, lazy patterns, or slop
- Focus on structural flaws, not style preferences
- Reset context to ensure detached every critique cycle
- Document critiques with severity (HIGH/MEDIUM/LOW) and file:line references
- Address and fix all HIGH-severity roast findings before proceeding to Stage A–D review

**Required skills to read (in order):**
1. `.agents/skills/code-quality-reviewer/SKILL.md` — quality review logic
2. `.agents/skills/truecourse-analyze/SKILL.md` — architecture analysis (if enabled)
3. `.agents/skills/truecourse-list/SKILL.md` — violation listing (if enabled)

## Context
- Working directory: `!echo -n "$(pwd)"`
- Active change ID: read from `.skillgrid/state/active_change` or derive from current branch
- Active slice: read from tasks file or infer from diff
- Artifact store mode: hybrid

## Preflight Checks

1. Verify `sdd-verify` has PASSED for this change
   - Check `.skillgrid/state/verification_status` or engram entry
   - If not verified: BLOCK and instruct user to run `sdd-verify` first

2. Confirm all tasks marked complete in `tasks.md`
   - Uncompleted tasks → warn, require `--force` to proceed

3. Ensure worktree is clean (no uncommitted changes)
   - `git status --porcelain` should return empty
   - If dirty: FAIL with "commit all changes before review"

## Review Pipeline

`sdd-review` orchestrates a multi-stage quality assessment pipeline:

```
Stage A: Security scan (Trivy MCP) — if configured
Stage B: Architecture analysis (TrueCourse) — if enabled
Stage C: Code quality review (code-quality-reviewer)
Stage D: Consolidation — merge all findings into single verdict
```

### Stage A: Trivy Vulnerability Scan

**Trigger:** Config key `review.security.trivy_scan = true` OR flag `--security`

**Prerequisites:**
- Trivy MCP configured at `.configs/mcp/trivy.json`
- `trivy` binary installed (aquasecurity/trivy) and in PATH

**Execution:**

Use MCP tool `trivy_scan_filesystem`:

```bash
# Parameters:
target: repository root
scanType: ["vuln", "secret", "config"]  # or as configured
severities: ["CRITICAL", "HIGH", "MEDIUM"]
outputFormat: "json"
```

Save report: `.agents/reviews/<change-id>/trivy-report.json`

**Parse:**
- CVE vulnerabilities → package, version, fixed version, CVE URL
- Hardcoded secrets → file:line, secret type
- Misconfigurations → resource, issue

**Report block:**
```markdown
## Security Scan (Trivy)

| Severity | Count | Issues |
|----------|-------|--------|
| CRITICAL | 0 | — |
| HIGH     | 2 | [CVE-2024-1234 in lodash@4.17.20](https://nvd.nist.gov/...), [Hardcoded AWS secret in src/aws.ts:42] |
| MEDIUM   | 3 | [Outdated express], [Insecure Dockerfile CMD]... |

**Fix before merge:** All CRITICAL and HIGH severity findings.
```

**Fail conditions** (if config `fail_on_cve` or `fail_on_secret` true):
- Any CRITICAL/HIGH CVE → automatic `CHANGES_REQUESTED`
- Any hardcoded secret → automatic `CHANGES_REQUESTED`

### Stage B: TrueCourse Architecture Analysis

**Trigger:** Config `review.architecture.truecourse_enabled = true` OR flag `--architecture`

**Prerequisites:**
- `npx -y truecourse` available
- Baseline `.truecourse/LATEST.json` exists (from full analysis on main)

**Mode:**
- Default: `--diff` (only new violations in current branch)
- `--full-analysis` flag: full repo scan (slower, comprehensive)
- LLM rules: only if user approves (`--llm` flag or config `truecourse_llm: true`)

**Execution:**
```bash
# Diff mode (preferred)
npx -y truecourse analyze --diff --no-llm 2>&1 | tee .agents/reviews/<change-id>/truecourse-diff.json

# Get violation list (first page)
npx -y truecourse list --diff --limit 20 2>&1 | tee .agents/reviews/<change-id>/truecourse-violations.txt
```

**Parse:**
- New violations count by severity
- Resolved violations count
- Categories: circular dependency, layer violation, missing interface, etc.

**Report block:**
```markdown
## Architecture Analysis (TrueCourse)

**Mode:** Diff against baseline
**New violations:** 3 (2 high, 1 medium)
**Resolved:** 1

| Severity | New | Resolved | Typical Categories |
|----------|-----|----------|-------------------|
| Critical | 0   | 0        | — |
| High     | 2   | 1        | Circular dependency, missing abstraction |
| Medium   | 1   | 0        | Layer violation, naming drift |

**Action:** Fix HIGH violations before merge. MEDIUM recommended.
```

**Fail conditions** (if `fail_on_new_violations: true` and `min_severity_to_fail: "high"`):
- Any new HIGH or CRITICAL violation → `CHANGES_REQUESTED`

If TrueCourse unavailable: WARNING and continue.

### Stage C: Code Quality Review (Core)

Invoke `code-quality-reviewer` skill:

- Scope: all files modified in this change
- Checks: readability, DRY, error handling, test quality, security (in addition to Trivy), performance, maintainability
- Output: severity-tagged issues (CRITICAL/IMPORTANT/MINOR)

### Stage D: Consolidation

Merge findings from A, B, C into unified report.

**Severity hierarchy (highest优先):**
1. CRITICAL (any source)
2. HIGH (Trivy CVE/secret, TrueCourse HIGH)
3. IMPORTANT (code quality)
4. MEDIUM (Trivy MEDIUM, TrueCourse MEDIUM)
5. MINOR (code quality, LOW/INFO from tools)

**Verdict logic:**
```
IF any CRITICAL from ANY stage → CHANGES_REQUESTED
ELIF (TrueCourse enabled AND new HIGH violations) → CHANGES_REQUESTED
ELIF (Trivy fail_on_cve/secret enabled AND CRITICAL/HIGH security finding) → CHANGES_REQUESTED
ELIF IMPORTANT count > 0 → CHANGES_REQUESTED
ELSE → APPROVED
```

**Final report saved to:**
- `openspec/changes/<id>/reviews/<YYYY-MM-DD-HHMM>-review.md`
- Engram: `sdd/<id>/review` (hybrid mode)

Report includes all three sections (Security, Architecture, Code Quality) with consolidated verdict.

## Flags

| Flag | Effect |
|------|--------|
| `--security` | Force Trivy scan even if config disabled |
| `--no-security` | Skip Trivy |
| `--architecture` | Force TrueCourse analysis |
| `--no-architecture` | Skip TrueCourse |
| `--full-analysis` | TrueCourse full scan (slow) vs default diff |
| `--llm` | Enable TrueCourse LLM-powered rules (costs tokens) |
| `--slice <slug>` | Review specific slice only |
| `--re-review` | Focus on previously flagged issues |
| `--reviewer <persona>` | Delegate review to persona (thor, heimdall, frigg, …) |
| `--force` | Skip already-reviewed check |

## Two-Stage Review Integration

`sdd-review` implements **Stage 2** of the two-stage review pipeline:

```
sdd-apply → sdd-verify (spec compliance — Stage 1) → sdd-review (multi-stage quality — Stage 2) → sdd-archive
```

You receive:
- **Spec compliance PASSED** status from `sdd-verify`
- Implementation is fully coded and tested

Your job: orchestrate comprehensive quality assessment across security, architecture, and code health.

## Quality Checklist

When invoking `code-quality-reviewer`, include these aspects:

### Style & Readability
- Names are descriptive and consistent
- No magic numbers or strings
- Functions fit on screen (≤50 lines)
- Comments explain why, not what
- Language idioms followed

### DRY & Duplication
- No copy-pasted code blocks
- Shared logic extracted
- Repeated literals → constants

### Error Handling
- Errors are explicit (no bare `catch` that swallows)
- User-friendly error messages
- Appropriate error types (validation, not-found, etc.)
- Errors logged with context

### Test Quality
- Tests are deterministic (no flaky timing/network)
- Arrange-Act-Assert structure clear
- Edge cases covered (null, empty, boundary values)
- mocks used judiciously — tests real behavior, not mocks

### Security
- No secrets hardcoded
- Input validation on boundaries
- SQL/command injection prevention
- Authentication/authorization checks present where needed

### Performance
- No O(n²) in loops
- Database queries indexed
- Caching where appropriate
- No N+1 query patterns

### Maintainability
- Coupling low, cohesion high
- Single responsibility per function/module
- Public interfaces documented
- Breaking changes isolated

## Severity Tagging & Consolidation

`code-quality-reviewer` produces CRITICAL/IMPORTANT/MINOR tags.

Additional severity inputs from external tools:

**Trivy:**
- CRITICAL, HIGH → treated as **CRITICAL** for verdict (if `fail_on_cve/secret` enabled)
- MEDIUM → **IMPORTANT** or **MINOR** based on config

**TrueCourse:**
- critical / high → **CRITICAL** (if `fail_on_new_violations` and `min_severity_to_fail: "high"`)
- medium → **IMPORTANT**
- low/info → MINOR or advisory

**Report format:**
```markdown
# Code Review Report — <change-id>

## Critical Issues (must fix)

- [ ] CVE-2024-1234 [src/deps.ts]: lodash 4.17.20 has prototype pollution
  - **Fix:** Upgrade to lodash 4.17.21
  - **Source:** Trivy

- [ ] Circular dependency [service.ts ← repository.ts]
  - **Fix:** Extract shared interface
  - **Source:** TrueCourse

## Important Issues (should fix)

- [ ] DRY [utils.ts:22-30, helpers.ts:10-18]: Duplicate date formatting
- [ ] Readability [payment.ts:88]: Function `process` is 80 lines

## Minor Issues (optional)

- [ ] Naming [middleware.ts:12]: `tmp` → `requestId`

## Security Scan (Trivy)

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 1 |
| MEDIUM   | 2 |

## Architecture Analysis (TrueCourse)

- New HIGH violations: 1 (circular dependency)
- New MEDIUM: 1 (layer violation)
- Resolved: 0

## Strengths

- Comprehensive test coverage
- Clear error messages

**Final Verdict:** CHANGES_REQUESTED
**Conditions for approval:** Fix all CRITICAL + HIGH issues, re-run `sdd-review --re-review`
```

## Review Loop

If issues found:
1. Report issues with CRITICAL/IMPORTANT/MINOR tags
2. Implementer fixes issues (can use `sdd-apply --continue` or manual)
3. Re-run `sdd-review --re-review`
4. Review again — MUST re-check ALL previously flagged items
5. Loop until APPROVED or user overrides

**Do not proceed to `sdd-archive` with CRITICAL or unresolved IMPORTANT issues.**

## Multi-Slice Review

If change has multiple slices:
- Review can be slice-by-slice or full change
- Use `--slice <slug>` to review specific slice
- All slices must be approved before `sdd-archive`

## Reviewer Assignment

**Default (no flag):** Current agent acts as reviewer

**With `--reviewer <persona>`:**
- Delegate review to persona: `--reviewer thor` (quality enforcer), `--reviewer heimdall` (security)
- Persona runs `code-quality-reviewer` from their perspective
- Report returns to you for consolidation

## Output

Save report to:
- `openspec/changes/<change-id>/reviews/<timestamp>-review.md`
- Engram: `sdd/<change-id>/review` (if hybrid)

Update task metadata:
- Mark tasks as `review-passed` or `review-issues`
- Record review SHA for traceability

## Exit Conditions

**APPROVED:**
- Zero CRITICAL from ANY source (code quality, Trivy, TrueCourse)
- Zero HIGH TrueCourse violations (if enabled and configured to fail on HIGH)
- Zero Trivy CRITICAL/HIGH CVEs/secrets (if `fail_on_cve/secret` enabled)
- Zero or resolved IMPORTANT issues
- Re-review iterations within `max_iterations`

**CHANGES_REQUESTED:**
- Any CRITICAL present (any source)
- HIGH TrueCourse violations (if fail-on enabled)
- HIGH/CRITICAL Trivy findings (if fail-on enabled)
- IMPORTANT issues remain after 1 fix attempt
- Reviewer judgment: quality unacceptable

**HITL (human):**
- 3 iterations without resolution
- Architectural concerns → `sdd-architecture-review`
- Disagreement on severity/scope

## After Review

**APPROVED:**
- Record `review: approved` in state
- Clear for `pre-merge-verification` → `sdd-archive`

**CHANGES_REQUESTED:**
- Set `review: failed` or leave incomplete
- Implementer fixes
- Re-run `sdd-review --re-review`

## Tool Requirements

**Trivy MCP:**
- MCP server configured at `.configs/mcp/trivy.json`
- Binary: `trivy` (install from https://github.com/aquasecurity/trivy)
- Scans: filesystem for vulnerabilities, secrets, misconfigurations

**TrueCourse:**
- Node.js + npm available (`npx` works)
- First full analysis on `main` required to establish baseline
- Baseline `.truecourse/LATEST.json` committed to main for diff scans
- Skills present: `truecourse-analyze`, `truecourse-list`, `truecourse-fix`, `truecourse-hooks`

## Edge Cases

**Already reviewed:** If `review-approved` flag set → skip (already done) unless `--force`

**Spec non-compliant:** If `sdd-verify` failed → refuse review until spec issues fixed

**Multiple reviewers:** Not supported — one reviewer at a time. For team review, merge individual reports manually.

## Commands

Invocation:
```
/sdd-review
/sdd-review --slice <slug>
/sdd-review --re-review
/sdd-review --reviewer thor
/sdd-review --security --architecture    # enable all external scans
/sdd-review --full-analysis              # TrueCourse full scan
/sdd-review --force                      # skip already-reviewed check
```

## Configuration

`.skillgrid/config.json`:

```json
{
  "review": {
    "required": true,
    "allow_auto_approve": false,
    "max_iterations": 3,
    "require_two_reviewers": false,
    "auto_assign_persona": null,

    "security": {
      "trivy_scan": true,
      "trivy_severity": ["CRITICAL", "HIGH", "MEDIUM"],
      "fail_on_cve": true,
      "fail_on_secret": true
    },

    "architecture": {
      "truecourse_enabled": false,
      "truecourse_mode": "diff",
      "truecourse_llm": false,
      "fail_on_new_violations": true,
      "min_severity_to_fail": "high"
    }
  }
}
```

## See Also

- Stage 1: `spec-compliance-verifier` (`sdd-verify`)
- Final gate: `pre-merge-verification` (combines verify + review + other checks)
- Security scanner: `trivy` MCP integration — https://github.com/aquasecurity/trivy
- Architecture analysis: `truecourse` skills — https://github.com/truecourse-systems/truecourse
- Skills: `truecourse-analyze`, `truecourse-list`, `truecourse-fix`, `truecourse-hooks`
- Superpowers pattern: two-stage code quality review
