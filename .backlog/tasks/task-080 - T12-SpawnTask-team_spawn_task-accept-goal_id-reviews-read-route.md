---
id: TASK-080
title: T12 SpawnTask + team_spawn_task accept goal_id; reviews read route
status: needs-triage
assignee: []
created_date: '2026-10-09 13:12'
labels:
  - skill-pi
  - swarm
  - go
dependencies:
  - TASK-079
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - mnemonic/internal/service/teams.go
  - mnemonic/internal/mcp/tools_teams.go
  - mnemonic/internal/http/teams.go
  - mnemonic/internal/integration/step07_team_goals_test.go
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add GoalID *string to SpawnTask param struct; include goal_id in the INSERT. Add optional goal_id param to team_spawn_task MCP + HTTP. Add ListReviews(ctx, taskID) service method + GET /teams/tasks/{id}/reviews route (defect A: the two-review gate's read path). Tracer B, Wave 9.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SpawnTask with GoalID set -> task's goal_id equals the goal id; SpawnTask with GoalID nil -> goal_id IS NULL
- [ ] #2 Submit a spec_compliance review and a code_quality review, then ListReviews(taskID) returns both with the right ReviewType/Passed
- [ ] #3 Fails when: go test ./internal/integration/ -run 'TestSpawnByGoal|TestListReviews' -v exits non-zero
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
