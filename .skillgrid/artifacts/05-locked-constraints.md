# Locked Constraints

Project-wide boundaries locked by the user. These override per-change decisions and are the hard limits a change must respect. A constraint is locked only when the user says so — inferred limits belong in `.skillgrid/ASSUMPTIONS.md` or an ADR, not here.

The `### Rules` section of the agent config block (`AGENTS.md`) is rendered from this file, one bullet per constraint. `state.yaml` `constraints_ref` points here.

## Locked

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Conventional commits only; no AI-attribution trailers (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).

## Notes

- A constraint that is later unlocked gets moved to a `## Unlocked (historical)` section below with the date — it is not deleted, so the boundary history stays readable.

## Unlocked (historical)

- (none yet)
