---
id: TASK-030
title: 'Epic: mnemonic memory improvements'
status: needs-triage
assignee: []
created_date: '2026-10-02 08:58'
labels:
  - mnemonic-memory-improvements
dependencies: []
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/briefing.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
documentation:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Owner-scoped mem_search gains RRF fusion and additive search signals, reinforcement decay, a query embedding cache, entity aliases, and a fail-open compact hook. Per ADR-0018. C6 is out of scope.

Current state: mem_search is BM25 via SearchOwnerScoped. BlendedSearch exists but skips the visibility filter.
Expected state: SearchOwnerScopedBlend returns signals and matched_via; decay, cache, aliases, and compact hook land as sliced tickets.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All six tickets in tasks.md are done and their go tests pass
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level scenarios in acceptance.feature are green
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Track TASK-030.01 through TASK-030.06 from tasks.md
2. Execute serially on release/2 per ADR-0018
3. Verify each ticket's go test before the next
<!-- SECTION:PLAN:END -->
