---
id: TASK-073
title: T02 jscpd parser + scannerCommands arm (duplication source)
status: done
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 21:39'
labels:
  - quality-report
milestone: m-10
dependencies: []
references:
  - .skillgrid/specs/2026-10-17-quality-report/briefing.md
  - .skillgrid/specs/2026-10-17-quality-report/blueprint.md
  - .skillgrid/specs/2026-10-17-quality-report/tasks.md
  - .skillgrid/specs/2026-10-17-quality-report/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-17-quality-report/blueprint.md
  - .skillgrid/artifacts/04-adr-0038-unified-quality-report.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Scope: parseJscpd (clones -> Finding rule_id=duplication), Parse arm, scannerCommands[jscpd]. jscpd already in GlobalTools (no installer change).

SATISFIES: duplication-jscpd-report

Build shape: tracer thread. Fails-when: see tasks.md TICKET-02.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Tests pass (`go test ./...` for touched packages)
- [x] #2 Lint and formatting pass
- [x] #3 Edge cases covered
- [x] #4 No new warnings introduced
- [x] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
