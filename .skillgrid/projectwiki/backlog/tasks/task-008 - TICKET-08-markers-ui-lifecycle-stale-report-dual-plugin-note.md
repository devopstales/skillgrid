---
id: TASK-008
title: 'TICKET-08: markers + ui - lifecycle, stale report, dual-plugin note'
status: In Progress
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 18:30'
labels: []
milestone: m-0
dependencies:
  - TASK-005
  - TASK-007
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create src/markers.ts (markWaiting, markWorking, clearMarker - the exact argv from findings.md R1; sole marker writer). Modify src/ui.ts (dialog half), src/scratch.ts + src/wt.ts (import markers, drop direct marker writes). Update README.md (dual-plugin note). Tests: marker group (lifecycle sequence in wt list, stale detection, manual clear). SATISFIES: marker-follows-lifecycle, stale-marker-reported.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/markers.test.ts passes (lifecycle: session_start->waiting, agent_start->working, agent_end->waiting, session_shutdown->clear)
- [ ] #2 Stale: kill-before-shutdown -> reported
- [ ] #3 Clear: wt config state marker clear removes it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/markers.test.ts passes (lifecycle: session_start->waiting, agent_start->working, agent_end->waiting, session_shutdown->clear)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing tests (lifecycle sequence, stale detection, manual clear)
2. bun test tests/markers.test.ts - verify FAIL (markers.ts missing)
3. Implement src/markers.ts (markWaiting = wt config state marker set waiting --cwd <cwd>, markWorking = set working, clearMarker = clear; event wiring per findings.md R1 marker table; stale detection = marker present + no live session)
4. Modify src/ui.ts, src/scratch.ts, src/wt.ts (import markers, drop direct writes); update README.md (dual-plugin note)
5. bun test tests/markers.test.ts - verify passes
6. Commit: feat(markers): lifecycle markers, stale report, dual-plugin README note
<!-- SECTION:PLAN:END -->
