---
name: reflect
description: "Use when a change has been shipped (folder moved to .skillgrid/archive/) and you need to close the cycle with a sourced retrospective and the final-state archive report. The terminal phase after ship."
license: MIT
disable-model-invocation: true
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# Reflect

**Announce at start:** "I'm using the skillgrid:reflect skill to close this change."

You are the **TERMINAL** phase — the close of the SDD cycle. `ship` already integrated the work and moved the change folder to `.skillgrid/archive/YYYY-MM-DD-<topic>/`. You do exactly three things:

1. **Complete `report.md`** — the merged report already exists (qa wrote the QA half in the spec folder, ship moved the folder); you complete its retro half IN PLACE in the archive: final-state facts, gate results, sourced learnings (Decisions / Lessons / Patterns / Surprises), acceptance verdict, move evidence, lineage.
2. **Environment retro** — invoke the Skill tool with `skillgrid:environment-retro` to close the loop on the *environment* (checks, navigation, steering no-ops, tool economy), appending its `## Environment Retro` section to `report.md`.
3. **Persist to Mnemonic** — the report + any high-value learning.
4. **Close the session** — `mem_session_summary` + `mem_session_end`.

You are the **completion checkpoint**: you are the last chance to honestly say what worked, what did not, and whether the change is accepted. You are also the **lineage endpoint** — the report is the record a future reader consults to learn what shipped and when.

## Overview

The terminal SDD phase. After `ship` has integrated the work and moved the change folder to `.skillgrid/archive/`, `reflect` closes the cycle: it completes the merged `report.md` in place in the archive (qa's QA half is already there; you write the retro half from `## Final-State Facts` onward — integration evidence + retrospective + final-state facts + lineage), persists to Mnemonic, and closes the session. It is the last honest checkpoint of what shipped and the lineage endpoint a future reader consults to learn what landed and when.

## When to Use

- After completing a change that has been shipped — its folder moved to `.skillgrid/archive/YYYY-MM-DD-<topic>/` by `ship`.
- Before closing a work unit / the session — you need the sourced report and the session close.

**When NOT to use:** mid-task before work is done — `reflect` captures the final state at close, not in-flight state. It also does not run on a change that never shipped (the Ship Gate blocks it).

## What You Receive

- **Change folder:** now at `.skillgrid/archive/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `report.md` with the QA half complete, `review.md` — the committed code-review audit record).
- **Ship context** — from `ship`'s Return Envelope: base branch, chain strategy, integration test evidence, PR/merge outcome, worktree state, gate results, `diff -r` readback. (If ship's envelope is unavailable, recover from `mem_search("skillgrid/YYYY-MM-DD-<topic>/ship")` or git history.)
- **`report.md` → `## Gate Decision` verdict** (the QA half, written by qa) + any human override.
- **`review.md` → `## Verdict` floor + `## Independence`** — the review gate and the independence grade, to cite in the lineage (which findings were caught pre-merge, and on which grade).
- **All planning artifacts** (briefing, blueprint, design/threat-matrix, tasks) — the source material for sourced learnings.
- **Fast-track waiver class** (`trivial` / `small`) if present — drives the light variant.

## Phase Order

After `ship`. Terminal. The full chain is in `_shared/rules/sdd-structure.md`.

`prev-phase: [ship]` · `next-phase: []` (terminal) · `artifact: report`.

## Final-State Authority

The report is the terminal record of the cycle. It describes the state of the change **at close**, not at earlier points. The QA half of `report.md` (written by qa) and the execution ledger are **intermediate snapshots** — true for the moment written, but work routinely continues after they are persisted (verify warnings fixed in later commits, blocked tasks completed, test counts change). A snapshot's "done" stays true — work does not un-complete — but its "pending / blocked / open gap" claims are only valid for the instant they were written. **Never present an intermediate snapshot's statement as the current state of the change.**

When sources disagree about a fact, rank them — most authoritative first:

1. **Current repository / filesystem state at close** — what is actually on disk and in git now. The strongest evidence of what shipped.
2. **Ship context** — the integration + move record (base branch, merge/PR commit, `diff -r` readback). The most recent close-out account.
3. **The persisted `tasks.md`** — completion visibility.
4. **The QA half of `report.md` and the execution ledger** — intermediate snapshots. Lowest rank: valid history of what was true at their time, never evidence of final state.

Reporting rules that follow:

- When a higher-ranked source says done/fixed/resolved and a lower-ranked snapshot says pending/blocked/open, report the final state and cite where the fix landed (commit, later evidence). Do **not** echo the stale claim.
- When a contradiction cannot be ranked (a claim no higher-ranked source or repo evidence corroborates), record it explicitly: both statements, their sources, and when each was written. Never resolve it silently.
- Attribute snapshot-derived claims to their source and time ("per `report.md` (QA half) at verification time …"). Do not restate them in bare present tense as current facts.
- Carry final numbers (test counts, warnings, open issues) from the highest-ranked source that covers them.
- Never merge distinct defects or failures into a single causal story. A cause is recorded as confirmed only with evidence; otherwise record the failure as undiagnosed.

This hierarchy governs how the report **reports** facts. It does not weaken the gates: a `FAIL` verdict in `report.md` → `## Gate Decision` / an unresolved CRITICAL still blocks reflect (you did not ship), and the gates below keep their own authority.

## Gates (all must pass before ANY write)

### Ship Gate (hard)

- **Folder not in `archive/`** → **`blocked`** (reason `not-shipped`). Nothing shipped, nothing to reflect.
- **Ship context missing** (no Return Envelope, no mnemonic, no git evidence of a clean move) → **`blocked`** (reason `ship-incomplete`). The folder move was not verified.
- **Ship context present** + `diff -r` empty → proceed.

### QA Gate (hard)

- **`report.md` → `## Gate Decision` verdict `FAIL`** or an unresolved **CRITICAL** with no human override → **`blocked`** (reason `qa-failed`). You cannot close work the quality gate refused.
- **`PASS`** / **`WAIVED`** / **`CONCERNS`** (human-owned) → proceed.

### Verdict Gate (advisory — never blocks)

The acceptance verdict you produce is **recorded, not enforced**. A `rejected` verdict does **not** block the archive — it is recorded in the report and surfaced to the human, who decides whether a follow-up change is needed. Never block the cycle on the verdict alone.

If a hard gate fails, **STOP and return `blocked`** with the failing gate named. Do not write the report.

## What to Do

### Step 1: Load All Artifacts (recovery + lineage)

1. Read the moved folder `.skillgrid/archive/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `report.md` — QA half complete, retro half empty, review artifacts, findings).
2. Recover the ship context: from `ship`'s Return Envelope if available, else `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/ship")` → `mem_get_observation(id)`, else git history (`git log --oneline` on the base branch for the merge commit).
3. Recover the Mnemonic copies (previews are not enough — always fetch full content). **Record the observation ID of every artifact you read — they go into the report for lineage.**
   - `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/blueprint")` → `mem_get_observation(id)`
   - `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/tasks")` → `mem_get_observation(id)`
   - (and any `findings` / ADR observations the change produced)
4. Read `config.yaml` if present — `rules.reflect` bind this phase.

> If `mnemonic.enabled` is `false`, the in-repo artifacts are the sole source — skip the Mnemonic recovery (degrade explicitly).

### Step 2: Pass the Gates (before ANY write)

Confirm, in order: **Ship Gate** → **QA Gate** → **Verdict Gate** (advisory). If a hard gate fails, STOP and return `blocked`.

### Step 3: Complete `report.md` (retro half)

Complete the existing file IN PLACE in the moved folder (`.skillgrid/archive/YYYY-MM-DD-<topic>/report.md`) — do not recreate it. The QA half (through `## Gate Decision` + `## Human Override`) is already written by qa; fill the retro half from `## Final-State Facts` onward. Use the merged template [../../verification/qa/templates/report.md](../../verification/qa/templates/report.md) (see [templates/report.md](templates/report.md) for the pointer).

**Final-state facts** — what shipped, to which base, at which commit/PR (from ship context).

**Sourced learnings — every entry MUST point at evidence** (a file:line, a commit, a ticket ID, or a scenario). A finding with no source is a guess, not a learning. Four categories:

- **Decisions** — architecture / design / tooling choices made, with the tradeoff and why.
- **Lessons** — what would be done differently next time (root cause, not symptom).
- **Patterns** — reusable approaches or conventions established (name them so a future change can reuse them).
- **Surprises** — non-obvious gotchas, edge cases, or behaviors discovered (the highest-signal category for future sessions).

**Acceptance verdict** — `accepted` / `accepted-with-open-items` / `rejected`:

- Ground it in the goal stated in `briefing.md` and the evidence in `report.md`'s QA half (Goal-Backward Verification + Traceability Matrix).
- `accepted` — goal met, no open items.
- `accepted-with-open-items` — goal met, but name the open items (they become next-change candidates).
- `rejected` — goal not met (a scenario unmet, a regression, the goal-backward check failed). Record why.

**Prior-change follow-through** — scan this change's own open items and the most recent archived changes' `report.md` open items: which did this change address? Which are still open? Name them.

**Lineage** — the observation IDs of every artifact read (the lineage endpoint), **plus** the `review.md` facts: the review verdict floor, each axis's independence grade, and the Critical/Important findings caught pre-merge (so a future reader can see what review caught vs. what escaped).

**Lift durable findings.** From the sourced learnings above (Surprises, Patterns, Decisions that constrain *future* work), append the ones that outlive this change to `.skillgrid/artifacts/06-research-findings.md` — the durable, cross-change distillation. A finding that is only true for this one change stays in `report.md` alone. This is the terminal catch: anything `research`/`prototype`/`sketch` missed while the sources were fresh still lands here at close.

**Finding triage — process bug vs one-off.** For each Lesson and Surprise, classify it before it closes:

- **Process bug** — the finding names a *reusable* failure class that would likely recur in other changes (a navigation trap, a steering-file no-op, a verification gap, a convention that was silently violated). It is a *process* fix: name the governing skill or `_shared` rule it belongs to, and record it as a **follow-up ticket** (an Open Item with the path to the process file to edit). Do **not** hand-tune only this change's artifacts to absorb it.
- **One-off** — the finding is specific to this change's context and would not generalize. It stays in `report.md` alone.

The test mirrors the distillation heuristic: *would the same lesson strengthen a future change, or only this one?* If the same argument would apply elsewhere, it is a process bug and must route back to the process — a process bug recorded only as a one-off Lesson is a finding that never reaches the file that would have prevented it next time.

### Step 3.5: Update `state.yaml`

Set `pipeline.current_phase: reflect` (the terminal phase). `ship` already appended the topic to `progress.completed_changes` and cleared `current_change`; leave those as-is. If `current_change` was not cleared (e.g. ship degraded), clear it here so the pipeline reads idle after close.

### Step 3.6: Environment Retro (sub-phase)

Invoke the Skill tool with `skillgrid:environment-retro`. It sweeps the six environment categories (navigation, automated checks, coding standards, steering-file no-ops, tool economy, information access) and appends a `## Environment Retro` section to the same `report.md`, every row sourced and naming where each fix lands. It **proposes** — real builds (a new hook, linter rule, CI job, steering-file edit) come back as `mem_save` findings and offered follow-up tickets, not inline code edits. A clean session legitimately produces no findings; do not force one. Run it after the retro half is written and before session close, so the environment loop closes in the same session that ran.

### Step 4: Persist to Mnemonic + Close the Session (you own this)

`reflect` is the **only** phase that closes the session. Follow [`../../_shared/rules/mnemonic-memory.md`](../../_shared/rules/mnemonic-memory.md).

```
mem_save(
  title:      "report — YYYY-MM-DD-<topic>",
  topic_key:  "skillgrid/YYYY-MM-DD-<topic>/report",
  type:       "learning",
  scope:      "project",
  session_id: "{sid}",
  content:    "{report markdown: final-state facts, gates, sourced Decisions/Lessons/Patterns/Surprises, acceptance verdict, open items, follow-through, move evidence, lineage}"
)
# Save any single high-value learning as its own observation too (type: decision|pattern|bugfix|discovery).

# THEN close the session:
mem_session_summary(session_id: "{sid}", summary: <structure below>)
mem_session_end(session_id: "{sid}", summary: "one-line outcome")
```

The `mem_session_summary` body uses the **six-section structure** (Goal / Instructions / Discoveries / Accomplished / Next Steps / Relevant Files) defined in [`../../_shared/rules/mnemonic-memory.md`](../../_shared/rules/mnemonic-memory.md) — all six must be filled.

> If `mnemonic.enabled` is `false`, the in-repo `report.md` is the sole record — skip the Mnemonic saves and session close (degrade explicitly, never fail silently).

### Step 5: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` + `mem_session_summary` + `mem_session_end` (Step 4) *before* this text.

```markdown
## Change Closed

**Change**: YYYY-MM-DD-<topic>
**Verdict**: accepted | accepted-with-open-items | rejected
**Location**: `.skillgrid/archive/YYYY-MM-DD-<topic>/report.md` · Mnemonic `skillgrid/YYYY-MM-DD-<topic>/report`
**Status**: success | blocked

### Gates
| Gate | Result |
|------|--------|
| Ship gate | ✅ success + diff -r empty |
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Verdict gate (advisory) | {accepted \| accepted-with-open-items \| rejected} — recorded, not enforced |

### Learnings (sourced)
- **Decision**: {choice + tradeoff} — {source}
- **Lesson**: {do differently + root cause} — {source}
- **Pattern**: {reusable approach} — {source}
- **Surprise**: {gotcha/edge case} — {source}

### Open Items (→ next change)
- {item + path to resolution} (or "None")

### Follow-Through
- {prior open items this change addressed / still open} (or "None")

**Lineage (observation IDs)**: briefing {id} · blueprint {id} · tasks {id} · ship {id} · qa {id} · …
**Mnemonic**: session `{sid}` closed · observations `{ids}`
**Open questions**: {list, or "None"}
**Risks**: {list, or "None"}
**Next**: none — cycle complete
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up.

## Rules

- You are **read-only over the archive** — you do not modify shipped code or the folder layout. You complete exactly one file in place (`report.md` — its retro half, from `## Final-State Facts` onward) in the moved folder and persist to Mnemonic.
- **Every learning is sourced.** A finding with no file:line / commit / ticket / scenario is a guess — either cite it or mark it undiagnosed.
- **The acceptance verdict is advisory.** `rejected` is recorded and surfaced, never a hard block. The human decides follow-up.
- **The report reflects FINAL state** per the Final-State Authority hierarchy — never echo stale `report.md` (QA half) / ledger "pending" claims as current facts; record unrankable contradictions explicitly.
- **You own session close** — `mem_session_summary` + `mem_session_end` run here and only here. Sub-agents in earlier phases emit `## Key Learnings` only and do not close the session.
- **Record every observation ID you read** into the report — the report is the lineage endpoint.
- If a hard gate (Ship or QA) fails, STOP and return `blocked` — do not write the report.
- No external binaries. Mnemonic (`mem_*`) and the project's files are the only tools.
- Apply any `rules.reflect` from `config.yaml`.
- Return envelope per Step 5 — final action is text, not a tool call.

## Fast-Track Variant

For a `trivial` / `small` change (waiver recorded): skip the full `report.md` document. Instead:

1. `mem_save` a single compact observation (`topic_key: skillgrid/YYYY-MM-DD-<topic>/report`, `type: learning`) with the acceptance verdict + any single high-value learning.
2. Emit a **one-line verdict** in the return text (e.g. "Verdict: accepted — fast-track, no report doc").
3. Still close the session (`mem_session_summary` + `mem_session_end`) — session close is never fast-tracked.

The waiver is honored, never silently dropped — note `Fast-track: {trivial|small} (waiver in briefing.md)` in the return text.

## Gotchas

- **A learning with no source is a guess.** "It was confusing" is not a learning. "The `feature-branch-chain` base boundary for PR #3 pointed at PR #1, not PR #2 — `tasks.md:47`" is.
- **The verdict is not a gate.** `rejected` records a real problem; it does not stop the cycle. Surface it and let the human decide. Don't let a `rejected` verdict be mistaken for a block — and don't wave a `FAIL`/CRITICAL through as "just a verdict."
- **Intermediate snapshots lie about "pending."** `report.md` (QA half) "open gap" lines are point-in-time. If the ship context or the repo shows the fix landed later, report the final state and cite where — do not carry the stale "pending" into the report as if it were open.
- **Do not re-verify the move.** `ship` already did the `diff -r` readback. You *reference* that evidence in the report; you do not re-run the move or re-diff.
- **Session close is yours.** No earlier phase closes the session. If you skip it, the next session starts blind.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `rules/mnemonic-memory.md`.)
- **A `PASS WITH` style note is not a block, and a `FAIL` is not a verdict.** The QA Gate (hard) is separate from the Verdict Gate (advisory). Keep them apart.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "Nothing changed since QA, so skip the report." | The report is where sourced learnings + the acceptance verdict land. Skipping it means the next session starts with no record of what worked or what didn't. |
| "I'll write the report later." | It is the final-state record of what actually shipped, with observation-ID lineage. There is no "later" — the session closes here. |
| "The discoveries are obvious, no need to source them." | Unsourced is a guess, not a learning. Every entry needs a file:line, commit, ticket, or scenario — otherwise mark it undiagnosed. |
| "The QA half of `report.md` says pending, so I'll carry that into the retro half." | Per Final-State Authority, the QA half is a point-in-time snapshot. Rank against ship context / the repo at close and report the final state, citing where the fix landed. |
| "A `rejected` verdict should block the cycle." | The verdict is advisory — recorded and surfaced, never a hard block. The human decides follow-up. Don't conflate it with the (hard) QA Gate. |

## Red Flags

- A learning in the report with no source — no file:line, commit, ticket, or scenario cited.
- The report restates a `report.md` (QA half) / ledger "pending / blocked / open" claim in bare present tense instead of attributing it to its source and time.
- The report was written even though a hard gate (Ship or QA) failed — those force `blocked` with no write.
- A `rejected` verdict was treated as a block (or a `FAIL` waved through as "just a verdict").
- A reusable Lesson/Surprise was recorded only as a one-off in `report.md`, never triaged as a process bug routed to the governing skill or `_shared` rule — the next change repeats it.
- The session was closed by an earlier phase, or not closed at all — close is `reflect`'s job and only its job.
- The report is missing observation IDs for the artifacts read (the lineage endpoint is incomplete).

## Verification

- [ ] All gates passed before any write: Ship Gate (folder in `archive/`, ship context present, `diff -r` empty) + QA Gate (`PASS` / `WAIVED` / `CONCERNS`, no unresolved CRITICAL) — or `blocked` returned with no write.
- [ ] `report.md` written into the moved folder with every learning sourced (file:line / commit / ticket / scenario) and an acceptance verdict recorded.
- [ ] `report.md` includes final-state facts, gate results, the `diff -r` reference, and the observation IDs of every artifact read.
- [ ] Every Lesson/Surprise was triaged process-bug vs one-off: process bugs named the governing skill/`_shared` rule and were recorded as follow-up tickets; one-offs stayed in `report.md` alone.
- [ ] Session persisted and closed: `mem_session_summary` present with all 6 sections (Goal / Instructions / Discoveries / Accomplished / Next Steps / Relevant Files) non-empty, followed by `mem_session_end`.
- [ ] Change marked closed: return envelope states the change, verdict, location, and "Next: none — cycle complete" (or the blocked reason, if gated).

## References

- [templates/report.md](templates/report.md) — thin pointer to the merged template at [../../verification/qa/templates/report.md](../../verification/qa/templates/report.md) (the `report.md` artifact shape: qa half + retro half — final-state + sourced learnings + verdict + lineage).
- [`../ship/SKILL.md`](../ship/SKILL.md) — upstream; moved the folder to `.skillgrid/archive/` and produced the ship context (the Ship Gate evidence).
- [`../../verification/qa/SKILL.md`](../../verification/qa/SKILL.md) — upstream; it wrote the QA half of `report.md` (through `## Gate Decision`), whose verdict drives the QA Gate and grounds the acceptance verdict.
- [`../../_shared/rules/sdd-structure.md`](../../_shared/rules/sdd-structure.md) — the `archive/` layout, terminal phase order, and Mnemonic slots.
- [`../../_shared/rules/mnemonic-memory.md`](../../_shared/rules/mnemonic-memory.md) — save shape, recovery ladder, session-close structure, and the observation-ID lineage the report must carry.
- [`../../_shared/planning/fast-track.md`](../../_shared/planning/fast-track.md) — the light-variant waiver this phase honors.
- [`../environment-retro/SKILL.md`](../environment-retro/SKILL.md) — the Step 3.6 sub-phase; sweeps the agent environment and appends the `## Environment Retro` section to this same `report.md` before session close.
