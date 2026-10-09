---
id: TASK-098
title: >-
  [FEATURE] CTX T10 observability endpoints: /stats, /stats-history, /metrics,
  /api/session-summaries (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:37'
updated_date: '2026-10-09 13:41'
labels:
  - mnemonic
  - http
  - observability
milestone: m-10
dependencies:
  - TASK-091
  - TASK-097
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
mnemonic/internal/http/ctx.go (new) — fold GET /stats, GET /stats-history, GET /metrics (Prometheus text), and GET /api/session-summaries into the existing registerRoutes() mux (server.go). Read-only, fail-open (writeJSON pattern from clm.go).

Current State:
- No observability HTTP routes exist; registerRoutes() mux pattern + fail-open handler already in place

Expected State:
- 4 read-only routes registered in the serve mux
- JSON consistent with stored data
- Loopback-restricted
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/http/... green: the 4 routes are registered and answer 200 on loopback
- [ ] #2 Each endpoint returns JSON consistent with stored data
- [ ] #3 A non-loopback origin is rejected (no data leak)
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
1. Create mnemonic/internal/http/ctx.go with 4 read-only handlers: GET /stats, GET /stats-history, GET /metrics (Prometheus text, hand-emitted), GET /api/session-summaries.
2. Register them in registerRoutes() (server.go) using the withProjectHandle + writeJSON fail-open pattern from clm.go.
3. Loopback-restrict (no data leak to non-loopback origins).
4. Tests: the 4 routes registered + 200 on loopback; JSON consistent with stored data; non-loopback rejected.
5. SATISFIES scenarios: http-stats, http-metrics, http-loopback-only.
<!-- SECTION:PLAN:END -->
