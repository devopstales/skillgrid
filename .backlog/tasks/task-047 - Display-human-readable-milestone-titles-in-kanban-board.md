---
id: TASK-047
title: Display human-readable milestone titles in kanban board
status: ready-for-agent
assignee: []
created_date: '2026-10-06 13:43'
labels: []
milestone: m-6
dependencies: []
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The kanban board renders the raw milestone ID (e.g. "m-1") from task frontmatter instead of the human-readable title from `.backlog/milestones/`. The milestone files exist (m-0 through m-5) with `title` fields, but the Go adapter never reads them and there is no HTTP endpoint to serve them.

**Scope:**
1. Add `Milestones() ([]Milestone, error)` method to `TicketProvider` interface (tracker.go) with `Milestone` struct (ID, Title, Description)
2. Implement in `backlogAdapter` by reading `.backlog/milestones/*.md` frontmatter
3. Add `GET /tracker/milestones` route in server_tracker.go
4. Update TS `api.ts` to fetch milestones and build an id→title map
5. Update `BoardView.tsx` to display milestone titles instead of raw IDs
6. Update `epicTree.ts` `MilestoneGroup` to carry the display title
7. Update `TaskCard.tsx` milestone badge to show title
8. Update `ListView.tsx` and `TaskDetail.tsx` to show title
9. Fall back to raw ID when no milestone file matches
10. Unit tests for the Go Milestones() method and TS rendering

**Acceptance criteria:**
- Board milestone rows show human-readable titles (e.g. "mnemonic-memory-improvements" not "m-1")
- Task cards show milestone title in badge
- List view shows milestone title
- Task detail shows milestone title
- Tasks with unknown milestone IDs show the raw ID (fallback)
- GET /tracker/milestones returns milestone list
- Go tests pass
- UI tests pass
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Board milestone rows display human-readable titles from milestone files
- [ ] #2 Task card milestone badge shows title not raw ID
- [ ] #3 List view and task detail show milestone title
- [ ] #4 Unknown milestone IDs fall back to raw ID display
- [ ] #5 GET /tracker/milestones endpoint returns milestone list with id and title
- [ ] #6 Go unit tests for Milestones() method pass
- [ ] #7 UI tests for milestone title rendering pass
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
