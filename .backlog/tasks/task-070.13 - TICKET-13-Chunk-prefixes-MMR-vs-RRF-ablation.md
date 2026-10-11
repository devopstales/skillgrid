---
id: TASK-070.13
title: 'TICKET-13: Chunk prefixes + MMR-vs-RRF ablation'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:35'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.04
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deterministic path-symbol-scope prefix on embedding text only; ablation numbers to findings.md win-or-lose; rank.go untouched (a ranker change is a NEW ticket).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestChunkPrefix_Context PASS; rank.go diff empty
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
