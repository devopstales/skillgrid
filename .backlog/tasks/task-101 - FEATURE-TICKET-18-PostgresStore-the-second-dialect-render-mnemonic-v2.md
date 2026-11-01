---
id: TASK-101
title: '[FEATURE] TICKET-18 PostgresStore (the second dialect render) (mnemonic-v2)'
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
  - TASK-084
  - TASK-085
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The second dialect render (one-way door D3): store/postgres.go with the Open dialect switch, schema_<project> scoping + fail fast, the full PG render of the local schema (migrations_pg/*.sql, tsvector in place of FTS5, HNSW index for the vector leg per S-1 parity), on lib/pq (ADR-0041 - a driver change later means re-testing the whole PG render). Store.Search/VectorSearch must return the same Go result shape on both dialects (the invariant the code-index and llm-wiki functions rely on).
<!-- SECTION:DESCRIPTION:END -->

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
--ac
<!-- SECTION:PLAN:END -->
