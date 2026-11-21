# Deterministic Boundary (script vs. prose)

The classification that decides whether a step in a skill is a script or a
prose judgment. Cite this; do not restate the test.

## The test

> "If I ran this step twice with identical input, would the correct output be
> identical both times?"

- **Yes** → deterministic → extract to a script.
- **No** → judgment → keep as prose.

| Step | Deterministic? | Action |
|------|---------------|--------|
| Parse Trivy JSON, filter by severity | Yes | Extract to `scripts/trivy-parse.mjs` |
| Classify an "unconfirmed" OWASP pattern | No | Keep as judgment prose |
| Check if state file matches spec zone | Yes | Extract to `scripts/state-drift-check.mjs` |
| Decide which test layer fits | No | Keep as judgment prose |

## The stop point (mixed steps)

If a step is *mixed* (deterministic parse + judgment classification), split
it: the deterministic part goes to a script, the judgment part stays in prose.
Name both. Never let a judgment hide inside a script, or a procedure hide
inside a paragraph.

## Why scripts cost nothing

Scripts are executed, not read into the model's window. A 200-line script costs
zero tokens at load time; the prose describing it does not. So the boundary is
not about token savings at load — it is about **reliability**: a script returns
the same verdict every run, prose drifts. When in doubt, the question that
decides it is the test above, not "is this long enough to be a script."
