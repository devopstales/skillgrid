---
id: TASK-070.14
title: 'TICKET-14: Bundle moves + sweep + ADR-0039 (STOP-gated)'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:35'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.11
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
git mv 37 ADRs to decisions/, archive to xarchive, splits, root index/log/reverse-index, frontmatter pass, full reference sweep, ADR-0039 amending ADR-0012. Two zone-clean commits. STOP: LOCKED OK required.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 okf-lint exit 0 DRIFT clean SCOPE COMPLETE; LOCKED resolves; history intact
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
