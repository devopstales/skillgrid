---
id: TASK-009
title: '[FEATURE] T03 RED: frontend /decisions view acceptance tests (ui)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:50'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-5
dependencies: []
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
documentation:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write the failing React acceptance tests (no components yet) for the /decisions view.

Current State:
- No /decisions route, no DecisionsPage/DecisionCard components.

Expected State:
- DecisionsPage.test.tsx + DecisionCard.test.tsx contain RED tests for: list renders the pending decision inbox (title+body+created), clicking a card opens detail with a textarea+submit, submitting POSTs to POST /mnemonic/decisions/{id}/answer and on success shows state=answered + stored answer, a decision without visual renders no iframe, a decision with visual renders the sandboxed iframe at /prototype/{visual}.
- The UI test runner exits non-zero (RED) but the UI package compiles.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 UI test runner (pnpm --filter skillgrid-ui test) exits non-zero on the decision view tests (RED) but the package compiles
- [ ] #2 Tests assert: inbox render, detail open, submit POSTs + shows state=answered, no iframe without visual, sandboxed iframe with visual
- [ ] #3 SATISFIES scenario names recorded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 UI package compiles
- [ ] #7 Tests are RED with the expected assertions
- [ ] #8 SATISFIES scenario names recorded
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 3 (frontend) in blueprint.md for the /decisions view contract
2. Identify the project UI test runner (vitest) and existing test patterns in skillgrid-ui
3. Write DecisionsPage.test.tsx + DecisionCard.test.tsx with RED tests (mock the api fetch)
4. Run the UI test runner; confirm RED (not a build error)
<!-- SECTION:PLAN:END -->
