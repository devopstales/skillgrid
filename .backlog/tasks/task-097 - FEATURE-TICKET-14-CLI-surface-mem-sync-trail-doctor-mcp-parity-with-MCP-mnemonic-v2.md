---
id: TASK-097
title: >-
  [FEATURE] TICKET-14 CLI surface (mem / sync / trail / doctor / mcp) + parity
  with MCP (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 19:36'
labels: []
milestone: m-11
dependencies:
  - TASK-087
  - TASK-096
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The CLI dispatch (stdlib flag, v1 pattern): mem (save/get/search/lookup/context/judge/compare/consolidate/scan/update/delete/session/review/doctor/stats/timeline/merge-projects), sync (stubs - real in TICKET-16), trail recent|show, doctor (resolved config, FTS present, vector index present, embed round-trip, degraded on any gap; --strict exits non-zero), mcp. K6: CLI and MCP output must be byte-identical (same handler, parity test).
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
created: 2026-11-01 18:31
---
Wave 4 (2026-11-01): implementer dispatched (sync journal + pull-apply seam).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Sync journal core: ApplyPulled dispatch (observation→AddPulledObservation through-store FTS trigger fires; relation→AddPulledRelation both-endpoints-or-park; session/prompt/project no-op until 16/17). Tombstone floor before delete (upsert→delete→late-upsert stays dead). Deferred-apply 5-cap with monotonic retry counter. Enrollment gates journal (reversible). Idempotent pull (double-apply = one effect). Append-only outbox (pull neither appends to nor rewrites local journal). Canonical payload fix in txngate (session_id/type/title/content/project/scope — cross-machine apply). 13 tests. Review Approved, 0 C/I. Commits 9df8b63b. Refs: TASK-097
<!-- SECTION:FINAL_SUMMARY:END -->
