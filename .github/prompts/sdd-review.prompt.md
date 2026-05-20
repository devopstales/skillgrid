---
name: sdd-review
id: sdd-review
category: Workflow
description: Perform code quality and security review on completed implementation
agent: odin
subtask: true
---

You are an SDD sub-agent. Read **`.agents/skills/sdd-review/SKILL.md`** FIRST, then follow it exactly.

**Workflow detail (stages, flags, report templates):** `.agents/workflows/sdd-review.md`

**VDD ROAST (optional, before stages):** `.agents/skills/vdd-roast/SKILL.md` — fix HIGH findings before Stage A.

## Context

- Working directory: `!echo -n "$(pwd)"`
- Active change: `.skillgrid/state/active_change` or conversation / branch
- Artifact store mode: read from `.skillgrid/config.json` → `artifactStore.mode` (default hybrid)

## Preflight (mandatory)

```bash
.skillgrid/scripts/sdd-gate.sh review --change {change-name}
```

On failure: `status: blocked`, `next_recommended: "/sdd-verify {change-name}"`.

## Skills (by stage)

| Stage | Skill |
| --- | --- |
| A | `trivy-security` |
| A.5 | `vulnerability-scanner` |
| B | `truecourse-review` (`run-truecourse-review.sh`) |
| C | `security-review`, `code-quality-reviewer` |

## Flags

`--security`, `--no-security`, `--pattern-scan`, `--architecture`, `--slice`, `--re-review`, `--reviewer`, `--force` — see workflow file.

## Return

Standard envelope: `.agents/skills/_shared/sdd-return-envelope.md`
