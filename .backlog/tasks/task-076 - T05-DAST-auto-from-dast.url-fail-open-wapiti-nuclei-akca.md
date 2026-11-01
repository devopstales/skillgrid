---
id: TASK-076
title: T05 DAST auto from dast.url + fail-open (wapiti/nuclei/akca)
status: done
assignee: []
created_date: '2026-10-17 18:49'
updated_date: '2026-10-17 20:34'
labels:
  - quality-report
milestone: m-10
dependencies:
  - TASK-072
  - TASK-074
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
Scope: collectDAST: empty DastURL -> nil; set -> HEAD probe 3s; unreachable -> 3 unavailable; reachable -> run wapiti/nuclei/akca via scan.Service; per-tool error -> that tool error, others continue. Wire rep.Dynamic into Collect.

SATISFIES: dast-auto-from-config, dast-unreachable-fail-open, dast-unset-is-absent

Build shape: tracer thread. Fails-when: see tasks.md TICKET-05.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
