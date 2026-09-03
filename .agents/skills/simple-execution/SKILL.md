---
name: simple-execution
description: >
  Execute SDD (or general) implementation plans inline — one task at a time in
  the current context, with a strict RED/GREEN/TRIANGULATE/REFACTOR cycle when
  the project requires it, marking each task [x] as it completes, and producing
  the per-step evidence that sdd-verify will audit.
  Use when the plan is small, tightly coupled, or below the dispatch threshold —
  not when the workload decision recommends chained or stacked PRs.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# simple-execution

Inline per-task execution. One task at a time, in THIS context. Not delegation.

**Core principle:** read task → RED (if Strict TDD) → GREEN → TRIANGULATE → REFACTOR → run focused test → mark `[x]` → next. Every step produces evidence you can point `sdd-verify` at.

**Violating the letter of the loop is violating the spirit of the loop.** If a task is marked `[x]` without a focused-test line in the Step Evidence, the work is not done.

## When to use

- A single small task (1–2 files, clear spec).
- A tightly coupled group of tasks where a fresh subagent per task adds overhead.
- Workload decision is `single-pr` (no chaining), under the 400-line budget.
- The orchestrator explicitly chose the inline path (see `sdd-apply` Step 5).

**Do NOT use when:**
- The workload decision is `auto-chain`, `chained-PR`, or any shape that needs a work-unit slice per review gate → use `../subagent-execution/SKILL.md` instead.
- The step has ≥ 4 independent tasks → delegate.
- Your context is already 40k+ tokens and each new task will bloat it further.

## The inline task loop

```
FOR each assigned task in steps/<NN-name>/tasks.md (in NN.{i} order):
  1. READ
     ├── Task description
     ├── The matching acceptance.feature scenario (this IS the acceptance criterion)
     ├── The plan's WHAT / decisions for this task (approach constraints)
     └── The target files, confirmed via code index (code_status → code_search → code_read)

  2. WRITE
     ├── (Strict TDD only) RED:  write the failing test for the next behavior FIRST
     ├── (Strict TDD only) GREEN: minimal code to pass
     ├── (Strict TDD only) TRIANGULATE: two independent assertions proving the behavior
     └── (Strict TDD only) REFACTOR: extract, rename, clean — keep green

  3. RUN
     └── The smallest command that proves this task, capture: exit code + key lines

  4. MARK
     └── Change `- [ ]` → `- [x]` in the step's tasks.md IMMEDIATELY (as you go)

  5. RECORD
     └── Append one row to the Step Evidence table:
        | Task | Focused test (cmd + result) | Acceptance scenario → test → result | Runtime | Rollback |
```

Keep each task completable in one sitting. If a task needs more than one sitting, the step decomposition was wrong — flag it, do not silently split.

## Strict TDD

If the change's testing resolution is `strict_tdd: true` (resolved by `sdd-apply` Step 4) AND a test runner exists, the STRICT TDD module **overrides** the task loop for the WRITE step:

[`references/strict-tdd.md`](references/strict-tdd.md)

Load it only when Strict TDD is active. Its RED → GREEN → TRIANGULATE → REFACTOR cycle and the TDD Cycle Evidence table are the gate. If Strict TDD **is not active**, do NOT read it — zero TDD instructions should be loaded, and the plain loop above is the whole task.

**No silent fallback**: if you resolved Strict TDD as active and then hit an infrastructure failure, mark the row FAILED and return `partial` — do NOT quietly drop to Standard Mode for the same task.

## Marking progress

- Update each assigned step's `steps/<NN-name>/tasks.md` **as you go**, not in one batch at the end.
- The `tasks.md` file is the recovery record — a session that compacts mid-task recovers from the `[x]` state, not from chat memory.
- Never rewrite a completed line. Append new lines. Preserve prior batches' state verbatim (per `sdd-apply` Step 8 Merge Protocol).

## Reporting evidence

After each task complete, capture these rows — they feed the Step Evidence table in the apply-progress envelope:

| Field | Purpose |
|---|---|
| Task id (e.g. `01.3`) | Traceable |
| Focused test command + exit code | Minimal proof |
| Acceptance scenario name → test that ran it → pass/fail | Maps to `acceptance.feature` |
| Runtime harness (cmd or scenario + result, or `N/A` + reason) | Real integration path |
| Rollback boundary | Reversal scope (files/behavior) |

`sdd-verify` reads these directly. A missing row for a completed task is a partial apply.

## Rationalizations to reject

| Excuse | Reality |
|---|---|
| "Let me batch 3 tasks and mark them all at once" | Batch marking = no recovery record. Mark as you go. |
| "The GREEN already proves behavior; TRIANGULATE is overkill" | One assertion passing on one path proves less. Two independent assertions prove the behavior is real. |
| "The test runner timed out — I'll note it and move on" | A timeout in the proof run is not evidence. Fix the infra, re-run, or mark the task failed. |
| "This task is a simple one-liner, TDD is overkill" | One-liners are where the accidental regressions land. Red first. |
| "The task said do X; I did X-Y" | Doing X-Y is a deviation. Note it and re-run the proof. |
| "Let me skip the code-index check, I know the file path" | A task that cites a file it has not read is a task with a hole. Confirm via `code_search` + `code_read`. |
| "I'll run the full suite to be safe" | The focused test is the proof. Run the full suite at the step boundary, not per task — per-task full runs cost time and bury the specific signal. |
| "I'll do task 5 first, it's easier" | The task loop is ordered. Reordering breaks the ledger and the evidence chain. |

## Integration with SDD

- This skill is one of **two execution routes** `sdd-apply` dispatches to. The other is `subagent-execution` (for larger, chained, or independent work).
- Its output — per-task `[x]` marks in `tasks.md` + the Step Evidence rows + the Mnemonic `sdd/<NNN-slug>/apply-progress` upsert — is what `sdd-verify` judges.
- It does NOT dispatch subagents. If the workload decision says dispatch, `sdd-apply` routes to `subagent-execution` instead.
- Persistence is hybrid: filesystem `tasks.md` + Mnemonic `sdd/<NNN-slug>/apply-progress`. Both must move together.

## References

- [`references/strict-tdd.md`](references/strict-tdd.md) — the Strict TDD module (RED → GREEN → TRIANGULATE → REFACTOR, TDD Cycle Evidence table, assertion-quality rules, test-layer selection). Load only when `strict_tdd` is active.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — the dispatcher that routes here (Step 5).
- [`../subagent-execution/SKILL.md`](../subagent-execution/SKILL.md) — the alternate route when the workload decision / step shape requires dispatch.
- [`../tdd/SKILL.md`](../tdd/SKILL.md) — the general TDD discipline (applies when Strict TDD is not active).
- [`../verification/SKILL.md`](../verification/SKILL.md) — before marking any task `[x]`, the evidence gate: fresh test run + output in the current message.
- [`../review-reception/SKILL.md`](../review-reception/SKILL.md) — how to receive findings if a review pass surfaces them (from `sdd-verify` or `judgment-day`).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session close, recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, step numbering, artifact paths.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — commit contract for the apply commit.
