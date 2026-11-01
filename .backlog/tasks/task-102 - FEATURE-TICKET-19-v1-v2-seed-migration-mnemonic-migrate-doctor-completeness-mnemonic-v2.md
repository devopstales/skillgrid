---
id: TASK-102
title: >-
  [FEATURE] TICKET-19 v1 -> v2 seed migration (mnemonic migrate) + doctor
  completeness (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
  - TASK-097
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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
