---
id: TASK-056
title: '[FEATURE] MCP registration: registerScanTools + registerDepTools (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-10-07 11:32'
updated_date: '2026-10-07 11:35'
labels: []
milestone: m-7
dependencies:
  - TASK-051
  - TASK-053
  - TASK-054
  - TASK-055
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
Current State: mcp/server.go registers mem_*/code_*/web_* surfaces only.\n\nExpected State: tools_scan.go (scan_start, scan_store_findings, scan_list, scan_get, scan_status, scan_diff with old/new/latest params) + tools_dep.go (dep_ingest, dep_list, dep_get, dep_affected, dep_graph, dep_runtime with purl/scan_id/sbom/source_file params); registerScanTools(s) + registerDepTools(s) wired in server.go alongside existing registrations (server.go:42-72 pattern); services constructed at boot.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/mnemonic/mcp/ passes: registration tests find all 12 tool names (scan_start, scan_store_findings, scan_list, scan_get, scan_status, scan_diff, dep_ingest, dep_list, dep_get, dep_affected, dep_graph, dep_runtime)
- [ ] #2 all pre-existing register*/Registered tests still pass (additive, not replacing)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 scan.Service constructed with dataDir from webcache.DefaultDataDir(); dep.Service at server boot
- [ ] #7 Thin adapters only: decode params -> service -> encode result
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 7 (tasks.md TICKET-07); work from mnemonic/ module
2. TDD: tools_scan_test.go (TestScanToolsRegistered, 6 names) + tools_dep_test.go (TestDepToolsRegistered, 6 names) RED first
3. Implement tools_scan.go (scan_start, scan_store_findings, scan_list, scan_get, scan_status, scan_diff with old/new/latest) + tools_dep.go (dep_ingest, dep_list, dep_get, dep_affected, dep_graph, dep_runtime) as thin adapters; wire registerScanTools(s) + registerDepTools(s) in server.go (server.go:42-72 pattern); construct scan.Service (dataDir from webcache.DefaultDataDir()) + dep.Service at boot
4. go test ./internal/mnemonic/mcp/ -v + go build ./... -> green; pre-existing Registered tests still pass
5. Commit: feat(mnemonic): register scan + dep MCP tools + Refs: TASK-056
<!-- SECTION:PLAN:END -->
