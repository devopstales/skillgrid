---
id: TASK-070.11
title: 'TICKET-11: okf-lint.mjs knowledge lint script + fixtures'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:34'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies: []
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Read-only Node guard (exit 0/1/2, SCOPE line, yaml only): 9 checks incl. provenance-required + no-wikilink syntax. Clean + drifted fixtures. Lands before TICKET-14 executes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 G13: clean exits 0, drifted exits 1 with all four finding classes
- [ ] #2 Fixtures unmodified by runs (read-only proof)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
