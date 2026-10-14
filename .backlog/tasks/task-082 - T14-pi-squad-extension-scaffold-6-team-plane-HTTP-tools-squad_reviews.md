---
id: TASK-082
title: T14 pi squad extension scaffold (6 team-plane HTTP tools + squad_reviews)
status: done
assignee: []
created_date: '2026-10-09 13:12'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - swarm
dependencies:
  - TASK-080
  - TASK-078
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/skillgrid-squad.ts
  - skill-pi/extensions/skillgrid-squad.test.ts
  - agents/pi/roles/worker.md
  - agents/pi/roles/reviewer.md
  - skill-pi/package.json
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Port plugins/opencode/skillgrid-squad.ts 6-tool logic to skill-pi/extensions/skillgrid-squad.ts (pi ExtensionAPI.tool() shape). Add squad_reviews read tool (calls GET /teams/tasks/{id}/reviews from T12). Add goal_id pass-through to squad_spawn_task. Create role placeholder files. Tracer B, Wave 10.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each of the 7 tools posts/gets the right endpoint with the right body (byte-compatible with the opencode plugin for the 6; squad_reviews -> GET /teams/tasks/{id}/reviews)
- [ ] #2 A mutation failure sets the in-memory failure latch
- [ ] #3 squad_spawn_task with a goal_id posts the field; without goal_id posts null/omits it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
