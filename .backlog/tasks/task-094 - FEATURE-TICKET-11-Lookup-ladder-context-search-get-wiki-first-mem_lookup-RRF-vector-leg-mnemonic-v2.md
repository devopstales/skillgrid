---
id: TASK-094
title: >-
  [FEATURE] TICKET-11 Lookup ladder (context / search / get) + wiki-first
  mem_lookup + RRF + vector leg (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 21:10'
labels: []
milestone: m-11
dependencies:
  - TASK-085
  - TASK-088
  - TASK-090
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The retrieval surface: mem_search compact rows (300-char preview + truncated flag) with the full boost stack; mem_lookup (exact topic_key -> single upserted row or none; id/sync_id -> full row; phrase -> <=5 FTS rows, no boosts; never widens - wiki-first per OPEN-12); mem_context byte budget; Store.VectorSearch real via the vec0 leg + RRF fusion recording the route (fts/vector/rrf); the trail writer records every retrieval (query, route, filters, ids, degraded) and bumps retrieval_usage (feeds the importance formula); MarkReviewed resets review_after = now + type-months.
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
created: 2026-11-01 20:42
---
Wave 5 (2026-11-01): implementer dispatched (retrieval trails + importance recomputation).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Retrieval trails + importance recompute + lookup ladder: retrieval_trails table (003 migration), trail recorded on Search+Lookup (zero-row search also appends — matches brief's own test sketch). Importance recompute over real retrieval_usage (v1 formula 0.6·min(usage,50)/50 + 0.37·recency + 0.2·durability) AFTER the BumpRetrievalUsage (cache doesn't lag one retrieval; monotonic 3→10 tested). VectorSearch: in-Go cosine over stored blobs (no vec0/ADR), dimension-mismatch skip, empty→degraded, k=2·limit, RRF 1/(60+rank) fusion; null embedder→degraded (INV-3). Context: pinned-first, ≤20 pinned cap, 16KiB default/64KiB max byte budget. Compact rows (300-char preview + Truncated + single relations scan, INV-6). Review: MarkReviewed clock restart load-bearing+tested; list path no-op (named Task 13 deferral). 10 SATISFIES scenarios + 4 triangulations. Review Approved, 0 C/I. Commits ad80c23c. Refs: TASK-094
<!-- SECTION:FINAL_SUMMARY:END -->
