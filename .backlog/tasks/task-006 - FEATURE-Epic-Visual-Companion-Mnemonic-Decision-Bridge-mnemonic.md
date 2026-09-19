---
id: TASK-006
title: '[FEATURE] Epic: Visual Companion (Mnemonic Decision Bridge) (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:48'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Close the interview loop: the agent posts interview questions as governed Mnemonic decision observations; the user answers in the existing skillgrid serve dashboard; the agent reads the decision back via mem_search. No new table, migration, transport, or process. A second visual function - throwaway interactive HTML prototypes at .skillgrid/prototype/<topic>/<variant>.html - is served by a new GET /prototype route and rendered in the existing sandboxed iframe inside the decision card.

Sliced into 7 tickets in 6 waves (acceptance-first). See tasks.md for the dependency graph and delivery strategy (High review workload -> 3 chained work units).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All 7 tickets done
- [ ] #2 go test ./internal/mnemonic/http/... and pnpm --filter skillgrid-ui test pass
- [ ] #3 Live dashboard round-trip: seed decision -> answer -> agent reads back
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 All 7 tickets done and merged
- [ ] #7 Chained PRs (3 work units) merged per delivery strategy
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Execute Wave 1 (T01 RED backend, T03 RED frontend) in parallel
2. Wave 2 T02 backend read+answer
3. Wave 3 T04 /decisions view
4. Wave 4 T05 RED prototype
5. Wave 5 T06 prototype route + iframe
6. Wave 6 T07 openapi+docs+round-trip
7. qa -> review -> ship
<!-- SECTION:PLAN:END -->
