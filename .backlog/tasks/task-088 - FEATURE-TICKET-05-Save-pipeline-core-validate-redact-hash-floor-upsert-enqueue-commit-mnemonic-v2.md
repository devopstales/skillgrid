---
id: TASK-088
title: >-
  [FEATURE] TICKET-05 Save pipeline core (validate -> redact -> hash floor ->
  upsert -> enqueue -> commit) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:44'
updated_date: '2026-11-01 17:16'
labels: []
milestone: m-11
dependencies:
  - TASK-085
  - TASK-087
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The 10-step mem_save contract: validation, project/session resolution, redaction + truncation, the normalized_hash dedup floor (identical content -> same id, duplicate_count bumped; same topic_key -> in-place revision + observation_versions snapshot), enqueue of the sync mutation, commit, and the result shape {id, sync_id, revision_count, dedup{verdict,reason,merged_into_sync_id}}. The eventbus is created here; the Publish call site lands in TICKET-07. This is the tracer thread working save path.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Save persists with valid sync id + immutable provenance; retrievable by numeric id and sync id
- [ ] #2 Redaction + truncation applied before the hash is computed
- [ ] #3 Identical content -> same id, duplicate_count bumped, last_seen_at updated, no new row
- [ ] #4 Same topic_key -> revision_count incremented + pre-state snapshot in observation_versions
- [ ] #5 Validation errors (empty title, bad type) reject before any write; result JSON shape matches spec 001-observations section 3
<!-- AC:END -->

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
1. Read blueprint Task 5 + spec 01-memory/001-observations section 2
2. TDD: pipeline tests RED (acceptance.feature save-* scenarios)
3. Implement pipeline.go, dedup.go (hash branch only), importance.go, eventbus.go (Publish unwired)
4. go test ./internal/memory/... -run TestSave GREEN
5. Commit feat(mnemonic-v2): save pipeline core
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: orchestrator
created: 2026-11-01 16:25
---
Wave 3 start (2026-11-01): implementer dispatched (save pipeline core — tracer thread working save path).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Save pipeline core: 10-step contract steps 1-3 + 5-floor (validate → resolve → redact/truncate → normalized_hash dedup floor → upsert → enqueue → commit → result). Hash dedup (identical content = same id + dup_count bump, no new row; same topic_key = in-place revision + observation_versions snapshot). INV-1/2/3 hold (provenance immutable, store-interface-only, hash floor = degraded path). LLM dedup branch stubbed for T08. eventbus Publish created, not wired (T07). 24 memory tests, real SQLite (no mock store). Review Approved, 0 C/I, 6 Minor deferred to final review. Commits f8983234, dd1a0a3f. Refs: TASK-088
<!-- SECTION:FINAL_SUMMARY:END -->
