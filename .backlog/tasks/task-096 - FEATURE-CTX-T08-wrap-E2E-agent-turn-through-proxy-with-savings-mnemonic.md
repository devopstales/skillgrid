---
id: TASK-096
title: '[FEATURE] CTX T08 wrap E2E: agent turn through proxy with savings (mnemonic)'
status: done
assignee: []
created_date: '2026-10-09 13:35'
updated_date: '2026-10-12 11:10'
labels:
  - mnemonic
  - wrap
  - e2e
milestone: m-10
dependencies:
  - TASK-093
  - TASK-094
  - TASK-095
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
End-to-end: a wrapped agent completes a full turn through the proxy and the savings headers are present end-to-end (not just unit-level). Verifies the launcher + proxy + store path together.

Current State:
- T07 launcher + T05/T06 proxy pieces exist in isolation; not yet verified as one end-to-end turn

Expected State:
- Wrapped agent turn completes through the proxy with savings headers
- Agent source tree unchanged
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/wrap/... (E2E test) green: the wrapped agent's turn reaches the proxy, completes, and X-CTX-Savings is present on the response
- [ ] #2 The agent's source tree is byte-identical to before the launch
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
1. Add an E2E test in internal/ctx/wrap: start the real proxy on a loopback port, wrap a stub agent that issues one LLM call through it.
2. Assert the stub's turn reaches the proxy, completes, and X-CTX-Savings is present on the response (end-to-end, not unit-level).
3. Assert the agent's source tree is byte-identical before/after the launch.
4. SATISFIES scenario: wrap-e2e-savings.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Test-only ticket: added `mnemonic/internal/ctx/wrap/e2e_test.go` proving the launcher (T07) + proxy (T04/T05) + store (T02) path works end-to-end. A wrapped claude (bash+curl stub that reads the injected ANTHROPIC_BASE_URL) POSTs one real turn to `<proxy>/v1/messages`; the real proxy (token mode, SavingsHeaders on, real shared store) transforms the body ([ctx-summarized] marker), forwards to an httptest upstream, and writes X-CTX-Savings non-zero + status 200 back. Telemetry lands on the real shared store (ListProxySavings/GetProxySession for the x-ctx-session tag) and the agent source tree is byte-identical before/after. SATISFIES wrap-core4-launches end-to-end Then clause. Commits: 6035d03a (E2E) + 5a5c38fb (review F1 proxyEnvMu env-guard + NOTE, F4 upstreamMu race-guard; -race clean). No production code changed.
<!-- SECTION:FINAL_SUMMARY:END -->
