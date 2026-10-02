# ASSUMPTIONS

> Create this as the root `.skillgrid/ASSUMPTIONS.md`. It is the understanding
> record a session reads first: what is true, what is a hypothesis, which
> decisions are in force, and what is locked. The longer product requirements
> live in `.skillgrid/artifacts/00-prd.md` (`templates/PRD.md`); decision bodies
> live in `.skillgrid/artifacts/04-adr-NNNN-slug.md`. This file stores the path
> to each. For a multi-project repo, merge rather than overwrite.
>
> Maintenance: VERIFIED is written by brainstorming as facts are confirmed during
> the interview. INFERRED is a hypothesis (name the assumption it rests on).
> LOCKED requires an explicit user OK. Supersede an ADR by adding a new file and
> flipping the old row. Never delete an ADR file or its row.

**Read order:** VERIFIED → INFERRED → LOCKED → Open Questions.

---

## VERIFIED

Confirmed against the code, a spike, or a primary source.

**Product.** [One paragraph: what this is, who runs it, where it runs.]

**Stack.** [Measured from the repo.]

**Known gaps (verified against the tree YYYY-MM-DD).** [What is stubbed, missing, or stale.]

### Product requirements

Target users, personas, the MoSCoW feature table, user flows, non-functional requirements, and the release line are in `.skillgrid/artifacts/00-prd.md`. Open it when a change touches scope or acceptance.

---

## INFERRED (HYPOTHESIS)

Treated as true but not yet confirmed. Each rests on a named assumption; promote
to VERIFIED (with evidence) or LOCKED (with a user OK) before it carries a
decision.

- **H1:** [Hypothesis] — Assumed [what]. *Not confirmed* by [what].

---

## LOCKED

The in-force ADR index and user-locked constraints. Adding a row requires an
explicit user OK.

### In-force set

**In force** = `status: accepted` AND no later ADR's `supersedes` names it.
Superseded / deprecated rows stay in the table (frozen) but are marked out of
force. **IRON RULE: never delete an ADR file or its row — supersede by adding a
new file that names it and flipping its row here.** The Record column is the
path. The body is the file.

| # | Title | Status | Supersedes | Date | In force | Record |
|---|-------|--------|------------|------|----------|--------|
| 0001 | [title] | accepted | — | YYYY-MM-DD | yes | `.skillgrid/artifacts/04-adr-0001-slug.md` |

**Highest sequence in use:** 0001 (next ADR is `04-adr-0002-slug.md`).

### Locked constraints

These override per-change decisions and are the hard limits a change must respect.
A constraint is locked only when the user says so — inferred limits belong in
VERIFIED or an ADR, not here. The `### Rules` section of `AGENTS.md` is rendered
from this list, one bullet per constraint. `state.yaml constraints_ref` points at
this file.

- [constraint 1]
- [constraint 2]

*Unlocked (historical):* (none yet). A constraint that is later unlocked gets moved
here with the date — it is not deleted, so the boundary history stays readable.

### Locked assumptions

- [locked assumption 1]

---

## Open Questions

Questions that are not yet decided and not yet spiked.

1. [Open question]
2. [Open question]
