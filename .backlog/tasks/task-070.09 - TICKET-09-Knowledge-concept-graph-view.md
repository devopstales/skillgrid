---
id: TASK-070.09
title: 'TICKET-09: Knowledge-concept graph view'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:34'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies: []
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Per-call derive (nodes/edges/backlinks, Concept*-prefixed) + open GET /knowledge/graph route + SPA page reusing code-graph D3 chrome. Fixture-first; tail verification on real bundle recorded fixture-only if R6 unlanded.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Go knowledge tests + vitest render PASS; go vet clean
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
