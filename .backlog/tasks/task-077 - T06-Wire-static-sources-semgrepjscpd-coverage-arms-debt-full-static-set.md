---
id: TASK-077
title: >-
  T06 Wire static sources (semgrep+jscpd) + coverage arms + debt + full static
  set
status: in-progress
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 20:34'
labels:
  - quality-report
milestone: m-10
dependencies:
  - TASK-072
  - TASK-073
  - TASK-075
  - TASK-076
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
Scope: collectSemgrep + collectJscpd (Start->StoreFindings->readback), parseCoverTS/Py/Rust/Java via parseCover, collectDebt (grep TODO|FIXME|XXX, walk root skip node_modules/.git/vendor), collectCoverage wired into Collect (upsert, unavailable on missing tool). Final Static = [trivy,semgrep,jscpd,debt,coverage].

SATISFIES: sast-semgrep-report, duplication-jscpd-report, coverage-dispatch-ts, coverage-dispatch-py, coverage-dispatch-rs, coverage-dispatch-java, debt-markers-report, coverage-tool-missing-unavailable

Build shape: tracer thread. Fails-when: see tasks.md TICKET-06.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
