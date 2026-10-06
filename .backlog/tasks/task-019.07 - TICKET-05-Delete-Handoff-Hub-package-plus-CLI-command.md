---
id: TASK-019.07
title: 'TICKET-05: Delete Handoff Hub package plus CLI command'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies:
  - TASK-019.04
  - TASK-019.03
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
Delete internal/mnemonic/handoff/, cmd/skillgrid/handoff.go plus tests, strip main.go dispatch.

Current: skillgrid handoff records snapshots, checkpoints, rollups.

Expected: skillgrid handoff reports unknown subcommand; no mnemonic/handoff importers remain; build green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestHandoffCommandGone passes
- [ ] #2 rg mnemonic/handoff over Go sources is empty
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
1. Failing gone-test (RED)
2. Delete package plus command plus dispatch (GREEN)
3. Build plus suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 3 review: 10a0618 verified (13 files, dispatch stripped, pre-delete safety checks done). Joint conditions (http import, mcp refs) resolved by Wave 3/4 siblings.
<!-- SECTION:NOTES:END -->
