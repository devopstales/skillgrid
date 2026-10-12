---
id: TASK-097
title: >-
  [FEATURE] CTX T09 observability store: per-request writer + periodic rollup
  (mnemonic)
status: done
assignee: []
created_date: '2026-10-09 13:37'
updated_date: '2026-10-12 18:51'
labels:
  - mnemonic
  - observability
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
modified_files:
  - mnemonic/internal/ctx/observability/observability.go
  - mnemonic/internal/ctx/observability/observability_test.go
  - mnemonic/internal/http/ctx.go
  - mnemonic/internal/http/ctx_test.go
  - mnemonic/internal/http/server.go
  - mnemonic/internal/ctx/proxy/proxy.go
  - mnemonic/cmd/mnemonic/ctx_proxy.go
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
mnemonic/internal/ctx/observability/ — a per-request store writer (tokens, mode, model) plus a periodic rollup that writes ctx_stats_history. The GET /stats handler reads the aggregate rollup.

Current State:
- Proxy (T04) serves requests but per-request stats are not yet written; no rollup job

Expected State:
- Per-request stats written to ctx_proxy_stats / ctx_stats_history
- Periodic rollup without restart
- /stats backed by stored aggregate
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/observability/... green: after N requests the store holds per-request, per-mode, per-model rows
- [ ] #2 A periodic rollup writes ctx_stats_history without a proxy restart
- [ ] #3 GET /stats JSON matches the stored aggregate rollup
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
1. Create mnemonic/internal/ctx/observability/ — a per-request store writer (tokens, mode, model) invoked from the proxy path.
2. Implement a periodic rollup goroutine that aggregates ctx_proxy_stats into ctx_stats_history (no proxy restart).
3. Expose an aggregate-rollup read used by GET /stats.
4. Tests: after N requests the store holds per-request/per-mode/per-model rows; periodic rollup writes ctx_stats_history; GET /stats JSON matches the stored aggregate.
5. SATISFIES scenarios: obs-per-request, obs-rollup, obs-stats.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
T09 observability complete. New internal/ctx/observability package: pure Roll (no store) → WriteSavings/Recompute store folds → StartRollup goroutine (stoppable, idempotent via deterministic <granularity>-<bucketStart> BucketID) → Stats/StatsHistory/Snapshot readers, all fail-open (ADR-0016). internal/http/ctx.go: GET /stats + GET /stats-history via withProjectHandle → ctxstore.New(h.Store()) → writeJSON (single-open invariant, ADR-0033), registered in server.go registerCtxRoutes(). Production wiring: recordTelemetry (proxy.go) calls observability.WriteSavings after PutProxySaving+PutProxySession (fail-open); runCtxProxy (ctx_proxy.go) starts StartRollup(cs, 1h) stopped via signal-ctx (no goroutine leak). Bucket keying: 7-day/epoch-aligned weekly (comment fixed from "ISO week"), INSERT OR REPLACE merges re-runs. /metrics + /api/session-summaries out of scope (T10/T11); Stats.Mode seam left for T11/T14. Stdlib only. Gates green: build 0, ctx/... + http/... ok (incl. single_open_test.go), proxy+observability 58 PASS, vet 0, gofmt clean. Commits 440ad7aa (impl) + d44a5c26 (wiring + nit). Review PASS/PASS.
<!-- SECTION:FINAL_SUMMARY:END -->
