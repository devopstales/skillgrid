---
id: TASK-103
title: >-
  [FEATURE] TICKET-20 Concurrency backstop (two-process WAL) + full-suite gate
  (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
  - TASK-096
  - TASK-100
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
