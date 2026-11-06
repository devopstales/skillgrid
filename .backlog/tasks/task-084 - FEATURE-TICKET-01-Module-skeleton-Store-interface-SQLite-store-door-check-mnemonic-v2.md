---
id: TASK-084
title: >-
  [FEATURE] TICKET-01 Module skeleton + Store interface + SQLite store (door
  check) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:37'
updated_date: '2026-11-01 13:58'
labels: []
milestone: m-11
dependencies: []
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The memory function's foundation: a fresh Go module mnemonic-v2/ implementing the dialect-agnostic Store interface and the SQLite dialect with FTS5 trigram retrieval. This ticket is the door check — if a Go module cannot open a per-project SQLite file with WAL through the Store and find a saved observation by trigram search (with the <3-rune LIKE fallback), the blueprint's foundation (INV-2 + the FTS retrieval backbone, spec 001-storage section 3) is invalidated and the change stops here. T0 checklist is already complete (module extraction ADR-0040, v1 archived to mnemonic_back/).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/core/... passes: per-project file opens (two projects, two files)
- [ ] #2 AddObservation/GetObservation round-trip returns obs- sync id (16 hex) + immutable provenance + active state
- [ ] #3 TestDoorCheckFTSRoundTrip: trigram query returns exactly the seeded row; the <3-rune query goes through the LIKE fallback
- [ ] #4 Ledger FK is ON DELETE SET NULL: a hard-deleted observation leaves its save-operation ledger row with observation_id NULL
- [ ] #5 syncid.NewSyncID emits ^<prefix>-[0-9a-f]{16}$ for obs|rel|ses|prm and never collides within a call pair
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Door check TestDoorCheckFTSRoundTrip passes before any later task's code commit
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: orchestrator
created: 2026-11-01 13:23
---
Wave 1 start (2026-11-01): implementer subagent dispatched (door check TICKET-01, one-way D5).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 1 door check complete. Fresh module mnemonic-v2/ (github.com/devopstales/skillgrid/mnemonic-v2, go 1.26.0, cgo-free, modernc.org/sqlite v1.60.1 only). Full Store interface (spec 001-storage §7 + 5 additive methods) with compile-time conformance pin; SQLiteStore with WAL DSN + 10/25/50/100/200ms write-retry; FTS5 trigram external-content tables + 8 triggers; syncid (obs|rel|ses|prm + 16 hex); ledger FK ON DELETE SET NULL; refcounted handle pool with MNEMONIC_DISABLE_CACHE bypass. Door check TestDoorCheckFTSRoundTrip GREEN (trigram hit + 2-rune LIKE fallback). 2 fix rounds (PromptRow full-row shape + test; write-retry on additive writes; 0/0 counters; MarkReviewed branch; InsertPrompt project resolution). Commits 15e009e1, 2c7ad9fe, 6b19bc56. Refs: TASK-084
<!-- SECTION:FINAL_SUMMARY:END -->
