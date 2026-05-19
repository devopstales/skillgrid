---
name: spec-compliance-verifier
description: >
  Validates that implementation fully satisfies slice specification.
  Produces traceability matrix: every requirement → code/test evidence.
  Used by sdd-verify command.
license: Apache-2.0
metadata:
  author: aiskillgrid-integration
  version: "1.0"
  mode: reviewer
  triggers:
    - "spec_verification"
    - "sdd-verify_invoke"
---

# Spec Compliance Verification

## Overview

Systematically verifies that the implementation matches the slice specification exactly. Creates traceability matrix, identifies gaps, and produces PASS/FAIL verdict.

**Core principle:** Every requirement must have concrete evidence in code or tests.

## When to Use

Invoked by:
- `sdd-verify` command (primary)
- `sequential-agent-executor` stage 1 review (reuses this skill)
- `pre-merge-verification` final gate (double-check)

Manual usage:
- `/verify-spec --slice <slice-slug>` — verify specific slice
- `/verify-spec --change <change-id>` — verify entire change

## Input

Expects to find:
- Slice spec file: `openspec/changes/<change-id>/specs/<slice-slug>/spec.md`
- Implementation files (from git diff or working tree)
- Test files (test suite output available)
- Optional: task list for task→spec mapping

Artifact store mode aware (engram/openspec/hybrid).

## Verification Process

### Step 1: Parse Specification

Read spec file and extract all requirements:

```markdown
## Acceptance Criteria

- [ ] REQ-001: User can register with email/password
- [ ] REQ-002: System sends verification email on registration
- [ ] REQ-003: Login returns JWT access token
```

Extract structured requirements:
```json
{
  "requirements": [
    {
      "id": "REQ-001",
      "description": "User can register with email/password",
      "type": "functional",
      "complexity": "simple"
    }
  ]
}
```

Also extract:
- In-scope boundary (what slice covers)
- Out-of-scope items (explicit exclusions)
- Non-functional requirements (performance, security)
- UI/UX specifics if present

### Step 2: Gather Evidence

For each requirement, search for evidence:

**Evidence sources:**
1. **Code** — implementation files contain logic fulfilling requirement
2. **Tests** — test file explicitly validates requirement behavior
3. **Configuration** — config keys enable feature
4. **Documentation** — user/developer docs mention feature

**Evidence collection commands:**
```bash
# Search code for requirement patterns
grep -r "register" src/ tests/ --include="*.ts" --include="*.py"

# Check test files
grep -r "email.*password" tests/ --include="*.test.*"

# Verify specific function or endpoint exists
rg "def register" src/ || rg "function register" src/
```

**Record per requirement:**
```yaml
REQ-001:
  evidence:
    - type: code
      location: src/auth/register.ts:15-45
      confidence: high
    - type: test
      location: tests/auth/register.test.ts:8-32
      confidence: high
  status: satisfied

REQ-002:
  evidence:
    - type: missing  (no code or test found)
    confidence: none
  status: missing

REQ-003:
  evidence:
    - type: code
      location: src/auth/login.ts:22-40
      confidence: medium  (partial — token refresh not covered)
    - type: test
      location: tests/auth/login.test.ts:45-67
      confidence: high
  status: partial
```

### Step 3: Traceability Matrix

Produce table:

| Requirement | Evidence Location | Status | Notes |
|-------------|------------------|--------|-------|
| REQ-001 | `src/auth/register.ts:15-45`, `tests/auth/register.test.ts` | ✅ SATISFIED | Full coverage |
| REQ-002 | — | ❌ MISSING | No email service found |
| REQ-003 | `src/auth/login.ts:22-40` | ⚠️ PARTIAL | Test covers happy path only |

### Step 4: Out-of-Scope Detection

Scan implementation for features NOT in spec:
```bash
git diff origin/main...HEAD --name-only | xargs grep -l "feature-flag"
```

Flag extra work:
- New routes not in acceptance criteria
- Unspecified configuration options
- Stretch deliverables not approved

**Out-of-scope verdict:** Does not FAIL verification but must be documented. User may choose to keep or remove extra work.

### Step 5: Spec Gap Analysis

Identify:
- **Missing requirements** — not implemented → FAIL
- **Partial requirements** — implemented but incomplete → CONDITIONAL PASS (with notes)
- **Extra features** — out-of-scope but harmless → WARNING
- **Spec ambiguity** — requirement unclear → HITL decision needed

### Step 6: Verdict

```json
{
  "change_id": "<id>",
  "slice": "<slice-slug>",
  "verdict": "PASS|FAIL|PARTIAL",
  "requirements_met": 5,
  "requirements_total": 8,
  "missing": ["REQ-002", "REQ-007"],
  "partial": ["REQ-003"],
  "out_of_scope": ["admin-dashboard"],
  "evidence_matrix": { /* table above */ },
  "next_recommended": "Implement missing requirements or mark them deferred with HITL approval"
}
```

**PASS:** All requirements satisfied (no missing)
**FAIL:** One or more requirements missing
**PARTIAL:** Implementation present but incomplete (edge cases, tests lacking)

## Output Artifacts

Saved to:
- `openspec/changes/<id>/verification/<slice>-report.md`
- Engram: `sdd/<id>/verification/<slice>` (if hybrid/engram)

Report format:
```markdown
# Spec Verification Report

**Change:** <id>
**Slice:** <slice-slug>
**Verdict:** PASS / FAIL / PARTIAL

## Requirements Traceability

| ID | Description | Evidence | Status |
|----|-------------|----------|--------|
| REQ-001 | ... | `file:line` | ✅ |
| REQ-002 | ... | — | ❌ |

## Missing Requirements
- REQ-002: [description]
- REQ-007: [description]

## Out-of-Scope Additions (for review)
- admin-dashboard (was not in spec)

## Next Steps
- [ ] Address missing requirements
- [ ] Get HITL approval for scope change
- [ ] Re-run verification

**Evidence collected:** {timestamp}
**Commit SHA:** {sha}
```

## Integration with Commands

**`sdd-verify`** invokes this skill:
- Reads all slice specs for change
- Runs compliance check per slice
- Aggregates verdicts
- Returns PASS only if all slices PASS

Exit code: 0 for PASS, 1 for FAIL/PARTIAL

## Error Conditions

**Spec not found:** FAIL with "spec missing" error
**No implementation detected:** FAIL with "no changes detected"
**Ambiguous requirement:** MARK HITL — requires human interpretation
**Conflicting evidence:** Record conflict, ask for clarification

## Performance

- Fast: grep-based search for evidence
- Cached: store evidence between runs to avoid re-scanning unchanged files
- Incremental: re-verify only modified files if baseline already passed

## Configuration

`.skillgrid/config.json`:
```json
{
  "verification": {
    "require_full_coverage": true,
    "allow_partial": false,
    "auto_evidence_collection": true,
    "search_timeout_seconds": 30
  }
}
```

## See Also

- Source: Superpowers `spec-compliance-reviewer` pattern
- Related: `code-quality-reviewer` (stage 2)
- Workflow: `sdd-verify` command
