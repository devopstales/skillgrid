---
id: TASK-019.02
title: 'TICKET-09: Per-plan sdd workspace scripts'
status: done
assignee: []
created_date: '2026-09-21 15:10'
updated_date: '2026-09-21 15:27'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/briefing.md
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Port sdd-workspace, task-brief, review-package (superpowers paths repathed to .skillgrid/sdd) plus test-sdd-workspace.sh.

Current: no per-plan workspace scripts; sdd layout flat.

Expected: distinct plan files resolve distinct dirs; artifacts land per-plan; parent .gitignore keeps git status clean; missing plan exits 2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 test-sdd-workspace.sh passes (distinct dirs, per-plan artifacts, self-ignore, missing-plan error)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing test-sdd-workspace.sh (RED)
2. Port the three scripts (GREEN)
3. Test passes, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 1 review: implementer done. Coordinator verified: test-sdd-workspace.sh 24/24 PASS on live tree. Note: shared-worktree commit 2d81707 carries foreign files (commit-impure, tree-correct); no action — integration will not rewrite live commits. Ready for Wave 2.
<!-- SECTION:NOTES:END -->
