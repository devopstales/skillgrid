---
id: TASK-019.06
title: 'TICKET-06: Delete hub and relay MCP tools'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies:
  - TASK-019.05
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Delete tools_handoff.go (5 tools), tools_session_handoff.go, tools_session_status.go; deregister in server.go. ONE-WAY: breaks UI, skills, external clients; needs human checkpoint before commit.

Current: 9 hub/relay tools registered.

Expected: registry holds session_changes but none of the 9 removed names; mcp suite green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Registry test: 9 names absent, session_changes present
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
1. Failing registry test (RED)
2. STOP: get human approval (one-way removal)
3. Delete files plus deregister (GREEN)
4. Suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 3 review: human checkpoint APPROVED; committed as 4d14a13 (+64/-743). Registry gate green. session_test.go alias lines + http/handoff.go import flagged for TICKET-07.
<!-- SECTION:NOTES:END -->
