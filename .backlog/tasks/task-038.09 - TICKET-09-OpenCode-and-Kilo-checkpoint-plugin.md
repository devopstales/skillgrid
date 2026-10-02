---
id: TASK-038.09
title: 'TICKET-09: OpenCode and Kilo checkpoint plugin'
status: needs-triage
assignee: []
created_date: '2026-10-02 14:47'
labels: []
dependencies:
  - TASK-038.04
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
skillgrid-checkpoint.ts claims on session.idle; setup copies it. SATISFIES setup-installs-checkpoint-plugin, plugin-does-not-prompt-when-not-due, plugin-fails-open-without-server.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 After setup the plugin file exists; dry-run writes nothing
- [ ] #2 Not-due and no-server send no prompt
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/setup -run TestSetupOpenCode_CheckpointPlugin|TestSetupKilo_CheckpointPlugin is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestSetupOpenCode_CheckpointPlugin and TestSetupKilo_CheckpointPlugin.
2. Add skillgrid-checkpoint.ts; copy from setup opencode|kilo.
3. Manual G7 in OpenCode; document in 04-hooks.md.
<!-- SECTION:PLAN:END -->
