---
id: TASK-038.10
title: 'TICKET-10: Observations in the session event feed'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:47'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
parent_task_id: TASK-038
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GET /sessions/{id}/events gains observations; list gains count. SATISFIES observation-row-links-to-full-text.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Saved observation appears with id type title; list count is 1
- [ ] #2 OpenAPI documents the additive fields
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/http -run TestSessionEvents is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Extend TestSessionEvents so a saved observation appears; list count is 1.
2. Add observations[] to GET /sessions/{id}/events and observations count on the list.
3. Update openapi.yaml.
<!-- SECTION:PLAN:END -->
