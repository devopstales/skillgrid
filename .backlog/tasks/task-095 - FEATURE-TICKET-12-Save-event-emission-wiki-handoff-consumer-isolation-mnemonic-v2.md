---
id: TASK-095
title: >-
  [FEATURE] TICKET-12 Save event emission (wiki handoff) + consumer isolation
  (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 22:00'
labels: []
milestone: m-11
dependencies:
  - TASK-090
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The llm-wiki function consumes saves through exactly one in-process event: SaveEvent{SyncID, Project, Scope, Type, TopicKey, Entity, Importance, Degraded} published once after commit. A consumer failure must never fail the save, and the memory package must not import the wiki package (import graph asserted by test) - this is the seam the sibling llm-wiki change builds against.
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
created: 2026-11-01 21:48
---
Wave 6 (2026-11-01): implementer dispatched (save event emission + consumer isolation).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Save event contract (entity + importance) + consumer isolation: SaveEvent.Entity filled at step-10 (EntityFor = topic_key family head / first significant title term / trimmed title — spec §5 best-effort entity name, not row content; TopicKey+SyncID carry the wiki's re-read handle). Consumer isolation verified+extended (per-subscriber panic recovery, bare-func contract doc) — a panicking consumer never fails the save, later consumers still fire. Wiki handoff = bus consumer post-commit (stub package outside internal/memory proves the dependency direction). Import guard (memory does not import wiki). Exactly-once pinned. topic_key correction (architecture/auth-model→auth/model) fixes the brief's own test against the committed ≤2-level rule. 6 new tests. Review Approved, 0 Critical. Watch-item: revision-path event Importance = fresh-state (same as row) ≠ read-time (Task 11). Commits 7bf9772b. Refs: TASK-095
<!-- SECTION:FINAL_SUMMARY:END -->
