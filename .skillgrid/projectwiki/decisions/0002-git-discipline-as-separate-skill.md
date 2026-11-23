# Adopt a `git-discipline` skill as a separate first-class skill (not folded into `work-unit-commits`)

---
status: accepted
date: 2026-11-22
supersedes: none
---

## Context and Problem Statement

The `intent-driven-template` repo's `openspec-git-discipline` skill is
a 3-KB state machine: a state table (moment → gate), a red-flags list,
and exact phrasing for each halt ("I should not apply this yet because
the proposal change has not reached `main`…"). The gates are
**phase-transition invariants** — the spec must reach `main` before
apply, the implementation must merge before archive — not commit-format
rules.

skillgrid's `_shared/work-unit-commits/README.md` owns the
**commit-time** discipline: conventional-commit format, atomic sizing,
when-to-commit, the `[skillgrid-context]` block, the `pre-commit` zone
guard. It does not own the **phase-transition** git state gates: the
spec must be on the base before `execute` dispatches Task 1, the
execution branch must merge before `ship` archives, one fan-out wave
must integrate before the next, a chained-PR slice's PR must exist
before the next slice.

Those phase-transition gates currently live as prose inside `execute`
(Step 0, the zone rule) and `ship` (Step 6) — they are not a
first-class skill, so they are not announced, not measured, and not
independently invokable.

## Considered Options

- **Fold the phase-transition gates into `_shared/work-unit-commits/README.md`**:
  the commit contract already names the zone guard; adding the
  phase gates keeps the git discipline in one place.
- **A separate `git-discipline` skill**: the phase-transition gates
  are a distinct concern (phase ordering, not commit format); a
  separate skill can be announced, measured, and invoked
  independently of the commit contract.
- **No skill, keep the gates as prose in `execute`/`ship`**: the
  cheapest option; the gates stay where they are.

## Decision Outcome

Chosen option: **A separate `git-discipline` skill**, because the
phase-transition gates are a distinct concern from commit format, and
a separate skill makes them first-class: announced at the start
("I'm using the skillgrid:git-discipline skill to check the git
state before this phase transition"), measured (the halt count per
change), and independently invokable (a human can say "run
git-discipline" without running `execute`).

The skill's state table has ten gates (G1–G10), each with a
deterministic check (read git), a prescribed phrasing, and a red-flag
list. The gates are **hard** — the skill halts, it does not warn.
The skill does not mutate git state; it checks, halts, and phrases.
The mutation is the user's or a downstream skill's.

### Consequences

- Good, because the phase-transition gates are first-class: they
  are announced, measured, and independently invokable.
- Good, because the commit contract (`_shared/work-unit-commits`)
  stays focused on commit format; the phase gates do not dilute
  it.
- Good, because the prescribed phrasing ("I should not start
  execution yet because…") is the template's strongest feature —
  the gate halt is not a silent failure, it is a named,
  phrased, ledgered event.
- Bad, because the skill is a new entry in the skill family; the
  surface-profile budget (`skillgrid:skill-creator` §12b) must
  account for it.
- Bad, because the gates overlap with the `pre-commit` zone guard
  (G2) and the no-added-changes rule (G10) — the skill is the
  prose mirror of the machine enforcement, and a drift between the
  prose and the hook is a risk (the skill's `References` section
  names the hook as the machine-enforcement, so the drift is
  catchable).

## Not adopted

- **Folding into `_shared/work-unit-commits/`** — the commit
  contract would grow from ~400 lines to ~600; the phase gates are
  a distinct concern and a separate skill is the cleaner split.
  Revisit when the skill family is reorganized.
- **A `git-discipline` hook** (machine enforcement of the phase
  gates) — the `pre-commit` hook enforces G2/G3; a phase-transition
  hook would need to know the phase (the `state.yaml` phase), which
  the hook does not currently read. Revisit when the hook
  infrastructure grows a phase-aware check.

## Related

- Supersedes: none.
- Depends on: `_shared/work-unit-commits/README.md` (the commit
  contract — the zone guard, the no-added-changes rule),
  `_shared/rules/chain-strategy.md` (the chain strategy — G9's
  context), `_shared/rules/sdd-structure.md` (the phase order —
  which phase is "next").
- Consumed by: `execute` (G1, G2, G4, G5), `ship` (G6, G7, G8,
  G10), `slicing` (G9, in a chained-PR strategy).
