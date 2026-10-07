---
id: TASK-055
title: '[FEATURE] Dep.Runtime declared-vs-imported via code index edges (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-10-07 11:32'
updated_date: '2026-10-07 11:34'
labels: []
milestone: m-7
dependencies:
  - TASK-054
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
Current State: no declared-vs-imported answer.\n\nExpected State: (*Service).Runtime(ctx, sourceFile) -> RuntimeResult{DeclaredOnly, ImportOnly, Shared}: resolve file_id from files, SELECT to_name,target_path FROM edges WHERE kind IN ('imports','dynamic_import') AND file_id=? for the imported set, declared set from dependencies (retired=0) provenance, three-way set diff in the tool layer.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 with seeded files + edges rows: flask lands in Shared (declared AND imported), werkzeug in DeclaredOnly (manifest dep never imported), os in ImportOnly (imported, not in any manifest)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Code index is read-only: no reindex, no new extractor, no schema change (blueprint WS7 scope)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 6 (tasks.md TICKET-06); work from mnemonic/ module
2. TDD: TestRuntimeFlagsDeclaredAndImportOnly (seed files row, edges kind='imports' rows, manifest deps) RED first
3. Implement Runtime: resolve file_id from files, SELECT to_name,target_path FROM edges WHERE kind IN ('imports','dynamic_import') AND file_id=? for imported set, declared set from dependencies (retired=0), three-way diff -> RuntimeResult{DeclaredOnly, ImportOnly, Shared}; code index read-only, no reindex/extractor
4. go test ./internal/mnemonic/dep/ -v + go build ./... -> green
5. Commit: feat(mnemonic): dep runtime — declared-vs-imported via code index edges + Refs: TASK-055
<!-- SECTION:PLAN:END -->
