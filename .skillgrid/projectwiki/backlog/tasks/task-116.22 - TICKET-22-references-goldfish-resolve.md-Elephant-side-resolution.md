---
id: TASK-116.22
title: 'TICKET-22: references/goldfish-resolve.md: Elephant-side resolution'
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
Scope: Create references/goldfish-resolve.md: 4 actions as adapter over code-review triage + receiving-code-review buckets + doc deltas, 4 rules, Elephant context, interactive gate (5 options), parked findings (R17), ledger round.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 22
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
