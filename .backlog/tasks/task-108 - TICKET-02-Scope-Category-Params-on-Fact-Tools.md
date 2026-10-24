---
id: TASK-108
title: 'TICKET-02: Scope + Category Params on Fact Tools'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-107
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** `facts.Scope` type (project/global/all), `facts.Route()` returning (dataStore, trailStore), `category` param on `Add/Search/Forget/Decay`, `scope` + `target` params on all MCP `fact_*` tools. Trail semantics: global `fact_add` trails to caller's project store (C3).

**Acceptance:** `fact_add(scope=global, target=memory)` → row in `global.sqlite` with `category='memory'`, trail in project store. `fact_add(scope=global, target=user)` → `category='user'`, user limit (1375). `fact_add(scope=project)` → project store only. All 5 fact tools accept `scope`. Unknown session → clean error.

**SATISFIES:** `happy path — global fact add writes to global store with project trail`, `happy path — project fact add writes to project store (unchanged)`, `edge — global fact add with target user uses user category`, `failure — global fact add with unknown session fails cleanly`

**Files:** `mnemonic/internal/facts/scope.go` (create), `mnemonic/internal/facts/scope_test.go` (create), `mnemonic/internal/facts/facts.go` (modify), `mnemonic/internal/mcp/tools_facts.go` (modify), `mnemonic/internal/mcp/tools_facts_test.go` (modify)

**Size:** ~500 (M)
**Blocked by:** TICKET-01
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 fact_add(scope=global, target=memory) → row in global.sqlite with category='memory'
- [ ] #2 trail in project store for global writes
- [ ] #3 fact_add(scope=global, target=user) → category='user', user limit (1375)
- [ ] #4 fact_add(scope=project) → project store only
- [ ] #5 all 5 fact tools accept scope
- [ ] #6 unknown session → clean error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
