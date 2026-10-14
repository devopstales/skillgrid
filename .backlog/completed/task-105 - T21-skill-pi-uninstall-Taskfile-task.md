---
id: TASK-105
title: 'T21 skill-pi:uninstall Taskfile task'
status: wontfix
assignee: []
created_date: '2026-10-13 09:54'
updated_date: '2026-10-13 10:02'
labels:
  - skill-pi
  - taskfile
dependencies:
  - TASK-076
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - Taskfile.yml
modified_files:
  - Taskfile.yml
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a `skill-pi:uninstall` task to Taskfile.yml — the inverse of `skill-pi:build` — and a backlog ticket for it. Scope (user-approved design, 2026-10-13): remove the dev npm link, remove skill-pi/node_modules, remove the per-project <project>/.skill-pi/ shim. ~/.skill-pi/ config is NOT touched (no --global). Never touches a plain pi install.

## Design (approved 2026-10-13, bounded path)

Add to Taskfile.yml next to `skill-pi:install` / `skill-pi:build` / `skill-pi:test`:

```yaml
  skill-pi:uninstall:
    desc: Uninstall skill-pi dev setup (inverse of skill-pi:build): npm unlink, remove node_modules, remove per-project .skill-pi/ shim. Leaves ~/.skill-pi/ config intact.
    cmds:
      - cd skill-pi && npm unlink --no-audit --no-fund
      - rm -rf skill-pi/node_modules
      - rm -rf .skill-pi
```

Also add the header comment line (line ~9 area):
`#   task skill-pi:uninstall uninstall the dev link, node_modules, and per-project .skill-pi/ shim`

Notes:
- `npm unlink` from inside skill-pi/ undoes the `npm link` done by skill-pi:build (Taskfile.yml:112).
- `.skill-pi` is removed from the repo root (per-project cwd shim location per T08/TASK-076 materialization). If the actual shim path differs, verify against skill-pi/bin/skill-pi.js shim materialization code before finalizing the rm target.
- Idempotent: all three steps are safe no-ops when already uninstalled.
- Verification: run `task skill-pi:build`, then `task skill-pi:uninstall`, then assert `command -v skill-pi` fails, `test ! -d skill-pi/node_modules`, `test ! -d .skill-pi`, `test -d ~/.skill-pi` (if present before).
- Add a node test under skill-pi/ if the shim removal logic lives in JS; otherwise a manual/Taskfile-level check suffices (task is shell-only).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Running `task skill-pi:uninstall` after `task skill-pi:build` leaves the `skill-pi` command unresolvable on PATH (npm link removed).
- [x] #2 skill-pi/node_modules is removed by the task.
- [x] #3 A materialized <project>/.skill-pi/ shim directory is removed by the task.
- [x] #4 ~/.skill-pi/ (global config + trust records) is left intact after the task runs.
- [x] #5 Taskfile.yml header comment block lists `task skill-pi:uninstall` alongside the other skill-pi tasks.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Tests pass (`go test ./...` for touched packages)
- [x] #2 Lint and formatting pass
- [x] #3 Edge cases covered
- [x] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented 2026-10-13: Taskfile.yml skill-pi:uninstall task. Gotcha: npm 11.17.0 `npm unlink` without -g does not remove the dev link created by `npm link` (only `npm unlink -g skill-pi` does). YAML desc needed quoting because it contains ': ' sequences.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added `skill-pi:uninstall` task to Taskfile.yml (inverse of `skill-pi:build`): `npm unlink -g skill-pi`, `rm -rf skill-pi/node_modules`, `rm -rf .skill-pi`. Header comment updated. Verified full build→uninstall cycle: skill-pi removed from PATH and global node_modules, project node_modules removed, per-project .skill-pi/ shim removed, ~/.skill-pi/ config intact. Note: no-arg `npm unlink` and `npm unlink skill-pi` (local) both fail in npm 11.17.0 — only the `-g` form undoes `npm link`.
<!-- SECTION:FINAL_SUMMARY:END -->
