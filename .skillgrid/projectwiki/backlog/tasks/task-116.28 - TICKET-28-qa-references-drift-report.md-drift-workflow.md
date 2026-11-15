---
id: TASK-116.28
title: 'TICKET-28: qa/references/drift-report.md: drift workflow'
status: done
assignee: []
created_date: '2026-11-12 09:20'
updated_date: '2026-11-12 11:22'
labels: []
dependencies:
  - TASK-116.22
  - TASK-116.11
  - TASK-116.29
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
Scope: Create qa/references/drift-report.md: invocation, target artifacts (R28), sequential + source grounding (R29), filtering (R30), categorize (R27), finding review (R33, codebase-details primary), reconcile (R32), report format (R31), size rule (R35).
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 28
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
