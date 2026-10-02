---
id: TASK-038
title: 'Epic: Memory checkpoint and memory index'
status: done
assignee: []
created_date: '2026-10-02 14:46'
updated_date: '2026-10-02 16:04'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/acceptance.feature
  - .skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Host agent is the memory observer (ADR-0022). Server-gated checkpoints write typed observations and a summary; prime injects a Memory Index at session start; private spans never persist; Sessions shows observations live.

Tickets: .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
Parked: prompt-capture hooks; Memories page upgrade.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All 11 tickets done
- [ ] #2 Acceptance gates G1-G14 have evidence
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level DoD in briefing.md fully checked
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Track the 11 child tickets in tasks.md.
2. Close when G1-G14 have evidence.
<!-- SECTION:PLAN:END -->
