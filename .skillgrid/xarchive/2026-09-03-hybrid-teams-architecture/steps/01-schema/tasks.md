# Tasks: 001-hybrid-teams-architecture — Step 01-schema

> Goal: Add SQLite migration for teams/tasks/messages/reviews tables + filesystem data-plane dirs.
> Depends on: none

## Review Workload Forecast (change-level)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | ~250 |
| Estimated changed lines (change) | ~1200–1500 |
| 400-line budget risk (change) | High |
| Chained PRs recommended (change) | Yes |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 01.1 Create `skillgrid-cli/internal/mnemonic/store/migrations/009_teams_schema.sql` with tables `teams`, `team_members`, `tasks`, `messages`, `task_results`, `reviews` (paths/status only; `_path` columns). Do not use `010_*` (003 owns it).
- [ ] 01.2 Create `skillgrid-cli/internal/mnemonic/files/content.go` — `ContentPlane` Write/Read under `{project}/.skillgrid/files/{tasks,messages,reviews}/…`; FS-first then caller SQL; delete file on SQL failure; no-op post-write hook seam for later 003 (no L0/L1/L2).
- [ ] 01.3 Add glossary terms Team Task, Team Member, Agent Review, Inbox to `docs/skillgrid/agents/glossary/technical.md`.
- [ ] 01.4 Verify WHAT: store open applies `009_*` via embed + `index_meta` without rewriting observations; SQL failure after FS write leaves no orphan file.
