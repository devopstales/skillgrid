---
id: TASK-039.02
title: '[FEATURE] TICKET-02 AttachSharedLLM and fail-open (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-10-03 09:38'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-4
dependencies:
  - TASK-039.01
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/acceptance.feature
parent_task_id: TASK-039
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A Completer that is never attached leaves ask, dedup, extraction, and dream on separate seams. One boot function must wire all four and still fall open to the deterministic floors when the client errors or feature flags stay off.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 happy path attach wires all seams from one client
- [ ] #2 disabled config clears all seams
- [ ] #3 happy path attached client with flags off uses floors
- [ ] #4 happy path llm error fails open to floors
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 G5-G8 gates green
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED TestAttachSharedLLMWiresAllSeams and TestAttachSharedLLMDisabledClears
2. Implement AttachSharedLLM adapters and call it from openProject
3. RED TestAttachedClientFlagsOffUsesFloors and TestLLMErrorFailsOpen
4. Fix only if attach forces LLM paths or swallows fail-open
<!-- SECTION:PLAN:END -->
