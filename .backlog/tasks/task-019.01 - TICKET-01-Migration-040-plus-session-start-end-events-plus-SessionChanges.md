---
id: TASK-019.01
title: 'TICKET-01: Migration 040 plus session start/end events plus SessionChanges'
status: done
assignee: []
created_date: '2026-09-21 15:10'
updated_date: '2026-09-21 15:27'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/briefing.md
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Tracer thread end-to-end: 040 schema, start/end event writes with commit range, ordered read.

Current: sessions table has no commit range, no counters, no events stream.

Expected: 040 adds agent_session_id, from_commit, to_commit plus 6 counters and session_events table; SessionStart writes session_start seq 0 with HEAD; SessionEnd writes session_end with HEAD; SessionChanges returns ordered events plus range.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestMigration040SessionEvents passes
- [ ] #2 TestSessionStartEndEvents passes
- [ ] #3 SessionChanges(sid) returns start/end in order with from/to commits
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
1. Write failing schema test (RED)
2. Write 040_session_events.sql (GREEN)
3. Wire SessionStart/End event writes plus SessionChanges (RED then GREEN)
4. Full store plus memory suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 1 review: implementer done. Coordinator verified on live tree: TestMigration040SessionEvents PASS, TestSessionStartEndEvents PASS, full store+memory suites green per implementer. Notable: quoted commit column in 040 (reserved word, matches 039 idiom). Tree recovery commit fde167e closed the hook-layout move. Ready for Wave 2.
<!-- SECTION:NOTES:END -->
