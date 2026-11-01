---
id: TASK-096
title: >-
  [FEATURE] TICKET-13 Shared handler + MCP stdio server (agent profile, write
  queue, envelope) (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-01 12:48'
labels: []
milestone: m-11
dependencies:
  - TASK-087
  - TASK-090
  - TASK-091
  - TASK-093
  - TASK-094
  - TASK-095
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The public surface (one-way door D1): one shared handler set (CLI and MCP call it - they can never diverge), all 26 mem_* tools via mark3labs/mcp-go, the agent profile exposing exactly the 20 agent rows and the all profile adding the 6 admin rows (a contract test pins both sets literally - K5; the opencode/pi plugins consume this contract). The write queue serializes concurrent writes; the response envelope carries degraded.
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
