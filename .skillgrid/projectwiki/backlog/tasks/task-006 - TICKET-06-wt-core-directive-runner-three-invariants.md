---
id: TASK-006
title: 'TICKET-06: wt-core directive runner + three invariants'
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
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement runDirectedWt (directive protocol: mkdtemp + 0o600, WORKTRUNK_DIRECTIVE_CD_FILE, spawn without shell, drain-then-delete, in-process mutex), supportsDirectiveProtocol, observeWorktreeRoot (first-use one-line observation), checkBranchAllowList (regex + nearest-form suggestion), applyMergeDefaults (prepends --no-squash --no-commit unless explicit) in src/core/wt-core.ts. Pure-parse groups run without wt; runner groups skip when wt absent. SATISFIES: standard-root-observed, foreign-root-noted, allow-list-at-creation, allow-list-rejected-with-suggestion, primary-checkout-unconstrained, merge-gets-default-flags, explicit-squash-not-rewritten, directive-outside-repo-rejected.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/wt-core.test.ts passes (all groups; runner groups skipped with named reason when wt absent)
- [ ] #2 applyMergeDefaults([merge,main]) returns [merge,--no-squash,--no-commit,main]
- [ ] #3 checkBranchAllowList(feature-x, true) returns {ok:false, suggestion:/agent-feature-x/}
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/wt-core.test.ts passes (all groups; runner groups skipped with named reason when wt absent)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing tests (pure: applyMergeDefaults x2, checkBranchAllowList x3; runner: observeWorktreeRoot, directive-outside-repo, mutex, directive-file-cleanup - skip when wt absent)
2. bun test tests/wt-core.test.ts - verify FAIL (wt-core.ts missing)
3. Implement runDirectedWt (mkdtemp + 0o600, WORKTRUNK_DIRECTIVE_CD_FILE env, spawn without shell, drain-then-delete in finally, in-process mutex), supportsDirectiveProtocol, observeWorktreeRoot (git worktree list --porcelain), checkBranchAllowList, applyMergeDefaults
4. bun test tests/wt-core.test.ts - verify passes
5. Commit: feat(wt): directive runner + root-observation/allow-list/merge-default invariants
<!-- SECTION:PLAN:END -->
