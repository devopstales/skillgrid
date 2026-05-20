---
name: security-review
description: >
  OWASP-oriented security review on change diff for sdd-review and heimdall persona.
  Orchestrates vulnerability-scanner checklists; does not duplicate Trivy CVE data.
license: Apache-2.0
metadata:
  author: aiskillgrid
  version: "1.0"
  triggers:
    - sdd-review_invoke
    - security_sensitive
    - heimdall security-review
---

# Security Review

## Purpose

Human-style security assessment on **files changed in this change**. Complements automated scans (Trivy, `vulnerability-scanner` script).

**Do not** repeat CVE tables from Trivy — reference `trivy-report.json` and focus on logic, authZ, and design flaws.

## When to use

- `sdd-review` Stage C when `security_sensitive` or `--security`
- Persona **`heimdall`** → `security-review` per `sdd-persona-delegation.md`
- Optional deep pass: `/sdd-security` (future command)

## Inputs

- Change name and diff file list (`git diff --name-only` vs main)
- `openspec/changes/{change}/spec.md` — security requirements
- Prior scans: `.agents/reviews/{change}/trivy-report.json`, `vulnerability-scan.json` (if present)

## Process

### 1. Load checklists

Read:

- `skills/vulnerability-scanner/SKILL.md` — methodology and OWASP 2025 framing
- `skills/vulnerability-scanner/checklists.md` — audit checklists (auth, API, data)

### 2. Threat context (brief)

Answer in 4 bullets:

1. Assets at risk
2. Trust boundaries crossed in this diff
3. Likely attack vectors
4. Highest business impact

### 3. Diff-focused review

For each changed file in scope:

| Check | Look for |
| --- | --- |
| Access control | Missing authZ on new routes/handlers |
| Injection | String concat in queries/shell/HTML |
| Secrets | Keys, tokens, passwords in code or config |
| Crypto | Weak algorithms, hardcoded IVs, `verify=False` |
| Session/auth | Fail-open on errors, missing logout invalidation |
| Supply chain | New dependencies without justification |
| Logging | PII/secrets in logs |

Use `vulnerability-scanner` principles (sections 8–10) for pattern grep guidance.

### 4. Map spec security requirements

If spec lists security scenarios, trace each to code + test evidence (same rigor as `sdd-verify` matrix). Flag UNTESTED security scenarios as CRITICAL.

### 5. Persona report (heimdall)

When dispatched as persona, write:

`.skillgrid/tasks/research/{change}/heimdall-security-review.md`

Include `findings_severity` and `hitl_required` per `sdd-persona-delegation.md`.

## Severity

| Level | Examples |
| --- | --- |
| **CRITICAL** | Auth bypass, SQLi, hardcoded prod secret, mass data exposure |
| **IMPORTANT** | Missing rate limit, weak session timeout, missing input validation on one path |
| **MINOR** | Defense-in-depth gaps, logging improvements |

## Output format

```markdown
## Security Review

**Change:** {change}
**Sensitive:** yes/no
**Scope:** N files

### Threat context
…

### Findings

#### CRITICAL
- [ ] …

#### IMPORTANT
- [ ] …

### Spec security requirements
| Requirement | Evidence | Status |

### Scan cross-check
- Trivy: {summary or N/A}
- vulnerability-scanner: {summary or N/A}

**Recommendation:** PASS / CHANGES_REQUESTED
```

## Rules

- CRITICAL findings block APPROVED until fixed or explicit risk acceptance (HITL).
- Do not approve if spec security scenarios are UNTESTED.
- Reference file:line for every finding.

## See also

- `trivy-security`, `vulnerability-scanner`, `code-quality-reviewer`
- `sdd-review`, `sdd-verify` (security requirement matrix)
