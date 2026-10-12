---
id: TASK-070.10
title: 'TICKET-10: MCP scope block + descriptions'
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
Additive scope block (buckets_hit, per-bucket counts) on multi-bucket hits; ranking untouched; progressive-disclosure wording. Federated tests must pass UNMODIFIED.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestMemSearch_ScopeBlock PASS; federated tests green untouched
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
