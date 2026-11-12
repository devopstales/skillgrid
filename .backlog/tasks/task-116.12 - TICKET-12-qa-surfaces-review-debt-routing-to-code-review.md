---
id: TASK-116.12
title: 'TICKET-12: qa surfaces review debt + routing to code-review'
status: needs-triage
assignee: []
created_date: '2026-11-12 09:17'
updated_date: '2026-11-12 09:29'
labels: []
dependencies:
  - TASK-116.11
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
Scope: qa/SKILL.md: named Surface review debt step (ledger Step 5 carried in, cites shared reference; WARNING on file overlap), routing + When-NOT-to-use cite code-review, R17 goldfish-parked decision-debt line.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 12
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
