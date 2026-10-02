---
id: TASK-038.08
title: 'TICKET-08: Memory index at session start'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:47'
labels: []
dependencies:
  - TASK-038.01
references:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
parent_task_id: TASK-038
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
RenderIndex plus prime ## Memory. Index only; AutoPrepend stays resume-only. SATISFIES prime-lists-summaries-and-observation-index, index-respects-token-cap, index-absent-for-empty-project.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Pinned first; empty store has no section; over cap drops oldest
- [ ] #2 Footer names mem_get_observation mem_timeline mem_search
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/session_inject -run TestRenderIndex is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestRenderIndex and extend RenderPrime for ## Memory.
2. Implement RenderIndex; wire prime to fetch RecentContext/RecentObservations.
3. Document mnemonic.inject in 05-memory.md.
<!-- SECTION:PLAN:END -->
