---
id: TASK-125
title: '[FEATURE] Application commands and queries (APP)'
status: done
assignee: []
created_date: '2026-11-12 16:02'
updated_date: '2026-11-13 10:05'
labels: []
milestone: m-12
dependencies:
  - TASK-124
references:
  - .skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/briefing.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: not yet implemented. Expected State: listTasks, moveTask, createTask, editTask, linkTasks over ports.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 moveTask updates status file; linkTasks writes both sides
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Tests pass
<!-- DOD:END -->
