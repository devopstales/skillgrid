---
id: TASK-087
title: T19 Dashboard goal-progress view
status: needs-triage
assignee: []
created_date: '2026-10-09 13:14'
labels:
  - skill-pi
  - swarm
  - ui
dependencies:
  - TASK-081
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - <located dashboard route/component>
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Locate the existing dashboard surface (grep skillgrid-ui/ + skillgrid-cli/). Add a goal list (from GET /teams/goals) + goal detail (from GET /teams/goals/{id}/tasks) showing per-status counts + task list with worktree_branch. Route goal_id IS NULL tasks to a "(no goal)" bucket. Tracer B, Wave 13. Dashboard location is a find at exec time.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A goal with tasks in pending/in-progress/review_spec renders the 3 counts
- [ ] #2 A goal-less task renders under '(no goal)' and is not lost
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
