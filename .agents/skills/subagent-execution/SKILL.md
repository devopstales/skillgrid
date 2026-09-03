---
name: subagent-execution
description: "Execute an implementation plan by dispatching a fresh implementer subagent per task, running a task-scoped spec+quality review after each (not a final pass), closing findings in a bounded fix loop, and keeping a per-plan work directory of briefs, reports, review packages, and a progress ledger that survives context loss. Use when there is an implementation plan to execute and the work should be delegated to fresh subagents with review between tasks, rather than done inline in a single context."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# subagent-execution

Execute a plan by dispatching a fresh implementer subagent per task, a task-scoped review (spec compliance + code quality) after each, and a broad whole-branch review at the end.

**Core principle:** fresh subagent per task + task review (spec + quality) + broad final review = high quality, fast iteration.

## Why subagents

You delegate each task to a fresh subagent with isolated context. By precisely crafting its instructions and context, you keep it focused on its one task and prevent it from inheriting your session's history — anything you paste or print stays resident in *your* context for the rest of the session. Delegate the work, hold the coordination, keep your own context for planning and review.

**Narration:** between tool calls, narrate at most one short line — the ledger and the tool results carry the record.

**Continuous execution:** do not pause to check in with your human partner between tasks. Execute all tasks without stopping. The only reasons to stop are the four named in [When to stop](#when-to-stop-asking), or all tasks complete. "Should I continue?" prompts and progress summaries waste their time — they asked you to execute the plan, so execute it.

**Rulings, not stalls.** A running plan does not wait on a human. Conflicts, ambiguities, plan defects, a cap you would have asked to exceed — decide them. The spec is the binding authority, the plan is its argument, and your judgment settles what neither answers. Record every decision as a ledger ruling: `Ruling: <what you decided> — <why> — <what it costs if wrong>`. A wrong ruling costs rework your partner can see and undo; a parked session costs their whole day and buys nothing.

### When to stop (asking)

Only **four** things stop you and require asking:

1. an irreversible or destructive operation;
2. a security-sensitive action;
3. a side effect outside this worktree that norms say to confirm first (a merge, a push to a shared branch, a publish);
4. a plan so broken that every path forward is a guess.

For those, stop and ask. Everything else, rule and continue.

## When to use

```
have an implementation plan?
├─ no  -> brainstorm / plan first
└─ yes -> tasks mostly independent?
        ├─ no (tightly coupled)  -> plan first or do inline
        └─ yes -> delegate to fresh subagents (this skill)
```

**vs. doing it inline:** SDD wins when the plan has multiple independent tasks, the tasks touch code, or the change is large enough that a fresh, narrow context benefits each implementer. For a single small task where delegation overhead exceeds the work, do it inline.

## The process

Per task:

```
1. Record BASE = git rev-parse HEAD
2. Generate the task brief, dispatch a fresh implementer (references/implementer-prompt.md)
3. Handle its report (DONE | DONE_WITH_CONCERNS | NEEDS_CONTEXT | BLOCKED)
4. Build the review package
5. Dispatch a task reviewer (references/task-reviewer-prompt.md)
6. If findings: bounded fix loop (≤5 rounds) — scoped re-review (references/re-review-prompt.md)
7. Append to the ledger; mark the task complete
After all tasks: broad whole-branch review, one fix wave if needed, then finish.
```

## Setup

Every plan gets its own work directory of short-lived, git-ignored artifacts — task briefs, implementer reports, review packages, and the progress ledger. It lives under `.skillgrid/sdd/<NNN-slug>/` (see `scripts/sdd-workspace`). Run:

```
$(dirname of this skill)/scripts/sdd-workspace <PLAN_FILE>
```

It prints the absolute work directory and ensures it exists (with a self-ignoring `.gitignore` inside, so it never shows in `git status`). Use `$WORKSPACE` for every artifact this skill writes:

- Briefs:        `$WORKSPACE/task-<N>-brief.md`
- Reports:        `$WORKSPACE/task-<N>-report.md`
- Review packages:`$WORKSPACE/review-<base7>..<head7>.diff`
- Ledger:         `$WORKSPACE/progress.md`   ← the single source of truth

The ledger is a markdown file. On start, if it exists, parse it and resume at the first task whose line is not yet `Task <N>: complete`.

**Conversation memory does not survive compaction.** In real sessions, controllers that lost their place have re-dispatched *entire completed task sequences* — the single most expensive failure observed. Track progress in the ledger (the file), not only in your chat memory. The ledger and `git log` are the recovery record; a controller without one re-does work it already did.

## Model selection

Use the least powerful model that can handle the role — and **always specify it explicitly** (an omitted model silently inherits the session's most expensive one, which defeats this):

- **Mechanical implementation** (isolated functions, clear specs, 1–2 files): fast, cheap model. Most implementation is mechanical when the plan is well-specified.
- **Integration and judgment** (multi-file coordination, pattern matching, debugging): a standard model.
- **Architecture / design / the final whole-branch review:** the most capable available model.
- **Reviews:** scale to the diff's size, complexity, and risk. A small mechanical diff doesn't need the most capable model; a subtle concurrency bug or a large refactor does.
- **Fix-loop escalation (rounds 4–5):** one tier above the implementer that got stuck.

**Always specify the model explicitly when dispatching** — an omitted model inherits your session's model, often the most capable and most expensive, which silently defeats this section.

## The task loop

**Batch small same-shape work.** When the plan lists several tasks that each only need one identical or near-identical edit to different files — the same constant, the same import, the same field — do not dispatch one subagent per task. Compose ONE brief listing each file's edit and the exact value, send the whole batch to one cheap implementer, and review the combined diff as a single unit. Reserve the per-task loop for work that needs its own judgment or tests.

### 1. Dispatch the implementer

- Record `BASE = git rev-parse HEAD` *before* dispatching.
- Generate the task brief with `scripts/task-brief <PLAN_FILE> <N>`. This extracts the task's full text into `$WORKSPACE/task-<N>-brief.md` and prints the path — keep the task text in a file, not pasted through your context.
- Compose the dispatch prompt: (1) one line on where this task sits, (2) the **brief path** introduced as "read this first — it is your requirements, verbatim", (3) interfaces / decisions from earlier tasks that the brief cannot know, (4) your resolution of any ambiguity you saw in the brief, (5) the **report path** `$WORKSPACE/task-<N>-report.md` and a one-line contract ("Write your full report there, and end with status + commits + one-line test summary + concerns").
- **A dispatch prompt describes one task, not the session's history.** Do not paste accumulated prior-task summaries — a real dispatch hit 42k chars of which 99% was pasted history. A fresh subagent needs its task, the interfaces it touches, and the global constraints. Nothing else.
- **The dispatch carries the no-subagents rule** (it is in the implementer template): the implementer never spawns subagents — not a helper, and never a reviewer. Review arrives from you, after the report.
- Record the implementer's agent identity (task id) from the dispatch result — the fix-loop resume needs it.
- **Never dispatch multiple implementer subagents in parallel** (conflicts).
- **Hand artifacts over as files.** Anything you paste into the prompt and anything the subagent prints back stays resident in your context for the rest of the session; briefs, reports, and review packages live in `$WORKSPACE`, not in your context.

Template: [references/implementer-prompt.md](references/implementer-prompt.md)

### 2. Handle the report

| Status | Your move |
|---|---|
| **DONE** | Generate the review package (step 3) and dispatch the task reviewer. |
| **DONE_WITH_CONCERNS** | Read the concerns. If they are about correctness or scope, address them before review. If they are observations (e.g. "this file is getting large"), note them and proceed to review. |
| **NEEDS_CONTEXT** | Provide the missing context and re-dispatch. |
| **BLOCKED** | Assess: context gap (supply more context, re-dispatch same model) / reasoning gap (re-dispatch on a bigger model) / task too large (split it) / plan is wrong (rule on the correction, **ledger the ruling**, re-dispatch with the ruling carried in the dispatch). |

**Never** ignore a BLOCKED or NEEDS_CONTEXT and re-dispatch the same model unchanged — if it said it's stuck, something needs to change. If the implementer asks questions before starting or mid-task, answer clearly and completely, provide additional context if needed, and don't rush it into implementation.

### 3. Build the review package

```
scripts/review-package <PLAN_FILE> <BASE> <HEAD> <OUTFILE>
```

The output never enters your own context, and the reviewer sees the commit list, stat summary, and full diff (with context) in one file. Use the BASE you recorded before dispatching the implementer — **never `HEAD~1`**, which silently truncates multi-commit tasks.

### 4. Dispatch the task review

Per-task reviews are task-scoped gates. The broad review happens once, at the final whole-branch review. Never skip the task review, and never accept a report missing either verdict — **spec compliance AND task quality** are both required, and implementer self-review never replaces the task review.

- The reviewer gets three paths — the same brief file, the report file, and the review package — plus the global constraints that bind the task.
- **The global-constraints block is its attention lens.** Copy the binding requirements *verbatim* from the plan's Global Constraints section or the spec: exact values, exact formats, and the stated relationships between components ("same layout as X", "matches Y"). The reviewer's template already carries the process rules (YAGNI, test hygiene, review method) — the constraints block is for what *this plan/spec* demands.
- Do not add open-ended directives like "check all uses" or "run race tests if useful" without a concrete reason.
- Do not ask a reviewer to re-run tests the implementer already ran on the same code — the implementer's report carries the test evidence.
- Do not pre-judge findings — never instruct a reviewer to ignore or not flag a specific issue. If you believe a finding would be a false positive, let the reviewer raise it and adjudicate it in the fix loop. If your dispatch contains "do not flag," "at most Minor," or "the plan chose," stop: you are pre-judging, usually to spare yourself a fix round.
- The task reviewer may report **⚠️ Cannot verify from diff** items — requirements that live in unchanged code or span tasks. These do **not** block the rest of the review, but you must resolve each one yourself before marking the task complete: you hold the plan and cross-task context the reviewer lacks. If you confirm one is a real gap, treat it as a failed spec review and send it into the fix loop with the other findings.

Template: [references/task-reviewer-prompt.md](references/task-reviewer-prompt.md)

### 5. The fix loop

The loop triggers when the review reports **Spec ❌**, or any **Critical** or **Important** finding, or a ⚠️ item you confirmed as a real gap.

Before the loop starts, two routes leave it immediately:

- **Minor findings** do not enter the loop. Record them in the ledger as you go (`Task <N>: minor (deferred): <one-liner>`) and point the **final** whole-branch review at that list so it can triage which must be fixed before merge. A roll-up nobody reads is a silent discard.
- **A finding labeled plan-mandated** — or any finding that conflicts with what the plan's text requires — is yours to rule on: weigh the finding against the plan text, decide with the spec as the binding authority, and ledger the ruling before you act on it. Do not dismiss the finding because the plan mandates it, and do not dispatch a fix that contradicts the plan without a recorded ruling.

Everything else enters the loop. A **fix round = one fix dispatch + one scoped re-review**. Five rounds maximum per task.

**Rounds 1–3 — resume the original implementer.** Send it the open findings verbatim. Its context is intact: it knows the task, the code, and its own choices. If your harness cannot send another message to a live subagent, dispatch a fresh implementer carrying the brief path, the report path, and the findings — the report file is the persistent memory either way.

**Rounds 4–5 — dispatch a fresh implementer on a more capable model** (per Model Selection), with the brief, the report file, the open findings, and: "A prior implementer attempted this task N times; you own it now. Read the report file for what was tried." A loop that survived three resumes usually means the implementer cannot see its own problem — fresh eyes plus a capability bump in one move.

**Every round, either way:** the implementer fixes, **re-runs the tests covering the amended code, appends its fix report to the same report file**, and returns the short contract. Before re-dispatching the reviewer, confirm the fix report contains the covering tests, the command run, and the output; dispatch the re-review once all three are present. Name the covering test files in the fix message — a one-line fix does not need the whole suite.

**The re-review is scoped.** Build the package over `FIX_BASE..HEAD` where `FIX_BASE` is the head the previous review saw, and dispatch [references/re-review-prompt.md](references/re-review-prompt.md) with the findings list, the brief, the report file, and the diff path. The re-reviewer verdicts each finding `ADDRESSED` or `NOT ADDRESSED` and flags **new breakage in the fix diff only**. New Critical/Important breakage joins the open findings. Out-of-scope observations go to the ledger as deferred minors — they never extend the loop.

**After each round,** append to the ledger:
`Task <N>: fix round <R>/5 (<X> addressed, <Y> open — <finding one-liners>; commits <a7>..<b7>)`

**Never fix findings yourself in the controller session** — your context stays clean for coordination, and a controller fix skips the review you just ran.

**The breaker.** When round 5's re-review still leaves findings open, stop dispatching. Adjudicate each open finding yourself — you hold the plan and the cross-task context the reviewer lacks:

- **The reviewer is wrong, or controllable/contestable:** park it — `Task <N>: parked — <finding> — Ruling: <why the code stands>`. The final review sees both sides.
- **Real, but nothing downstream builds on it:** park it the same way, with a ruling that says it is real and deferred.
- **Real and load-bearing** — a later task builds on it, or it exposes a plan defect: rule on the smallest change that unblocks the dependent work, ledger it as `Task <N>: Ruling: <finding> — <what you decided>`, and carry it into the next task's dispatch. Parking a structural failure silently lets every dependent task build on it. Stop only when the defect leaves every path forward a guess.

Adjudicate **only at the cap** — adjudicating earlier to end a loop is pre-judging with a different name. Every adjudication is a ledger entry; a silent discard is forbidden.

### 6. Complete the task

When the review comes back clean — or every open finding is parked with a ruling at the cap — append the completion line:

- `Task <N>: complete (commits <base7>..<head7>, review clean)`
- `Task <N>: complete (commits <base7>..<head7>, <K> parked)` after a tripped breaker

Then mark the task done and move on.

## Final review

The broad whole-branch review gets a package too:

```
scripts/review-package <PLAN_FILE> <MERGE_BASE> <HEAD>
```

Where `MERGE_BASE` is the commit the branch started from, e.g. `git merge-base main HEAD` (for SDD, the commit before `Task 1`'s BASE, or the base branch tip per the plan). Include the printed path in the dispatch. Dispatch on the **most capable** model. The reviewer reads one file — the commit list, stat, and full diff with context — instead of re-deriving the branch diff with git commands.

If the final review returns findings: dispatch ONE fix subagent with the complete findings list (not one per finding — real sessions show that per-finding fixers each rebuild context and re-run suites, and a real session's final-review fix wave cost more than all its tasks combined). Then run one scoped re-review over the same fix range, and adjudicate residuals with the same four rules as the breakpoint: park with rulings, or rule on the load-bearing ones and ledger it. **There is no second fix wave** — residual load-bearing findings surface to your human partner before the finish.

## Finish

1. **Surface the rulings.** Before you delete `$WORKSPACE`, collect every ledger line containing `Ruling:` — pre-flight rulings, parked findings, breaker adjudications, all of them — and present them in order, each with what it costs if wrong. That list is the only place the decisions you made on your human partner's behalf reach them. A ruling that dies with the workspace is a decision made in secret.
2. **Clean up.** Delete the work directory for *this* plan: `rm -rf <this plan's workspace>`. Leave sibling plans' directories alone. The record now lives in `git log` and any persisted notes.
3. **Persist the ledger.** Save the completed ledger to Mnemonic so a resumed session finds it: `mem_save(title: "sdd/<NNN-slug>/apply-progress", topic_key: "sdd/<NNN-slug>/apply-progress", type: "architecture", scope: "project", content: <full ledger markdown>)`. See [../_shared/conventions/mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md).
4. **Hand off.** If the work needs to land somewhere, the `finish` is a decision about branch / PR / merge — do **not** push or merge from this skill unless the user explicitly asked. See [../_shared/conventions/commits.md](../_shared/conventions/commits.md).

## Common rationalizations

| Excuse | Reality |
|---|---|
| "Close enough, one more fix will land it" | The loop has a cap. You adjudicate at the cap, not before. |
| "I'll fix the reviewer's finding myself, dispatching a subagent is overhead" | Controller fixes skip review and pollute your context. Resume the implementer. |
| "The reviewer will just find something new anyway, skip the re-review" | A scoped re-review verifies the specific finding; it cannot wander. Skipping it is how unfixed regressions land. |
| "The plan says this IS the requirement, the reviewer is wrong" | Plan-mandated findings still get an explicit ruling in the ledger. You do not silently override a reviewer. |
| "The implementer spawned its own reviewer — extra assurance" | It's a duplicate seat for the same diff, at full cost, and its verdict counts for nothing in the process. The task review is the gate. |
| "Let me ask my human partner if I should keep going" | A running plan does not wait on a human. Decide, ledger the ruling, keep going — unless one of the four named stops applies. |
| "I'll do task 2 first, it's easier, and come back to task 1" | The task loop is ordered. Reordering tasks breaks the review and the ledger. |

## Example workflow

```
You: I'm using Subagent-Driven Development to execute this plan.

[Setup: worktree verified]
[Read plan file once: docs/plans/feature-plan.md]
[Resolve workspace: scripts/sdd-workspace docs/plans/feature-plan.md — no ledger inside, fresh start]
[Create todos for all tasks]

Task 1: hook-install script

[Run task-brief for Task 1; dispatch implementer with brief + report paths]
Implementer: "Before I begin — should the hook be installed at user or system level?"
You: "User level (~/.config/hooks/)"
Implementer:
  - Implemented the hook-install command
  - Wrote + ran 3 tests: 3/3 passing
  - Self-review: clean
  - Committed
[Run review-package PLAN_FILE BASE HEAD; dispatch task reviewer with the printed path]
Task reviewer: Spec ✅ — all requirements met, nothing extra. Task quality: Approved.
[Ledger: Task 1: complete (commits a1b2c3d..d4e5f6a, review clean)]

Task 2: recovery modes

[Run task-brief for Task 2; dispatch implementer]
Implementer:
  - Implemented verify + repair modes
  - Wrote + ran 5 tests: 5/5 passing
  - Committed
[Run review-package; dispatch task reviewer]
Task reviewer: Spec ❌ — Missing: progress-reporting (spec says "report at 10% intervals").
             Task quality: Approved.
[Fix round 1: resume the implementer with the finding]
Implementer: Added progress-reporting per 10% interval. Re-ran recovery.test.js — 6/6 passing. Fix report appended.
[Run review-package PLAN_FILE FIX_BASE HEAD; dispatch scoped re-review]
Re-reviewer: Missing progress-reporting — ADDRESSED (src/recovery.js:41). New breakage: none.
             Verdict: all findings addressed.
[Ledger: Task 2: fix round 1/5 (1 addressed, 0 open; commits d4e5f6a..b7c8d9e)]
[Ledger: Task 2: complete (commits d4e5f6a..b7c8d9e, review clean)]

[After all tasks: dispatch whole-branch reviewer, most capable model]
Reviewer: No issues.
[Ledger: <plan-slug>: complete. Rulings: <list or "none">]
[Remove $WORKSPACE]

Done. Ready for finishing.
```

## Gotchas

- **`HEAD~1` is not a review window.** A multi-commit task silently loses commits. Use the recorded `BASE`.
- **The task brief must be a file, not a paste.** `scripts/task-brief` extracts it; a pasted brief stays in your context forever.
- **Do not add "do not flag" instructions.** If a finding looks like a false positive, let the reviewer raise it and adjudicate it in the loop.
- **A review that returns "⚠️ Cannot verify from diff" still needs a ruling.** Do not let a ⚠️ block the rest of a clean review — resolve it yourself from your cross-task context and ledger the ruling.
- **A fix round with no re-review is an unreviewed fix.** Every round ends with a scoped re-review before the loop closes.
- **Round 5 with findings still open is a structural failure, not "one more iteration."** The breaker tells you what to do.
- **A worker-spawned reviewer duplicates the task review at full cost.** Its verdict counts for nothing in the process.
- **Mnemonic is the recovery copy for the ledger; the commit chain is the durable record.** `mem_save` the ledger state so a resumed session can re-dispatch tasks correctly.
- **A batch dispatch is ONE review, not N.** Compose one brief listing every file + its change + the exact value, and review the combined diff as a single unit.

## References

- [references/implementer-prompt.md](references/implementer-prompt.md) — implementer dispatch template.
- [references/task-reviewer-prompt.md](references/task-reviewer-prompt.md) — task-scoped reviewer dispatch template (spec + quality + calibration + output).
- [references/re-review-prompt.md](references/re-review-prompt.md) — scoped re-review template (addressee, new breakage, out-of-scope).
- [scripts/sdd-workspace](scripts/sdd-workspace) — resolves + creates the per-plan scratch dir (prints the absolute path).
- [scripts/task-brief](scripts/task-brief) — extracts one task's full text from the plan to `$WORKSPACE/task-<N>-brief.md`.
- [scripts/review-package](scripts/review-package) — writes the commit list + stat + diff with context for a reviewer to read in one call.
- [../isolated-workspace/SKILL.md](../isolated-workspace/SKILL.md) — optional; isolation for the work, when wanted.
