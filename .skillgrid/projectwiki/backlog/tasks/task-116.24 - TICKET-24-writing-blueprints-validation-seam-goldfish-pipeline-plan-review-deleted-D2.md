---
id: TASK-116.24
title: >-
  TICKET-24: writing-blueprints validation seam -> goldfish pipeline
  (plan-review deleted, D2)
status: done
assignee: []
created_date: '2026-11-12 09:19'
updated_date: '2026-11-12 11:47'
labels: []
dependencies:
  - TASK-116.20
  - TASK-116.21
  - TASK-116.22
  - TASK-116.23
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
Scope: git rm references/plan-review.md; rewrite SKILL.md validation block: Self-Review -> Owed-Decision Gate -> Goldfish review -> User Review Gate -> Execution Handoff; invocation params. One-way D2.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 24
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
