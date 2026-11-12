---
id: TASK-116.09
title: >-
  TICKET-09: 8 code-review/reviewers templates (security merged, gap canon
  cited)
status: needs-triage
assignee: []
created_date: '2026-11-12 09:16'
updated_date: '2026-11-12 09:29'
labels: []
dependencies:
  - TASK-116.08
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
Scope: Create 8 code-review/reviewers templates: code-reviewer, spec-reviewer, edge-case-hunter, verification-gap (cites shared canon), security (requesting 230-line OWASP template wins, R40), accessibility, performance, red-team.
Wave: (see tasks.md)
Blueprint: .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md -> Task 09
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
