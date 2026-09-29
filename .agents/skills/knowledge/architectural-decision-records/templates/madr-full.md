# ADR entry template — madr-full

> ADR entries live as `### ADR-NNNN` sections inside `.skillgrid/ASSUMPTIONS.md` § `## LOCKED`.
> Copy the body below directly under the entry's heading line — no `#` title, no
> `---` frontmatter. The heading line is the fixed machine-readable part; the body
> follows this shape. Also append a row to the `### In-force set` table in the same edit.

### ADR-NNNN — {short title, representative of solved problem and found solution}

**Status.** {proposed | rejected | accepted | deprecated | superseded}
**Date.** {YYYY-MM-DD}
**Supersedes.** {ADR-NNNN or `—` — required only when this decision replaces a prior in-force ADR}
**Decision-makers.** {people involved in the decision}
**Consulted.** {people consulted}
**Informed.** {people informed}

#### Context and Problem Statement

{Describe the context and problem statement.}

#### Decision Drivers

- {decision driver 1}
- {decision driver 2}

#### Considered Options

- {option 1}
- {option 2}
- {option 3}

#### Decision Outcome

Chosen option: "{option 1}", because {justification}.

##### Consequences

- Good, because {positive consequence}
- Bad, because {negative consequence}

##### Confirmation

{How compliance with the ADR will be confirmed.}

#### Pros and Cons of the Options

##### {option 1}

{description or pointer to more information}

- Good, because {argument}
- Neutral, because {argument}
- Bad, because {argument}

##### {option 2}

{description or pointer to more information}

- Good, because {argument}
- Neutral, because {argument}
- Bad, because {argument}

#### More Information

{Evidence, agreement notes, links, realization plan, revisit timing.}

---

Source: https://github.com/adr/madr/blob/4.0.0/template/adr-template.md
