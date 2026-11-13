---
id: TASK-130
title: '[FEATURE] File watcher refresh (INFRA)'
status: done
assignee: []
created_date: '2026-11-12 17:32'
updated_date: '2026-11-13 10:05'
labels: []
milestone: m-12
dependencies:
  - TASK-126
references:
  - .skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/briefing.md
priority: low
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: not yet implemented. Expected State: Debounced watcher re-renders views.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 external edit refreshes within 1s
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
