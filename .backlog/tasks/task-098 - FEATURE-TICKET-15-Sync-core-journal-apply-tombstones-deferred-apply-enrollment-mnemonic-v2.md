---
id: TASK-098
title: >-
  [FEATURE] TICKET-15 Sync core (journal apply, tombstones, deferred-apply,
  enrollment) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 10:32'
labels: []
milestone: m-11
dependencies:
  - TASK-088
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The local half of sync: pulled mutations must apply through the same store methods (INV-4 - a raw SQL bypass would let a remote write escape the dedup/review/tombstone floors); tombstone floors make a late upsert after a delete unable to resurrect the row; missing-FK mutations defer into sync_apply_deferred (5-retry cap then dead); sync_enrolled_projects gates the journal (an un-enrolled project cannot silently accumulate mutations).
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
created: 2026-11-02 09:30
---
Wave 8 start (2026-11-02): implementer dispatched (CLI surface + parity with MCP).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
CLI surface (mem/sync/trail/doctor/mcp) over the shared handler: thin dispatch (flag-parse → openRuntime → h.<Method>() → finish; no business logic in the adapter). K6 parity gate PROVEN — parity_test drives the same runCLI path vs the same mcp.CallTool path → byte-identical JSON (sync_id exempted). JSON-shape test catches silent floor + ghost degraded flag (INV-3/6). Doctor strict-mode pinned (exit 1 on capability gap, 0 without --strict). --project feeds handler resolution; --all + --project = usage error. Fix round 1: --all fan-out honest rejection (exit 2 'not yet wired' on read commands; real fan-out deferred to T16 cross-project store — brief's SATISFIES never claimed cli-all-fanout); extractDegraded moved to envelope.ExtractDegraded (both adapters call it, K6 seam). 22+ CLI tests. Binary builds + runs. Commits 9db4e83d, 81a38631. Refs: TASK-098
<!-- SECTION:FINAL_SUMMARY:END -->
