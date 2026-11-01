---
id: TASK-093
title: >-
  [FEATURE] TICKET-10 Judgment (judge / judge_many / compare) + consolidate +
  scan (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
  - TASK-092
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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
