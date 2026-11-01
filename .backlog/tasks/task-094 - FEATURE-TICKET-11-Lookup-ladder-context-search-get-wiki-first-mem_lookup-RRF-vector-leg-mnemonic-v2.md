---
id: TASK-094
title: >-
  [FEATURE] TICKET-11 Lookup ladder (context / search / get) + wiki-first
  mem_lookup + RRF + vector leg (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
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
