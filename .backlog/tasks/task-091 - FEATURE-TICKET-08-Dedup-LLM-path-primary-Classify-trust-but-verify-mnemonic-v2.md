---
id: TASK-091
title: >-
  [FEATURE] TICKET-08 Dedup LLM path (primary Classify + trust-but-verify)
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 20:42'
labels: []
milestone: m-11
dependencies:
  - TASK-086
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
Fills the non-nil-seam branch of dedup.Run: the LLM Classify call is primary, the hash floor (TICKET-05) remains the degraded branch - one code path, the degraded flag distinguishes them. Trust-but-verify: the LLM candidate id must exist and be in the redacted-hash pre-filter set, else the hash floor verdict stands. Noop is first-class (verdict=noop, no write).
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
created: 2026-11-01 20:01
---
Wave 5 (2026-11-01): implementer dispatched (LLM dedup branch — fills the Task 5 stub).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
LLM dedup branch (fills Task 5 stub): LLM-first Classify over a project-scoped recent-window pre-filter (≤20, most-recent first, non-deleted), trust-but-verify (positive CandidateID verified live, else repaired to pre-filter head; empty pre-filter → no-op). 4-way verdicts mapped to save-path: noop→MarkDuplicate, add→fresh insert, update→in-place ReviseObservation + MergedIntoSyncID, delete→SupersedeTarget (soft-delete + superseded_by_sync_id + enrollment-gated delete journal + pending supersedes relation) + fresh insert. INV-3 exact (nil seam→floor no retry; failing seam→one bounded retry→floor). Fixed a latent Task 5 FTS bug (old preFilter Search({Query:"trigram"}) returned 0 rows for arbitrary content → direct SQL scan). commitSave rollback-contract restructure (named-return err) for the SetMaxOpenConns(1) pool. 14 dedup tests. Review Approved, 0 Critical; scope-agnostic pre-filter tracked as follow-up (non-blocker). Commits 6c8d4d09. Refs: TASK-091
<!-- SECTION:FINAL_SUMMARY:END -->
