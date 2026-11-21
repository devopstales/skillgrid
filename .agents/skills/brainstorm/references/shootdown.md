# Goldfish Brainstorming (Shootdown)

The Goldfish stress test for the New Project / New Function paths: **seven**
fresh-context lenses attack the anchored hypothesis after the codebase
feasibility check passes and before the ADR manifest is written. This file is
the dispatch protocol — where the shootdown sits, how the lenses spawn, how the
synthesizer aggregates, and how the findings are dispositioned. The tone rules
it enforces live in `_shared/craft/review-discipline.md` (cited, not
restated). The lens + synthesizer persona briefs live in
`.agents/skills/_shared/elephant-goldfish/`.

## Dispatch point (R2)

Runs on the **New Project** and **New Function** paths only, **after the
codebase feasibility check passes and before the ADR manifest is written**.
The brainstorming flow is interview → approaches → design → feasibility →
**shootdown** → ADR manifest → briefing; the shootdown sits between feasibility
and the ADR manifest, so the manifest is written against a shootdowned design.

The **Bounded** and **Prototype** paths skip it — their path descriptions do
not name the shootdown. Path ratchet unchanged: if a bounded task upgrades to
New Function mid-task, the shootdown becomes mandatory at the new path.

## Readiness gate (before the lenses spawn)

The lenses are expensive (seven fresh-context subagents). They are not spawned
until the user confirms the anchored hypothesis is stable:

1. **Drill the problem first** — work the interview (the design tree in
   `references/questioning.md`) until the problem statement, constraints, and
   target behavior are thorough. Open-ended questioning; no structured
   multiple-choice candidate answers during problem exploration.
2. **Anchor the hypothesis** — summarize the anchored working hypothesis in
   2–3 sentences.
3. **Single-select readiness gate** — present the confirmation gate (one
   question, the sole tool call in the turn, no `(Recommended)` prefix):
   - **"Review with Goldfishes"** — the user confirms readiness. Proceed to
     the lens dispatch (next section).
   - **"Clarify additional constraints or assumptions in chat"** — the user
     refines the problem statement or adds write-in notes. Stay in the drill,
     yield the turn, incorporate the feedback into the hypothesis, and
     re-present the gate when ready.

Do NOT spawn subagents before this gate confirms readiness. This is the EGM
"multi-turn clarification & readiness gate" — the anchor must be confirmed
before the swarm runs against it.

## Freshness check (R36)

Before the seven lenses spawn, apply the R36 STOP from
`_shared/elephant-goldfish/protocol.md` § Freshness rule: a harness without
fresh-context subagents does not degrade — it stops, and a same-context run
MUST NOT count toward R1. No partial run, no "we'll count this one as a review
anyway."

## The seven lenses (R1)

All seven are dispatched **in parallel** (not sequentially): one freshly
spawned `reviewer` subagent (fallback `general`) per lens brief, all under
`.agents/skills/_shared/elephant-goldfish/`:

- `zoom-out.md` — Skeptical Architect (YAGNI, reuse, anti-weeds, altitude)
- `design-critic.md` — Hostile Design Critic (architecture, scalability, cost
  & quota economics, unmitigated risk)
- `comprehension.md` — Naive Reader (UX & developer ergonomics, cognitive
  overhead, self-containment)
- `implementation-readiness.md` — Pragmatic Tech Lead (migration, backward
  compatibility, rollout safety)
- `codebase-details.md` — Pedantic Technical Auditor (observability,
  telemetry, operational debuggability)
- `structural.md` — Document Architect (long-term tech debt, coupling,
  maintainability)
- `inventive-catalyst.md` — Inventive Catalyst (TRIZ: false-compromise
  detection, inventive transforms, Ideal Final Result)

Dispatch prompt = lens brief path + the anchored hypothesis + the explicit
reading list (the relevant existing code/files) + the R37 four-part isolation
contract re-asserted verbatim (from
`_shared/elephant-goldfish/protocol.md` § Four-part isolation contract) +
`Mode: BRAINSTORM`. Each subagent is given ONLY the anchored hypothesis +
relevant code — no conversation history, no author narrative. A review run in
the author's own context MUST NOT count toward R1.

## Synthesis (the synthesizer)

After the seven lenses report, the **synthesizer** aggregates. Per
`.agents/skills/_shared/elephant-goldfish/synthesizer.md`, `Mode: BRAINSTORM`:

- If the harness nests subagents, the main agent spawns the synthesizer, which
  spawns the seven lenses, collects their reports, deduplicates, groups by
  severity (🔴 Blocker / 🟡 Major / 🟢 Nit), normalizes references, and writes
  the consolidated analysis.
- If the harness cannot nest, the main agent spawns the seven lenses directly
  and performs the same dedup + severity grouping itself (the synthesizer
  persona is an optimization, not a requirement — the discipline is).

The synthesis tone follows `_shared/craft/review-discipline.md` (cited, not
restated): dedupe and group without softening — a 🔴 Blocker stays a 🔴 Blocker
until the user disposes of it. The synthesizer never invents a finding a lens
did not report, and never drops a Blocker to make the verdict look cleaner.

## Error handling

A lens that times out or returns no report: retry once, then record `not run`
— never zero findings. A lens that is `not run` caps its contribution at "not
verified," not "clean." The synthesizer's per-lens status table (completed /
not run) is part of the deliverable.

## Finding gate (R3)

The user is shown the 🔴 Blockers and disposes of each:

- **resolved** — the design is revised; loop back to the feasibility check.
- **accepted-with-note** — recorded in the briefing's `## Shootdown Findings`
  as a known risk.
- **downgraded-to-non-goal** — moved to the spec's Out of Scope.

🟡/🟢 are recorded but do not block. The spec is NOT written until all
🔴 Blockers are dispositioned — the dispositions land in the briefing
template's `## Shootdown Findings` section.

The **Inventive Catalyst** lens is advisory by nature: a false-compromise
finding with an inventive transform is a 🟡/🟢 Major/Nit (advisory), not a
🔴 Blocker — it surfaces a contradiction and a pattern, it does not assert the
design is broken. "No false compromise detected" is a valid, reportable
outcome and is recorded as such (never as a zero-finding silence).

## Consolidated chat response

The synthesis turn ends with:

- the per-lens status table (completed / not run, all seven named),
- the severity table (normalized vocabulary at the record boundary),
- every 🔴 finding with its disposition options,
- the path where the findings will land.

A turn that ends with only a file path does not satisfy this.

## Iteration loop

After the findings are dispositioned, the user decides how to address them
(pivot the approach, narrow scope, add a constraint). If the design changes
materially, the shootdown re-runs (fresh lenses against the revised anchor)
before the ADR manifest is written. The loop repeats until the concept has
survived scrutiny and the user is satisfied.
