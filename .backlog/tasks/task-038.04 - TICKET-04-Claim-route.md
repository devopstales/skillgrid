---
id: TASK-038.04
title: 'TICKET-04: Claim route'
status: done
assignee: []
created_date: '2026-10-02 14:47'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-3
dependencies:
  - TASK-038.01
  - TASK-038.02
  - TASK-038.03
  - TASK-038.07
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
POST /sessions/{id}/checkpoint/claim returns due reason prompt. SATISFIES claim-due-after-enough-events, claim-not-due-below-threshold, claim-for-unknown-session, disabled-checkpoint.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Five tool calls then claim is due; immediate re-claim is cooldown
- [x] #2 Unknown session is 200 with reason unknown_session
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/http -run TestCheckpointClaim|TestCheckpointPrompt is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestCheckpointClaim_DueAndCooldown, UnknownSession, Disabled, TestCheckpointPrompt_Content.
2. Implement handleCheckpointClaim; register POST /sessions/{id}/checkpoint/claim.
3. Document in openapi.yaml. go test ./internal/mnemonic/http -run TestCheckpointClaim|TestCheckpointPrompt.
<!-- SECTION:PLAN:END -->
