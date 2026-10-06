---
id: TASK-038.02
title: 'TICKET-02: Session checkpoint state in the store'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:46'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-3
dependencies: []
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
Migration 049 adds checkpoint_claimed_at and last_memory_write_at; CheckpointState, ClaimCheckpoint, touchMemoryWrite. One-way door. SATISFIES claim-resets-after-summary, claim-for-unknown-session.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Six events then SessionSummary resets EventsSinceWrite to 0
- [ ] #2 ClaimCheckpoint sets LastClaimedAt; unknown id Exists false no error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/memory ./internal/mnemonic/store -run TestCheckpointState is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Confirm with the user before applying 049 against a real store.
2. Write TestCheckpointState_CountsEventsSinceLastWrite and UnknownSession.
3. Add 049_session_checkpoint.sql and CheckpointState/ClaimCheckpoint/touchMemoryWrite.
4. go test ./internal/mnemonic/memory ./internal/mnemonic/store.
<!-- SECTION:PLAN:END -->
