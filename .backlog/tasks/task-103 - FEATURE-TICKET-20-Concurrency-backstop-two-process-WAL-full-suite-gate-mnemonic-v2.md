---
id: TASK-103
title: >-
  [FEATURE] TICKET-20 Concurrency backstop (two-process WAL) + full-suite gate
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 13:36'
labels: []
milestone: m-11
dependencies:
  - TASK-096
  - TASK-101
  - TASK-102
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The held-out backstop (T21): two real OS processes writing to the same project file - one holds the store open while the other runs mem_save - both complete without corruption (WAL + busy timeout + retry backoff from TICKET-01). If it fails, the fix lands here and re-enters the retry wrapper. Then the full-suite gate: the entire mnemonic-v2 test suite + go vet + build, and every gate command in acceptance.feature (G1-G16) executed with evidence captured.
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
created: 2026-11-02 13:02
---
Wave 11 start (2026-11-02): two-process WAL backstop + full-suite gate dispatched. Pre-state: all 19 prior tickets done; module green except a flagged one-off timing flake in TestWritePathNeverBlocksOnNetwork (121ms vs 100ms threshold, 10/10 isolated re-runs pass).
---

author: orchestrator
created: 2026-11-02 13:14
---
Wave 11 (final) start (2026-11-02): implementer dispatched (concurrency backstop two-process WAL + full-suite gate).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Two-process WAL backstop + full-suite gate (TICKET-20): TestConcurrentProcessWAL — two real OS processes (test-binary re-exec via MNEMONIC_CONCURRENT_WORKER, TestMain mode switch) on one t.TempDir project file; worker does 20 real AddObservation saves through store.Open's WAL DSN + retry wrapper while the parent holds a live session and interleaves 10 rows; parent-side readback asserts 21 worker + 11 parent rows. No retry-wrapper fix needed. Held at -count=5 and -race. Full gate: build+vet+test ./... green (13 packages); G1–G16 all green with EVIDENCE captured in acceptance.feature — G15 ran WITH the live PG DSN (8/8 TestPG); G5/G6/G7/G10/G16 name-pattern substitutions documented per gate; G1 count + G16 CHECK corrected in the review fix round (real packages). Flake: TestWritePathNeverBlocksOnNetwork ran 10×, no flake, threshold kept at 100ms. Review: NO-GO (C-1 G16 CHECK/EVIDENCE mismatch + C-2 missing reviewer-side test evidence) → fix round 809ff13d/d62059ca + controller-side evidence run (suite green, -race -count=3 pass). Reviewer-side test evidence gap (shell denied in reviewer runtime) documented in the review record — controller re-ran all five commands green. Commits bff3940c, 0bf051bd, 809ff13d, d62059ca. Refs: TASK-103
<!-- SECTION:FINAL_SUMMARY:END -->
