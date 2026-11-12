---
id: TASK-116.13
title: >-
  TICKET-13: citation migration: ~25 sites to code-review + shared ledger
  reference
status: done
assignee: []
created_date: '2026-11-12 09:17'
updated_date: '2026-11-12 10:58'
labels: []
dependencies:
  - TASK-116.04
  - TASK-116.05
  - TASK-116.06
  - TASK-116.09
  - TASK-116.10
  - TASK-116.12
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
Scope: Migrate every live citation of requesting-code-review/parallel-code-review/review-ledger-skill to code-review / _shared/craft/review-ledger.md (~25 sites incl. docs + projectwiki); receiving-code-review gains review-discipline cite; delete TICKET-08 interim pointer; verify grep-zero.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 13
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
