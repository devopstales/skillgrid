---
id: TASK-001
title: 'TICKET-01: Scaffold pi-worktree repo + contract constants + import-scan'
status: Done
assignee: []
created_date: '2026-11-22 15:40'
updated_date: '2026-11-22 16:20'
labels: []
milestone: m-0
dependencies: []
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create the devopstales/pi-worktree repo (git init, MIT LICENSE, .gitignore), package.json (pi.extensions, engines, scripts, files), src/core/constants.ts (REF_BASE, AGENT_BRANCH_RE, IGNORED_DIRS, MAX_UNTRACKED_FILE_SIZE, MAX_UNTRACKED_DIR_FILES, MAX_PER_SESSION, isSafeRefId), and tests/import-scan.test.ts (no Pi imports in src/core/ + constants match contract values). OD-1 (ref namespace refs/skillgrid/) and OD-3 (merge-default argv) are declared here. One-way: the namespace once in use is not renamable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/import-scan.test.ts passes (2 tests: import-scan + constants-match)
- [ ] #2 The constants module exports the exact contract values
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/import-scan.test.ts exits 0
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. git init devopstales/pi-worktree; MIT LICENSE; .gitignore (node_modules/, dist/, *.log)
2. Write package.json (name, version, license, type:module, files, pi.extensions, engines, scripts, devDependencies)
3. Write src/core/constants.ts (REF_BASE, AGENT_BRANCH_RE, IGNORED_DIRS, MAX_UNTRACKED_FILE_SIZE, MAX_UNTRACKED_DIR_FILES, MAX_PER_SESSION, isSafeRefId)
4. Write tests/import-scan.test.ts (no @earendil-works imports in src/core/; constants match contract values)
5. bun test tests/import-scan.test.ts - verify 2 tests pass
6. Commit: feat: scaffold pi-worktree repo + contract constants
<!-- SECTION:PLAN:END -->
