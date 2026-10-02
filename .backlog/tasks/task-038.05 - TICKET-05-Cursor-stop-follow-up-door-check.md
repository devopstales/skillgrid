---
id: TASK-038.05
title: 'TICKET-05: Cursor stop follow-up door check'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:47'
labels: []
dependencies:
  - TASK-038.04
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
checkpoint mode on capture script; stop hook followup_message; loop_limit 2. SATISFIES stop-returns-followup-when-due, stop-stays-silent-at-loop-limit, stop-fails-open-without-server.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Due plus loop_count 0 prints followup_message
- [ ] #2 loop_count 2 or dead port prints empty object exit 0
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 node scripts/test-hooks.mjs checkpoint reports 0 failed
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Add checkpoint group to scripts/test-hooks.mjs (due, loop_limit, fail-open).
2. Implement checkpoint mode; cursor-session-end.sh stop only; loop_limit 2 in hooks-cursor.json and setup cursor.
3. node scripts/test-hooks.mjs checkpoint; document in 04-hooks.md.
<!-- SECTION:PLAN:END -->
