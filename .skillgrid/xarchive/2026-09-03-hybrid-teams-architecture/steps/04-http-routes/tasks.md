# Tasks: 001-hybrid-teams-architecture — Step 04-http-routes

> Goal: Add HTTP routes for task/message/review CRUD (auth-gated writes).
> Depends on: 02-service-facade

## Review Workload Forecast
See change-level forecast in `01-schema/tasks.md` (ask-on-risk; High; Chained PRs Yes; Decision needed Yes; Chain strategy pending).

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 04.1 Modify `skillgrid-cli/internal/mnemonic/http/server.go` — add `/teams/…` CRUD for teams/tasks/messages/reviews (not `/memory/reviews`).
- [ ] 04.2 Wrap all team write handlers with existing `requireWriteAuth`; leave GETs open when `SKILLGRID_HTTP_TOKEN` is set.
- [ ] 04.3 Confirm no route collision with existing `POST /memory/reviews/{id}` in the same file.
- [ ] 04.4 Verify WHAT: unauthenticated POST → 401 when token set; authenticated write succeeds; GETs stay open.
