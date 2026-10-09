---
id: TASK-100
title: >-
  [FEATURE] CTX T12 traffic learner: deterministic zero-LLM candidate rules
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:38'
updated_date: '2026-10-09 13:42'
labels:
  - mnemonic
  - traffic
milestone: m-10
dependencies:
  - TASK-091
  - TASK-092
references:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
mnemonic/internal/ctx/traffic/ — a zero-LLM, rule-based, deterministic traffic learner. Given a message list, emits candidate ctx_traffic_memories rows (no upstream). Rules live in traffic/rules.yaml; categories: error_recovery, environment, preference, architecture. Deterministic: same input -> same candidates.

Current State:
- No traffic learner exists; ctx_traffic_memories table (T02) ready

Expected State:
- Deterministic candidate generation from rules.yaml
- Correct category assignment
- No upstream call
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/traffic/... green: a deterministic corpus produces the same candidate set on repeated runs
- [ ] #2 Categories assigned correctly (error_recovery/environment/preference/architecture)
- [ ] #3 A candidate is created only when its trigger rule matches; none otherwise
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create mnemonic/internal/ctx/traffic/ — a zero-LLM, rule-based, deterministic candidate generator.
2. Author traffic/rules.yaml with trigger rules per category: error_recovery, environment, preference, architecture.
3. Given a message list, emit candidate ctx_traffic_memories rows (no upstream call); same input -> same candidate set (deterministic).
4. Tests: a deterministic corpus produces the same candidate set on repeated runs; categories assigned correctly; a candidate is created only when its trigger rule matches.
5. SATISFIES scenarios: traffic-deterministic, traffic-categorize, traffic-no-false-positive.
<!-- SECTION:PLAN:END -->
