---
id: TASK-030.04
title: 'TICKET-02: Owner-scoped blend and mem_search signals'
type: feature
status: needs-triage
assignee: []
created_date: '2026-10-02 08:59'
updated_date: '2026-10-03 07:39'
labels:
  - mnemonic-memory-improvements
dependencies:
  - TASK-030.01
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
parent_task_id: TASK-030
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: mem_search calls SearchOwnerScoped (BM25) and BlendedSearch skips visibilityFilter.
Expected state: SearchOwnerScopedBlend fuses FTS and vector under the owner filter and mem_search returns additive signals, matched_via, and score. Per ADR-0018.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 No embedder yields matched_via keyword
- [ ] #2 Embedder on yields hybrid or vector
- [ ] #3 Another owner private row stays hidden
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test TestOwnerScopedBlend and TestMemSearchSignals pass
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: TestOwnerScopedBlendKeywordFloor, TestOwnerScopedBlendHidesPrivate, TestOwnerScopedBlendHybrid, TestMemSearchSignalsKeyword
2. Implement SearchOwnerScopedBlend and additive DTO fields
3. GREEN: go test those names
<!-- SECTION:PLAN:END -->
