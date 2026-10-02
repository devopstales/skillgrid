---
id: TASK-038.03
title: 'TICKET-03: Pure checkpoint package decide digest prompt'
status: done
assignee: []
created_date: '2026-10-02 14:46'
updated_date: '2026-10-02 15:37'
labels: []
dependencies:
  - TASK-038.01
references:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
parent_task_id: TASK-038
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
internal/mnemonic/checkpoint Decide, BuildDigest, RenderPrompt. Structured fields only. SATISFIES prompt-lists-new-events-and-existing-titles, prompt-digest-is-bounded, prompt-excludes-tool-output-text.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Decide table covers disabled unknown below-min cooldown due
- [x] #2 Digest never contains preview text and stays within 1500 chars
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/checkpoint is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestDecide, TestBuildDigest_Bounded, TestBuildDigest_StructuredFieldsOnly, TestRenderPrompt.
2. Implement Decide, BuildDigest, RenderPrompt with no I/O.
3. go test ./internal/mnemonic/checkpoint.
<!-- SECTION:PLAN:END -->
