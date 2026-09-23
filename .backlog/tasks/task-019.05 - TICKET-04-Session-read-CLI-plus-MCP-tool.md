---
id: TASK-019.05
title: 'TICKET-04: Session read CLI plus MCP tool'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-09-21 16:02'
labels: []
dependencies:
  - TASK-019.01
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
skillgrid session <id> [--show-diff] plus session_changes MCP tool over SessionChanges.

Current: no session-scoped read; resume reads checkpoint.json plus Hub.

Expected: CLI prints ordered events; --show-diff adds git diff stat from..to; unknown id errors; MCP tool returns events JSON.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestSessionShow passes (events, diff, unknown-id error)
- [ ] #2 TestSessionChangesTool passes
<!-- AC:END -->

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
1. Failing CLI plus MCP tests (RED)
2. session show subcommand plus session_changes tool (GREEN)
3. Suites green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 2 review: 954365a verified (TestSessionShow + 4/4 MCP tests PASS, cmd+mcp suites green). Tool-count pins now at 89 — TICKET-06 must rebump on removal. Noted pre-existing ollama-env suite timeout (environmental).
<!-- SECTION:NOTES:END -->
