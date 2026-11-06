---
id: TASK-096
title: >-
  [FEATURE] TICKET-13 Shared handler + MCP stdio server (agent profile, write
  queue, envelope) (mnemonic-v2)
status: done
assignee: []
created_date: '2026-11-01 12:47'
updated_date: '2026-11-02 09:30'
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
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//blueprint.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//tasks.md
  - .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory//acceptance.feature
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

## Comments

<!-- COMMENTS:BEGIN -->
author: orchestrator
created: 2026-11-02 07:55
---
Wave 7 start (2026-11-02): implementer dispatched (shared handler + MCP stdio server).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Shared handler + MCP stdio server: ONE handler (transport/handler) both MCP + CLI call (CLI/MCP parity seam — handler + envelope import neither mcp-go nor the CLI). MCP stdio server (agent profile 22 tools / all 28), write queue (single worker, channel, reads bypass, panic-recovery, JudgeMany = 1 queue slot = 1 txn), envelope (ok/error + degraded flag, snake_case golden-pinned, all brief error codes). marked_by_model flows agent model → Judge/JudgeMany. mem_current_project routes through detector (nil → error). mem_capture_passive honest degraded no-op. JudgeMany fail-safe on seam-less store. 27+ transport tests. 3 fix rounds. New dep: mark3labs/mcp-go v1.1.1 (ADR prerequisite, was in go.sum indirect). Commits 549fc498, 4c5a8137, 8a20f585, bbec0632. Refs: TASK-096
<!-- SECTION:FINAL_SUMMARY:END -->
