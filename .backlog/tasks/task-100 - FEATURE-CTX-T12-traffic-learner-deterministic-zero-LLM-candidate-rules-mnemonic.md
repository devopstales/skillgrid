---
id: TASK-100
title: >-
  [FEATURE] CTX T12 traffic learner: deterministic zero-LLM candidate rules
  (mnemonic)
status: done
assignee: []
created_date: '2026-10-09 13:38'
updated_date: '2026-10-12 16:50'
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
CTX traffic learner (zero-LLM deterministic candidate rules). internal/ctx/traffic/{rules,learner,evict}.go + rules.yaml (4 rules: er_fail_then_ok 0.8, env_repeated_command 0.7, pref_message_repetition 0.7, arch_path_prefix 0.7). Pure Learner.Extract (no http/model → extract path cannot LLM). Cross-rule first-match-wins via consumed map[int]bool keyed by event Seq (emitted candidate consumes evidence; below-floor skips without consuming). pathPrefix restricted to directory boundaries (no-shared-dir corpus → 0 candidates, AC#3). Deterministic identity memoryID=hex(sha256[:20](sessionID|ruleID|key)). EvictToBound enforces max_memories (confidence ASC, created_at ASC, fail-open). Additive store methods ListAllTrafficMemories/DeleteTrafficMemory (fail-open). Stdlib only, shared store (ADR-0033). Impls dc20f834 + rework f92e97f5 (closed C1 pathPrefix false-positive, C2 cross-rule first-match-wins, I2 sha1→sha256). Gates green: build 0, traffic 30/30, store ok, vet 0. Review Spec SATISFIED, Quality APPROVED (2 Minor parked).
<!-- SECTION:FINAL_SUMMARY:END -->
