---
id: TASK-004
title: 'TICKET-04: scratch-core session pruning + latency bench'
status: Done
assignee: []
created_date: '2026-11-22 15:43'
updated_date: '2026-11-22 17:13'
labels: []
milestone: m-0
dependencies:
  - TASK-003
references:
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/tasks.md
  - .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/acceptance.feature
priority: high
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement prune (50-cap), pruneOtherSessions (liveness-aware, PRUNE_MIN_AGE_MS=1h, injectable now) in src/core/scratch-core.ts (prune half). Create tests/bench.ts (capture latency at 20/500/5000/50000 files; exit 1 if 5k-file median > 150ms). 4 prune tests. SATISFIES: fifty-cap-prunes-oldest, cross-session-prune-keeps-fresh-sibling, exactly-50-no-prune.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t prune|cap passes (4 tests)
- [ ] #2 bun run bench prints the table with 5k-file median <= 150ms
- [ ] #3 Total scratch test count = 14
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 bun test tests/scratch.test.ts -t prune|cap passes (4 tests)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write failing tests (4: 51st-capture-prunes-oldest, pruneOtherSessions-removes-other, liveness-threshold-keeps-fresh-sibling, exactly-50-no-prune) + tests/bench.ts
2. bun test tests/scratch.test.ts -t prune - verify FAIL
3. Implement prune (50-cap), pruneOtherSessions (liveness-aware: delete only refs older than PRUNE_MIN_AGE_MS=1h; now injectable), PRUNE_MIN_AGE_MS constant
4. bun test tests/scratch.test.ts && bun run bench - verify 14 tests pass + bench table, 5k median <= 150ms
5. Commit: feat(scratch): session pruning + latency bench
<!-- SECTION:PLAN:END -->
