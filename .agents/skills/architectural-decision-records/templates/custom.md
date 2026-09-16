# Custom ADR Template

`adr_style: custom` is set in `.skillgrid/config.yaml`. Record the project's
house format below, then draft every ADR to match it. Keep the invariants from
`SKILL.md` regardless of shape: one decision per ADR, known facts only (mark
unknowns, don't invent), explicit rationale tied to requirements, honest
consequences, a status, and supersede-don't-rewrite.

**Fixed header (required, independent of house body style):** every ADR carries
a machine-readable header so the in-force set stays computable. Use frontmatter
when the house format allows it, otherwise a leading blockquote line:
`status` (proposed | accepted | deprecated | superseded), `date`, and
`supersedes: ADR-NNNN` (only when replacing a prior in-force ADR).

## House format

{Describe the required sections and any naming/location conventions.}

## Example

```md
{A filled-in example in the house format.}
```
