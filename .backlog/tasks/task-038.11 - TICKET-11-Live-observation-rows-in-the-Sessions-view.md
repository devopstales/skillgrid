---
id: TASK-038.11
title: 'TICKET-11: Live observation rows in the Sessions view'
status: done
assignee: []
created_date: '2026-10-02 14:47'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-3
dependencies:
  - TASK-038.10
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
ToolTimeline interleaves observation rows; SessionsPage appends from activity SSE. SATISFIES observation-appears-live, observation-row-links-to-full-text, stream-disconnected.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Observation row appears by timestamp and links to detail
- [x] #2 Mocked activity frame increments the card count
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 npx vitest run SessionsPage.test.tsx ToolTimeline.test.tsx all pass
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write Vitest cases for interleaved observation rows and live activity frames.
2. Wire ToolTimeline and SessionsPage; missing fields stay empty.
3. npx vitest run SessionsPage.test.tsx ToolTimeline.test.tsx.
<!-- SECTION:PLAN:END -->
