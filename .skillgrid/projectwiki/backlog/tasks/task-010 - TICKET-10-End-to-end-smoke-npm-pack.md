---
id: TASK-010
title: 'TICKET-10: End-to-end smoke + npm pack'
status: To Do
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 15:43'
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
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create tests/e2e.sh (full smoke: scripted Pi session via pi --print -p per turn, restore via tool execute with id set, the full checkpoint -> restore -> worktree-move -> recovery -> marker cycle). Run bun pack to verify the npm package structure. SATISFIES: npm-test-passes-without-pi.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bash tests/e2e.sh exits 0 (all assertions pass)
- [ ] #2 bun pack produces a valid .tgz with the src/ + references/ files
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bash tests/e2e.sh exits 0 (all assertions pass)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write tests/e2e.sh (pi --print -p per turn; restore via tool execute with id; full cycle: checkpoint -> restore -> worktree-move -> recovery -> marker)
2. bash tests/e2e.sh - verify exits 0
3. bun pack - verify .tgz structure
4. Commit: test: e2e smoke + npm pack verification
<!-- SECTION:PLAN:END -->
