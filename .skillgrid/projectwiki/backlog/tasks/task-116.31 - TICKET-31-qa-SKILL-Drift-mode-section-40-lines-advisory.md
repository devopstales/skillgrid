---
id: TASK-116.31
title: 'TICKET-31: qa SKILL: Drift mode section (<40 lines, advisory)'
status: needs-triage
assignee: []
created_date: '2026-11-12 09:20'
updated_date: '2026-11-12 09:30'
labels: []
dependencies:
  - TASK-116.28
  - TASK-116.29
  - TASK-116.30
references:
  - >-
    .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/tasks.md
  - >-
    .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/briefing.md
documentation:
  - >-
    .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md
parent_task_id: TASK-116
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Scope: qa/SKILL.md: add ## Drift mode (advisory) section (<40 lines): invocation, intake, 4 categories one-liners, sequential rule, artifact paths, not-a-gate, reference pointers. Size budget exits 0.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 31
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
