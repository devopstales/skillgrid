---
id: TASK-012
title: '[FEATURE] T06 prototype serve route + sandboxed iframe in card (api+ui)'
status: needs-triage
assignee: []
created_date: '2026-09-19 14:51'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-5
dependencies:
  - TASK-010
  - TASK-011
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
Implement GET /prototype/{id...} in internal/mnemonic/http/prototype.go (root = .skillgrid/prototype/ via sddRoot(), reuse the stitchFile traversal-guard shape: reject absolute ids, .., and anything escaping the root after filepath.Clean; serve text/html). Register the route in server.go (distinct subtree from the plural /prototypes). Wire DecisionCard to render the existing SandboxPreview iframe (src=/prototype/{content.visual}) only when visual is set.

Current State:
- No GET /prototype route; DecisionCard has no prototype iframe.

Expected State:
- TICKET-05 (TASK-011) tests now pass.
- With a seeded .skillgrid/prototype/<topic>/a.html, GET /prototype/<topic>/a.html returns the HTML, traversal GET /prototype/../../etc/passwd is rejected, and the dashboard decision card shows the prototype in a sandboxed iframe.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TICKET-05 (TASK-011) tests now pass
- [ ] #2 go test -count=1 ./internal/mnemonic/http/... passes for the prototype route
- [ ] #3 Seeded .skillgrid/prototype/<topic>/a.html served as text/html; traversal rejected; decision card shows the prototype in a sandboxed iframe
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 TICKET-05 tests pass
- [ ] #7 Route registered + traversal guard active
- [ ] #8 iframe renders sandboxed (allow-scripts, no allow-same-origin)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read Task 5 + stitchFile/stitchRoot in internal/mnemonic/http/prototypes.go
2. Implement GET /prototype/{id...} in prototype.go (root .skillguard/prototype/ via sddRoot(), stitchFile guard shape, text/html)
3. Register the route in server.go (distinct from /prototypes)
4. Wire DecisionCard to render SandboxPreview iframe (src=/prototype/{content.visual}) only when visual is set
5. Verify TICKET-05 tests pass + live serve + iframe render
<!-- SECTION:PLAN:END -->
