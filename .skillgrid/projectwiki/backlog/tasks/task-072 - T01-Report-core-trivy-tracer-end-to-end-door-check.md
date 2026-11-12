---
id: TASK-072
title: 'T01 Report core: trivy tracer end-to-end (door check)'
status: done
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 19:35'
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
Scope: report pkg skeleton (Service/Report/SourceResult/CoverageRow, Collect trivy source + fail-open shell), emit.JUnit (zero-finding empty testsuites) + emit.JSON, reportConfigFrom (dast.url from config.yaml), scan.FindingsByScan + scan.ExecRunner, GET /report/{fmt} (write-auth), minimal /reports page + route (CVE panel). DOOR CHECK: GET /report/json on Go fixture returns trivy section + 200.

SATISFIES: report-aggregates-static-sources, report-source-error-is-fail-open, junit-xml-per-tool-testsuite, junit-xml-zero-finding-source-is-empty-testsuite, json-report-shape, cve-trivy-report, report-http-endpoints, dashboard-reports-page

Build shape: tracer thread. Fails-when: see tasks.md TICKET-01.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
