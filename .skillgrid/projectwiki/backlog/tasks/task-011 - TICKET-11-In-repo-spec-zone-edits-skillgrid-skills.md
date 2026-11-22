---
id: TASK-011
title: 'TICKET-11: In-repo spec-zone edits (skillgrid-skills)'
status: In Progress
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 18:30'
labels: []
milestone: m-0
dependencies:
  - TASK-009
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
In skillgrid-skills: update root README.md (Tool Dependency list: add the pi-worktree line), update .agents/skills/_shared/work-unit-commits/README.md section Scratch checkpoints (reference-implementation pointer line + the ~~~skillgrid-constants fence at column 0). Run the contract-drift guard against the edited contract. SATISFIES: drift-guard-passes-on-match.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The ~~~skillgrid-constants fence is present at column 0 in work-unit-commits/README.md with the exact values from Task 1 constants.ts
- [ ] #2 SKILLGRID_CONTRACT_PATH=.agents/skills/_shared/work-unit-commits/README.md bun test tests/contract-drift.test.ts (in the plugin repo) passes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 The ~~~skillgrid-constants fence is present at column 0 in work-unit-commits/README.md with the exact values from Task 1 constants.ts
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Edit root README.md (Tool Dependency: add pi-worktree line)
2. Edit work-unit-commits/README.md (reference-implementation pointer + ~~~skillgrid-constants fence at column 0 with exact constants values)
3. SKILLGRID_CONTRACT_PATH=... bun test tests/contract-drift.test.ts (in plugin repo) - verify passes
4. Commit (spec zone): docs: name pi-worktree as the scratch-checkpoint reference implementation
<!-- SECTION:PLAN:END -->
