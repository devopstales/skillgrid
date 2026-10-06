---
id: TASK-038.01
title: 'TICKET-01: Checkpoint and inject config sections'
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
Load mnemonic.checkpoint and mnemonic.inject with briefing defaults and validation. SATISFIES defaults-apply-without-keys, invalid-value.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 No keys yields defaults enabled, 5, 10m, 5 and 5, 20, 800
- [ ] #2 min_events 0 errors naming mnemonic.checkpoint.min_events
- [ ] #3 Keys documented in docs/user-guide/05-memory.md
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/config -run TestCheckpointConfig is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestCheckpointConfig (defaults + invalid min_events).
2. Add Checkpoint and Inject structs, yaml, defaults, validation in load.go.
3. Document keys in docs/user-guide/05-memory.md.
4. go test ./internal/mnemonic/config -run TestCheckpointConfig.
<!-- SECTION:PLAN:END -->
