---
id: TASK-076
title: T08 Installer extension + --where + trust persistence
status: done
assignee: []
created_date: '2026-10-09 13:10'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - tracing
  - go
dependencies:
  - TASK-070
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skillgrid-cli/
  - skill-pi/bin/skill-pi.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Extend `skillgrid-cli` (Go) install/sync-repo/setup to copy skill-pi/ -> ~/.skill-pi/, npm ci the pinned pi, materialize <project>/.skill-pi/ + shim, put skill-pi on PATH. Confirm --where + trust wiring from T02. Tracer A, Wave 5.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A stubbed npm ci that exits non-zero makes the install report failure (no ~/.skill-pi/node_modules/@earendil-works/pi-coding-agent marked installed)
- [ ] #2 A successful stubbed npm ci + copy + materialization produces ~/.skill-pi/node_modules/ at the pinned version, <project>/.skill-pi/ materialized, the shim built, and skill-pi on PATH
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
