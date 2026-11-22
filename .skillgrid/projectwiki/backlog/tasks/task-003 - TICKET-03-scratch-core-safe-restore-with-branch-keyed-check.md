---
id: TASK-003
title: 'TICKET-03: scratch-core safe restore with branch-keyed check'
status: To Do
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 15:43'
labels: []
milestone: m-0
dependencies:
  - TASK-002
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement restoreCheckpoint, loadCheckpoint, listCheckpoints, safeClean in src/core/scratch-core.ts (restore half). 5 tests: restore-on-matching-branch (deleted tracked returns, post-checkpoint untracked removed, pre-existing survives), before-restore checkpoint, refused-branch-mismatch-named, refused-missing-checkpoint, detached-HEAD-restore-refused. Branch identity is the full safety check (a branch name exists in exactly one live worktree). SATISFIES: restore-on-matching-branch, restore-refused-cross-branch, restore-refused-missing-checkpoint.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t restore passes (5 tests)
- [ ] #2 A restore on a mismatched branch is refused with reason branch + both branch names
- [ ] #3 A before-restore checkpoint always exists
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t restore passes (5 tests)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing tests (5: restore-on-matching-branch, before-restore checkpoint, refused-branch-mismatch, refused-missing, detached-HEAD)
2. bun test tests/scratch.test.ts -t restore - verify FAIL
3. Implement loadCheckpoint (isSafeRefId check, parse metadata), listCheckpoints, restoreCheckpoint (branch check, before-restore snapshot, reset --hard, read-tree, safeClean via git diff --name-only --diff-filter=AD, read-tree index)
4. bun test tests/scratch.test.ts - verify 9 tests total pass
5. Commit: feat(scratch): safe restore with branch-keyed check
<!-- SECTION:PLAN:END -->
