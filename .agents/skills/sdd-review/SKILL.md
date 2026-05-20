---
name: sdd-review
description: >
  Stage 2 quality gate: security scans, architecture drift, and code health after sdd-verify PASS.
  Trigger: Orchestrator runs /sdd-review after spec compliance; blocks sdd-archive until APPROVED.
license: Apache-2.0
metadata:
  author: aiskillgrid
  version: "1.0"
---

## Purpose

Orchestrate **Stage 2** review after `/sdd-verify` PASS. You do **not** re-check spec compliance — that is Stage 1.

**Pipeline:** `sdd-apply` → `sdd-verify` → **`sdd-review`** → `sdd-archive`

Read `skills/_shared/sdd-phase-common.md` and `skills/_shared/sdd-return-envelope.md` for persistence and return format.

## Norse persona invocations (coordinator)

Per `skills/_shared/sdd-persona-delegation.md`:

| Required | Persona | Capability | When |
| --- | --- | --- | --- |
| no | thor | implementation-enforcement | `--reviewer thor` or default quality pass |
| yes* | heimdall | security-review | `security_sensitive` tag or `--security` |
| no | frigg | ux-clarity | user-facing change |
| no | loki | assumption-stress-test | conflicting scan vs manual findings |

\*Required when `.skillgrid/state/{change}/security_sensitive` exists or verify-report marks `security-sensitive: true`.

## Preflight (mandatory)

```bash
.skillgrid/scripts/sdd-gate.sh review --change {change-name}
```

On non-zero exit: return `status: blocked`, `next_recommended: "/sdd-verify {change-name}"`.

Also:

1. Tasks in `tasks.md` complete (or `--force`).
2. `git status --porcelain` empty (commit first).
3. Optional: `vdd-roast` before stages below — read `skills/vdd-roast/SKILL.md`; fix HIGH before continuing.

## Configuration

Read `.skillgrid/config.json` → `review.security`, `review.architecture`, `review.max_iterations`.

## Review stages

Run in order; skip a stage only when config/flags disable it.

```
Stage A:   trivy-security (MCP) — CVE, secrets, misconfig
Stage A.5: vulnerability-scanner — fallback when Trivy unavailable or --no-trivy
Stage B:   truecourse-review — if review.architecture.truecourse_enabled (wraps analyze + list)
Stage C:   security-review + code-quality-reviewer on changed files
Stage D:   Consolidate → verdict + artifacts
```

### Stage A — Trivy

**Trigger:** `review.security.trivy_scan` or `--security`

**Skill:** `skills/trivy-security/SKILL.md`

**Artifacts:** `.agents/reviews/{change}/trivy-report.json`

### Stage A.5 — Vulnerability scanner (fallback)

**Trigger:** Trivy skipped/failed OR `review.security.fallback_scan` (default true) OR `--pattern-scan`

**Skill:** `skills/vulnerability-scanner/SKILL.md`

```bash
python .agents/skills/vulnerability-scanner/scripts/security_scan.py . --scan-type all
```

**Artifacts:** `.agents/reviews/{change}/vulnerability-scan.json`

### Stage B — TrueCourse

**Trigger:** `review.architecture.truecourse_enabled` or `--architecture`

**Skill:** `skills/truecourse-review/SKILL.md`

**Run:**

```bash
.skillgrid/scripts/run-truecourse-review.sh --change {change-name} --no-llm
```

Use `--llm` only when `review.architecture.truecourse_llm` is true and the user approved token cost.

**Artifacts:** `.agents/reviews/{change}/truecourse-analyze.txt`, `truecourse-violations.txt`, `truecourse-summary.json`

**Interactive detail:** `truecourse-analyze`, `truecourse-list`, `truecourse-fix`, `truecourse-hooks` — https://github.com/truecourse-ai/truecourse

### Stage C — Code quality + security review

1. **`security-review`** when sensitive or `--security` — `skills/security-review/SKILL.md` (invokes `vulnerability-scanner` checklists + diff scope).
2. **`code-quality-reviewer`** always on changed files — `skills/code-quality-reviewer/SKILL.md`.
3. **`--re-review`:** only re-check prior report open items + files touched since last review.

**Scope:** `git diff --name-only origin/main...HEAD` (or merge-base with main); honor `--slice`.

### Stage D — Consolidation

Merge A / A.5 / B / C / persona reports into one report.

**Severity (verdict):**

| Source | Maps to |
| --- | --- |
| Trivy CRITICAL/HIGH (when `fail_on_cve` / `fail_on_secret`) | CRITICAL → CHANGES_REQUESTED |
| vulnerability-scanner critical/high | CRITICAL |
| TrueCourse new high/critical (when fail_on_new_violations) | CRITICAL |
| code-quality CRITICAL | CRITICAL |
| code-quality IMPORTANT | CHANGES_REQUESTED if any remain |
| All clear | APPROVED |

**Persist:**

- `openspec/changes/{change}/reviews/{timestamp}-review.md`
- `.skillgrid/state/{change}/review_status` → `approved` | `changes_requested`
- Engram (hybrid): `mem_save` topic `sdd/{change}/review`
- On APPROVED:

```bash
.skillgrid/scripts/checkpoint-record.sh --change {change} --name review-pass --trigger review-pass --phase review --evidence "review APPROVED"
```

## Flags

| Flag | Effect |
| --- | --- |
| `--security` | Force Stages A + security-review |
| `--no-security` | Skip Trivy and security-review |
| `--pattern-scan` | Force Stage A.5 |
| `--architecture` | Force TrueCourse |
| `--slice <slug>` | Limit diff scope |
| `--re-review` | Prior findings only |
| `--reviewer <persona>` | Delegate (thor, heimdall) |
| `--force` | Skip already-reviewed guard |

Full command examples: `.agents/workflows/sdd-review.md`

## Rules

- Do **not** approve with open CRITICAL from any stage (when fail-on config enabled).
- Do **not** fix code — report only; orchestrator routes fixes to `sdd-apply`.
- Set `review_status` file whenever verdict is final.
- **Return:** standard SDD envelope; `status: completed` if APPROVED, `failed` if CHANGES_REQUESTED.

## See also

- Stage 1: `sdd-verify`, `spec-compliance-verifier`
- Archive: `pre-merge-verification`, `sdd-archive`
- Workflow detail: `.agents/workflows/sdd-review.md`
