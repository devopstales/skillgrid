---
id: TASK-005
title: >-
  TICKET-05: scratch.ts extension wiring - turn_end, restore tool, checkpoint,
  footer
status: To Do
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 15:43'
labels: []
milestone: m-0
dependencies:
  - TASK-004
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create src/scratch.ts (extension file 1: turn_end -> capture + footer, session_start -> cache rebuild + resume dedup + background prune, scratch-restore tool, /checkpoint command) and src/ui.ts (updateFooter, rebuildCache, restoreDialog - the pinned shapes from Stage 9 F1). Create tests/scratch-ext.smoke.md. SATISFIES: checkpoint-after-mutating-turn, dedup-on-unchanged-turn, resume-unchanged-turn-no-duplicate.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pi -e ./src/scratch.ts in a temp repo: edit a file -> footer shows diamond 1 checkpoint
- [ ] #2 /checkpoint lists it with label
- [ ] #3 Restore after a second edit -> file back, dialog confirmed, diamond 2 checkpoints
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 pi -e ./src/scratch.ts in a temp repo: edit a file -> footer shows diamond 1 checkpoint
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write smoke script (tests/scratch-ext.smoke.md: pi -e ./src/scratch.ts in temp repo -> edit -> footer -> /checkpoint -> restore -> file back)
2. pi -e ./src/scratch.ts - verify FAIL (module missing)
3. Implement src/scratch.ts (turn_end -> createCheckpoint + updateFooter; session_start -> resetSessionState + rebuildCache + background pruneOtherSessions; scratch-restore tool; /checkpoint command) + src/ui.ts (updateFooter, rebuildCache, restoreDialog)
4. pi -e ./src/scratch.ts - verify smoke observations hold
5. Commit: feat(scratch): turn_end wiring, restore tool, /checkpoint, footer
<!-- SECTION:PLAN:END -->
