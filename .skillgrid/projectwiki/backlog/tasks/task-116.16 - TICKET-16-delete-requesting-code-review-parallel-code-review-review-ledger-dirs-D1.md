---
id: TASK-116.16
title: >-
  TICKET-16: delete requesting-code-review, parallel-code-review, review-ledger
  dirs (D1)
status: done
assignee: []
created_date: '2026-11-12 09:17'
updated_date: '2026-11-12 11:04'
labels: []
dependencies:
  - TASK-116.14
  - TASK-116.15
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
Scope: Precondition: zero live citations outside the 3 dirs. git rm -r requesting-code-review/ parallel-code-review/ review-ledger/; verify D-entry count. One-way D1 (approved by user execute 2026-11-12).
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 16
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
