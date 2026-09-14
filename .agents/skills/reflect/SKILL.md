---
name: reflect
description: "Use when a change has been shipped (folder moved to .skillgrid/archive/) and you need to close the cycle with a sourced retrospective and the final-state archive report. The terminal phase (qa → review → ship → reflect)."
---

# Reflect

**Announce at start:** "I'm using the skillgrid:reflect skill to close this change."

You are the **TERMINAL** phase — the close of the SDD cycle. `ship` already integrated the work and moved the change folder to `.skillgrid/archive/YYYY-MM-DD-<topic>/`. You do exactly three things:

1. **Run a retrospective** — extract sourced learnings (Decisions / Lessons / Patterns / Surprises) and record an acceptance verdict (`accepted` / `accepted-with-open-items` / `rejected`).
2. **Record the archive report** — the final-state record of what actually shipped, with observation-ID lineage.
3. **Close the session** — `mem_session_summary` + `mem_session_end`.

You are the **completion checkpoint**: you are the last chance to honestly say what worked, what did not, and whether the change is accepted. You are also the **lineage endpoint** — the archive report is the record a future reader consults to learn what shipped and when.

## What You Receive

- **Change folder:** now at `.skillgrid/archive/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `qa-report.md`, `ship-report.md`, review artifacts).
- **`ship-report.md`** — the move evidence (`diff -r` readback), base branch, integration outcome, gate results.
- **`qa-report.md` verdict** + any human override.
- **All planning artifacts** (briefing, blueprint, design/threat-matrix, tasks) — the source material for sourced learnings.
- **Fast-track waiver class** (`trivial` / `small`) if present — drives the light variant.

## Phase Order

```
qa → review → ship → reflect
```

`prev-phase: [ship]` · `next-phase: []` (terminal) · `artifact: retrospective + archive-report`.

## Final-State Authority

The archive report is the terminal record of the cycle. It describes the state of the change **at close**, not at earlier points. `qa-report.md` and the execution ledger are **intermediate snapshots** — true for the moment written, but work routinely continues after they are persisted (verify warnings fixed in later commits, blocked tasks completed, test counts change). A snapshot's "done" stays true — work does not un-complete — but its "pending / blocked / open gap" claims are only valid for the instant they were written. **Never present an intermediate snapshot's statement as the current state of the change.**

When sources disagree about a fact, rank them — most authoritative first:

1. **Current repository / filesystem state at close** — what is actually on disk and in git now. The strongest evidence of what shipped.
2. **`ship-report.md`** — the integration + move record (base branch, merge/PR commit, `diff -r` readback). The most recent close-out account.
3. **The persisted `tasks.md`** — completion visibility.
4. **`qa-report.md` and the execution ledger** — intermediate snapshots. Lowest rank: valid history of what was true at their time, never evidence of final state.

Reporting rules that follow:

- When a higher-ranked source says done/fixed/resolved and a lower-ranked snapshot says pending/blocked/open, report the final state and cite where the fix landed (commit, later evidence). Do **not** echo the stale claim.
- When a contradiction cannot be ranked (a claim no higher-ranked source or repo evidence corroborates), record it explicitly: both statements, their sources, and when each was written. Never resolve it silently.
- Attribute snapshot-derived claims to their source and time ("per `qa-report` at verification time …"). Do not restate them in bare present tense as current facts.
- Carry final numbers (test counts, warnings, open issues) from the highest-ranked source that covers them.
- Never merge distinct defects or failures into a single causal story. A cause is recorded as confirmed only with evidence; otherwise record the failure as undiagnosed.

This hierarchy governs how the archive report **reports** facts. It does not weaken the gates: a `qa-report` `FAIL` / unresolved CRITICAL still blocks reflect (you did not ship), and the gates below keep their own authority.

## Gates (all must pass before ANY write)

### Ship Gate (hard)

- **No `ship-report.md`** in the moved folder → **`blocked`** (reason `not-shipped`). Nothing shipped, nothing to reflect.
- **`ship-report.md` status `blocked`** or its `diff -r` readback non-empty → **`blocked`** (reason `ship-incomplete`). The folder move was not clean.
- **`ship-report.md` status `success`** + `diff -r` empty → proceed.

### QA Gate (hard)

- **`qa-report.md` verdict `FAIL`** or an unresolved **CRITICAL** with no human override → **`blocked`** (reason `qa-failed`). You cannot close work the quality gate refused.
- **`PASS`** / **`WAIVED`** / **`CONCERNS`** (human-owned) → proceed.

### Verdict Gate (advisory — never blocks)

The acceptance verdict you produce is **recorded, not enforced**. A `rejected` verdict does **not** block the archive — it is recorded in the archive report and surfaced to the human, who decides whether a follow-up change is needed. Never block the cycle on the verdict alone.

If a hard gate fails, **STOP and return `blocked`** with the failing gate named. Do not write the retrospective or archive report.

## What to Do

### Step 1: Load All Artifacts (recovery + lineage)

1. Read the moved folder `.skillgrid/archive/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `qa-report.md`, `ship-report.md`, review artifacts, findings).
2. Recover the Mnemonic copies (previews are not enough — always fetch full content). **Record the observation ID of every artifact you read — they go into the archive report for lineage.**
   - `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/blueprint")` → `mem_get_observation(id)`
   - `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/tasks")` → `mem_get_observation(id)`
   - `mem_search(query: "skillgrid/YYYY-MM-DD-<topic>/ship-report")` → `mem_get_observation(id)`
   - (and any `research` / `findings` / ADR observations the change produced)
3. Read `config.yaml` if present — `rules.reflect` bind this phase.

> If `mnemonic.enabled` is `false`, the in-repo artifacts are the sole source — skip the Mnemonic recovery (degrade explicitly).

### Step 2: Pass the Gates (before ANY write)

Confirm, in order: **Ship Gate** → **QA Gate** → **Verdict Gate** (advisory). If a hard gate fails, STOP and return `blocked`.

### Step 3: Write `retrospective.md`

Write into the moved folder (`.skillgrid/archive/YYYY-MM-DD-<topic>/retrospective.md`). Use [templates/retrospective.md](templates/retrospective.md).

**Sourced learnings — every entry MUST point at evidence** (a file:line, a commit, a ticket ID, or a scenario). A finding with no source is a guess, not a learning. Four categories:

- **Decisions** — architecture / design / tooling choices made, with the tradeoff and why.
- **Lessons** — what would be done differently next time (root cause, not symptom).
- **Patterns** — reusable approaches or conventions established (name them so a future change can reuse them).
- **Surprises** — non-obvious gotchas, edge cases, or behaviors discovered (the highest-signal category for future sessions).

**Acceptance verdict** — `accepted` / `accepted-with-open-items` / `rejected`:

- Ground it in the goal stated in `briefing.md` and the evidence in `qa-report.md` (Goal-Backward Verification + Traceability Matrix).
- `accepted` — goal met, no open items.
- `accepted-with-open-items` — goal met, but name the open items (they become next-change candidates).
- `rejected` — goal not met (a scenario unmet, a regression, the goal-backward check failed). Record why.

**Prior-change follow-through** — scan this change's own open items and the most recent archived changes' `retrospective.md` open items: which did this change address? Which are still open? Name them.

### Step 4: Write `archive-report.md`

Write into the moved folder (`.skillgrid/archive/YYYY-MM-DD-<topic>/archive-report.md`). Use [templates/archive-report.md](templates/archive-report.md). Per the **Final-State Authority** hierarchy above, record:

- Final-state facts (what shipped, to which base, at which commit/PR).
- Gate results (Ship Gate, QA Gate, Verdict Gate).
- The observation IDs of **every** artifact read (the lineage endpoint).
- The `ship-report.md` `diff -r` readback (the move evidence — reference it).
- Any overrides / waivers / unrankable contradictions.

### Step 5: Persist to Mnemonic + Close the Session (you own this)

`reflect` is the **only** phase that closes the session. Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md).

```
mem_save(
  title:      "retrospective — YYYY-MM-DD-<topic>",
  topic_key:  "skillgrid/YYYY-MM-DD-<topic>/retrospective",
  type:       "learning",
  scope:      "project",
  session_id: "{sid}",
  content:    "{retrospective markdown: sourced Decisions/Lessons/Patterns/Surprises, acceptance verdict, open items, follow-through}"
)
mem_save(
  title:      "archive-report — YYYY-MM-DD-<topic>",
  topic_key:  "skillgrid/YYYY-MM-DD-<topic>/archive-report",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{archive-report markdown: final-state facts, gate results, observation-ID lineage, diff -r reference, overrides}"
)
# Save any single high-value learning as its own observation too (type: decision|pattern|bugfix|discovery).

# THEN close the session:
mem_session_summary(session_id: "{sid}", summary: <structure below>)
mem_session_end(session_id: "{sid}", summary: "one-line outcome")
```

`mem_session_summary` structure (required):

```
## Goal
[What we were working on this session]

## Instructions
[User preferences or constraints discovered — skip if none]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains to be done — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]
```

> If `mnemonic.enabled` is `false`, the in-repo `retrospective.md` + `archive-report.md` are the sole record — skip the Mnemonic saves and session close (degrade explicitly, never fail silently).

### Step 6: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` + `mem_session_summary` + `mem_session_end` (Step 5) *before* this text.

```markdown
## Change Closed

**Change**: YYYY-MM-DD-<topic>
**Verdict**: accepted | accepted-with-open-items | rejected
**Location**: `.skillgrid/archive/YYYY-MM-DD-<topic>/` (retrospective.md + archive-report.md) · Mnemonic `skillgrid/YYYY-MM-DD-<topic>/{retrospective,archive-report}`
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

**Lineage (observation IDs)**: blueprint {id} · tasks {id} · ship-report {id} · qa {id} · …
**Mnemonic**: session `{sid}` closed · observations `{ids}`
**Open questions**: {list, or "None"}
**Risks**: {list, or "None"}
**Next**: none — cycle complete
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up.

## Rules

- You are **read-only over the archive** — you do not modify shipped code or the folder layout. You write exactly two files (`retrospective.md`, `archive-report.md`) into the moved folder and persist to Mnemonic.
- **Every learning is sourced.** A finding with no file:line / commit / ticket / scenario is a guess — either cite it or mark it undiagnosed.
- **The acceptance verdict is advisory.** `rejected` is recorded and surfaced, never a hard block. The human decides follow-up.
- **The archive report reflects FINAL state** per the Final-State Authority hierarchy — never echo stale `qa-report` / ledger "pending" claims as current facts; record unrankable contradictions explicitly.
- **You own session close** — `mem_session_summary` + `mem_session_end` run here and only here. Sub-agents in earlier phases emit `## Key Learnings` only and do not close the session.
- **Record every observation ID you read** into the archive report — the archive is the lineage endpoint.
- If a hard gate (Ship or QA) fails, STOP and return `blocked` — do not write the retrospective or archive report.
- No external binaries. Mnemonic (`mem_*`) and the project's files are the only tools.
- Apply any `rules.reflect` from `config.yaml`.
- Return envelope per Step 6 — final action is text, not a tool call.

## Fast-Track Variant

For a `trivial` / `small` change (waiver recorded): skip the full `retrospective.md` / `archive-report.md` documents. Instead:

1. `mem_save` a single compact observation (`topic_key: skillgrid/YYYY-MM-DD-<topic>/retrospective`, `type: learning`) with the acceptance verdict + any single high-value learning.
2. Emit a **one-line verdict** in the return text (e.g. "Verdict: accepted — fast-track, no retro doc").
3. Still close the session (`mem_session_summary` + `mem_session_end`) — session close is never fast-tracked.

The waiver is honored, never silently dropped — note `Fast-track: {trivial|small} (waiver in briefing.md)` in the return text.

## Gotchas

- **A learning with no source is a guess.** "It was confusing" is not a learning. "The `feature-branch-chain` base boundary for PR #3 pointed at PR #1, not PR #2 — `tasks.md:47`" is.
- **The verdict is not a gate.** `rejected` records a real problem; it does not stop the cycle. Surface it and let the human decide. Don't let a `rejected` verdict be mistaken for a block — and don't wave a `FAIL`/CRITICAL through as "just a verdict."
- **Intermediate snapshots lie about "pending."** `qa-report` "open gap" lines are point-in-time. If `ship-report` or the repo shows the fix landed later, report the final state and cite where — do not carry the stale "pending" into the archive as if it were open.
- **Do not re-verify the move.** `ship` already did the `diff -r` readback. You *reference* that evidence in the archive report; you do not re-run the move or re-diff.
- **Session close is yours.** No earlier phase closes the session. If you skip it, the next session starts blind.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md`.)
- **A `PASS WITH` style note is not a block, and a `FAIL` is not a verdict.** The QA Gate (hard) is separate from the Verdict Gate (advisory). Keep them apart.

## References

- [templates/retrospective.md](templates/retrospective.md) — the `retrospective.md` artifact shape (sourced learnings + verdict + follow-through).
- [templates/archive-report.md](templates/archive-report.md) — the `archive-report.md` artifact shape (final-state + lineage).
- [`../ship/SKILL.md`](../ship/SKILL.md) — upstream; moved the folder to `.skillgrid/archive/` and produced `ship-report.md` (the Ship Gate evidence).
- [`../qa/SKILL.md`](../qa/SKILL.md) — upstream; its `qa-report.md` verdict drives the QA Gate and grounds the acceptance verdict.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — the `archive/` layout, terminal phase order, and Mnemonic slots.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, recovery ladder, session-close structure, and the observation-ID lineage the archive report must carry.
- [`../_shared/conventions/fast-track.md`](../_shared/conventions/fast-track.md) — the light-variant waiver this phase honors.
