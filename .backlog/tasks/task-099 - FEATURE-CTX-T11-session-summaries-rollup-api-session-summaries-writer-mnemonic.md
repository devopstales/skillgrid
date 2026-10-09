---
id: TASK-099
title: >-
  [FEATURE] CTX T11 session summaries rollup + /api/session-summaries writer
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:38'
updated_date: '2026-10-09 13:42'
labels:
  - mnemonic
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
Session summary rollup: at session end (or interval), compute a summary and write ctx_session_summaries; expose it via GET /api/session-summaries. Reuses the observability rollup loop from T09.

Current State:
- Observability writer/rollup (T09) exists but no session-level summary is written

Expected State:
- Session summary written to ctx_session_summaries
- Exposed via /api/session-summaries
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/observability/... green: after a session, a summary row is written to ctx_session_summaries
- [ ] #2 GET /api/session-summaries returns the written summary
- [ ] #3 The rollup interval from config is honored (no double-write)
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
1. Extend the observability rollup loop (T09): at session end/interval, compute a session summary.
2. Write it to ctx_session_summaries (session_id, totals, summary_json).
3. Ensure GET /api/session-summaries (T10) returns the written summary.
4. Honor the config rollup interval (no double-write).
5. Tests: after a session a summary row is written; /api/session-summaries returns it; interval honored (no double-write).
6. SATISFIES scenarios: session-summary-write, session-summary-read.
<!-- SECTION:PLAN:END -->
