---
id: TASK-089
title: >-
  [FEATURE] TICKET-06 Replay-safe saves (operation idempotency ledger)
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 21:11'
labels: []
milestone: m-11
dependencies:
  - TASK-088
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Agent saves are retried: without an idempotency ledger a replayed mem_save double-writes (second upsert, second sync mutation, second event). The ledger step (4.5, before dedup) makes replay identical-payload return the committed id, changed-payload a 409 operation_conflict, and deleted-target a 410 operation_expired (relying on TICKET-01 SET NULL FK).
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
created: 2026-11-01 17:16
---
Wave 4 start (2026-11-01): implementer dispatched (replay-safe saves).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Replay-safe saves: ledger step 4.5 before dedup. Identical OperationID+payload → committed id, Replayed:true, no second upsert/mutation/event, exactly one ledger row. Changed payload (any dedup-identity field) → 409 operation_conflict. Ledger row with NULL observation_id (SET NULL) → 410 operation_expired. Multi-field length-prefixed fingerprint (Type/Title/Project/Scope/TopicKey/Content) consistent on both compute + stored sides. 6 tests. Fix round 1 widened the content-only fingerprint. Commits 41adbfb0, 1789e77d. Refs: TASK-089
<!-- SECTION:FINAL_SUMMARY:END -->
