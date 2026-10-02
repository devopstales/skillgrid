---
id: TASK-027
title: 'TICKET-06: mem_save MCP → SaveWithAction + additive response'
status: done
assignee: []
created_date: '2026-10-01 18:23'
updated_date: '2026-10-02 07:30'
labels:
  - bitemporal-audn
dependencies:
  - TASK-026
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
TICKET-06 of 2026-09-24-bitemporal-audn (blueprint Task 6, work unit 3).

Current State: the mem_save MCP tool calls Save directly and returns only id + project — agents cannot see what action the save path took, or which fact was superseded.

Expected State: handleMemSave calls SaveWithAction; the response map gains action (when non-empty) + superseded_id (when > 0) alongside the existing id + project fields. Purely additive.

Blocked by: TICKET-05 (task-026).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestMemSaveMCPAdditive: new observation → response has id, project, action == "add"
- [ ] #2 Hash-dup → action == "noop", no superseded_id
- [ ] #3 Existing id/project fields intact (old consumers unaffected)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenario green: mem_save MCP response is additive
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: extend mcp/tools_memory_test.go with TestMemSaveMCPAdditive (add + noop cases); confirm failing.
2. Switch handleMemSave in mcp/tools_memory.go (~183-265) to call SaveWithAction; add action (when non-empty) + superseded_id (when > 0) to the response map alongside id + project.
3. GREEN: go test ./internal/mnemonic/mcp/ -count=1.
4. Commit: feat(memory): mem_save MCP SaveWithAction + additive response. Refs: task-027.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
handleMemSave now calls SaveWithAction; response is purely additive — id (=res.ObservationID, existing row id on noop floor) + project intact, action added when non-empty (string of DedupVerdict), superseded_id added only when > 0 (delete arm). HTTP handler (http/server.go:1239) left unchanged (out of scope, MCP-specific). Review: PASS; 1 SUGGESTION fixed (test now asserts superseded_id key ABSENT via raw-map unmarshal, not just struct-zero). Commits: 966ae70a + 9089fddf.
<!-- SECTION:FINAL_SUMMARY:END -->
