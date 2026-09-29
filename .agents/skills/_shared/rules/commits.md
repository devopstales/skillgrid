# Commit Convention (shared across all Skillgrid skills)

The commit contract for `subagent-execution`, `simple-execution`, and any skill that commits.

## Rules

- **Conventional Commits**: `type(scope): subject` — imperative present tense, ≤ 72 chars.
- **No AI trailers**: never `Co-Authored-By: Claude`, `Co-Authored-By: Copilot`, or any AI attribution.
- **One logical change per commit**: implementation + its tests + enabling config.
- **Ticket close-token footer**: if a ticket ID exists for this work, include `Refs: <ID>` in the body. The closing commit uses `Closes <ID>`.
- **Commit after green**: never commit work whose tests are red or whose evidence is incomplete.
- **`[skillgrid-context]` block**: every work-unit commit carries the context block (see `skillgrid:work-unit-commits`).

## Types

| Type | Use |
|---|---|
| `feat` | New capability |
| `fix` | Bug fix |
| `refactor` | Structural change, no behavior change |
| `test` | Test-only change |
| `docs` | Documentation-only change |
| `chore` | Config, tooling, housekeeping |

## Multi-commit Batches

When a work unit spans multiple commits (e.g., RED commit + GREEN commit in TDD):
- Each commit is a checkpoint.
- The final commit of the unit carries the ticket close-token.
- All commits in the unit reference the same ticket via `Refs: <ID>`.
