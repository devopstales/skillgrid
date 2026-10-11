---
id: TASK-070.02
title: 'TICKET-02: Health recommendations (5 rules)'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:33'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.01
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Extend pure healthRecommendations: dedup HIGH, embeddings MEDIUM with no-nag gate, stale MEDIUM, cleanup LOW, review LOW. Stable ordering, no new DB queries.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestHealthRecommendations PASS with all 5 rules and severities
- [ ] #2 No new DB queries; embedder-off produces no nag
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
