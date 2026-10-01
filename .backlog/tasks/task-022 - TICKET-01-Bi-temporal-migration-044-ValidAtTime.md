---
id: TASK-022
title: 'TICKET-01: Bi-temporal migration 044 + ValidAtTime'
status: needs-triage
assignee: []
created_date: '2026-10-01 18:15'
updated_date: '2026-10-01 18:16'
labels:
  - bitemporal-audn
dependencies: []
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
TICKET-01 of 2026-09-24-bitemporal-audn (blueprint Task 1, tracer thread).

Current State: observations have no validity window — superseded facts keep ranking in search and "what was true at time T?" is unanswerable.

Expected State: additive migration 044 adds valid_at/invalid_at/superseded_by (nullable) + partial index; Save stamps valid_at=created_at; all 7 read sites + 4 inline FTS/admin SELECTs apply the bi-temporal filter; ValidAtTime(ctx, query, atTime) answers the time-travel query.

Tracer thread: proves the temporal window is queryable before the AUDN classifier thickens the save path.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 PRAGMA table_info(observations) shows valid_at, invalid_at, superseded_by
- [ ] #2 Migration re-run is a no-op (runner-tracked, second Open on same dir)
- [ ] #3 Saved observation reads back ValidAt==CreatedAt, InvalidAt=="", SupersededBy invalid
- [ ] #4 Row with invalid_at set absent from SearchWithScope/SearchOwnerScoped/SearchOwner/AdminCrossOwnerList
- [ ] #5 ValidAtTime(q, T1) returns A-not-B and at T3 returns B-not-A for adjacent windows
- [ ] #6 TestSquashShimUpgrade still passes (fixture already carries observations)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenarios green: New observations are valid from creation; Superseded observations are excluded from search; Time-travel query returns the fact valid at time T
- [ ] #7 Human checkpoint: migration 044 is one-way — confirm TestSquashShimUpgrade baseline before applying
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: write store/migrations_044_test.go (TestMigration044Columns, TestMigration044Idempotent) + memory/bitemporal_test.go (TestSaveStampsBiTemporalColumns, TestValidAtTimeWindow, TestSupersededExcludedFromSearch); confirm failing.
2. Create store/migrations/044_bitemporal.sql (3 ALTER TABLE ADD COLUMN + CREATE INDEX IF NOT EXISTS idx_obs_invalid_at partial).
3. Add 3 Observation struct fields (ValidAt/InvalidAt string omitempty, SupersededBy sql.NullInt64 omitempty).
4. Add 3 columns to obsSelectCols + scanObservations + the 4 inline FTS/admin SELECTs (service.go SearchWithScope/SearchOwnerScoped, governance.go SearchOwner/AdminCrossOwnerList).
5. Add bi-temporal filter (invalid_at IS NULL OR '' OR strftime now) to the 7 read sites.
6. Stamp valid_at on the Save INSERT; implement ValidAtTime (FTS5 match + valid_at <= T < invalid_at, extract obsSelectColsAliased helper).
7. GREEN: go test ./internal/mnemonic/store/ ./internal/mnemonic/memory/ -count=1; TestSquashShim baseline first (Precondition).
8. Commit: feat(memory): migration 044 bi-temporal columns + ValidAtTime. Refs: task-022.
<!-- SECTION:PLAN:END -->
