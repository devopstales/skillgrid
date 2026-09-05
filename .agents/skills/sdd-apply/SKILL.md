---
name: sdd-apply
description: Execute unblocked SDD steps from change.md + tasks.md + acceptance.feature — sequential by default, parallel only when Depends clear; mark [x], bump State, hand off to verify. Use when the user gate approves Implement, or when resuming phase apply.
disable-model-invocation: true
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Apply

Stage owner (v4). Only phase that writes production code. Consume specs — do not invent requirements. Progress is cumulative across batches.

## Hard Rules

- Execute **unblocked** steps only. Default **sequential** (NN order). Parallel only when every `Depends on:` predecessor is done (tasks `[x]` and, once verified, Verdict PASS/WARNINGS).
- If `change.md` lists **`Prototype:`**, read that path before implementing — reuse spike learnings; do not treat prototype as production.
- Mark `- [ ]` → `- [x]` in change-level `tasks.md` **as you go**; bump `## State` (`phase: apply`).
- Leave `### Verification` as `PENDING` — `sdd-verify` owns verdicts.
- Honor every `Run:` / `Expected:` line. Threat `[RED]` before production change.
- Hybrid: disk `tasks.md` + Mnemonic `sdd/<NNN-slug>/apply-progress` and upsert `…/tasks`.
- No `steps/` tree. Edit only assigned / allowed roots.

## Workflow

```
[ ] 1. Guard + load context
[ ] 2. Workload / isolation
[ ] 3. Route execution
[ ] 4. Mark [x] + State + evidence
[ ] 5. Persist + hand off to verify
```

### 1. Guard + load context

From `## State` / orchestrator:

- `blocked` / unsatisfied `Depends on:` → STOP, return blocked.
- `done` → Next: `sdd-verify`.
- Else: only assigned pending tasks.

Read: `change.md`, `tasks.md` (`## NN-<name>`), `acceptance.feature` (`@step-NN`), prior `apply-progress` (MERGE — never overwrite). Code ladder before edit. Apply `rules.apply` from config.

### 2. Workload / isolation

- Review workload High / chained / unresolved `ask-on-risk` → STOP until delivery path resolved (`auto-chain` | `exception-ok` | explicit `size:exception`).
- Prefer **`isolated-workspace`** for non-trivial branches.
- Commits: **`work-unit-commits`** / [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — checkpoint when green. **Per step always**, even under `single-pr` (one PR ≠ one commit).

### 3. Route execution (do not freestyle inline)

For each unblocked assigned step:

| Route | When | Skill |
|---|---|---|
| Subagent | auto-chain / ≥4 independent tasks / large context | **`subagent-execution`** (+ `dispatching-parallel-agents` when 2+ independent) |
| Inline | small / tightly coupled / single-pr under budget | **`simple-execution`** |
| TDD | `rules.apply.tdd` / testing-capabilities say so | **`tdd`** (+ strict-tdd module when active) |
| Bug mid-apply | unexpected failure | **`debugging`** before patching |

Pass Strict TDD flag and `Run:`/`Expected:` lines into the chosen route. Match existing code patterns; note deviations.

### 4. Mark [x] + State + evidence

Under assigned `## NN-<name>` → `### Tasks`, flip checkboxes as completed. Bump `## State`. Record Step Evidence (focused test, `@step-NN` coverage, runtime harness or N/A+reason, rollback boundary). If Strict TDD active, also TDD Cycle Evidence (RED→GREEN→TRIANGULATE→REFACTOR). Do not invent Verification PASS.

### 5. Persist + hand off

1. Filesystem `tasks.md` already updated.
2. Mnemonic session: upsert cumulative `apply-progress` (merged) + full `tasks` observation.
3. Return envelope → **Next: sdd-verify**. Do not self-launch archive or invent remediation cycles beyond the assigned batch.

```markdown
## Implementation Progress
**Change**: {NNN-slug} · **Mode**: Strict TDD | Standard
**Status**: success | partial | blocked
**Completed / Remaining**: …
**Step Evidence**: …
**Deviations / Issues**: …
**Workload boundary**: …
**Next**: sdd-verify
```

## Gotchas

- Crossing an unfinished `Depends on:` silently breaks verify's evidence chain — return `blocked`.
- Upserting `apply-progress` without reading prior progress drops earlier batches.
- `mem_search` previews are not acceptance criteria — `mem_get_observation(id)`.
- Prototype path is a learning aid, not a license to ship spike code unmarked.
- Do not write Verification verdicts here.

## References

- [`../simple-execution/SKILL.md`](../simple-execution/SKILL.md) · [`../subagent-execution/SKILL.md`](../subagent-execution/SKILL.md)
- [`../tdd/SKILL.md`](../tdd/SKILL.md) · [`../debugging/SKILL.md`](../debugging/SKILL.md)
- [`../isolated-workspace/SKILL.md`](../isolated-workspace/SKILL.md) · [`../dispatching-parallel-agents/SKILL.md`](../dispatching-parallel-agents/SKILL.md)
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) · [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) · [`../sdd-verify/SKILL.md`](../sdd-verify/SKILL.md)
