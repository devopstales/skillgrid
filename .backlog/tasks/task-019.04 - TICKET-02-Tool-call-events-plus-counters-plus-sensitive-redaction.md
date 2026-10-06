---
id: TASK-019.04
title: 'TICKET-02: Tool-call events plus counters plus sensitive redaction'
status: done
assignee: []
created_date: '2026-09-21 15:10'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies:
  - TASK-019.01
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
post_tool_use hook branch: ordered event rows, counter bumps in-transaction, sensitive matcher with hash/preview-only storage.

Current: RunHook has no post_tool_use branch; no per-tool-call events.

Expected: Write maps to file_write, Shell to command_exec; sequence max-plus-1 with counter bump in one transaction; .env and key paths flag is_sensitive with masked preview.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Two hook calls yield sequences 1,2 with counters bumped
- [ ] #2 .env write flags sensitive with no full secret in payload
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Failing ordering plus redaction tests (RED)
2. HookPayload extension plus post_tool_use branch plus sensitive.go (GREEN)
3. Memory suite green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 2 review: efcc655 verified on integrated tree (focused tests PASS, build clean). Enrich-after-append keeps single insert path; same-tx UPDATE is SQLite-safe. Pre-existing budget.go:114 vet note acknowledged.
<!-- SECTION:NOTES:END -->
