---
id: TASK-079
title: T11 goals table + tasks.goal_id migration (ADR-0039)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:11'
labels:
  - skill-pi
  - swarm
  - go
  - migration
dependencies:
  - TASK-078
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
  - .skillgrid/artifacts/04-adr-0039-*.md
modified_files:
  - mnemonic/internal/store/migrations/056_team_goals.sql
  - mnemonic/internal/integration/step07_team_goals_test.go
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create migration 056_team_goals.sql: goals table, tasks.goal_id nullable FK, tasks.worktree_branch nullable column, both indexes. Integration test asserting schema. Tracer B, Wave 8. Reversibility: one-way (ADR-0039: schema migration, table + index shape).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The goals table exists
- [ ] #2 Insert a goal, insert a task with goal_id set — the task's goal_id column reads back the goal id
- [ ] #3 A task with no goal has goal_id IS NULL
- [ ] #4 The migration file is numbered above the newest existing migration (verified at exec time)
- [ ] #5 Fails when: go test ./internal/integration/ -run TestTeamGoals -v exits non-zero
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
