# Tasks: 001-hybrid-teams-architecture — Step 02-service-facade

> Goal: Add Service methods (spawn, pull, read, submit, review, done) following openProject pattern.
> Depends on: 01-schema

## Review Workload Forecast
See change-level forecast in `01-schema/tasks.md` (ask-on-risk; High; Chained PRs Yes; Decision needed Yes; Chain strategy pending).

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 02.1 Create `skillgrid-cli/internal/mnemonic/service/teams.go` with `SpawnTask` (brief via ContentPlane → SQL `pending`, return id).
- [ ] 02.2 Add `PullNextTask` (claim highest-priority unassigned), `ReadTask` (brief.md from disk).
- [ ] 02.3 Add `SubmitOutput` (`output.md` + status `review_spec` + `task_results`), `SubmitReview` (review `.md` + `reviews.passed`), `MarkDone` (status `done`).
- [ ] 02.4 Modify `skillgrid-cli/internal/mnemonic/service/service.go` to wire teams via `openProject` / project handle as needed.
- [ ] 02.5 Verify WHAT: spawn returns id; pull claims priority order; output→`review_spec`; review sets `passed`; mark done→`done` + `task_results` row.
