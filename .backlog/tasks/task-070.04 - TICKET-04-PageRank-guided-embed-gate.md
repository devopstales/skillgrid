---
id: TASK-070.04
title: 'TICKET-04: PageRank-guided embed gate'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:33'
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
Centrality rank from call/reference graph; --embed-top flag default 100; gate=50 embeds at most half plus docs/recent carve-out; counts reported.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestIndex_EmbedTop PASS with carve-out honored
- [ ] #2 Gate 100 reproduces ungated vector set
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
