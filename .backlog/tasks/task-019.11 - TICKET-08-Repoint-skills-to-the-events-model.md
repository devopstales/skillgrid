---
id: TASK-019.11
title: 'TICKET-08: Repoint skills to the events model'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-09-22 08:32'
labels: []
dependencies:
  - TASK-019.09
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: medium
type: docs
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Rewrite 14 skill files plus sdd-structure.md to the events model; regenerate site partial.

Current: skills document checkpoint.json, handoff verify/archive, cleave bundles.

Expected: zero checkpoint.json, snapshot/restore, handoff, cleave refs; resume documents session_changes plus to_commit; site regenerated.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Stale-ref grep across the 14 files is empty; site partial regenerated
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
1. Failing stale-ref check (RED)
2. Rewrite skills plus regenerate site (GREEN)
3. Check passes, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
14 skill files repointed + checkpoint.md template deleted + site partial regenerated (08c5624); forbidden rg across .agents/skills/ empty
<!-- SECTION:NOTES:END -->
