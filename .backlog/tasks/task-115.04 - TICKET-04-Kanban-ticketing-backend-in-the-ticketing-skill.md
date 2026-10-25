---
id: TASK-115.04
title: 'TICKET-04: Kanban ticketing backend in the ticketing skill'
status: done
assignee: []
created_date: '2026-10-25 08:02'
updated_date: '2026-10-25 11:45'
labels: []
dependencies:
  - TASK-115.03
references:
  - .skillgrid/specs/2026-10-24-hermes-integration/blueprint.md
  - .skillgrid/specs/2026-10-24-hermes-integration/tasks.md
  - .skillgrid/specs/2026-10-24-hermes-integration/acceptance.feature
  - >-
    Kanban board: skillgrid (board is source of truth; this entry is a mirror —
    see .agents/skills/_shared/rules/ticketing/hermes-kanban.md)
parent_task_id: TASK-115
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add the hermes-kanban backend to the ticketing skill. Read ticketing.type from .skillgrid/config.yaml; when hermes-kanban, publish tickets via hermes kanban create with --assignee, parents, --board, --skill. Map skillgrid ticket fields to Kanban task fields. Update Backlog.md entry with Kanban task ID (mirror). When type: backlogmd, unchanged behavior.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 When type: hermes-kanban, ticketing skill calls hermes kanban create
- [ ] #2 Tickets with dependencies get parents
- [ ] #3 Coding tickets use worktree workspace; research tickets use scratch
- [ ] #4 Backlog.md entry updated with Kanban task ID (mirror)
- [ ] #5 When type: backlogmd, writes to .backlog/ (unchanged)
- [ ] #6 SATISFIES: happy path — ticket published to Kanban; happy path — ticket with dependencies; edge — board created if missing; failure — board creation non-fatal
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 hermes kanban list shows published ticket
<!-- DOD:END -->
