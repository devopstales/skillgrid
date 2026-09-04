# Tasks: 006-structured-session-handoff — Step 01-relay-schema

> Goal: Migrations for handoffs/archives.
> Depends on: none

## Review Workload Forecast (change-level)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | ~80 |
| Estimated changed lines (change) | ~900–1400 |
| 400-line budget risk (change) | High |
| Chained PRs recommended (change) | Yes |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 01.1 Create `skillgrid-cli/internal/mnemonic/store/migrations/012_session_relay.sql` with additive `session_handoffs` and `session_archives` (leave `009_*`/`010_*`/`011_*` for 001/003/004).
- [ ] 01.2 Cover WHAT: opening a store applies `012_*` and creates handoff/archive tables without rewriting existing sessions/observations (`store.Open` / migrate path).
- [ ] 01.3 Cover WHAT edge: DB through `008_*` migrates once; re-open is idempotent with no duplicate tables (`upgrade_test.go` or sibling migrate test).
