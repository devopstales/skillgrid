---
id: TASK-102
title: >-
  [FEATURE] TICKET-19 v1 -> v2 seed migration (mnemonic migrate) + doctor
  completeness (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 13:14'
labels: []
milestone: m-11
dependencies:
  - TASK-097
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The cutover path (one-way door D4): v1 store files are not auto-migrated (different schema philosophy - spec 002-schema section 6); mnemonic migrate opens the v1 file read-only and seeds a fresh v2 store (observations + sessions + sync-state), leaving v1 files untouched (mtime + bytes asserted). Choosing a different cutover later means re-importing on every machine. Doctor gains the completeness checks.
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
created: 2026-11-02 11:54
---
Wave 10 (2026-11-02): implementer dispatched (v1→v2 seed migration — ONE-WAY DOOR D4).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
v1→v2 seed migration (mnemonic migrate) + doctor completeness — ONE-WAY DOOR D4 PASSED (learning→discovery mapping verified). Migration: v1 read-only (mode=ro, MaxOpenConns(1)), sessions first then observations in id order, deleted v1 rows not seeded, idempotent (OperationID='v1migrate:<id>' → ledger replay no-op), per-row errors don't wedge the seed, {exported,skipped,errors} report. Doctor completeness: pending/done/absent arms. Fix round 1: provenance.writer='v1-migrate' plumbed (SaveRequest.Writer → gateInsert → Provenance.Writer on both dialect legs; seed sets WriterV1Migrate, normal saves leave ''; test asserts json_extract). 14 tests (TestV1* 6 tests + 5 type-mapping subtests + doctor 3 arms). NOTE: commit b26e2cad (concurrent T17 fix) swept in the migrate files — commit-hygiene artifact, not correctness. Commits b26e2cad, acc948b8, 1f482b31, a69f61a3. Refs: TASK-102
<!-- SECTION:FINAL_SUMMARY:END -->
