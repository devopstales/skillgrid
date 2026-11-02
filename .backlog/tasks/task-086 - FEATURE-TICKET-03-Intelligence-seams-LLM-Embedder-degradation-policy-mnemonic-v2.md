---
id: TASK-086
title: >-
  [FEATURE] TICKET-03 Intelligence seams (LLM + Embedder + degradation policy)
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:41'
updated_date: '2026-11-01 15:56'
labels: []
milestone: m-11
dependencies:
  - TASK-084
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The LLM-first seams with explicit deterministic degradation (INV-3): the LLM interface (4-way Classify, Judge, SuggestCandidates), the Embedder interface, an OpenAI-compatible provider over one base URL, and the degrade package (bounded retry, degraded flag + reason, never a silent floor). Every semantic operation routes through these seams, so their contract — including the zero-value-decision = degraded path — must exist before the save pipeline can degrade correctly.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 httptest OpenAI-compat server: 4-way classify payload returns the typed DedupDecision
- [ ] #2 Nil seam -> zero-value decision + degraded: true with reason
- [ ] #3 Failing seam (HTTP 500) -> exactly one bounded retry then the deterministic floor with reason
- [ ] #4 Embedder round-trip returns fixed-dim vectors; null embedder -> degraded
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
created: 2026-11-01 15:24
---
Wave 2 (2026-11-01): implementer dispatched.
---

author: orchestrator
created: 2026-11-01 15:46
---
Review complete (2026-11-01): Approved with 1 plan-mandated Important (degrade.Run signature bend) + 4 Minor. Controller ruling: keep the any-based Run, correct the ponytail comment, restore the exact brief signatures for the LLM/Embedder seams (already exact per reviewer). Fix round 1 dispatched.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
LLM seam (4-way Classify → DedupDecision, Judge, Compare, Extract) + Embedder seam (Embed/Model/Dimensions + Null adapter) + OpenAI-compatible provider (one base URL, chat + embeddings, stdlib net/http, no new dep) + degrade package (nil seam → floor + degraded, no retry; failing seam → exactly one bounded retry → floor; per-call isolation; ctx passthrough). INV-3 tested per-call. 23/23 tests. Fix round 1: ponytail comment corrected (real any-vs-generic tradeoff), ctx param on Run, doJSON dedup, dead branch, Null doc. Commits 0dda6b64, 49269d7a. Refs: TASK-086
<!-- SECTION:FINAL_SUMMARY:END -->
