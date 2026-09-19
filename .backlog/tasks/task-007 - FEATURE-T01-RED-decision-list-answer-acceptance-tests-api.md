---
id: TASK-007
title: '[FEATURE] T01 RED: decision list + answer acceptance tests (api)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:49'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
documentation:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write the failing acceptance tests (no implementation yet) for the two backend decision routes.

Current State:
- No GET /mnemonic/decisions or POST /mnemonic/decisions/{id}/answer routes; no memory.UpdateContent.
- Decision content convention (state=pending|answered|superseded) is agreed but untested.

Expected State:
- decisions_test.go contains RED tests for: GET /mnemonic/decisions (filters type=decision + content.state=pending, ascending created_at, returns {decisions,total}) and POST /mnemonic/decisions/{id}/answer (state flip pending->answered, appends observation_versions, bumps revision_count, actor=user:<name>, idempotent no-op on re-answer, 404 bad id, 409 when content.state missing/invalid).
- go test -count=1 ./internal/mnemonic/http/... -run TestDecision fails (expected RED) but the package compiles.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 decisions_test.go compiles against a forward-declared memory.UpdateContent signature
- [ ] #2 go test -count=1 ./internal/mnemonic/http/... -run TestDecision exits non-zero (RED) with the expected assertions
- [ ] #3 Tests assert: filter type=decision+state=pending, ascending order, state flip, version append, revision_count bump, idempotent re-answer, 404, 409
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Test file compiles (no build error)
- [ ] #7 Tests are RED with the exact expected assertions
- [ ] #8 SATISFIES scenario names recorded
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 1 (door check) + Task 2 in blueprint.md for the decision content convention and route contracts
2. Add a forward declaration (or interface stub) of memory.UpdateContent so the test compiles
3. Write decisions_test.go with RED table-driven tests for the list + answer routes
4. Run go test -count=1 ./internal/mnemonic/http/... -run TestDecision; confirm RED (not a build error)
<!-- SECTION:PLAN:END -->
