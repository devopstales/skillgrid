---
id: TASK-025
title: 'TICKET-04: SaveWithAction 4-way routing + Save delegates'
status: done
assignee: []
created_date: '2026-10-01 18:19'
updated_date: '2026-10-02 06:41'
labels:
  - bitemporal-audn
dependencies:
  - TASK-024
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
TICKET-04 of 2026-09-24-bitemporal-audn (blueprint Task 4, work unit 3).

Current State: Save is a single straight-line INSERT/upsert path with no action routing — every write is treated the same, so stale facts are never superseded at save time.

Expected State: SaveResult + SaveWithAction with the 4-way routing table (noop→BumpDuplicate, add→INSERT, update→topic-key upsert, delete→INSERT then MarkSuperseded + supersedes edge); INSERT extracted into insertObservation helper; Save becomes a delegate preserving its (int64, error) contract.

Blocked by: TICKET-03 (task-024).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Routing table holds for all 4 verdicts via fake seams
- [ ] #2 Deterministic floor (no seam): hash-hit → noop + BumpDuplicate (duplicate_count +1, last_seen_at refreshed), hash-miss → add
- [ ] #3 LLM error → non-fatal, degrades to floor
- [ ] #4 TestSaveDelegatesToSaveWithAction: Save returns the same ID, row shape unchanged
- [ ] #5 Full memory package green
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenarios green: Save returns an action label; Delete arm produces the supersede chain (save-path half); Deterministic floor without LLM; LLM error degrades to deterministic floor; Existing Save callers are unaffected
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: extend memory/service_test.go with TestSaveWithActionRouting (4 verdicts + floor + error) and TestSaveDelegatesToSaveWithAction; confirm failing.
2. Extract the Save INSERT into an insertObservation helper in memory/service.go (~581-611).
3. Add SaveResult + Service.SaveWithAction with the 4-way routing table: noop→BumpDuplicate; add→insertObservation; update→topic-key upsert; delete→insertObservation then MarkSuperseded (task-023) + supersedes edge.
4. Make Save a delegate to SaveWithAction preserving the (int64, error) contract.
5. GREEN: go test ./internal/mnemonic/memory/ -count=1.
6. Commit: feat(memory): SaveWithAction 4-way routing + Save delegates. Refs: task-025.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
SaveWithAction 4-way AUDN routing (noop/add/update/delete) + deterministic floor (hash-hit→noop+bump returning existing row id; hash-miss→add) + LLM-error non-fatal degradation. Save is now a thin delegate preserving the (int64, error) contract for all 7 production callers. insertObservation extracted (preserves valid_at/FTS); applyUpdateToCandidate shared by the LLM update arm and the topic-key upsert path. Review: PASS (no BLOCKER/WARNING); 2 SUGGESTIONS fixed (topic-key delegate parity test, obsCount comment). Commits: 37caa728 + 3f5dc333.
<!-- SECTION:FINAL_SUMMARY:END -->
