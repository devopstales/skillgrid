---
id: TASK-013
title: '[DOCS] T07 openapi + docs + MCP round-trip guard (docs+api)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:52'
updated_date: '2026-09-19 14:52'
labels: []
dependencies:
  - TASK-012
references:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md
  - .skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md
documentation:
  - .skillgrid/specs/2026-09-19-embed-visual-companion/blueprint.md
priority: medium
type: docs
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add /mnemonic/decisions, /mnemonic/decisions/{id}/answer, /prototype/{id...} to ui/openapi.yaml. Add docs/user-guide/10-decision-companion.md (content schema + poll loop + .skillgrid/prototype/ convention). Add the round-trip guard test: GET /mnemonic/decisions lists exactly the seeded decision; mem_search(state pending) is consistent with the list; the answer round-trip is visible to a fresh MCP mem_search.

Current State:
- openapi.yaml has no decision/prototype routes; no user-guide doc for the companion; no round-trip guard test.

Expected State:
- openapi.yaml validates (or the existing openapi test passes); the docs file exists and is referenced; go test -count=1 ./internal/mnemonic/http/... -run TestDecisionRoundTrip passes (list <-> mem_search consistency + answer visible to fresh MCP mem_search).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 openapi.yaml validates (or existing openapi test passes) with the 3 new routes
- [ ] #2 docs/user-guide/10-decision-companion.md exists and documents content schema + poll loop + .skillgrid/prototype/ convention
- [ ] #3 go test -count=1 ./internal/mnemonic/http/... -run TestDecisionRoundTrip passes (list <-> mem_search consistency + answer visible to fresh MCP mem_search)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 openapi test passes
- [ ] #7 docs file present + referenced
- [ ] #8 round-trip guard test passes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 4 (docs/MCP) in blueprint.md
2. Add the 3 routes to internal/mnemonic/http/ui/openapi.yaml (run the openapi test)
3. Write docs/user-guide/10-decision-companion.md (content schema + poll loop + .skillgrid/prototype/ convention + sketch.dir note)
4. Add TestDecisionRoundTrip to decisions_test.go (list <-> mem_search consistency + answer visible to fresh MCP mem_search)
5. Verify go test + openapi test pass
<!-- SECTION:PLAN:END -->
