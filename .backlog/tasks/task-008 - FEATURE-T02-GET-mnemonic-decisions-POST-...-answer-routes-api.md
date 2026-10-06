---
id: TASK-008
title: '[FEATURE] T02 GET /mnemonic/decisions + POST .../answer routes (api)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:49'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-5
dependencies:
  - TASK-007
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
documentation:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement the read route (filter type=decision + content.state=pending from GET /mnemonic/memories output, ascending created_at, return {decisions,total}) and the write route (parse {answer,rationale?,answerer?}, require content.state present+valid, set state=answered+answer+rationale, actor=user:<name> via UpdateContent, 404 bad id, 409 missing/invalid content.state, idempotent no-op when already answered). Add UpdateContent to memory with version append. Register both routes in server.go before registerUIRoutes().

Current State:
- No decision routes; no memory.UpdateContent.

Expected State:
- TICKET-01 tests now pass.
- Live skillgrid serve + curl round-trip: seed type=decision/state=pending via mem_save, GET /mnemonic/decisions lists it, POST .../answer flips it, re-GET shows state=answered with the answer, observation_versions gained a row, re-POST no-op, bad id 404, memory without content.state 409.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TICKET-01 (TASK-007) tests now pass
- [ ] #2 go test -count=1 ./internal/mnemonic/http/... passes
- [ ] #3 Live round-trip: list -> answer -> state=answered -> version row -> idempotent re-answer -> 404/409 edge cases
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test -count=1 ./internal/mnemonic/http/... passes
- [ ] #7 Live curl round-trip demonstrated
- [ ] #8 Routes registered in server.go
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Run the Task 1 door check (mem_search over type=decision on a live store) before committing to the filter
2. Implement GET /mnemonic/decisions in internal/mnemonic/http/decisions.go (filter + ascending + {decisions,total})
3. Add memory.UpdateContent (content JSON replace + actor + observation_versions append + revision_count bump) with tests in memory/governance.go
4. Implement POST /mnemonic/decisions/{id}/answer (state flip + idempotent + 404/409)
5. Register both routes in server.go before registerUIRoutes()
6. Verify go test + live curl round-trip
<!-- SECTION:PLAN:END -->
