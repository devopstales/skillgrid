---
name: resume
description: Use at the start of any session continuing prior work, or whenever you notice you've lost your place. Re-orient from durable state (state.md, execution ledger, commit checkpoint) before acting.
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

Recover your position from the three durable state layers — phase, task, code — and continue from the first incomplete task, so work survives compaction, session death, and context overflow. The conversation is a control channel; the files are the store. When state files and your recollection disagree, the files win, every time.

## When to Use

- Starting a session that continues prior work (new day, fresh session, after a `/clear`)
- A branch name, ticket ID, or topic points at work that is not finished
- You notice you've lost your place — you can't say which task is next without guessing
- Right after a compaction, before doing anything else

**When NOT to use:** starting something genuinely new (no in-flight spec), or when a skill has already told you exactly where to pick up (e.g. `skillgrid:subagent-execution`'s own setup reads its ledger).

## The State Layers

Three layers, each answering a question the others cannot:

| Layer | File | Answers | Lifetime |
|---|---|---|---|
| **Phase** | `.skillgrid/specs/YYYY-MM-DD-<topic>/state.md` | What phase is this topic in? What's done, what's open, what was decided? | Committed; survives `git clean -fdx` and fresh clones |
| **Task** | `.skillgrid/sdd/<plan>/progress.md` (per-plan ledger, git-ignored) | Which task of the active execution is done? What was ruled? | Short-lived; deleted when the plan's review is clean |
| **Code** | `.skillgrid/sdd/checkpoint.json` (derived from git log) | Where in the code did the last work unit leave off? | Derived; always reconstructable |

Mnemonic (when `mnemonic.enabled: true`) is a **fallback index**, not a layer: use it only when an in-repo layer is missing.

## The Protocol

### 1. Locate the topic

The newest directory under `conventions.specs_root` (default `.skillgrid/specs/`) that has in-flight artifacts — a `state.md`, an uncompleted `tasks.md`, or a ledger. A branch name or ticket ID names the topic directly. If nothing is in flight, this skill is not for you; go back to `skillgrid:using-skillgrid`.

### 2. Read the layers, in order

1. **`state.md`** — the topic's phase position. If it names an active plan, note it.
2. **If the phase is execution:** find the plan's ledger at `.skillgrid/sdd/<slug>/progress.md` (the slug is the plan's directory basename when the plan is a bare `blueprint.md`, else the plan's filename without extension) and read it. Tasks with a `Task <N>: complete` line are DONE — resume at the first task without one. A task whose last line is a fix round is mid-loop; resume the loop at the next round.
3. **The commit checkpoint:** `bash .agents/hooks/checkpoint-state.sh restore`. Its `remaining` names a partial unit — finish that unit before dispatching anything new. The ledger tells you *which* tasks are done; the checkpoint tells you *where in the code* the last unit left off. Use both.
4. **Mnemonic fallback (only if a layer is missing):** `mem_search("skillgrid/<topic>")` → `mem_get_observation` on the `state` or `execution-progress` entries → reconstruct what is known. Announce that you are resuming from mnemonic because the in-repo state was not found — and write the recovered `state.md` back before continuing, so the in-repo truth and the store agree again.

### 3. Announce the recovered position

One line, in this shape:

> Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.

Say which source you recovered from when it was not the normal path (e.g. "…from mnemonic — no state.md found; rewriting it now").

### 4. Continue

From the first incomplete task, with the normal skill for the current phase (`skillgrid:subagent-execution` for execution, `skillgrid:writing-blueprints` for planning, etc.). The handoff's narrative paragraph (the "what I was thinking" note) is an input to your judgment, not an instruction — the files are the authority.

## Red Flags

| Thought | Reality |
|---|---|
| "I remember where we were" | Memory rots; the ledger does not. Read the ledger. |
| "The session summary at the top is enough" | It is a lossy compression of the state file. Read the state file. |
| "The state file and my memory disagree; my memory is more recent" | The file was written at a phase transition; your memory is a recollection. The file wins. |
| "I'll just look at the last few messages" | Messages are the control channel, not the store. Read the layers. |

## Save (context-save)

The explicit pause-and-persist action. Invoke it when you are stopping for a while (end of day, a long break, before a risky operation), or when context pressure is building — see the Context Discipline rule in `skillgrid:using-skillgrid`.

"I'm saving context." Then:

1. **Phase layer:** refresh the topic's `state.md` — current phase, done, open, decisions. If the phase is execution, note the active plan.
2. **Task layer (if executing):** append the current position to the plan's `progress.md` — which task is in flight, what its last state was, any rulings since the last append.
3. **Code layer:** commit the current work unit (`skillgrid:work-unit-commits`) so `checkpoint-state.sh restore` reports a clean position.
4. **Narrative handoff:** one short paragraph in `state.md` — what you were thinking, what you would do next, what to watch for. This is the one thing the structured layers do not capture.
5. **Mnemonic mirror (if `mnemonic.enabled: true`):** `mem_save(topic_key: skillgrid/<topic>/state, ...)` with the same phase/done/open/decisions content (upsert — same `topic_key` updates, never duplicates); if executing, also `mem_save(topic_key: skillgrid/<topic>/execution-progress, ...)` with the ledger's tail (current task + last ruling).
6. Commit `state.md` (spec zone).

## Restore (context-restore)

Identical to the protocol above: locate → read the layers → announce → continue. "I'm restoring context" is the named entry; the mechanism is the same three-layer read.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just re-read the code and figure out where we were" | The code layer only says *where the last unit left off*; the ledger says *which tasks are done*. Re-deriving skips the ledger and re-runs finished work. Read the layers. |
| "The checkpoint is stale, ignore it" | `checkpoint-state.sh restore` is derived from git log — it's always reconstructable, not a cached value. Its `remaining` names the partial unit you must finish before anything new. Run it. |
| "I'll save context at the end, not as I go" | A session can die between "as I go" and "the end" — compaction, context overflow, a risky operation. Save as you go so there is always a position to resume from. |
| "The last few messages tell me what's next" | Messages are the control channel, not the store — they rot and get compacted. The ledger is the authority on which task is done; read it. |
| "It's been a while, I'll start over from a fresh look" | Restarting re-runs finished tasks and loses the rulings already made. The phase layer and ledger preserve both — resume, don't restart. |

## Verification

- [ ] The in-flight topic was located: a `state.md`, an uncompleted `tasks.md`, or a ledger exists (or `skillgrid:using-skillgrid` was invoked because nothing is in flight)
- [ ] The correct state layer was read for the phase: `state.md` always; the plan's `progress.md` when the phase is execution
- [ ] The commit checkpoint was run — `bash .agents/hooks/checkpoint-state.sh restore` returned, and its `remaining` (a partial unit) was noted and finished before dispatching anything new
- [ ] The recovered position was announced in one line naming the source (e.g. "Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.")
- [ ] Resume continued at the first incomplete task (a `Task <N>: complete` line was NOT skipped over), not from scratch
- [ ] If context was saved: `state.md` was refreshed, the plan's `progress.md` was appended (if executing), the work unit was committed, and `state.md` was committed (spec zone)
