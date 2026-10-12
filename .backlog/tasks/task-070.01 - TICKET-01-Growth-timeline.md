---
id: TASK-070.01
title: 'TICKET-01: Growth timeline'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:33'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies: []
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrowthTimeline(project, granularity day|week|month) via strftime bucketing; exclude soft-deleted/archived; mem_lifecycle growth action; empty store returns [].
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestGrowthTimeline_Monthly PASS, cumulative totals verified
- [ ] #2 Soft-deleted rows excluded; empty store returns []
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
