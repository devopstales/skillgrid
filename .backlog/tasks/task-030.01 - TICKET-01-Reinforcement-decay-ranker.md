---
id: TASK-030.01
title: 'TICKET-01: Reinforcement decay ranker'
type: feature
status: needs-triage
assignee: []
created_date: '2026-10-02 08:59'
updated_date: '2026-10-03 07:39'
labels:
  - mnemonic-memory-improvements
dependencies: []
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
documentation:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/acceptance.feature
parent_task_id: TASK-030
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: mem_search does not apply reinforcement decay.
Expected state: Reinforcement and rankByDecay order a hot observation ahead of a cold twin and keep equal factors stable. Per ADR-0018.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Usage 20 at 40 days outranks usage 0
- [ ] #2 Importance >= 4 freezes the half-life term
- [ ] #3 Equal factors keep input order
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/memory/ -run TestReinforcementDecayRanksHotRow passes
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: TestReinforcementDecayRanksHotRow, TestDecayImmunityFreezesHalfLife, TestEqualDecayKeepsOrder
2. Implement Reinforcement and rankByDecay in memory/decay.go per blueprint Task 1
3. GREEN: go test ./internal/mnemonic/memory/ -run 'TestReinforcement|TestDecayImmunity|TestEqualDecay'
<!-- SECTION:PLAN:END -->
