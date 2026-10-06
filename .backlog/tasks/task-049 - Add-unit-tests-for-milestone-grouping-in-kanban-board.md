---
id: TASK-049
title: Add unit tests for milestone grouping in kanban board
status: ready-for-agent
assignee: []
created_date: '2026-10-06 13:44'
labels: []
milestone: m-6
dependencies:
  - TASK-047
priority: medium
type: enhancement
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The milestone grouping functions (`groupTasksByMilestone` in epicTree.ts) and the BoardView milestone row rendering were implemented in commit a15e55d3 but have no dedicated unit tests. The existing BoardView.test.tsx only tests epic/parent-child grouping, not milestone grouping.

**Scope:**
1. Add tests for `groupTasksByMilestone` in a new `epicTree.test.ts` (or extend existing test file):
   - Groups tasks by milestone value
   - Tasks without milestone are grouped under "" (renders as "Unassigned")
   - Groups are sorted alphabetically, unassigned last
   - Single milestone, multiple milestones, all unassigned
   - Empty task list returns empty groups
2. Add BoardView milestone rendering tests:
   - Milestone rows render with correct name
   - Progress bar shows correct done/total
   - Collapse/expand toggles visibility of column body
   - Column header bar shows correct per-column counts
   - DnD still works within milestone rows
3. Add sortTasks tests for milestone sort key (if not already covered)

**Acceptance criteria:**
- groupTasksByMilestone has tests for all edge cases
- BoardView milestone row rendering is tested
- Collapse/expand behavior is tested
- All tests pass with `pnpm test`
- Test coverage for epicTree.ts improved
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 groupTasksByMilestone tested: grouping, sorting, unassigned, empty
- [ ] #2 BoardView milestone row rendering tested
- [ ] #3 Collapse/expand behavior tested
- [ ] #4 All pnpm test pass
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
