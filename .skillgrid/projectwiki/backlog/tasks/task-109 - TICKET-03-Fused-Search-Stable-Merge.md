---
id: TASK-109
title: 'TICKET-03: Fused Search (Stable Merge)'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-108
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** `facts.FusedSearch()` — stable merge by each store's BM25 rank (not RRF), labels each hit with scope + category. Trail goes to project store only. `scope=all` MCP param.

**Acceptance:** Both stores searched; hits interleaved by rank; each labeled with scope + category. One-store match returns only that store's hit. Trail in project store, not global. Empty stores → empty result, no error.

**SATISFIES:** `happy path — fused search returns hits from both stores`, `edge — fused search with no matches in one store`, `edge — fused search trail goes to project store only`, `failure — fused search with empty stores returns empty`

**Files:** `mnemonic/internal/facts/scope.go` (modify: add FusedSearch), `mnemonic/internal/facts/facts_test.go` (modify), `mnemonic/internal/mcp/tools_facts.go` (modify)

**Size:** ~300 (S)
**Blocked by:** TICKET-02
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 both stores searched on scope=all
- [ ] #2 hits interleaved by each store's BM25 rank
- [ ] #3 each hit labeled with scope and category
- [ ] #4 one-store match returns only that store's hit
- [ ] #5 trail in project store, not global
- [ ] #6 empty stores → empty result, no error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
