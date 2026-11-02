---
id: TASK-092
title: >-
  [FEATURE] TICKET-09 Relations: conflict-candidate detection + relevance gate
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 18:31'
labels: []
milestone: m-11
dependencies:
  - TASK-088
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
After a committed save, conflicting memories must be surfaced, not silently coexist: same project+scope, OR-FTS title pre-filter, relevance gate (shared topic_key OR >=2 significant shared title terms), materialized as pending memory_relations and returned in SaveResult.Candidates with judgment_required: true. The gate is what keeps candidate noise down at scale.
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
created: 2026-11-01 18:05
---
Wave 4 (2026-11-01): implementer dispatched (relations: conflict-candidate detection + relevance gate).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Relations: conflict-candidate detection = OR-FTS title pre-filter (same project+scope) + relevance gate (shared topic_key OR ≥2 significant shared title terms); candidates materialized as pending memory_relations + returned in SaveResult.Candidates, JudgmentRequired:true. AddRelation implemented on SQLiteStore (rel- sync_id, full readback). RawQuery read seam added (store.Search's short-term LIKE fallback misroutes OR queries — ruled legitimate; PG home = Store method for T18). 8 tests incl. the ≥2 boundary (one-term overlap does NOT surface — RED against shared>=1, GREEN against shared>=2). Fix round 1: gate boundary row, gate-reaches re-seed, doc, readback. Commits 30f30600, e4cef7d1, 71078018. Refs: TASK-092
<!-- SECTION:FINAL_SUMMARY:END -->
