---
id: TASK-070.05
title: 'TICKET-05: Confidence-gated capture + regression'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:33'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.03
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deterministic ClassifyCapture (10 patterns, no LLM); mem_capture_suggest threshold 0.75, never persists; agent confirms via mem_save. Then full module suite green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestCaptureSuggest PASS incl. strong/weak/negative matrix
- [ ] #2 Full go test ./internal/... green (G7)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
