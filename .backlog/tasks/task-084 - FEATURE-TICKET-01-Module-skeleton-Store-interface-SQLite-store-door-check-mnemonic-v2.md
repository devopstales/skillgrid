---
id: TASK-084
title: >-
  [FEATURE] TICKET-01 Module skeleton + Store interface + SQLite store (door
  check) (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:37'
labels: []
milestone: m-11
dependencies: []
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
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
