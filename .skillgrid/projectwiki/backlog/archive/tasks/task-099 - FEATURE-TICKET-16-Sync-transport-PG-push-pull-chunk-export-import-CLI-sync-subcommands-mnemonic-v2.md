---
id: TASK-099
title: >-
  [FEATURE] TICKET-16 Sync transport (PG push/pull + chunk export/import + CLI
  sync subcommands) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 11:16'
labels: []
milestone: m-11
dependencies:
  - TASK-097
  - TASK-098
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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

author: orchestrator
created: 2026-11-02 11:12
---
Resumed 2026-11-02: working tree carried uncommitted T16 fix-round work (transport.go: MaxPushBytes byte gate + lib/pq time.Time decode in pull). Full suite green EXCEPT a latent T16 regression — TestSyncExportImportCLI fails (sync import exits non-zero; export/manifest ok). Fixing before T17 dispatch.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Sync transport (PG push/pull + chunk export/import + CLI sync subcommands): MutationTransport seam (PG + in-memory fake), Push/Pull/PullAndApply cycle (outbox → PG batch one txn → ack local journal; pull → ApplyPulled INV-4), schema-per-project isolation (double-quoted schema_<canonical>, fail-fast EnsureSchema), pause flag (project_controls.paused → ErrPaused), chunk codec (content-addressed 8-hex SHA-256, key-order-stable canonicalization, project stamp), export/import (gzip JSONL archive + manifest, idempotent via sync_chunks dedupe). CLI sync push/pull/export/import wired (Task 14 stubs replaced). Fix round 1: pushLim+maxPushB enforced at seam (ErrPushTooLarge), occurred_at time.Time/string type-switch (PG TIMESTAMPTZ), errors.Is(ErrPaused), markChunkImported returns error. 18+ tests. Review Approved. NOTE: PG transport SQL is DSN-gated (seam+fake = unit surface; real-PG transport gate tracked for T20/final review). Commits d24ce971, 490d7b4a. Refs: TASK-099
<!-- SECTION:FINAL_SUMMARY:END -->
