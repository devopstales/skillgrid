---
id: TASK-074
title: T03 akca parser + scannerCommands arm
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
Scope: parseAkca (evidence findings[] -> Finding rule id+file+proof), Parse arm, scannerCommands[akca], toolSeverityMaps[akca] (ERROR->HIGH, WARNING->MEDIUM).

SATISFIES: dast-auto-from-config (akca leg)

Build shape: tracer thread. Fails-when: see tasks.md TICKET-03.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
