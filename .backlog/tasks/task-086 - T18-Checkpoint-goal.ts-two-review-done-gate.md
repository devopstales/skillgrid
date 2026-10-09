---
id: TASK-086
title: T18 Checkpoint (goal.ts) + two-review done gate
status: needs-triage
assignee: []
created_date: '2026-10-09 13:14'
labels:
  - skill-pi
  - swarm
dependencies:
  - TASK-082
  - TASK-084
  - TASK-085
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/goal.ts
  - skill-pi/extensions/skillgrid-squad.test.ts
  - skill-pi/extensions/skillgrid-squad.ts
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement goalMark(taskID, status) (atomic write: temp + rename, shape {current, status, updated}) + goalResume() (read + validate, corrupt/absent -> null). Implement bothReviewsPassed(taskID) (calls squad_reviews, true only if both a passing spec_compliance AND a passing code_quality review exist). Gate squad_mark_done on it. On a passing gate, trigger commitWorktree for worktree:true tasks (commit-on-done is a code guarantee, not conversational). Tracer B, Wave 13.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 goalMark then goalResume round-trips {current, status}
- [ ] #2 A corrupt file (truncated JSON) -> goalResume returns null (not a throw)
- [ ] #3 Seed a task with only a passing spec_compliance review -> bothReviewsPassed is false and squad_mark_done refuses
- [ ] #4 Add a passing code_quality -> bothReviewsPassed true and squad_mark_done proceeds
- [ ] #5 A code_quality with passed=false -> still false
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
