---
id: TASK-101
title: >-
  [FEATURE] CTX T13 traffic learner human-gate: ctx learn apply/revoke
  (mnemonic)
status: done
assignee: []
created_date: '2026-10-09 13:38'
updated_date: '2026-10-13 09:54'
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
- [x] #1 go test ./internal/ctx/traffic/... green: apply marks a candidate promoted and it appears in subsequent compression context
- [x] #2 revoke reverts a promoted candidate; it no longer appears
- [x] #3 A non-candidate id is rejected with a clear error (human-gate: only existing candidates promote)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Tests pass (`go test ./...` for touched packages)
- [x] #2 Lint and formatting pass
- [x] #3 Edge cases covered
- [x] #4 No new warnings introduced
- [x] #5 Spec/docs updated if behavior changes
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Human-gate promotion lifecycle for traffic memories (plan §18 M23, no automatic promotion). Migration 068_ctx_traffic_memory_status.sql adds status (candidate/applied/revoked, default candidate) + applied_at to ctx_traffic_memories idempotently (duplicate-column skip makes re-runs safe; CREATE TABLE carries cols for fresh DB). Store: TrafficMemory gains Status/AppliedAt; SetTrafficMemoryStatus(memoryID, status, appliedAt) returns (bool,error) — appliedAtMode "set" (apply) vs "clear" (demote → applied_at NULL); pre-068 fail-open (ADR-0016) in trafficMemorySelectCols(), PutTrafficMemory(), and a hasTrafficMemoryStatus() probe so the UPDATE degrades to a no-op when the cols are absent. Gate: internal/ctx/traffic/gate.go NewGate(*ctxstore.Store) with Apply(id), Revoke(id), ActiveMemories(sessionID) — all nil-store safe; not-found error only surfaces in the CLI, never the request path. Context seam: internal/ctx/proxy/memcontext.go activeMemoryContext/prependMemoryContext + proxy.go SetGate/gateRef/memoryContext wired into serveCompress (compress.go) so applied memories are prepended to subsequent compression context; default proxy runs nil gate → fail-open, no behavior change until T14 wires cfg.Traffic. CLI: cmd/mnemonic/ctx_learn.go runCtxLearnAction(apply|revoke) + ctx.go dispatch, openCtxStoreForLearn reuses the runCtxProxy store-resolution pattern. Stdlib only. Gates green: go build ./... exit 0; go test ./internal/ctx/traffic/... (6 gate tests) ok; ./internal/ctx/store/... (incl. TestMigration068TrafficMemoryStatus) ok; ./internal/ctx/proxy/... (5 context-injection tests) ok; go vet exit 0; gofmt -l on the 10 touched files clean. AC1 apply-promoted+appears (TestGate_ApplyPromotesAndIsActive + memcontext tests), AC2 revoke-reverts+disappears (TestGate_RevokeRevertsAndDisappears, row kept status=revoked applied_at=NULL), AC3 non-candidate id rejected (TestGate_NotFoundRejected, no row created). Commit 41ec139e.
<!-- SECTION:FINAL_SUMMARY:END -->
