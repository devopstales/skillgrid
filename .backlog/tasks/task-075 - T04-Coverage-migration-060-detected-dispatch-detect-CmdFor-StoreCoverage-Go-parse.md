---
id: TASK-075
title: >-
  T04 Coverage: migration 060 + detected dispatch (detect + CmdFor +
  StoreCoverage + Go parse)
status: done
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 20:15'
labels:
  - quality-report
milestone: m-10
dependencies: []
references:
  - .skillgrid/change-specs/2026-10-17-quality-report/briefing.md
  - .skillgrid/change-specs/2026-10-17-quality-report/blueprint.md
  - .skillgrid/change-specs/2026-10-17-quality-report/tasks.md
  - .skillgrid/change-specs/2026-10-17-quality-report/acceptance.feature
documentation:
  - .skillgrid/change-specs/2026-10-17-quality-report/blueprint.md
  - .skillgrid/artifacts/04-adr-0038-unified-quality-report.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Scope: 060_coverage.sql (coverage table + index), DetectKind (manifest-based), CmdFor (5 arms), StoreCoverage (single signature ctx,db,project,rows,now; upsert), parseCoverGo + parseCover(k,data) dispatcher. ONE-WAY DOOR (migration 060).

SATISFIES: coverage-dispatch-go

Build shape: tracer thread. Fails-when: see tasks.md TICKET-04.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
