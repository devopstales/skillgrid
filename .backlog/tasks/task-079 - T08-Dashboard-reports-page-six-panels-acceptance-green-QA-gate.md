---
id: TASK-079
title: 'T08 Dashboard /reports page: six panels + acceptance green + QA gate'
status: needs-triage
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 18:49'
labels:
  - quality-report
milestone: m-10
dependencies:
  - TASK-078
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
Scope: Extract SevBadge to components/ui/Badges.tsx (exported; SecurityPage re-imports). Six panels (CVE/SAST/DAST/duplication/debt/coverage; d3 heat bar for coverage). ReportsPage renders static always, DastPanel only when rep.dynamic?.length. Confirm acceptance.feature (24 scenarios) green; full go test ./... + npx vitest run; QA gate (L3).

SATISFIES: dashboard-reports-page, dashboard-dast-conditional, dashboard-dast-unavailable

Build shape: tracer thread. Fails-when: see tasks.md TICKET-08.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
