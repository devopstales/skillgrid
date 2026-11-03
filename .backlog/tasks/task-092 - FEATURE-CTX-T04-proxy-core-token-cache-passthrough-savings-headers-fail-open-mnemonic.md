---
id: TASK-092
title: >-
  [FEATURE] CTX T04 proxy core: token/cache/passthrough + savings headers +
  fail-open (mnemonic)
status: done
assignee: []
created_date: '2026-10-09 13:34'
updated_date: '2026-10-09 13:40'
labels:
  - mnemonic
  - proxy
milestone: m-10
dependencies:
  - TASK-089
  - TASK-090
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
mnemonic/internal/ctx/proxy/ — a net/http + httputil.ReverseProxy loopback proxy bound to 127.0.0.1:8787. Three modes (token, cache default, passthrough). Forwards Anthropic- and OpenAI-compatible requests, computes X-CTX-Savings accurately, adds savings headers, fails open to passthrough on any compression fault mid-request. Provider retry with provider_retry_backoff_ms; after final retry returns the provider error verbatim.

Current State:
- No proxy exists; loopback serve entry needed in cmd/mnemonic/ctx_proxy.go

Expected State:
- Three-mode loopback proxy forwarding both provider dialects
- Accurate X-CTX-Savings + savings headers
- Fail-open to passthrough on compression fault
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/proxy/... green: Anthropic-compat + OpenAI-compat request complete with accurate X-CTX-Savings and savings headers
- [ ] #2 A fault mid-compression still completes in passthrough (fail-open)
- [ ] #3 Upstream timeout retries with configured backoff then returns the provider error verbatim
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Loopback-only (127.0.0.1:8787); no new 3rd-party dep (stdlib net/http + httputil.ReverseProxy)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create mnemonic/internal/ctx/proxy/proxy.go: net/http + httputil.ReverseProxy bound to 127.0.0.1:8787 (loopback-only).
2. Implement three modes: token, cache (default), passthrough; mode from config + CTX_PROXY_MODE env.
3. Request path: detect Anthropic-compat vs OpenAI-compat, compress per mode, forward upstream, add X-CTX-Savings + savings headers (accurate token delta).
4. Fail-open: any compression fault mid-request -> passthrough (still completes).
5. Provider retry: provider_retry_backoff_ms; after final retry return provider error verbatim.
6. Add cmd/mnemonic/ctx_proxy.go serve entry (loopback).
7. Tests: Anthropic-compat + OpenAI-compat complete with accurate headers; mid-compression fault still completes in passthrough; upstream timeout retries w/ backoff then verbatim error.
8. SATISFIES scenarios: proxy-token, proxy-cache, proxy-passthrough, proxy-fail-open, proxy-retry.
<!-- SECTION:PLAN:END -->
