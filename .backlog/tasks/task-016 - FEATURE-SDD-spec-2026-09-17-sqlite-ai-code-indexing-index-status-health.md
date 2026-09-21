---
id: TASK-016
title: '[FEATURE] SDD spec: 2026-09-17-sqlite-ai-code-indexing (index status + health)'
status: done
assignee: []
created_date: '2026-09-21 07:15'
labels: []
dependencies: []
references:
  - .skillgrid/archive/2026-09-17-sqlite-ai-code-indexing/tasks.md
  - .skillgrid/archive/2026-09-17-sqlite-ai-code-indexing/findings.md
  - skillgrid-cli/internal/mnemonic/mcp/tools_code.go
  - skillgrid-cli/internal/mnemonic/mcp/server_test.go
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track the SDD spec **2026-09-17-sqlite-ai-code-indexing**: fast index-freshness/health signal for the agent before it relies on `code_explore` / `code_hybrid_search` (modeled on ory/lumen's `health_check` + `index_status`), sliced into TICKET-01 (index_status) + TICKET-02 (health_check).

**Current State:**
- Delivered via `code_status` in `skillgrid-cli/internal/mnemonic/mcp/tools_code.go`: coverage, staleness (fingerprint drift), import cycles, provider availability (fts/vector/graph), extractor stamp — registered in the MCP server and covered by `server_test.go`
- Both tickets' scope is subsumed by `code_status`; no separate `code_health_check`/`code_index_status` files exist
- The compact-search slice (`context`/`unfold` params) is a separate spec: 2026-09-17-compact-search-output (still stalled)

**Expected State:**
- Done. Archived to `.skillgrid/archive/2026-09-17-sqlite-ai-code-indexing/` on 2026-09-21.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `code_status` MCP tool reports index state (file/symbol/chunk counts, freshness) on a seeded store
- [ ] #2 Unindexed repo returns a clear 'not indexed' state, not an error
- [ ] #3 Code tool surface remains hidden by default (SKILLGRID_MCP_CODE_TOOLS filter)
- [ ] #4 Fast health path: no embed round-trip, no full doctor
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed during the 2026-09-21 specs audit: both tickets' scope is delivered by the live `code_status` MCP tool (tools_code.go), which reports index state, fingerprint-drift staleness, import cycles, and provider availability — verified present in the registered tool list and server_test.go. Spec folder archived to .skillgrid/archive/2026-09-17-sqlite-ai-code-indexing/. The sibling compact-search-output spec was kept open (stalled) and is NOT covered by this ticket. Ticket was missing from the backlog — created retroactively and marked done.
<!-- SECTION:FINAL_SUMMARY:END -->
