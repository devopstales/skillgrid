---
id: TASK-015
title: '[FEATURE] SDD spec: 2026-09-18-memfs-code-index (mem fs code index)'
status: done
assignee: []
created_date: '2026-09-21 07:14'
labels: []
dependencies: []
references:
  - .skillgrid/archive/2026-09-18-memfs-code-index/blueprint.md
  - .skillgrid/archive/2026-09-18-memfs-code-index/acceptance.feature
  - skillgrid-cli/internal/mnemonic/memfs/
  - skillgrid-cli/cmd/skillgrid/mem.go
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track the SDD spec **2026-09-18-memfs-code-index**: expose the SQLite code index as an OpenViking-style `mem fs` filesystem (ls / tree / find / cat over the code-index namespace with code paths like `<project>/src/auth/login.go#Handler`).

**Current State:**
- Implemented: `skillgrid-cli/internal/mnemonic/memfs/` (memfs.go, codepath.go, find.go, cat.go, shared.go) + `skillgrid-cli/cmd/skillgrid/mem.go` wiring; `mem_fs_test.go` + `cat_code_test.go` + `codepath_test.go` present
- Blueprint checkboxes were never ticked — completion verified against code on 2026-09-21

**Expected State:**
- Done. Archived to `.skillgrid/archive/2026-09-18-memfs-code-index/` on 2026-09-21.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `mem fs ls` lists files/subdirs (dirs) and symbols (files) from the code index without returning memory observations
- [ ] #2 `mem fs tree` renders the repo tree with per-file symbol counts
- [ ] #3 `mem fs find` globs paths and symbol names, with scope narrowing
- [ ] #4 `mem fs cat` returns node source for files and symbol signatures for symbols
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed during the 2026-09-21 specs audit: all acceptance.feature scenarios delivered — memfs package with codepath resolution (dir/file/symbol forms), ls over files+symbols, tree with symbol counts, find (path glob + symbol name + scope), cat for node source. Blueprint checkboxes were never ticked; completion verified directly in code before archiving. Ticket was missing from the backlog — created retroactively and marked done.
<!-- SECTION:FINAL_SUMMARY:END -->
