---
id: TASK-030.05
title: 'TICKET-03: Decay config'
type: feature
status: needs-triage
assignee: []
created_date: '2026-10-02 08:59'
updated_date: '2026-10-03 07:39'
labels:
  - mnemonic-memory-improvements
dependencies:
  - TASK-030.01
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
parent_task_id: TASK-030
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: there is no mnemonic.decay section.
Expected state: absent config enables decay with a 30-day half-life and enabled false keeps BM25 order.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Absent section is enabled with half-life 30
- [ ] #2 enabled false keeps BM25 order
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 config and blend tests pass
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: absent mnemonic.decay enables defaults; enabled false stays false
2. Wire SetDecay from config on service open
3. GREEN: TestDecayDisabledKeepsBM25
<!-- SECTION:PLAN:END -->
