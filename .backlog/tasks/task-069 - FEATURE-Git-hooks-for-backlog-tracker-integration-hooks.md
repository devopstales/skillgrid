---
id: TASK-069
title: '[FEATURE] Git hooks for backlog + tracker integration (hooks)'
status: in-progress
assignee: []
created_date: '2026-10-10 07:39'
updated_date: '2026-10-10 07:45'
labels: []
dependencies: []
references:
  - hooks/checkpoint-state.js
  - git-hooks/pre-commit.js
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State:
- Ticket lifecycle (Refs/Closes footers, status transitions) and backlog completeness gate live only in skill prose; no git-hook enforcement.
- Beads-style lifecycle hooks (prepare-commit-msg, pre-push, post-merge) do not exist in Skillgrid.
Expected State:
- Shared ticketing lib + pre-commit completeness guard + commit-msg footer validation + prepare-msg injector seam + pre-push outgoing check + post-merge doctor, all fail-open, tracker-selected via ticketing.type.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pre-commit blocks staged backlog tasks missing type/references/DoD/plan
- [ ] #2 commit-msg validates Refs/Closes footer ID format when present, never requires
- [ ] #3 prepare-commit-msg + pre-push + post-merge shims exist and fail open
- [ ] #4 Go hook tests cover guard + footer + prepare-msg
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Tests pass
- [ ] #7 No new warnings introduced
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented in 803cba1a: hooks/ticketing.js + ticketing-guard/pre-push/post-merge, checkpoint-state wiring (guard/guard-msg/prepare-msg/pre-push/post-merge-check), git-hooks shims (prepare-commit-msg/pre-push/post-merge), Go tests green (install pkg ok), docs updated. Remaining: review + AC check-off + close.
<!-- SECTION:NOTES:END -->
