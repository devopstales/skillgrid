---
id: TASK-097
title: >-
  [FEATURE] CTX T09 observability store: per-request writer + periodic rollup
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:37'
updated_date: '2026-10-09 13:41'
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
