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

## The agentic script contract

A step that passes the test above becomes a script with this contract (the
same rules the existing helpers — `state-drift-check.mjs`, `task-brief.mjs`,
`area-scope.mjs` — already follow):

1. **Stdout is machine-shaped.** One JSON object (or a documented lean TSV).
   Usage and parse failures are JSON on stderr too — never free-form prose that
   the agent has to re-parse.
2. **Stderr is diagnostics.** Human-facing one-liners go there; stdout stays
   clean for piping.
3. **Exit codes are a vocabulary.** 0 = pass, 1 = findings/fail, 2 = usage or
   bad input. Some linters always exit 0 and put findings in the JSON so the
   *caller* decides severity — say which mode yours is.
4. **Writes are atomic.** temp file → replace; roll back if post-write
   validation fails.
5. **No interactive prompts.** Flags only. A script that calls `read()` or
   waits on a keypress blocks the agent.
6. **Tests sit beside the script.** `test-<name>.*` next to the script, run by
   the QA path. A helper without a test is prose with a shebang.
7. **The SKILL.md names the split.** One line: "the agent decides *which*;
   the script owns *everything after*." When the script fails, the agent tells
   the user and offers the fix flow — it does not guess.

The gap this closes: planning skills (brainstorm, blueprint, slicing, resume)
historically described deterministic procedures in prose while QA/ship
called scripts. A prose "check that every requirement has a test" drifts run
to run; the same check as a script returns the same verdict every run.

## Why scripts cost nothing

Scripts are executed, not read into the model's window. A 200-line script costs
zero tokens at load time; the prose describing it does not. So the boundary is
not about token savings at load — it is about **reliability**: a script returns
the same verdict every run, prose drifts. When in doubt, the question that
decides it is the test above, not "is this long enough to be a script."
