---
id: TASK-116.07
title: >-
  TICKET-07: code-review/SKILL.md: merged review skill (topology + shared
  protocol)
status: needs-triage
assignee: []
created_date: '2026-11-12 09:16'
updated_date: '2026-11-12 09:29'
labels: []
dependencies: []
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
Scope: Create .agents/skills/code-review/SKILL.md: topology step (3-axis <50 lines/low-risk; core-five + conditional Accessibility/Performance >=50/high-risk; red team last; tier floor), Blindness Contract + grades A/B/C + six-value verdict table + resilience rule each exactly once, dispatch, output contract (review.md + REVIEW-PASS/BACK-TO-APPLY + ledger round).
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 07
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
