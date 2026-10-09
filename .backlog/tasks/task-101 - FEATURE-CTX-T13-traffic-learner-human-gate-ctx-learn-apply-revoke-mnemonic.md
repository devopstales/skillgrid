---
id: TASK-101
title: >-
  [FEATURE] CTX T13 traffic learner human-gate: ctx learn apply/revoke
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:38'
updated_date: '2026-10-09 13:44'
labels:
  - mnemonic
  - traffic
milestone: m-10
dependencies:
  - TASK-100
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
mnemonic/internal/ctx/traffic/ (continued) — the human-gate promotion: ctx learn apply <id> promotes a candidate ctx_traffic_memories into active context; ctx learn revoke <id> reverts. Only existing candidates can be promoted (human-gate).

Current State:
- Traffic learner (T12) emits candidates but no promotion/revoke path

Expected State:
- ctx learn apply/revoke manage candidate -> active -> reverted lifecycle
- Only existing candidates promotable
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/traffic/... green: apply marks a candidate promoted and it appears in subsequent compression context
- [ ] #2 revoke reverts a promoted candidate; it no longer appears
- [ ] #3 A non-candidate id is rejected with a clear error (human-gate: only existing candidates promote)
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
1. In mnemonic/internal/ctx/traffic/, add the human-gate promotion: ctx learn apply <id> flips a candidate's status to active.
2. ctx learn revoke <id> reverts an active candidate back to candidate (or archived).
3. Guard: only existing candidate ids may be promoted; a non-candidate id -> clear error.
4. Ensure a promoted candidate appears in the compression context for subsequent requests; a revoked one does not.
5. Tests: apply marks promoted + appears in context; revoke reverts + disappears; non-candidate id rejected.
6. SATISFIES scenarios: learn-apply, learn-revoke, learn-gate.
<!-- SECTION:PLAN:END -->
