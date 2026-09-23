---
id: TASK-019.10
title: 'TICKET-10: Migration 041 drops old tables'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-09-22 08:31'
labels: []
dependencies:
  - TASK-019.07
  - TASK-019.06
  - TASK-019.09
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Drop change_snapshots, checkpoints, handoff_refs, session_handoffs, session_archives plus six indexes. ONE-WAY: Hub UI history gone (reconstructable from git log only); needs human checkpoint before commit.

Current: 039/019 tables coexist with the events layer.

Expected: Hub-era DB migrates with the five tables absent and sessions/session_events intact; store suite green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Drop test passes; full store suite green
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
1. Failing drop test (RED)
2. STOP: get human approval (one-way drop)
3. Write 041 migration (GREEN)
4. Store suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
041 drop migration committed 850e7f7; 5 tables + 6 indexes dropped, data-intact replay test green; one-way human checkpoint approved
<!-- SECTION:NOTES:END -->
