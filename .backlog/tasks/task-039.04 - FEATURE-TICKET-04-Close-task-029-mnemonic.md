---
id: TASK-039.04
title: '[FEATURE] TICKET-04 Close task-029 (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-10-03 09:39'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-4
dependencies:
  - TASK-039.02
  - TASK-039.03
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/tasks.md
  - >-
    .backlog/tasks/task-029 -
    FOLLOWUP-shared-LLM-client-attach-for-ask-extraction-dedup-seams.md
parent_task_id: TASK-039
priority: medium
type: docs
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
task-029 still tracks the missing shared LLM attach. Once the Completer, attach, and install land, that follow-up is the same work and must be marked done so a second client is not built beside it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 happy path task-029 superseded by this change
- [ ] #2 task-029 status is done after TICKET-02 and TICKET-03 land
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 G9 match on the spec path
- [ ] #7 backlog task-029 is Done
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirm task-029 references 2026-10-02-mnemonic-llm-provider
2. After attach and install land, mark task-029 done via the backlog CLI
<!-- SECTION:PLAN:END -->
