# Custom ADR entry template

`adr_style: custom` is set in `.skillgrid/config.yaml`. Record the project's
house format below, then draft every ADR as `.skillgrid/artifacts/04-adr-NNNN-slug.md`.
Add a path row to `.skillgrid/ASSUMPTIONS.md` § `### In-force set`. Do not paste
the body into `ASSUMPTIONS.md`. Keep the invariants from `SKILL.md` regardless of
shape: one decision per file, known facts only (mark unknowns, don't invent),
explicit rationale tied to requirements, honest consequences, a status, and
supersede-don't-rewrite (never delete a file).

**Fixed heading (required, independent of house body style):** every ADR file
carries frontmatter so the in-force set stays computable:

```
# {title}

---
status: "{proposed | accepted | deprecated | superseded}"
date: {YYYY-MM-DD}
supersedes: {ADR-NNNN or none}
---
```

## House format

{Describe the required sections and any naming/location conventions.}

## Example

```md
# {a filled-in example in the house format}

---
status: accepted
date: YYYY-MM-DD
supersedes: none
---
...
```
