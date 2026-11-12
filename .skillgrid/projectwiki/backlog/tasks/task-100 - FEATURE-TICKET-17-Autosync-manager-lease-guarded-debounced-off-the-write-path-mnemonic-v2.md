---
id: TASK-100
title: >-
  [FEATURE] TICKET-17 Autosync manager (lease-guarded, debounced, off the write
  path) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 13:02'
labels: []
milestone: m-11
dependencies: []
references:
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
When sync.auto=true the manager debounces 500 ms, holds a SQLite-backed lease so only one worker cycles at a time, and keeps network I/O off the SQLite write path (the onWrite hook is a buffered-1 channel - a slow transport must not block a save commit). Without the lease, two MCP servers on one data dir double-push; without the channel, every save pays for a network round-trip.
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Autosync manager (TICKET-17): SQLite-backed lease in sync_state (004_sync_lease.sql additive lease_owner/lease_until; atomic conditional-UPDATE takeover with RowsAffected arbiter), 500ms coalescing debounce over a buffered-1 dirty channel, onWrite hook = store txn hook (non-blocking — a 500ms transport never delays a save commit, pinned by a real timing test), sync.auto gating, start/stop in mcp.go (ctx-cancel -> Stop -> releaseLease -> tr.Close), push-then-pull cycle with backoff + 10-strike disable, PG dialect refused with a warning + run-loop recover so a cycle failure can't kill the process. Fix round 1: strengthened TestAutosyncSingleLease (proves one-wins/other-stands-down), atomic lease takeover, PG gate + recover, srv.Start error-path shutdown ordering, cycle-ctx in batchProject, tr.Calls() assertion in the write-path test. 8 autosync tests pass -race -count=3. Commits 340f9ae7, b26e2cad. Refs: TASK-100
<!-- SECTION:FINAL_SUMMARY:END -->
