---
id: TASK-030.02
title: 'TICKET-04: Query embedding cache'
status: ready-for-agent
assignee: []
created_date: '2026-10-02 08:59'
labels:
  - mnemonic-memory-improvements
dependencies: []
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
parent_task_id: TASK-030
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: every EmbedQuery recomputes the vector.
Expected state: query_cache reuses a SHA-256 hash of model and query for 7 days and never stores the query text.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Second identical query does not call the embedder
- [ ] #2 An 8-day-old row and a different model miss
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test query cache and migration 045 pass
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: TestQueryCacheHit, TestQueryCacheMiss, TestMigration045
2. Add query_cache in 045_search_aids.sql and CachedEmbedQuery
3. GREEN: go test those names
<!-- SECTION:PLAN:END -->
