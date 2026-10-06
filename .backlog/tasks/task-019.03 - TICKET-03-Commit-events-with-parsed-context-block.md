---
id: TASK-019.03
title: 'TICKET-03: Commit events with parsed context block'
status: done
assignee: []
created_date: '2026-09-21 15:10'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies:
  - TASK-019.01
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Post-commit record path writes commit events (sha plus parsed Task/Decisions/Remaining/Tried); block-less commits still recorded.

Current: change_snapshots holds the per-commit index (Hub).

Expected: commit event per work-unit commit on the active session; empty payload when no block; clean error with no repo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestCommitEventParsesContextBlock passes for block and block-less commits
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
1. Failing parse test (RED)
2. Record path with block parser (GREEN)
3. Suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 2 review: 64a9bbe verified (8/8 subtests PASS on integrated tree). Append-only changes.go respected. Trigger wiring deferred to later wave per brief.
<!-- SECTION:NOTES:END -->
