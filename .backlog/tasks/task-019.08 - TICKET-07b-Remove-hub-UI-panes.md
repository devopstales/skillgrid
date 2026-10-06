---
id: TASK-019.08
title: 'TICKET-07b: Remove hub UI panes'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies:
  - TASK-019.06
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: medium
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Delete HandoffsPane.tsx plus ChangesPane.tsx, strip handoff calls from api.ts, fix remaining UI references.

Current: sessions UI renders hub-backed panes.

Expected: UI builds with no handoff-pane imports; no handoff references under skillgrid-ui/src/features/sessions/.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 UI builds clean; sessions dir has no handoff references
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Failing reference check (RED)
2. Delete panes plus strip api.ts (GREEN)
3. UI build green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 4 review: 91a2d28 verified (panes deleted, api.ts stripped, tsc + 3/3 vitest green per implementer). Sessions dir handoff-free.
<!-- SECTION:NOTES:END -->
