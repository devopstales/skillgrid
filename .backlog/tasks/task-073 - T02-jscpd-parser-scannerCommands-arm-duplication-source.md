---
id: TASK-073
title: T02 jscpd parser + scannerCommands arm (duplication source)
status: needs-triage
assignee: []
created_date: '2026-10-17 18:49'
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
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
