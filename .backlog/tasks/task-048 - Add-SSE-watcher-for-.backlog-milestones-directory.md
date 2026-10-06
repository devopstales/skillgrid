---
id: TASK-048
title: Add SSE watcher for .backlog/milestones directory
status: ready-for-agent
assignee: []
created_date: '2026-10-06 13:44'
labels: []
milestone: m-6
dependencies:
  - TASK-047
priority: medium
type: enhancement
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The SSE stream at `/tracker/stream` (server_tracker_stream.go) only watches `.backlog/tasks/` via fsnotify. Edits to milestone files (rename, create, delete, archive) do not trigger a board refresh, so the kanban board can show stale milestone groupings until the page is manually reloaded.

**Scope:**
1. Add a second fsnotify watcher for `.backlog/milestones/` in the SSE handler
2. Emit a `milestones-changed` SSE event type when milestone files change
3. Update the TS `hooks.ts` `useTrackerBoard` to re-fetch on `milestones-changed` events (in addition to `tasks-changed`)
4. Handle the case where `.backlog/milestones/` does not exist (skip watcher, no error)
5. Unit test for the watcher wiring

**Acceptance criteria:**
- Creating/renaming/deleting a milestone file triggers SSE `milestones-changed` event
- UI re-fetches and re-renders board groups on milestone changes
- Missing milestones directory does not cause errors
- SSE handler still works when only tasks dir exists
- Go tests pass
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SSE emits milestones-changed event on milestone file changes
- [ ] #2 UI re-fetches and re-renders on milestones-changed
- [ ] #3 Missing milestones directory is handled gracefully
- [ ] #4 Existing tasks-changed SSE still works
- [ ] #5 Go tests pass
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
