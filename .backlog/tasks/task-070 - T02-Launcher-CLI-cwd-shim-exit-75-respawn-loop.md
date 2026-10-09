---
id: TASK-070
title: T02 Launcher CLI + cwd shim + exit-75 respawn loop
status: needs-triage
assignee: []
created_date: '2026-10-09 13:08'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-069
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/bin/skill-pi.js
  - skill-pi/bin/skill-pi.test.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement `skill-pi/bin/skill-pi.js`: env relocation, shim build (rebuild-from-scratch every launch), chdir, spawn (not exec), exit-75 loop (capped), owned subcommands (`init`/`sync`/`version`), `--where` 8-key JSON, trust persistence, `PI_SKIP_VERSION_CHECK` + `--no-skills` args. Tracer A, Wave 2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A fake child exiting 75 is re-spawned (second spawn asserted); a child exiting 0 is not re-spawned; a child exiting 75 eleven times stops the loop (cap)
- [ ] #2 After buildShim, the shim's package.json has piConfig.configDir = .skill-pi; a new file in the real root is visible through the shim on rebuild
- [ ] #3 A missing root exits non-zero, no spawn
- [ ] #4 --where emits the 8-key JSON with configDir === <project-root>/.skill-pi and agentDir === ~/.skill-pi
- [ ] #5 version prints the pinned version without TUI; a pi flag like --session is forwarded
- [ ] #6 Spawn env has PI_SKIP_VERSION_CHECK=1 + --no-skills --no-prompt-templates --no-themes --extension <root>
- [ ] #7 Fresh run fires trust prompt; after accept, ~/.skill-pi/trust.json has a key = shim realpath; second run no prompt
- [ ] #8 Fails when: node --test skill-pi/bin/skill-pi.test.js exits non-zero
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
