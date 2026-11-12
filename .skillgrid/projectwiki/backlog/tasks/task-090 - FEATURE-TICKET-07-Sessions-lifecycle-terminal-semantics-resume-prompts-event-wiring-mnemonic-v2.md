---
id: TASK-090
title: >-
  [FEATURE] TICKET-07 Sessions (lifecycle, terminal semantics, resume, prompts)
  + event wiring (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 20:01'
labels: []
milestone: m-11
dependencies:
  - TASK-089
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Session lifecycle: start (byte-exact id, runtime_lease_expires_at = now+30min, local-only), missing session auto-register (gate decision S-3), ended session 409 session_already_ended, resume continuation, finalize (atomic six-section session_summary + ended_at + enqueue + session_finalizations row + already_finalized retry semantics). Also user_prompts capture (tri-state capture_prompt: absent -> true) + mem_search_prompts, and this ticket wires the bus.Publish(SaveEvent) call site that TICKET-05 left dangling.
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
created: 2026-11-01 19:36
---
Wave 5 start (2026-11-01): implementer dispatched (events: save → bus → trails).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Event wiring: step-10 bus.Publish at the marked hook (exactly one event per committed save; replayed/noop saves do NOT publish — fix round 1 exposed a real bug where the noop save WAS publishing, now gated `if !row.replayed`). Trail consumer (trails.go) appends session_trails (additive 002 migration); missing session auto-registered (pipeline CheckSession + consumer ensureSession FK guard). Post-commit bus side-effect INV documented (audit row can drop, fact never lost). 10 tests incl. exactly-once, no-republish-on-replay, no-publish-on-noop, ensureSession isolation. Review Approved, 1 fix round. Commits 60cadd77, 92acc30b. Refs: TASK-090
<!-- SECTION:FINAL_SUMMARY:END -->
