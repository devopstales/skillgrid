---
id: TASK-099
title: >-
  [FEATURE] TICKET-16 Sync transport (PG push/pull + chunk export/import + CLI
  sync subcommands) (mnemonic-v2)
status: in-progress
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 10:32'
labels: []
milestone: m-11
dependencies:
  - TASK-097
  - TASK-098
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The network half of sync: push inserts pending mutations (<=100/batch, <=8 MiB) into schema_<project>.mutations in one transaction and acks last_acked_seq; pull is scoped to schema_<project> and fails fast (no public fallback) when the project schema cannot be resolved; the chunk codec makes export/import deterministic (chunk id = first 8 hex of SHA-256 of the canonical payload); sync export -> manifest.json + chunks/*.jsonl.gz; sync import loads the archive dialect-neutrally. One code path, two callers (K4: CLI + the autosync manager).
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
created: 2026-11-02 10:32
---
Wave 9 start (2026-11-02): implementer dispatched (sync transport — PG push/pull + chunk export/import).
---
<!-- COMMENTS:END -->
