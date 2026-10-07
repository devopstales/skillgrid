---
id: TASK-053
title: '[FEATURE] Scan Diff + Status + List + Get (mnemonic)'
status: done
assignee: []
created_date: '2026-10-07 11:31'
updated_date: '2026-10-07 13:53'
labels: []
milestone: m-7
dependencies:
  - TASK-051
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: no diff/status/list/get on the scan service.\n\nExpected State: (*Service).Diff(ctx, oldID, newID) -> DiffResult{Added,Removed,Unchanged} via set difference of dedup_hash per scan_id; LatestDiff (two most recent scans by started_at); Status (counts by tool + severity + latest scan time per tool); List (filter tool/status + limit); Get (scan row). Removed findings never deleted (audit trail).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 unchanged re-scan: Diff reports added=0 removed=0 (identical hash sets)
- [ ] #2 fixed-CVE fixture: the CVE's dedup_hash appears in Removed and the original findings row count is unchanged (audit survival — removed findings are NOT deleted)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 LatestDiff resolves the two most recent scans by started_at
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 4 (tasks.md TICKET-04); work from mnemonic/ module
2. TDD: TestUnchangedRescanDiffsToZeroAdded + TestFixedCveAppearsAsRemoved in service_test.go RED first
3. Implement Diff (set difference of dedup_hash per scan_id), LatestDiff (two most recent by started_at), Status (counts by tool+severity + latest per tool), List, Get; removed findings NOT deleted (audit)
4. go test ./internal/mnemonic/scan/ -v + go build ./... -> green
5. Commit: feat(mnemonic): scan diff (hash-set), status, list, get + Refs: TASK-053
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
scan package read surface complete: Diff (set difference of dedup_hash per scan), LatestDiff, Status, List, Get. Removed findings NOT deleted (audit). 17/17 tests GREEN (TestUnchangedRescanDiffsToZeroAdded, TestFixedCveAppearsAsRemoved + Status/List/Get). Commit 0bba45e0 scoped to mnemonic/internal/scan only.
<!-- SECTION:FINAL_SUMMARY:END -->
