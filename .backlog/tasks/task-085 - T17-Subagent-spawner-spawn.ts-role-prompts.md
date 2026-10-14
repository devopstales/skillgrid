---
id: TASK-085
title: T17 Subagent spawner (spawn.ts) + role prompts
status: done
assignee: []
created_date: '2026-10-09 13:14'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - swarm
dependencies:
  - TASK-083
  - TASK-084
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/spawn.ts
  - skill-pi/extensions/skillgrid-squad.test.ts
  - skill-pi/extensions/skillgrid-squad.ts
  - agents/pi/roles/worker.md
  - agents/pi/roles/reviewer.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement swarmSpawn({role, taskID, worktree?}): resolve role via discoverRoles, create worktree if requested, spawn pi subagent session with role prompt + owner=taskID + cwd. Serial guard: a second swarmSpawn while one is active throws. Register swarm_spawn tool. Write full worker/reviewer role prompts. Tracer B, Wave 12.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 swarmSpawn({role:worker, taskID:t1}) -> the fake SDK received owner=t1, cwd=<repoRoot>, worker role prompt
- [ ] #2 With worktree:true -> cwd is the worktree path
- [ ] #3 A second swarmSpawn while the first is active throws 'a worker is already active'
- [ ] #4 role:nope -> unknown-role error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
