---
id: TASK-050
title: '[FEATURE] Migration 050: additive scan + dep tables (mnemonic)'
status: done
assignee: []
created_date: '2026-10-07 11:29'
updated_date: '2026-10-07 12:37'
labels: []
milestone: m-7
dependencies: []
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: embedded migrations end at 049_session_checkpoint.sql; scanner output has no durable home.\n\nExpected State: migration 050_scan_findings.sql adds (additive, CREATE IF NOT EXISTS only) scans, findings (+ findings_fts FTS5 with AI/AD/AU triggers), dependencies (PK purl), dep_edges (UNIQUE(from_purl,to_purl)). No changes to existing tables; code index edges table untouched. Test asserts all five names in sqlite_master after store.Open and idempotent re-open.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/mnemonic/store/ -run TestScanTablesMigrateIdempotent passes: scans/findings/findings_fts/dependencies/dep_edges all present after migration
- [ ] #2 re-opening the same store file re-runs the runner without error (idempotent)
- [ ] #3 migration is additive only (CREATE ... IF NOT EXISTS); no ALTER on existing tables
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 One-way door: user sign-off on additive table creation recorded (blueprint one-way-door #1)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 1 (tasks.md TICKET-01); work from mnemonic/ module (cd mnemonic)
2. TDD: write scan_tables_test.go (TestScanTablesMigrateIdempotent) first, confirm RED (no such table: scans)
3. Write 050_scan_findings.sql per blueprint Step 3 (scans, findings + findings_fts + AI/AD/AU triggers, dependencies, dep_edges; CREATE IF NOT EXISTS only)
4. go test ./internal/mnemonic/store/ -run TestScanTablesMigrateIdempotent -v -> green; go build ./...
5. Commit: feat(mnemonic): additive scan + dependency graph tables (050) with [skillgrid-context] block + Refs: TASK-050
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-07 11:47
---
Execution began (subagent-execution, Wave 1). One-way door checkpoint passed: user "execute" = sign-off on additive migration 050.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Migration 050 (mnemonic/internal/store/migrations/050_scan_findings.sql) adds additive scan + dependency graph tables.

## What
- `050_scan_findings.sql`: `scans`, `findings` (+ `findings_fts` FTS5 with AI/AD/AU triggers), `dependencies` (PK purl), `dep_edges` (UNIQUE(from_purl,to_purl)). All `CREATE ... IF NOT EXISTS`; no ALTER on existing tables; code index `edges` table untouched.
- `scan_tables_test.go`: `TestScanTablesMigrateIdempotent` asserts all five table/view names present in sqlite_master after `store.Open` and that re-opening the same file re-runs the migration runner without error.

## Why
TASK-050 of the mnemonic-scan-findings change — durable home for scanner output + dependency graph (ADR-0030). SATISFIES `happy path scan migration applies idempotently`.

## Where
- mnemonic/internal/store/migrations/050_scan_findings.sql (create)
- mnemonic/internal/store/scan_tables_test.go (create)

## Verified
- `go test ./internal/store/ -run TestScanTablesMigrateIdempotent -v -count=1` → PASS
- FTS + triggers mirror the 001 symbol_fts pattern; dependencies PK purl (upsert identity); dep_edges UNIQUE(from_purl,to_purl); findings UNIQUE(scan_id, dedup_hash) is the cross-scan contract.
<!-- SECTION:FINAL_SUMMARY:END -->
