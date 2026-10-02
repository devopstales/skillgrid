---
id: TASK-030.03
title: 'TICKET-06: Compact continuity hook'
status: ready-for-agent
assignee: []
created_date: '2026-10-02 08:59'
labels:
  - mnemonic-memory-improvements
dependencies: []
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
parent_task_id: TASK-030
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: RunHook rejects an unknown compact type.
Expected state: compact writes one upserted continuity observation, uses a 3s budget, and fails open.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 topic_key compaction/<session> is saved
- [ ] #2 A hook that waits on ctx.Done returns nil error and Distilled false
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test TestCompactHook passes
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: TestCompactHookSaves, TestCompactHookFailOpen
2. Accept hook type compact with a 3s fail-open budget
3. GREEN: go test TestCompactHook
<!-- SECTION:PLAN:END -->
