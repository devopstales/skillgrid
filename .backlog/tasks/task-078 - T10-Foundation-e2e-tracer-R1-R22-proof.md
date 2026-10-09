---
id: TASK-078
title: T10 Foundation e2e tracer (R1-R22 proof)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:11'
labels:
  - skill-pi
  - tracing
  - integration
dependencies:
  - TASK-073
  - TASK-074
  - TASK-075
  - TASK-077
  - TASK-076
  - TASK-072
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/e2e.test.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Integration test driving a real skill-pi launch in a temp skillgrid repo, asserting the full R1-R22 surface. Wire the foundation modules together (integration, not new code). TRACER A COMPLETE GATE — must pass before T11 (swarm migration) starts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A skill-pi launch in a temp repo reads global config from ~/.skill-pi/ + project config through the cwd shim (R1/R2)
- [ ] #2 Re-materializes the shim (R3); materializes a real gitignored .skill-pi/ (R4)
- [ ] #3 Exposes the five ports' tools (R5); gates edit via permission (R6); exposes ticket tool (R7)
- [ ] #4 Renders context commands (R8); renders TUI at breakpoint (R9)
- [ ] #5 Exports /memory-sync JSONL (R10); lists synced skills (R11); honors version pin (R12)
- [ ] #6 Installs via Go installer (R13); forwards pi flags + owns subcommands (R14); emits --where (R15)
- [ ] #7 Keeps SYSTEM.md a delta (R16); overrides base URL (R17)
- [ ] #8 Spawns with skip-version + no-skills args (R18); exposes mcp__* tools (R19); persists trust (R20); injects date/time (R21); re-spawns on exit 75 (R22)
- [ ] #9 ~/.pi/ and <project-root>/.pi/ are byte-identical before and after
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
