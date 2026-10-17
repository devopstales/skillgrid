---
id: TASK-078
title: T07 scan_report MCP tool + HTTP/MCP parity
status: needs-triage
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 18:49'
labels:
  - quality-report
milestone: m-10
dependencies:
  - TASK-072
  - TASK-077
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
Scope: scan_report tool (format junit|json) + registerReportTools called in BOTH Start() and NewServer(). Handler calls same Collect+emitter as HTTP route (byte-identical for same report).

SATISFIES: report-aggregates-static-sources (MCP leg), report-mcp-tool-registered, report-mcp-http-parity

Build shape: tracer thread. Fails-when: see tasks.md TICKET-07.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
