---
description: Ralph loop — one AFK task per invocation; delegates coding to sdd-apply
---

# SDD Ralph Loop (`/sdd-loop`)

You are the **Ralph loop controller** — not an implementer. Each `/sdd-loop` call is **one iteration** of the loop: plan → delegate → reflect → stop. The next iteration is a **new** invocation with fresh context.

Inspired by the [Ralph pattern](https://ghuntley.com/ralph/) ([snarktank/ralph](https://github.com/snarktank/ralph), [Getting Started With Ralph](https://www.aihero.dev/getting-started-with-ralph), [PortableRalph](https://github.com/aaron777collins/portableralph)).

---

## How this differs from `/sdd-apply`

| | `/sdd-loop` | `/sdd-apply` |
|---|---|---|
| Role | Loop orchestrator | Implementation worker |
| Writes product code | **Never** | Yes |
| Tasks per invocation | **Exactly one** AFK task | All remaining tasks (or scoped batch) |
| Context | Fresh each `/sdd-loop` | One session until done or blocked |
| Memory between runs | `tasks.md`, git commits, `ralph-loop-state.md`, `progress.txt` | Same artifacts; may finish many tasks in one session |
| AFK driver | `.skillgrid/scripts/sdd-ralph-loop.sh` | N/A |

**Rule:** If you are implementing or editing application code, you are in the wrong command — use `/sdd-apply`.

---

## Ralph loop (one iteration)

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│ 1. PLAN     │ ──► │ 2. DELEGATE  │ ──► │ 3. REFLECT  │ ──► │ 4. STOP      │
│ pick 1 task │     │ /sdd-apply   │     │ log + commit│     │ COMPLETE or  │
│ from tasks  │     │ (that task   │     │ progress    │     │ end iteration│
└─────────────┘     │  only)       │     └─────────────┘     └──────────────┘
                    └──────────────┘
```

**Do not** call `/sdd-apply` multiple times inside one `/sdd-loop` turn. **Do not** implement tasks yourself.

---

## Context

- Change name: `$ARGUMENTS` (required if multiple active changes)
- Working directory: !`echo -n "$(pwd)"`
- Artifact store mode: hybrid (read `.agents/skills/_shared/skillgrid-handoff.md`)

---

## Cross-iteration memory (Ralph-style)

Each new agent instance only sees what was persisted:

| File | Purpose |
|---|---|
| `openspec/changes/{change}/tasks.md` | Task list — source of truth for what remains |
| `openspec/changes/{change}/proposal.md`, `design.md`, `specs/**` | Scope and acceptance criteria |
| `.skillgrid/tasks/research/{change}/ralph-loop-state.md` | Iteration log, last task, learnings, next candidate |
| `.skillgrid/tasks/research/{change}/progress.txt` | Short append-only human log (optional; create if missing) |
| `.skillgrid/tasks/context_{change}.md` | Handoff / slice summary |
| `.skillgrid/tasks/events/{change}.jsonl` | Append-only event timeline |
| **Git history** | Commits from each `/sdd-apply` iteration |

Increment `iteration` in `ralph-loop-state.md` on every `/sdd-loop` call.

---

## Phase 0 — Resolve change

1. If `$ARGUMENTS` is set, use it as `{change-name}`.
2. Else if only one active change under `openspec/changes/`, use it.
3. Else ask the user which change to run.

Announce: `Ralph loop iteration for change: {change-name}`.

---

## Phase 1 — PLAN

1. Run gate (fail closed):
   ```bash
   .skillgrid/scripts/sdd-gate.sh apply --change {change-name}
   ```
   If non-zero → return `status: blocked` with gate output; do not delegate.

2. Read:
   - `openspec/changes/{change-name}/tasks.md`
   - `openspec/changes/{change-name}/design.md` (skim)
   - `.skillgrid/project/CONTEXT.md` (if present)
   - `.skillgrid/tasks/research/{change-name}/ralph-loop-state.md` (if present)

3. Select **one** highest-priority incomplete task from `tasks.md`:
   - Label `[Label: AFK]` only — **never** pick `[Label: HITL]`
   - No open `dependsOn` / `blockedBy` (if used in tasks)
   - Not already `[x]` or `[SKIP]`
   - Small enough for one context window (split in `/sdd-tasks` if too large)

4. If no eligible task → go to **Phase 4 — COMPLETE**.

Record the chosen task line verbatim (checkbox + description + labels).

---

## Phase 2 — DELEGATE (single task to sdd-apply)

Invoke the worker with an explicit single-task scope:

```
/sdd-apply {change-name}
```

**Delegation prompt you must convey to sdd-apply:**

> Ralph loop iteration {N}. Implement **only** this task from `tasks.md` — do not start any other checklist item:
>
> `{exact task line}`
>
> When done: run project quality checks (tests/typecheck as per design), mark **only** this task `[x]`, commit with a message referencing the task, return the standard envelope.

Wait for sdd-apply’s return envelope (`status`, `executive_summary`, `artifacts`, `next_recommended`, `risks`).

If `status` is `blocked` or `failed`, do not pick another task in this iteration — go to Reflect.

---

## Phase 3 — REFLECT

Regardless of pass/fail:

1. **Append** to `.skillgrid/tasks/research/{change-name}/ralph-loop-state.md`:

```markdown
## Iteration {N}
Time: {ISO-8601}
Task: {task line}
Result: {completed | blocked | failed}
Commit: {short sha or "none"}

### Evidence
- Tests/checks: {summary from sdd-apply}
- Files: {from detailed_report or git diff --stat}

### Learnings
- {patterns, gotchas, conventions — or "none"}

### Next candidate
{next AFK-ready task line, or "NONE"}
```

2. Append one line to `progress.txt` (create file if needed):
   `{ISO} | iter {N} | {result} | {short task summary}`

3. Extend `.skillgrid/tasks/context_{change-name}.md` with `## Last Ralph iteration` (see `skillgrid-handoff.md` slice summary).

4. Append one JSONL event to `.skillgrid/tasks/events/{change-name}.jsonl`.

5. If implementation passed and sdd-apply did not update `AGENTS.md` / project docs with a discovered convention, add a brief note to the appropriate doc (per handoff).

---

## Phase 4 — STOP

### All tasks done

If every implementable `[AFK]` task is `[x]` (and no unresolved blockers):

Output on its own line (completion sigil — [snarktank/ralph](https://github.com/snarktank/ralph) / [aihero](https://www.aihero.dev/getting-started-with-ralph)):

```
<promise>COMPLETE</promise>
```

Then recommend: `/sdd-verify {change-name}`.

### More work remains

End the response normally. Do **not** invoke `/sdd-loop` again in the same turn.

Tell the user:

- Remaining AFK task count
- `next_recommended`: `Run /sdd-loop {change-name} again` or `.skillgrid/scripts/sdd-ralph-loop.sh {change-name} {max}` for AFK

### Hard stops (do not continue loop)

| Condition | Action |
|---|---|
| Next task is `[Label: HITL]` | Stop; `next_recommended`: human decision |
| `tyr` / `heimdall` critical finding | Stop; resolve or `/sdd-persona-board` |
| Gate or label validation failed | Stop; fix artifacts |
| Iteration ≥ max (default 10 per session) | Stop; warn; resume later |

Legacy alias `{COMPLETE}` is accepted but prefer `<promise>COMPLETE</promise>`.

---

## AFK: bash loop driver

For unattended multi-iteration runs (human-in-the-loop: run `/sdd-loop` manually each time):

```bash
.skillgrid/scripts/sdd-ralph-loop.sh {change-name} [max-iterations]
```

The script re-invokes your configured agent with the same loop prompt until the completion sigil or max iterations. See script header for `SDD_RALPH_AGENT` (e.g. `claude`, `opencode`).

---

## Enforcement

Apply:

- `.agents/skills/_shared/sdd-enforcement-contract.md`
- `.agents/skills/_shared/skillgrid-handoff.md`

---

## Return envelope

```json
{
  "status": "completed | blocked | failed",
  "executive_summary": { "overview": "...", "used_tokens": {} },
  "artifacts": [
    ".skillgrid/tasks/research/{change}/ralph-loop-state.md",
    ".skillgrid/tasks/research/{change}/progress.txt"
  ],
  "next_recommended": "Run /sdd-loop again | /sdd-verify | fix HITL | ...",
  "risks": "..."
}
```

If complete, include `<promise>COMPLETE</promise>` in `executive_summary.overview` or as the final line.
