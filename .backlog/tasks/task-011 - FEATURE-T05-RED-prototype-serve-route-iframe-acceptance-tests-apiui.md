---
id: TASK-011
title: '[FEATURE] T05 RED: prototype serve route + iframe acceptance tests (api+ui)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:51'
updated_date: '2026-09-19 14:51'
labels: []
dependencies:
  - TASK-010
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
Write the failing acceptance tests (no implementation yet) for the prototype function.

Current State:
- No GET /prototype/{id...} route; DecisionCard does not render a prototype iframe.

Expected State:
- prototype_test.go contains RED tests for: GET /prototype/{id...} serves .skillgrid/prototype/<topic>/<variant>.html with Content-Type text/html, 404 for a missing file, 400/404 for a path-traversal escape (e.g. /prototype/../../.stitch/foo.html).
- DecisionCard.test.tsx (add iframe cases) contains RED tests: renders the sandboxed iframe only when content.visual is set (attr sandbox=allow-scripts and NOT allow-same-origin).
- go test -count=1 ./internal/mnemonic/http/... -run TestPrototype and the UI iframe test fail (RED) but packages compile.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test -count=1 ./internal/mnemonic/http/... -run TestPrototype exits non-zero (RED) but compiles
- [ ] #2 UI iframe test exits non-zero (RED) but compiles
- [ ] #3 Tests assert: serve .skillgrid/prototype/<topic>/<variant>.html text/html, 404 missing, traversal rejected, iframe only when visual set with sandbox=allow-scripts and not allow-same-origin
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Go + UI test files compile
- [ ] #7 Tests are RED with the expected assertions
- [ ] #8 SATISFIES scenario names recorded
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 5 (prototype) + the stitchFile guard in internal/mnemonic/http/prototypes.go
2. Write prototype_test.go with RED table-driven tests (serve, 404, traversal)
3. Add iframe RED cases to DecisionCard.test.tsx (only when visual, sandbox attr)
4. Run go test + UI test runner; confirm RED (not a build error)
<!-- SECTION:PLAN:END -->
