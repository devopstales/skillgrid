---
id: TASK-009
title: 'TICKET-09: evals + changelog + README + reference docs + contract-drift guard'
status: To Do
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 15:43'
labels: []
milestone: m-0
dependencies:
  - TASK-008
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create evals/evals.json (array of {name,description,input,expect}; ported scratch_dedup_readonly_turn + scratch_branch_safety_refused + marker + approval scenarios), changelog/releases/v0.1.0/ (pi-worktrunk structure), references/wt-commands.md (one-page pointer), references/scratch-restore.md (restore protocol), README.md (full). Create tests/contract-drift.test.ts (parses ~~~skillgrid-constants fence from SKILLGRID_CONTRACT_PATH, diffs against exported constants). SATISFIES: drift-guard-passes-on-match, drift-named, missing-fence.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test (full suite) passes
- [ ] #2 SKILLGRID_CONTRACT_PATH=... bun test tests/contract-drift.test.ts passes when constants match; fails naming the mismatch
- [ ] #3 evals/evals.json is valid JSON with the pinned format
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test (full suite) passes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing test (contract-drift.test.ts: parse ~~~skillgrid-constants fence, diff against constants; missing fence -> FAIL; missing env -> FAIL)
2. Run - verify FAIL; mutate one constant to verify failure names the mismatch; restore
3. Write evals/evals.json, changelog/releases/v0.1.0/, references/wt-commands.md, references/scratch-restore.md, README.md (full)
4. bun test - full suite passes
5. Commit: chore: evals, changelog skeleton, README, reference docs, contract-drift guard
<!-- SECTION:PLAN:END -->
