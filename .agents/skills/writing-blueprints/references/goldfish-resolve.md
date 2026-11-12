# Goldfish Resolve — Elephant-Side Resolution Workflow (Track B)

Findings from the 5-stage pipeline (`goldfish-review.md`) are resolved by
the main agent (the Elephant) — never silently — before the blueprint
advances. The Goldfish reviewers never resolve their own findings: the
deletion test this file guards is findings triaged by the reviewers
themselves, the same-context rubber-stamp R10 exists to kill.

## The four actions

Each finding is categorized into exactly one:

- **CONTEST** — inaccurate; cite evidence the reviewer could read.
- **ESCALATE** — inaccurate, but the evidence is out-of-doc; flag it for
  the user.
- **PROPOSE-FIX** — valid; name the exact edit.
- **DECLINE** — nit; record, don't fix.

**Triage adapter (A1 — an adapter, not a new system):** CONTEST = the
code-review triage verdict `false` (state what disproves it — R39);
ESCALATE = `could-not-verify` → `receiving-code-review`'s human-look
bucket; PROPOSE-FIX = the fix-now bucket; DECLINE = the low/noise
bucket. The buckets are cited from `receiving-code-review`
(`.agents/skills/receiving-code-review/SKILL.md` → Triage and Fix Loop),
not restated — this file adds only the doc-review deltas below.

## The four rules

- A contest must be checkable ("this was intentional" is not evidence).
- Do not add context just to satisfy a reviewer (if only a zero-context
  subagent would need it → ESCALATE).
- Do not contest comprehension gaps (a reviewer that read the target and
  still missed it describes a self-containment defect — PROPOSE-FIX).
- Do not change severity to win.

The author wrote the document — the desire to contest is not evidence
the reviewer is wrong.

## Resolution context

Resolution runs in the Elephant's context (the main agent, never a
subagent): it has the institutional knowledge to write good fixes and
defend design choices. Reviewers are never asked to self-critique.

## Interactive gate

After each round's resolution the user picks:

- **Auto accept** — batch-apply all PROPOSE-FIX
- **Continue** — no changes
- **Interactive break down** — per-item: Accept / Skip / Discuss
- **Comment on file** — the user edits the report directly
- **Custom instruction** — free text

The gate never auto-advances — the loop stays idle (turn-taking per
`goldfish-review-loop.md`).

## Parked findings (R17)

A finding may be parked with its path recorded; the gate verdict is
otherwise unaffected (WARNING, never CRITICAL). At qa time the
debt-surfacing step (`qa/SKILL.md` → Step 9.9: Surface review debt)
surfaces it as a WARNING naming the parked finding and its path.

## Ledger round

Every resolution round writes a ledger round entry per
`_shared/craft/review-ledger.md` (dispositions: fixed / parked / waived;
severities normalized at this boundary per the record-boundary rule in
`goldfish-review.md`).
