---
id: TASK-083
title: T15 Role discovery (roles.ts)
status: done
assignee: []
created_date: '2026-10-09 13:13'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - swarm
dependencies:
  - TASK-082
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/roles.ts
  - skill-pi/extensions/skillgrid-squad.test.ts
  - agents/pi/roles/worker.md
  - agents/pi/roles/reviewer.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement discoverRoles(rolesDir): Role[] (glob *.md, parse model: frontmatter, strip frontmatter from prompt). Register a swarm_role tool. Give worker.md/reviewer.md real minimal prompts. Tracer B, Wave 11.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 In a temp roles dir with a.md, b.md (with model: x), and a non-.md file: discoverRoles returns {a},{b} (not the non-.md), b.model === x
- [ ] #2 Write c.md and re-call — c appears (new file = new role, no code change)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
