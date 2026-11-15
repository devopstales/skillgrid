---
id: TASK-116.21
title: 'TICKET-21: references/goldfish-review-loop.md: sharpening loop'
status: done
assignee: []
created_date: '2026-11-12 09:18'
updated_date: '2026-11-12 11:27'
labels: []
dependencies:
  - TASK-116.20
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
Scope: Create references/goldfish-review-loop.md: multi-round (cap 5), convergence (2 consecutive clean rounds), settled vs fixed, churn, turn-taking, frozen-candidate re-check.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 21
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
