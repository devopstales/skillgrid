---
id: TASK-023
title: 'TICKET-02: MarkSuperseded + supersede chain'
status: done
assignee: []
created_date: '2026-10-01 18:17'
updated_date: '2026-10-01 20:25'
labels:
  - bitemporal-audn
dependencies:
  - TASK-022
references:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/briefing.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/tasks.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/acceptance.feature
  - .skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md
documentation:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TICKET-02 of 2026-09-24-bitemporal-audn (blueprint Task 2, work unit 2).

Current State: no primitive stamps an observation as superseded — the bi-temporal window from TICKET-01 exists but nothing populates invalid_at/superseded_by/status, and no supersedes edge is recorded.

Expected State: Service.MarkSuperseded (lifecycle.go, mirrors BumpDuplicate) stamps invalid_at=now, superseded_by=?, status='superseded' in one transaction and writes the supersedes edge in memory_relations.

Blocked by: TICKET-01 (task-022).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 After MarkSuperseded(a, b) + the edge, Get(a) shows InvalidAt set
- [ ] #2 Get(a).SupersededBy == b
- [ ] #3 Get(a).Status == "superseded"
- [ ] #4 A memory_relations row (src=a, dst=b, relation="supersedes") exists
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenario green: Delete arm produces the supersede chain
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: extend memory/bitemporal_test.go with TestMarkSupersededChain; confirm failing.
2. Implement Service.MarkSuperseded in memory/lifecycle.go mirroring BumpDuplicate (~104): single transaction — UPDATE observations SET invalid_at=now, superseded_by=?, status='superseded' WHERE id=?; upsert memory_relations edge (src, dst, relation='supersedes') per the upsert idiom in relations.go (~95-122).
3. GREEN: go test ./internal/mnemonic/memory/ -run 'TestMarkSuperseded' -count=1.
4. Commit: feat(memory): MarkSuperseded + supersede chain. Refs: task-023.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
MarkSuperseded(ctx, oldID, newID) in lifecycle.go: single tx — UPDATE observations (invalid_at=now, superseded_by=newID, status=superseded) + supersedes edge upsert in memory_relations (src=old, dst=new). Review: 2 WARNINGs fixed (bare literal → relSupersedes const, created_at immutable on re-supersede → SELECT-then-INSERT no-op). Commits: 7b7d7f68 (impl) + d0f8073e (fixes).
<!-- SECTION:FINAL_SUMMARY:END -->
