---
id: TASK-002
title: 'TICKET-02: scratch-core temp-index capture + tree-SHA dedup'
status: Done
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 16:20'
labels: []
milestone: m-0
dependencies:
  - TASK-001
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement createCheckpoint, git, repoRoot, Checkpoint/CheckpointResult types, lastTree dedup map, resetSessionState in src/core/scratch-core.ts (capture half). 5 tests: mutating-turn metadata, index-identical, read-only-skip, unborn-HEAD zero-SHA, ignored-paths-absent. SATISFIES: checkpoint-after-mutating-turn, readonly-turn-no-ref, unborn-head-capture, ignored-only-change-no-ref.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t capture|dedup|unborn|ignored passes (5 tests)
- [ ] #2 The real git index is byte-identical before/after capture
- [ ] #3 A read-only turn returns {status:skipped,reason:unchanged}
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t capture|dedup|unborn|ignored passes (5 tests)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing tests (5: mutating-turn metadata, index-identical, read-only-skip, unborn-HEAD zero-SHA, ignored-paths-absent)
2. bun test tests/scratch.test.ts - verify FAIL (createCheckpoint is not defined)
3. Implement createCheckpoint (temp-index: GIT_INDEX_FILE in mkdtemp, read-tree HEAD, add --all filtered in batches of 100, write-tree, commit-tree with metadata, update-ref refs/skillgrid/<id>; dedup on worktree tree SHA)
4. Implement git helper (spawn, no shell), repoRoot (git -C <cwd> rev-parse --show-toplevel), Checkpoint type, lastTree map, resetSessionState
5. bun test tests/scratch.test.ts - verify 5 tests pass
6. Commit: feat(scratch): temp-index capture + tree-SHA dedup
<!-- SECTION:PLAN:END -->
