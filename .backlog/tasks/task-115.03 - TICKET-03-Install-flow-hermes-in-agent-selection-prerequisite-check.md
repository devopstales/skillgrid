---
id: TASK-115.03
title: 'TICKET-03: Install flow — hermes in agent selection + prerequisite check'
status: done
assignee: []
created_date: '2026-10-25 08:02'
updated_date: '2026-10-25 11:24'
labels: []
dependencies:
  - TASK-115.02
references:
  - .skillgrid/specs/2026-10-24-hermes-integration/blueprint.md
  - .skillgrid/specs/2026-10-24-hermes-integration/tasks.md
  - .skillgrid/specs/2026-10-24-hermes-integration/acceptance.feature
parent_task_id: TASK-115
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add hermes to the skillgrid installer's agent selection, prerequisite check (hermes binary on PATH, non-fatal, prints uv tool install hermes-agent), and install flow. Wire setup.RunSetup(hermes, ...) into the install sequence.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 skillgrid install --agents hermes selects hermes
- [ ] #2 Prerequisite check: hermes on PATH (non-fatal, prints install command)
- [ ] #3 Calls SetupHermes
- [ ] #4 skillgrid install --dry-run --agents hermes prints planned changes without writing
- [ ] #5 skillgrid install --agents opencode still works (no regression)
- [ ] #6 SATISFIES: happy path — full install configures all layers; edge — dry-run; failure — install continues when hermes missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 cd skillgrid-cli && go test ./... passes
<!-- DOD:END -->
