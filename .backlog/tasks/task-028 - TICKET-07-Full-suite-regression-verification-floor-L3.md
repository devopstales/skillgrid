---
id: TASK-028
title: 'TICKET-07: Full-suite regression + verification floor (L3)'
status: needs-triage
assignee: []
created_date: '2026-10-01 18:24'
updated_date: '2026-10-01 18:24'
labels:
  - bitemporal-audn
dependencies:
  - TASK-027
references:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/briefing.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/tasks.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/acceptance.feature
  - .skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md
documentation:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TICKET-07 of 2026-09-24-bitemporal-audn (blueprint Task 7, work unit 4) — the L3 verification floor.

Current State: each ticket is verified in its own package, but no whole-change regression gate exists; the bi-temporal + AUDN change (TICKET-01..06) has not been validated end-to-end.

Expected State: verification only — build, full go test ./..., go vet, package coverage on ./internal/mnemonic/.... No production code changes (test files only if a regression surfaces). The success-criteria backstop.

Blocked by: TICKET-06 (task-027).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go build ./... clean
- [ ] #2 go test ./... -count=1 all ok (known pre-existing http/tracker flake passes in isolation — not a regression)
- [ ] #3 go vet ./... clean except the known pre-existing budget.go:114 context-cancel leak (out of scope, do not fix here)
- [ ] #4 memory package coverage not below its prior level
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenario green: All existing tests pass (no regression) — the success-criteria backstop
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. cd skillgrid-cli && go build ./...
2. go test ./... -count=1 (known http/tracker flake passes in isolation — re-run in isolation if it fails there)
3. go vet ./... (known pre-existing budget.go:114 context-cancel leak is out of scope; flag any NEW finding)
4. go test -cover ./internal/mnemonic/memory/ — confirm coverage not below prior level; record the number.
5. If a regression surfaces: fix in test files only, re-run, note in ## Comments.
6. Commit: test(memory): L3 regression floor for bitemporal-audn. Refs: task-028.
<!-- SECTION:PLAN:END -->
