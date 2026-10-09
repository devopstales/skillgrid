---
id: TASK-094
title: >-
  [FEATURE] CTX T06 /v1/compress endpoint (loopback-only) + mode-switch runtime
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:34'
updated_date: '2026-10-09 13:41'
labels:
  - mnemonic
  - proxy
milestone: m-10
dependencies:
  - TASK-092
references:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add POST /v1/compress (compresses a message list, no upstream completion) — loopback-only: returns 404 (not 403) from a non-loopback origin. Add runtime mode switch (writes ctx_proxy_mode_history old+new, emits proxy-mode-changed, exposes current mode in GET /stats). Support CTX_PROXY_MODE env + restart-to-mode.

Current State:
- Proxy core (T04) forwards but /v1/compress endpoint and runtime mode switch not yet present

Expected State:
- Loopback-only POST /v1/compress with no upstream call
- Runtime mode switch with history + event + stats exposure
- CTX_PROXY_MODE env honored on restart
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/proxy/... green: loopback POST returns a compressed list with no upstream call
- [ ] #2 Non-loopback POST returns 404 (assert not 403)
- [ ] #3 Switching cache->token writes a history row + emits proxy-mode-changed + GET /stats shows token
- [ ] #4 Setting CTX_PROXY_MODE env + restart starts in that mode
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
1. Add POST /v1/compress to the proxy mux: compresses a message list, returns it WITHOUT an upstream call.
2. Loopback-only guard: non-loopback origin -> 404 (not 403).
3. Add runtime mode switch (ctx proxy mode <m> + env): writes ctx_proxy_mode_history old+new, emits proxy-mode-changed event, exposes current mode in GET /stats.
4. Support CTX_PROXY_MODE env + restart-to-mode.
5. Tests: loopback POST returns compressed list with no upstream call; non-loopback POST -> 404 (assert not 403); cache->token switch writes history + emits event + GET /stats shows token; env+restart starts in that mode.
6. SATISFIES scenarios: compress-loopback, compress-404, mode-switch, mode-env.
<!-- SECTION:PLAN:END -->
