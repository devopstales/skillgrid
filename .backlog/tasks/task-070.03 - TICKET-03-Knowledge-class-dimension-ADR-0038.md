---
id: TASK-070.03
title: 'TICKET-03: Knowledge-class dimension + ADR-0038'
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
Nullable knowledge_class + index; inferKnowledgeClass on save/update; re-runnable backfill; mem_search filter + labels; ADR-0038. ONE-WAY migration: snapshot first, STOP for approval.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestSave_KnowledgeClassInfer PASS; unfiltered search unchanged
- [ ] #2 Migration 051 applies once; ADR-0038 filed with LOCKED row
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
