---
id: TASK-007
title: 'TICKET-07: wt-core recovery + wt.ts worktrunk tool, session moves, approvals'
status: To Do
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 15:43'
labels: []
milestone: m-0
dependencies:
  - TASK-006
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement recordWorktrees (in-memory Map), recover (repository identity re-verified, default-branch preference via git symbolic-ref, no-replay, linked-session creation via ctx.sessionManager) in src/core/wt-core.ts. Create src/wt.ts (worktrunk tool {command,args?} sequential, /wt command, directive follow -> linked-session move, print/json -> stopped, approval dialog passthrough, recovery hook). Update README.md (min wt version pin). SATISFIES: session-moves-to-worktree, print-mode-stopped, recovery-no-replay, no-surviving-worktree, approval-approved-and-retried, approval-declined.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/wt-core.test.ts -t recovery passes (2 tests)
- [ ] #2 pi -e ./src/wt.ts smoke: wt switch --create agent-smoke -> session cwd is new worktree, source in /resume
- [ ] #3 Print mode -> stopped + path; unapproved hook -> dialog, approve -> runs once
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/wt-core.test.ts -t recovery passes (2 tests)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write recovery tests (2: cwd-removed -> recover to survivor, no-surviving -> report) + smoke script
2. bun test tests/wt-core.test.ts -t recovery - verify FAIL
3. Implement recordWorktrees (Map<worktreePath,{gitCommonDir,recordedAt}>), recover (identity re-check, default-branch preference via git symbolic-ref --short refs/remotes/origin/HEAD fallback main/master, linked-session via ctx.sessionManager, no-replay), src/wt.ts (worktrunk tool, /wt command, directive follow, approval passthrough)
4. bun test + pi -e ./src/wt.ts - verify tests + smoke pass
5. Update README.md (min wt version)
6. Commit: feat(wt): no-replay recovery + worktrunk tool with session moves and approvals
<!-- SECTION:PLAN:END -->
