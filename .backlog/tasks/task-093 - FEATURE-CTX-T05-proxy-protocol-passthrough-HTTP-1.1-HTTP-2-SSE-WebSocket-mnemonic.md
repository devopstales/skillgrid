---
id: TASK-093
title: >-
  [FEATURE] CTX T05 proxy protocol passthrough: HTTP/1.1, HTTP/2, SSE, WebSocket
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:34'
updated_date: '2026-10-09 13:40'
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
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Ensure the proxy passes through HTTP/1.1, HTTP/2, SSE (chunked streaming), and WebSocket frames intact (no body buffering that breaks streaming). Wire the serve entry to enable the needed features (h2, upgrade).

Current State:
- Proxy core (T04) forwards requests but streaming protocols not yet verified

Expected State:
- SSE, WebSocket, and HTTP/2 pass through intact
- No stream truncated by the proxy
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/proxy/... green: SSE stream chunks arrive in order and complete
- [ ] #2 A WebSocket handshake upgrades and a ping/pong round-trips
- [ ] #3 An HTTP/2 request completes; no stream is truncated by the proxy
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
1. In the proxy serve entry, ensure HTTP/2 (h2) is enabled and chunked/SSE streaming is not buffered.
2. Add WebSocket upgrade passthrough (httputil handles 101 if the upstream supports it; verify frame round-trip).
3. Add a test: SSE stream chunks arrive in order and complete (no truncation).
4. Add a test: WebSocket handshake upgrades and a ping/pong round-trips.
5. Add a test: an HTTP/2 request completes without truncation.
6. SATISFIES scenarios: proxy-sse, proxy-websocket, proxy-http2.
<!-- SECTION:PLAN:END -->
