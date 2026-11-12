---
id: TASK-101
title: '[FEATURE] TICKET-18 PostgresStore (the second dialect render) (mnemonic-v2)'
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 07:55'
labels: []
milestone: m-11
dependencies:
  - TASK-084
  - TASK-085
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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

## Comments

<!-- COMMENTS:BEGIN -->
author: orchestrator
created: 2026-11-01 22:00
---
Wave 6 (2026-11-01): implementer dispatched (PostgresStore — second dialect render). NOTE: gate G15 — needs MNEMONIC_PG_TEST_DSN (Postgres ≥15) or PG parity tests skip.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
PostgresStore (second dialect render): full Store interface (S-2, same Go types, no *sql.Rows escaping), tsvector FTS + GIN + weight classes A/B/C + ts_rank_cd ranking, <3-rune ILIKE/pg_trgm fallback, pgvector(768) + HNSW index, schema-per-project (double-quoted, dedicated conn + SET search_path), lib/pq v1.12.3 (ADR-0041). Migration parity re-run against migrations_pg/ (from-empty + idempotent reopen + no-catalog-inflation). 8/8 TestPG* pass WITH DSN (gate G15); skip cleanly without. Fix round 1: deleted the test init() + pgLocalhostDSN (bridge-IP rot hazard — brief's DSN works as-is), collapsed dead VectorSearch error branch, folded tsv into CREATE TABLE. Commits a86d0d46, 1df17db7. Refs: TASK-101
<!-- SECTION:FINAL_SUMMARY:END -->
