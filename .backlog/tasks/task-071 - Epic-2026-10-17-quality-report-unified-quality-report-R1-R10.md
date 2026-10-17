---
id: TASK-071
title: 'Epic: 2026-10-17-quality-report (unified quality report, R1-R10)'
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
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
One command produces a unified quality report (CVE, SAST, DAST, duplication, code debt, test coverage) for any project skillgrid is installed in. Report service reuses the findings store (ADR-0030) + new coverage table (060, ADR-0038); emits JUnit XML + JSON; served by GET /report/{fmt} + scan_report MCP tool; visualized on /reports dashboard. 8 tickets, tracer-thread build, fail-open (ADR-0016).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
