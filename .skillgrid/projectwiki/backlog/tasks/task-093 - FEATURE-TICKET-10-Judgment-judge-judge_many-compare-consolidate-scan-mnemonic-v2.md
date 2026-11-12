---
id: TASK-093
title: >-
  [FEATURE] TICKET-10 Judgment (judge / judge_many / compare) + consolidate +
  scan (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 21:48'
labels: []
milestone: m-11
dependencies:
  - TASK-092
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The human-in-the-loop loop over candidates: mem_judge sets judgment_status=judged with provenance (marked_by_actor/kind/model); mem_judge_many batches 1-32 in one queued write; mem_compare creates a judged relation directly (idempotent per pair); mem_consolidate preview/confirm with fingerprint staleness (stale -> fingerprint_mismatch + fresh plan; confirmed -> sources soft-deleted with superseded_by_sync_id + survivor consolidated_from footer); mem_scan re-surfaces stale candidates.
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
created: 2026-11-01 21:11
---
Wave 6 start (2026-11-01): implementer dispatched (judgment + consolidate + scan).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Judgment + consolidate + scan: Judge (pending→judged + provenance, raced judge = no-op via WHERE judgment_status='pending'), JudgeMany (1-32, one txn, rollback-on-error, already-judged no-op), Compare (LLM→judged relation directly, idempotent per unordered pair, double-check-after-seam TOCTOU guard), Consolidate (two-phase fingerprint-gated merge: preview resolves refusals, sha256 over canonical projection, confirm re-resolves, fingerprint mismatch→fresh plan no writes, survivor inherits newest source + ## consolidated_from footer, sources soft-deleted + superseded_by_sync_id), Scan (page-batched 500-row re-run, new pending only, reuses Task 9 pre-filter+gate). Two-txn consolidate = documented ceiling (monotonic-degrading, fingerprint-observable, INV-1 intact; Task 18 closes). Latent bug fixed: txngate scope hardcoded 'project'→real scope. 15 tests. Review Approved, 0 C/I. Commits 3591f147. Refs: TASK-093
<!-- SECTION:FINAL_SUMMARY:END -->
