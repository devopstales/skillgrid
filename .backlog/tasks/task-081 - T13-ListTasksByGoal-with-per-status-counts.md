---
id: TASK-081
title: T13 ListTasksByGoal with per-status counts
status: needs-triage
assignee: []
created_date: '2026-10-09 13:12'
labels:
  - skill-pi
  - swarm
  - go
dependencies:
  - TASK-080
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - mnemonic/internal/service/goals.go
  - mnemonic/internal/http/teams.go
  - mnemonic/internal/integration/step07_team_goals_test.go
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create mnemonic/internal/service/goals.go: ListGoals (per-status counts) + ListTasksByGoal (returns TaskRow{ID,Title,Status,WorktreeBranch}). Add GET /teams/goals + GET /teams/goals/{id}/tasks HTTP routes. Tracer B, Wave 10.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A goal with tasks in 3 statuses -> ListGoals returns the goal with correct per-status counts
- [ ] #2 ListTasksByGoal returns the goal's tasks (not the NULL-goal one)
- [ ] #3 A task with goal_id IS NULL appears under '(no goal)' and is not lost
- [ ] #4 Fails when: go test ./internal/integration/ -run TestGoalProgressQuery -v exits non-zero
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
