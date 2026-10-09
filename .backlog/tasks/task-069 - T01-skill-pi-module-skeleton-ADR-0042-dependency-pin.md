---
id: TASK-069
title: T01 skill-pi/ module skeleton + ADR-0042 dependency pin
status: needs-triage
assignee: []
created_date: '2026-10-09 13:07'
labels:
  - skill-pi
  - tracing
dependencies: []
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/package.json
  - skill-pi/pi-version.txt
  - skill-pi/README.md
  - .skillgrid/artifacts/04-adr-0042-skill-pi-dependencies.md
  - .skillgrid/ASSUMPTIONS.md
  - skill-pi/version.test.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create the `skill-pi/` top-level module with `package.json` (piConfig identity + exact pi pin), `pi-version.txt`, ADR-0042, and README. Tracer A (foundation), Wave 1. Reversibility: one-way (ADR-0042 shapes installer + TUI + min-version guard).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pi-version.txt content equals the package.json dependency string for @earendil-works/pi-coding-agent (exact, no ^/~/range)
- [ ] #2 ADR-0042 is in .skillgrid/artifacts/ and indexed in .skillgrid/ASSUMPTIONS.md section LOCKED
- [ ] #3 README names the min pi version for piConfig.configDir
- [ ] #4 Fails when: node --test skill-pi/version.test.js exits non-zero
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
