---
name: truecourse-review
description: >
  TrueCourse architecture diff review for sdd-review Stage B. Runs npx truecourse analyze --diff
  and list --diff; writes review artifacts. Upstream: https://github.com/truecourse-ai/truecourse
license: Apache-2.0
metadata:
  author: aiskillgrid
  version: "1.0"
  triggers:
    - sdd-review Stage B
    - architecture review
---

# TrueCourse Review (SDD Stage B)

Orchestrates [TrueCourse](https://github.com/truecourse-ai/truecourse) for **architecture + code intelligence** during `/sdd-review`.

Detects circular deps, layer violations, dead modules, security anti-patterns, and 1,200+ deterministic rules (JS/TS/Python).

## When to use

- `sdd-review` when `review.architecture.truecourse_enabled` is true or `--architecture` flag
- After `/sdd-verify` PASS, before `code-quality-reviewer`
- Optional standalone: user asks for architecture diff on current branch

## Prerequisites

| Requirement | Notes |
| --- | --- |
| Node.js | >= 20 |
| `npx` | Always use `npx -y truecourse` (avoids install prompt hang) |
| Baseline | `.truecourse/LATEST.json` committed on `main` (see Setup) |
| LLM rules | Optional; need `claude` on PATH. Default `--no-llm` in CI/review |

## Setup (once per repo)

On **`main`** after merge:

```bash
npx -y truecourse analyze --no-llm
git add .truecourse/LATEST.json .truecourse/config.json
git commit -m "chore: add truecourse baseline"
```

Do **not** commit `LATEST.json` from feature branches (merge conflicts).

Fresh clones/worktrees inherit baseline via git. For in-flight work use **`--diff`** only.

Optional: `.truecourseignore` at repo root (same syntax as `.gitignore`) for generated paths.

## Execution (preferred — script)

```bash
.skillgrid/scripts/run-truecourse-review.sh --change {change-id} --no-llm
```

With LLM rules (user must approve token cost):

```bash
.skillgrid/scripts/run-truecourse-review.sh --change {change-id} --llm
```

Full repo scan (slow, use `--full-analysis` flag only):

```bash
.skillgrid/scripts/run-truecourse-review.sh --change {change-id} --full --no-llm
```

**Artifacts:**

- `.agents/reviews/{change}/truecourse-analyze.txt`
- `.agents/reviews/{change}/truecourse-violations.txt`
- `.agents/reviews/{change}/truecourse-summary.json`

## Manual steps (if script unavailable)

1. **Analyze diff:** `npx -y truecourse analyze --diff --no-llm` (600s+ timeout)
2. **List violations:** `npx -y truecourse list --diff --limit 50`
3. Tee output to artifact paths above

Read `skills/truecourse-analyze/SKILL.md` and `skills/truecourse-list/SKILL.md` for interactive flows.

## Verdict mapping (sdd-review)

Read `.skillgrid/config.json` → `review.architecture`:

| Config | Effect |
| --- | --- |
| `fail_on_new_violations: true` | New **high** or **critical** in diff → CHANGES_REQUESTED |
| `min_severity_to_fail: "high"` | medium → IMPORTANT, not blocking |
| `truecourse_llm: true` | Pass `--llm` only after user approves cost |

Parse analyze summary line: `Summary: N new issues, M resolved`. If **stale baseline** warning appears, WARN and suggest full analyze on main.

## Report block (paste into review markdown)

```markdown
## Architecture Analysis (TrueCourse)

**Mode:** diff | **Baseline:** present | missing
**Command:** npx -y truecourse analyze --diff --no-llm

| Metric | Value |
| --- | --- |
| New issues | N |
| Resolved | M |
| Blocking (high+) | N |

Top findings: (from truecourse-violations.txt, first page)
```

## Fix loop

For auto-fixable violations: `skills/truecourse-fix/SKILL.md` → `npx -y truecourse list --diff`, apply `Fix:` blocks, re-run diff analyze.

## Pre-commit (optional)

`skills/truecourse-hooks/SKILL.md` — `npx -y truecourse hooks install` (blocks critical/high new violations; slower commits).

## See also

- CLI reference: https://github.com/truecourse-ai/truecourse#cli-commands
- `sdd-review`, `sdd-architecture-review` (human deepening after friction)
- `docs/19-external-tools.md` (TrueCourse section)
