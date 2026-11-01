---
id: TASK-091
title: >-
  [FEATURE] TICKET-08 Dedup LLM path (primary Classify + trust-but-verify)
  (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
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
