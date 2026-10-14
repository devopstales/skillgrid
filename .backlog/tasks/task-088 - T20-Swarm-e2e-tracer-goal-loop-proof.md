---
id: TASK-088
title: T20 Swarm e2e tracer (goal-loop proof)
status: done
assignee: []
created_date: '2026-10-09 13:15'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - swarm
  - integration
dependencies:
  - TASK-085
  - TASK-086
  - TASK-087
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/skillgrid-squad.test.ts
  - skill-pi/system/APPEND_SYSTEM.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Integration test against a real temp store + temp git repo: lead decomposes a goal -> squad_spawn_task with goal_id -> swarmSpawn worker (worktree) -> squad_submit_output -> reviewer spec -> reviewer code -> squad_mark_done -> assert goal shows 1 done + branch exists. Write the lead's goal-loop prompt (the conversational driver, R7). TRACER B COMPLETE.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The task's goal_id is set
- [ ] #2 The goal's per-status count shows done:1
- [ ] #3 The branch worktree-agent-swarm-<taskID> exists with a commit
- [ ] #4 The checkpoint progress.json reflects done
- [ ] #5 The lead's goal-loop prompt encodes R7's sequence + checkpoint discipline
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
