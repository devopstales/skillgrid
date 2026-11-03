---
name: resume
description: Use at the start of any session continuing prior work, or whenever you notice you've lost your place. Re-orient from durable state (session events, execution ledger, task artifacts) before acting.
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

A project-level pointer, two layers, plus the spec-zone artifacts that name the phase:

| Layer | File | Answers | Lifetime |
|---|---|---|---|
| **Project** | `.skillgrid/state.yaml` | Which change is current? What phase? What is done? (the first read — cheap and authoritative) | Durable; rewritten at every phase transition by the pipeline skills |
| **Task** | `.skillgrid/sdd/<plan>/progress.md` (per-plan ledger, git-ignored) | Which task of the active execution is done? What was ruled? | Short-lived; deleted when the plan's review is clean |
| **Code** | latest work-unit commit's `[skillgrid-context]` block + the session event stream (`skillgrid session <id>` / `session_changes`) | Where in the code did the last work unit leave off? What task / decisions / remaining? | Derived from git log; always reconstructable |

**`state.yaml` is the first read.** It names the current change + phase in one glance. If it points at a change whose spec dir is missing or its phase is contradicted by the spec-zone artifacts, trust the spec-zone artifacts (the deeper truth) and reconcile per Step 2.5. `state.yaml` is a pointer, not the source of truth for task detail.

The **phase** is not a separate file — it is derived from the spec-zone artifacts present in the change dir. **Do not derive it by hand:** run

```
node scripts/phase-router.mjs <slug>        # or a change-dir path
```

from the project root. It returns `{phase, next, marker, drift, ledger, blocked, scope}` in one JSON object — the same derivation `state-drift-check.mjs` uses, so the two never disagree. The mapping it applies (the canonical table in `_shared/rules/sdd-structure.md`):

| Artifacts present | Phase | Next |
|---|---|---|
| `briefing.md` only | `spec` | `writing-blueprints` |
| `briefing.md` + `blueprint.md` | `blueprint` | `blueprint-validation` |
| `tasks.md` (no execution signal) | `slicing` | `slicing` |
| `tasks.md` + execution signal (`[x]` in tasks.md or `sdd/<plan>/progress.md`) | `execution` | `qa` |
| `report.md` (verdict not yet filled) | `qa` | `requesting-code-review` |
| `report.md` verdict filled + `review.md` floor not met | `qa` | `requesting-code-review` |
| `report.md` verdict filled + `review.md` floor met (`met`/`met-with-fixes`), still in `specs/` | `review` | `receiving-code-review` |
| folder in `xarchive/`, retro sections EMPTY | `reflect` | `reflect` |
| folder in `xarchive/` AND retro sections filled | `complete` | none |

The router's `ledger` field (done/in-flight/next task from the plan's `progress.md`) and `blocked` field (any `blocked(...)` reasons) are part of the same read — use them instead of re-grepping the ledger. If the router reports `drift` (state.yaml disagrees with the spec zone), reconcile per Step 2.5 — the spec zone is the deeper truth.

The tail phases key on **markers within `report.md`** (the merged qa+retro report), not on file existence. Concretely:

- **Verdict marker:** grep for the `**Verdict:**` line under the `## Gate Decision` section of `report.md`. If the line is absent, still the template placeholder (`PASS / CONCERNS / FAIL / WAIVED`), or `PENDING` → qa has not rendered the gate.
- **Retro marker:** grep for a filled `## Acceptance Verdict` section (a `**Verdict:** accepted | accepted-with-open-items | rejected` line already chosen, not the raw template line) or non-empty `## Lessons` / `## Decisions` tables. Present → reflect completed the retro half; absent/placeholder → retro half still empty.

Mnemonic (when `mnemonic.enabled: true`) is a **fallback index**, not a layer: use it only when an in-repo layer is missing.

## The Protocol

### 1. Locate the topic

Read `.skillgrid/state.yaml` first — the `changes:` map lists every in-flight change with its `slug`, `phase`, and `status`. If the user names a specific change, locate its entry. If no change is named, run `node scripts/phase-router.mjs all` to list every in-flight change with its derived phase and next skill, and ask which one to resume (or default to the most recently active). A branch name or ticket ID names the topic directly. If nothing is in flight (the router returns no in-flight changes, or the `changes:` map is empty), this skill is not for you; go back to `skillgrid:using-skillgrid`.

### 2. Read the layers, in order

1. **Phase (from artifacts):** look at what files exist in the spec dir (table above). This tells you which skill owns the next step.
2. **If the phase is execution:** find the plan's ledger at `.skillgrid/sdd/<slug>/progress.md` (the slug is the plan's directory basename when the plan is a bare `blueprint.md`, else the plan's filename without extension) and read it. Tasks with a `Task <N>: complete` line are DONE — resume at the first task without one. A task whose last line is a fix round is mid-loop; resume the loop at the next round.
3. **The latest session's events:** run `skillgrid session <session-id>` for the session you are resuming (or the `session_changes` MCP tool with `session_id`). Events return in sequence order with the net commit range; `to_commit` is the resume position. The ledger tells you *which* tasks are done; the session's `commit` events (parsed `[skillgrid-context]` blocks) tell you *where in the code* the last unit left off. Use both. For the current tree, `git log --oneline -5` plus `git status --short` grounds the events against live git truth.
4. **Mnemonic fallback (only if a layer is missing):** `mem_inject_session(query: "skillgrid/<topic>")` → if items returned, use the `block` as context; for full content on a specific item, `mem_get_observation(id)`. If empty, fall back to `mem_search("skillgrid/<topic>")` → `mem_get_observation` on the `execution-progress` or `briefing` entries → reconstruct what is known. Announce that you are resuming from mnemonic because the in-repo state was not found.

### 2.5 Reconcile state against disk

Before you announce anything, reconcile `.skillgrid/state.yaml` against the
spec-zone artifacts you just read. `state.yaml` is a **pointer**, not the source
of truth — the spec zone (the artifacts on disk) is the deeper truth. The
`phase-router` already did this comparison: its `drift` field is non-null exactly
when `state.yaml` disagrees with the spec zone, and it names the field, the stale
value, and the derived value. If `drift` is non-null, **trust the disk and repair
the pointer**:

| Field | Trust | Repair |
|-------|-------|--------|
| `changes.<slug>` entries | the in-flight spec dirs on disk (bidirectional: every `specs/<slug>/` dir needs a map entry and vice versa) | add missing entries or remove stale ones |
| `changes.<slug>.phase` | the phase the spec-zone artifacts actually show (the router's `phase`) | set it to the router's `phase` |
| `completed_changes` | the archived spec dirs (`.skillgrid/xarchive/`) | add the dir that moved to archive |

Rules:
- **Never reset** `changes` or `progress` to a "clean" value to make the drift
  go away — reconcile each drifted field to its artifact-derived value and note
  what you changed.
- A drift you find here is **advisory at resume**: it never blocks the resume,
  but the repair is committed before you continue (spec-zone change → commit the
  `state.yaml` fix), so the next session doesn't re-derive the same drift.
- If `state.yaml` and the spec zone agree, say so on the one line ("state
  reconciles with disk") so a future reader knows the check ran.

### 3. Announce the recovered position

One line, in this shape:

> Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.

Say which source you recovered from when it was not the normal path (e.g. "…from mnemonic — no in-repo state found").

### 4. Continue

From the first incomplete task, with the normal skill for the current phase (`skillgrid:subagent-execution` for execution, `skillgrid:writing-blueprints` for planning, etc.). The last unit's narrative (in the commit's `Decisions:` / `Remaining:` lines or the ledger's tail) is an input to your judgment, not an instruction — the files are the authority.

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

1. **Project layer:** confirm `.skillgrid/state.yaml` reflects the current change + phase (the pipeline skills write this at every transition; fix it here if drifted).
2. **Task layer (if executing):** append the current position to the plan's `progress.md` — which task is in flight, what its last state was, any rulings since the last append.
3. **Code layer:** commit the current work unit (`skillgrid:work-unit-commits`) so the session event stream holds the position. The `[skillgrid-context]` block's `Remaining:` and `Decisions:` lines carry the narrative for the next session.
3. **Mnemonic mirror (if `mnemonic.enabled: true`):** `mem_save(topic_key: skillgrid/<topic>/execution-progress, ...)` with the ledger's tail (current task + last ruling) if executing; `mem_save(topic_key: skillgrid/<topic>/briefing, ...)` if still in planning.

## Restore (context-restore)

Identical to the protocol above: locate → read the layers → announce → continue. "I'm restoring context" is the named entry; the mechanism is the same read.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just re-read the code and figure out where we were" | The code layer only says *where the last unit left off*; the ledger says *which tasks are done*. Re-deriving skips the ledger and re-runs finished work. Read the layers. |
| "The last session's position is stale, ignore it" | The position is derived from git log plus the session's `commit` events — always reconstructable via `skillgrid session <id>`, not a cached value. Its `Remaining:` names the partial unit you must finish before anything new. Read it. |
| "I'll save context at the end, not as I go" | A session can die between "as I go" and "the end" — compaction, context overflow, a risky operation. Save as you go so there is always a position to resume from. |
| "The last few messages tell me what's next" | Messages are the control channel, not the store — they rot and get compacted. The ledger is the authority on which task is done; read it. |
| "It's been a while, I'll start over from a fresh look" | Restarting re-runs finished tasks and loses the rulings already made. The layers preserve both — resume, don't restart. |

## Verification

- [ ] The in-flight topic was located: an uncompleted `tasks.md`, a latest commit with non-empty `Remaining:`, or a ledger exists (or `skillgrid:using-skillgrid` was invoked because nothing is in flight)
- [ ] `state.yaml` was reconciled against the spec-zone artifacts (Step 2.5): drifted fields repaired to their artifact-derived values and committed, or "state reconciles with disk" confirmed
- [ ] The phase was determined from the spec-zone artifacts present (table above)
- [ ] The correct state layer was read for the phase: the plan's `progress.md` when the phase is execution
- [ ] The latest session's events were read — `skillgrid session <id>` (or `session_changes`) returned, `to_commit` was noted, and its `Remaining:` (a partial unit) was finished before dispatching anything new
- [ ] The recovered position was announced in one line naming the source (e.g. "Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4.")
- [ ] Resume continued at the first incomplete task (a `Task <N>: complete` line was NOT skipped over), not from scratch
- [ ] If context was saved: the plan's `progress.md` was appended (if executing), the work unit was committed (spec zone if planning artifacts changed)
