---
id: TASK-091
title: >-
  [FEATURE] CTX T02 ctx_ schema: migrations 056-061 (6 tables) + store package
  (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:33'
updated_date: '2026-10-09 13:39'
labels:
  - mnemonic
  - store
milestone: m-10
dependencies:
  - TASK-089
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
Create mnemonic/internal/ctx/store/ and six migrations 056-061: ctx_proxy_mode_history, ctx_proxy_stats, ctx_stats_history, ctx_session_summaries, ctx_traffic_memories, ctx_judgment_cache. Provide store accessors (open via shared SQLite handle, no 2nd DB — per ADR-0033) with typed CRUD + the aggregate rollup query.

Current State:
- No ctx_* tables exist; CLM renumber (T01) frees the 053-055 range

Expected State:
- Six ctx_* tables created on the shared SQLite store
- Typed CRUD accessors + aggregate rollup query
- No second DB handle
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go build ./... green
- [ ] #2 go test ./internal/ctx/store/... green: each of the 6 tables created on a fresh store, each typed insert+query round-trips
- [ ] #3 No created table lacks the ctx_ prefix
- [ ] #4 Opening the store takes no handle of its own (asserted via shared-handle fixture)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Six ctx_* tables present and prefixed; no 2nd DB handle (shared SQLite per ADR-0033)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create mnemonic/internal/ctx/store/ + migrations 056-061:
   - 056_ctx_proxy_mode_history (ts, old_mode, new_mode)
   - 057_ctx_proxy_stats (ts, mode, model, tokens_in, tokens_out, saved_tokens)
   - 058_ctx_stats_history (ts, rollup aggregates)
   - 059_ctx_session_summaries (session_id, ts, totals, summary_json)
   - 060_ctx_traffic_memories (id, category, fingerprint, content, status)
   - 061_ctx_judgment_cache (fingerprint, verdict_json, ts)
2. Implement a Store struct opened on the SHARED SQLite handle (pass *sql.DB / shared conn; no 2nd DB) with typed CRUD per table.
3. Add the aggregate rollup query (tokens by mode/model over a window) used by /stats.
4. Tests: each of the 6 tables created on a fresh store; typed insert+query round-trips; assert ctx_ prefix on every table; assert no 2nd DB handle (shared-handle fixture).
5. SATISFIES scenarios: schema-created, store-crud, rollup-query.
<!-- SECTION:PLAN:END -->
