# ADR entry template — madr-full

> Write this file as `.skillgrid/artifacts/04-adr-NNNN-slug.md`.
> Add one pointer row to the `## LOCKED` ADR index in `.skillgrid/ASSUMPTIONS.md`.
> Do not paste this body into `ASSUMPTIONS.md`. The `#` title and the frontmatter
> (`status`, `date`, `supersedes`) are the machine-readable heading.

# {short title, representative of solved problem and found solution}

---
status: "{proposed | rejected | accepted | deprecated | superseded}"
date: {YYYY-MM-DD}
supersedes: {ADR-NNNN or none}
---

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
