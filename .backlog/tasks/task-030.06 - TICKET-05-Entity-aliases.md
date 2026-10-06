---
id: TASK-030.06
title: 'TICKET-05: Entity aliases'
status: needs-triage
assignee: []
created_date: '2026-10-02 08:59'
updated_date: '2026-10-06 08:35'
labels:
  - mnemonic-memory-improvements
milestone: m-1
dependencies:
  - TASK-030.02
references:
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/blueprint.md
  - .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/tasks.md
parent_task_id: TASK-030
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current state: code search has no entity_aliases table.
Expected state: a case-insensitive alias resolves to qualified_name before FTS, and unknown aliases fall through.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The Renderer resolves to comp_renderer.cpp
- [ ] #2 Unknown alias returns no rows
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test TestEntityAlias passes
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. RED: TestEntityAliasLookup for The Renderer
2. Append entity_aliases to migration 045 and lookup before code FTS
3. GREEN: go test TestEntityAlias
<!-- SECTION:PLAN:END -->
