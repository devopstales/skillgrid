---
id: TASK-070.12
title: 'TICKET-12: Teaching graph page (single page, tour, locals)'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:35'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.09
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Node += lines; pure on-demand locals resolver (never writes); tree panel synced both ways; typed connections with source lines + preview; search-to-focus; filters + newcomer cap; concept toggle; client-side tour. No new endpoint.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 G14 PASS; go vet clean; route-table diff shows at most the one planned route
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
