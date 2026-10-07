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

Confirmed against the code, a prototype, or a primary source.

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

The ADR index — the in-force decisions. Adding a row requires an explicit user OK.

The ADR index. One row per decision: a short description and a link to the full
decision file. This table is an index, not the body — `status`, `supersedes`, and
`date` live in each file's frontmatter; open the linked file for the full
Context / Decision / Consequences.

**In force** = the file's `status: accepted` AND no later ADR's `supersedes:`
names it. Superseded rows stay in the index (frozen) but are marked `no`.
**IRON RULE: never delete an ADR file or its row — supersede by adding a new file
that names it and flipping its row here.**

| # | Description | In force | File |
|---|-------------|----------|------|
| 0001 | [short description] | yes | `.skillgrid/artifacts/04-adr-0001-slug.md` |

**Highest sequence in use:** 0001 (next ADR is `04-adr-0002-slug.md`).

Operational rules (the hard limits a change must respect) live in `AGENTS.md`'s
`### Rules` section, not here. A rule that is actually a decision belongs in this
index as an ADR instead.

---

## Open Questions

Questions that are not yet decided and not yet prototyped.

1. [Open question]
2. [Open question]
