---
id: TASK-085
title: >-
  [FEATURE] TICKET-02 Migration suite + FTS5 trigger parity (from-empty
  regression) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:41'
updated_date: '2026-11-01 15:24'
labels: []
milestone: m-11
dependencies:
  - TASK-084
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The full migration set must run from an empty DB exactly once, re-open must be a no-op (additive), and the FTS triggers must keep observations_fts/prompts_fts in sync on insert/update/delete. Also pins the BM25 weights (title 5.0, content 1.0, topic_key 3.0) and boost stack (pinned +0.10, recency, stability) that mem_search depends on. Without this, every later ticket's search behavior is untested against the real schema.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 From-empty DB: every migration file applies exactly once; re-open runs zero statements
- [ ] #2 Trigger sync: deleting a row drops the observations_fts count; update keeps it current
- [ ] #3 A <3-rune term falls back to LIKE and returns the seeded short-term row
- [ ] #4 Ranking test with pinned/stale/identical rows returns the spec-ordered ranking (boosts applied)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: orchestrator
created: 2026-11-01 13:58
---
Wave 2 start (2026-11-01): implementer dispatched.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Migration parity (from-empty exactly-once + proven skip on re-open via sqlite_master count) + FTS5 trigger sync (insert/update/soft-delete) + <3-rune LIKE fallback + composite ranking in SQL (BM25 title 5.0/content 1.0/topic_key 3.0 x boosts pinned 0.10/recency 0.06/stability 0.04, order-only assertions). Fix round 1 exposed + fixed 3 root-cause bugs: rank degeneracy (MAX(bm25,1.0) base), ORDER BY direction, Pinned persistence in UpdateObservationForProject. Commits dffdfd8d, bfd3b859. Refs: TASK-085
<!-- SECTION:FINAL_SUMMARY:END -->
