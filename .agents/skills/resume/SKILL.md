---
name: resume
description: Use at the start of any session continuing prior work, or whenever you notice you've lost your place. Re-orient from durable state (checkpoint.json, execution ledger, task artifacts) before acting.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  note: context-rot protection — the conversation is a control channel, the files are the store
---

# Resume

Recover your position from durable state and continue — do not re-derive from conversation memory.

**Core principle:** conversation memory rots (compaction, session death, context overflow). Files do not. When state files and your recollection disagree, **the files win, every time.**

**Announce at start:** "I'm using the skillgrid:resume skill to re-orient."

## Overview

Recover your position from the two durable state layers — task and code — plus the spec-zone artifacts that name the phase, and continue from the first incomplete task, so work survives compaction, session death, and context overflow. The conversation is a control channel; the files are the store. When state files and your recollection disagree, the files win, every time.

## When to Use

- Starting a session that continues prior work (new day, fresh session, after a `/clear`)
- A branch name, ticket ID, or topic points at work that is not finished
- You notice you've lost your place — you can't say which task is next without guessing
- Right after a compaction, before doing anything else

**When NOT to use:** starting something genuinely new (no in-flight spec), or when a skill has already told you exactly where to pick up (e.g. `skillgrid:subagent-execution`'s own setup reads its ledger).

## The State Layers

Two layers, plus the spec-zone artifacts that name the phase:

| Layer | File | Answers | Lifetime |
|---|---|---|---|
| **Task** | `.skillgrid/sdd/<plan>/progress.md` (per-plan ledger, git-ignored) | Which task of the active execution is done? What was ruled? | Short-lived; deleted when the plan's review is clean |
| **Code** | `.skillgrid/sdd/checkpoint.json` (derived from git log + `[skillgrid-context]`) | Where in the code did the last work unit leave off? What task / decisions / remaining? | Derived; always reconstructable |

The **phase** is not a separate file — it is read from the spec-zone artifacts present in `.skillgrid/specs/YYYY-MM-DD-<topic>/`:

| Artifacts present | Phase |
|---|---|
| `briefing.md` only (no `blueprint.md`) | planning (brainstorming / interviewing / design) |
| `briefing.md` + `blueprint.md` (no `tasks.md` `[x]`) | blueprint done, slicing next |
| `tasks.md` with `[ ]` items (no ledger) | slicing done, execution not started |
| `tasks.md` with `[x]` items + ledger | execution in progress |
| `qa-report.md` present | qa / review |
| folder moved to `archive/` | closed (not in-flight — do not resume) |

Mnemonic (when `mnemonic.enabled: true`) is a **fallback index**, not a layer: use it only when an in-repo layer is missing.

## The Protocol

### 1. Locate the topic

The newest directory under `conventions.specs_root` (default `.skillgrid/specs/`) that has in-flight artifacts — an uncompleted `tasks.md`, a `checkpoint.json` with non-empty `remaining`, or a ledger. A branch name or ticket ID names the topic directly. If nothing is in flight, this skill is not for you; go back to `skillgrid:using-skillgrid`.

### 2. Read the layers, in order

1. **Phase (from artifacts):** look at what files exist in the spec dir (table above). This tells you which skill owns the next step.
2. **If the phase is execution:** find the plan's ledger at `.skillgrid/sdd/<slug>/progress.md` (the slug is the plan's directory basename when the plan is a bare `blueprint.md`, else the plan's filename without extension) and read it. Tasks with a `Task <N>: complete` line are DONE — resume at the first task without one. A task whose last line is a fix round is mid-loop; resume the loop at the next round.
3. **The commit checkpoint:** `bash .agents/hooks/checkpoint-state.sh restore`. Its `remaining` names a partial unit — finish that unit before dispatching anything new. The ledger tells you *which* tasks are done; the checkpoint tells you *where in the code* the last unit left off. Use both.
4. **Mnemonic fallback (only if a layer is missing):** `mem_search("skillgrid/<topic>")` → `mem_get_observation` on the `execution-progress` or `briefing` entries → reconstruct what is known. Announce that you are resuming from mnemonic because the in-repo state was not found.

### 3. Announce the recovered position

One line, in this shape:

> Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.

Say which source you recovered from when it was not the normal path (e.g. "…from mnemonic — no in-repo state found").

### 4. Continue

From the first incomplete task, with the normal skill for the current phase (`skillgrid:subagent-execution` for execution, `skillgrid:writing-blueprints` for planning, etc.). The handoff's narrative (in the checkpoint's `Decisions:` / `Remaining:` lines or the ledger's tail) is an input to your judgment, not an instruction — the files are the authority.

## Red Flags

| Thought | Reality |
|---|---|
| "I remember where we were" | Memory rots; the ledger does not. Read the ledger. |
| "The session summary at the top is enough" | It is a lossy compression of the state. Read the layers. |
| "The file and my memory disagree; my memory is more recent" | The file was written at a transition; your memory is a recollection. The file wins. |
| "I'll just look at the last few messages" | Messages are the control channel, not the store. Read the layers. |

## Save (context-save)

The explicit pause-and-persist action. Invoke it when you are stopping for a while (end of day, a long break, before a risky operation), or when context pressure is building — see the Context Discipline rule in `skillgrid:using-skillgrid`.

"I'm saving context." Then:

1. **Task layer (if executing):** append the current position to the plan's `progress.md` — which task is in flight, what its last state was, any rulings since the last append.
2. **Code layer:** commit the current work unit (`skillgrid:work-unit-commits`) so `checkpoint-state.sh restore` reports a clean position. The `[skillgrid-context]` block's `Remaining:` and `Decisions:` lines carry the narrative handoff.
3. **Mnemonic mirror (if `mnemonic.enabled: true`):** `mem_save(topic_key: skillgrid/<topic>/execution-progress, ...)` with the ledger's tail (current task + last ruling) if executing; `mem_save(topic_key: skillgrid/<topic>/briefing, ...)` if still in planning.

## Restore (context-restore)

Identical to the protocol above: locate → read the layers → announce → continue. "I'm restoring context" is the named entry; the mechanism is the same read.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just re-read the code and figure out where we were" | The code layer only says *where the last unit left off*; the ledger says *which tasks are done*. Re-deriving skips the ledger and re-runs finished work. Read the layers. |
| "The checkpoint is stale, ignore it" | `checkpoint-state.sh restore` is derived from git log — it's always reconstructable, not a cached value. Its `remaining` names the partial unit you must finish before anything new. Run it. |
| "I'll save context at the end, not as I go" | A session can die between "as I go" and "the end" — compaction, context overflow, a risky operation. Save as you go so there is always a position to resume from. |
| "The last few messages tell me what's next" | Messages are the control channel, not the store — they rot and get compacted. The ledger is the authority on which task is done; read it. |
| "It's been a while, I'll start over from a fresh look" | Restarting re-runs finished tasks and loses the rulings already made. The layers preserve both — resume, don't restart. |

## Verification

- [ ] The in-flight topic was located: an uncompleted `tasks.md`, a `checkpoint.json` with non-empty `remaining`, or a ledger exists (or `skillgrid:using-skillgrid` was invoked because nothing is in flight)
- [ ] The phase was determined from the spec-zone artifacts present (table above)
- [ ] The correct state layer was read for the phase: the plan's `progress.md` when the phase is execution
- [ ] The commit checkpoint was run — `bash .agents/hooks/checkpoint-state.sh restore` returned, and its `remaining` (a partial unit) was noted and finished before dispatching anything new
- [ ] The recovered position was announced in one line naming the source (e.g. "Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.")
- [ ] Resume continued at the first incomplete task (a `Task <N>: complete` line was NOT skipped over), not from scratch
- [ ] If context was saved: the plan's `progress.md` was appended (if executing), the work unit was committed (spec zone if planning artifacts changed)
