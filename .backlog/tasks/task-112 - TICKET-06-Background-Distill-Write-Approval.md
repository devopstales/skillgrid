---
id: TASK-112
title: 'TICKET-06: Background Distill + Write Approval'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-110
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** `facts.Distill()` — one LLM pass (ADR-0023), scope-classifying prompt, coalescing (latest snapshot wins). Write approval: `pending_facts` table (in migration 073), stage on passive write, `fact_pending_list/approve/reject` MCP tools. Durable (survives restart).

**Acceptance:** One LLM call per session close. Project fact → project store, global fact → global store. Coalescing: one call, latest wins. LLM down → session closes, no facts written. Write approval: staged in `pending_facts`, survives restart, approve → in `facts` + deleted from `pending_facts`. Explicit writes bypass gate. Reject → discarded.

**SATISFIES:** `happy path — distill extracts and routes facts by scope`, `edge — distill coalescing (latest snapshot wins)`, `failure — distill failure does not break session close`, `happy path — passive write is staged and approved`, `edge — pending queue survives server restart`, `edge — explicit writes bypass the approval gate`, `failure — rejected pending fact is discarded`

**Files:** `mnemonic/internal/facts/distill.go` (create), `mnemonic/internal/facts/distill_test.go` (create), `mnemonic/internal/store/migrations/073_skill_usage_outcome.sql` (create: pending_facts table), `mnemonic/internal/mcp/tools_facts.go` (modify), `mnemonic/internal/facts/batch.go` (modify)

**Size:** ~600 (M)
**Blocked by:** TICKET-04
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 one LLM call per session close
- [ ] #2 project fact → project store, global fact → global store
- [ ] #3 coalescing: one call, latest wins
- [ ] #4 LLM down → session closes, no facts written
- [ ] #5 staged in pending_facts, survives restart
- [ ] #6 approve → in facts + deleted from pending_facts
- [ ] #7 explicit writes bypass gate
- [ ] #8 reject → discarded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
