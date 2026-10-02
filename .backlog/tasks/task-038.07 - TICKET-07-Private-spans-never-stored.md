---
id: TASK-038.07
title: 'TICKET-07: Private spans never stored'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:47'
labels: []
dependencies: []
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
memory.StripPrivate on every write seam and in the capture hook. SATISFIES tool-call-with-private-span, unterminated-private-tag, private-span-in-summary, prompt-omits-private-spans.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Stored preview summary and content never contain span inner text
- [ ] #2 Hook request body is stripped before POST
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test and node scripts/test-hooks.mjs private pass
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestStripPrivate, TestToolCalls_PrivateSpan, TestSessionSummary_PrivateSpan; test-hooks private group.
2. Implement memory.StripPrivate and call it on every write seam plus the hook.
3. go test and node scripts/test-hooks.mjs private.
<!-- SECTION:PLAN:END -->
