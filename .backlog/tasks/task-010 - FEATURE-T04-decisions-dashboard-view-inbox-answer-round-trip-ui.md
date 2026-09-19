---
id: TASK-010
title: '[FEATURE] T04 /decisions dashboard view: inbox + answer round-trip (ui)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:50'
updated_date: '2026-09-19 14:50'
labels: []
dependencies:
  - TASK-008
  - TASK-009
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
documentation:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the /decisions route + DecisionsPage (left inbox list of pending decisions, right detail), DecisionCard (title/body/created + answer textarea + submit -> POST .../answer, on success show state=answered + stored answer, idempotent re-answer is a no-op), api.ts (fetch GET /mnemonic/decisions, POST answer). Add the /decisions route to app.tsx and a nav entry to AppLayout.tsx.

Current State:
- No /decisions view.

Expected State:
- TICKET-03 (TASK-009) tests now pass.
- In the live dashboard: the pending decision appears in the inbox, the user types an answer and submits, the card flips to state=answered and shows the stored answer, a second submit is a no-op, and the decision no longer appears in the pending list on reload.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TICKET-03 (TASK-009) tests now pass
- [ ] #2 pnpm --filter skillgrid-ui test passes for the decision view
- [ ] #3 Live dashboard: inbox shows pending decision -> answer -> card flips to answered -> re-answer no-op -> gone from pending list on reload
- [ ] #4 /decisions route + nav entry added to app.tsx + AppLayout.tsx
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 TICKET-03 tests pass
- [ ] #7 Live dashboard round-trip demonstrated
- [ ] #8 Route + nav entry present
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 3 (frontend) in blueprint.md for the /decisions view contract
2. Create skillgrid-ui/src/features/decisions/api.ts (fetch GET /mnemonic/decisions, POST answer)
3. Build DecisionsPage.tsx (inbox list + detail) and DecisionCard.tsx (textarea + submit + answered state)
4. Add the /decisions route to app.tsx and a nav entry to AppLayout.tsx
5. Verify TICKET-03 tests pass + live dashboard round-trip
<!-- SECTION:PLAN:END -->
