# Custom ADR entry template

`adr_style: custom` is set in `.skillgrid/config.yaml`. Record the project's
house format below, then draft every ADR entry to match it. ADR entries live as
`### ADR-NNNN` sections inside `.skillgrid/ASSUMPTIONS.md` § `## LOCKED`. Keep the
invariants from `SKILL.md` regardless of shape: one decision per entry, known facts
only (mark unknowns, don't invent), explicit rationale tied to requirements, honest
consequences, a status, and supersede-don't-rewrite (never delete an entry).

**Fixed heading line (required, independent of house body style):** every ADR entry
carries a machine-readable heading so the in-force set stays computable:

```
### ADR-NNNN — {title}
**Status.** {proposed | accepted | deprecated | superseded}
**Date.** {YYYY-MM-DD}
**Supersedes.** {ADR-NNNN or `—` — only when replacing a prior in-force ADR}
```

## House format

{Describe the required sections and any naming/location conventions.}

## Example

```md
### ADR-NNNN — {a filled-in example in the house format}
**Status.** accepted
**Date.** YYYY-MM-DD
**Supersedes.** —
...
```
