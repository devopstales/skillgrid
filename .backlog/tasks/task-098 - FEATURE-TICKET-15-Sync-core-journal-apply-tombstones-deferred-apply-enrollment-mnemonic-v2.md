---
id: TASK-098
title: >-
  [FEATURE] TICKET-15 Sync core (journal apply, tombstones, deferred-apply,
  enrollment) (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
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
The local half of sync: pulled mutations must apply through the same store methods (INV-4 - a raw SQL bypass would let a remote write escape the dedup/review/tombstone floors); tombstone floors make a late upsert after a delete unable to resurrect the row; missing-FK mutations defer into sync_apply_deferred (5-retry cap then dead); sync_enrolled_projects gates the journal (an un-enrolled project cannot silently accumulate mutations).
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
