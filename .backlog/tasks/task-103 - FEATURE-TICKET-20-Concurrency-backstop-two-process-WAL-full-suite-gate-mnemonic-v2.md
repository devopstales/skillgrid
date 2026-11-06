---
id: TASK-103
title: >-
  [FEATURE] TICKET-20 Concurrency backstop (two-process WAL) + full-suite gate
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-03 18:01'
labels: []
milestone: m-11
dependencies:
  - TASK-096
  - TASK-101
  - TASK-102
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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
Two-process WAL concurrency backstop + full-suite gate. TestConcurrentProcessWAL (real re-exec, MNEMONIC_CONCURRENT_WORKER=1, TestMain branch) + TestConcurrentProcessWALBusyExhaustion (10ms busy_timeout, parent 750ms hold, cross-process retry target). Both PASS (-count=1, -count=5, -race). Full internal/... suite: 18/18 packages ok. CONVERGENCE NOTE: the concurrent session already completed + SHIPPED this change (commit 77474091, Nov 2) including its own T20 (bff3940c, 0bf051bd, 809ff13d, d62059ca). This T20 (b46df781, Nov 3) is a second, independent implementation that landed after the ship — redundant but valid. Both on release/2; the change is shipped. All 20 tickets (TASK-084..103) are done. Refs: TASK-103
<!-- SECTION:FINAL_SUMMARY:END -->
