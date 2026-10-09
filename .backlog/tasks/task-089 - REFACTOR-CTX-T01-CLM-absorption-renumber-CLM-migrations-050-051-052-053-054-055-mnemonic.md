---
id: TASK-089
title: >-
  [REFACTOR] CTX T01 CLM absorption: renumber CLM migrations 050/051/052 ->
  053/054/055 (mnemonic)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:32'
updated_date: '2026-10-09 13:32'
labels:
  - mnemonic
  - store
milestone: m-10
dependencies: []
references:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Rename the three CLM migration files (050_tool_outputs, 051_indexed_files, 052_context_revisions) to 053/054/055 so CTX's ctx_* tables can start at 056 on the shared SQLite store (ADR-0033). No schema change — CLM tables already exist from the CLM change; this is a pure file rename + any hard-coded path refs in the migration runner.

Current State:
- CLM migrations are numbered 050/051/052 and occupy the range CTX needs
- CTX spec reserves 056+ for its six ctx_* tables

Expected State:
- CLM migrations numbered 053/054/055, applied in order
- No file numbered 050/051/052 remains for CLM
- CTX may number its first migration 056
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ls mnemonic/internal/store/migrations/ shows 053/054/055 for tool_outputs/indexed_files/context_revisions, none at 050/051/052
- [ ] #2 go test ./internal/store/... green: migrations apply cleanly in order
- [ ] #3 Existing CLM rows intact after re-apply
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 CLM rows verified intact after re-apply (row counts unchanged)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Locate the three CLM migration files (050_tool_outputs, 051_indexed_files, 052_context_revisions) in mnemonic/internal/store/migrations/.
2. Rename 050->053, 051->054, 052->055 (git mv to preserve history).
3. Update any hard-coded path/ref to the old numbers in the migration runner (internal/store/migrate*).
4. Run go build ./... and go test ./internal/store/... — confirm migrations apply in order on a fresh store.
5. Verify existing CLM rows are intact after re-apply (row counts unchanged).
6. SATISFIES scenario: clm-renumber.
<!-- SECTION:PLAN:END -->
