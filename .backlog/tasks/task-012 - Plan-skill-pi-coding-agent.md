---
id: TASK-006
title: Plan 012-skill-pi-coding-agent.md
status: needs-triage
assignee: []
created_date: '2026-09-08 17:53'
updated_date: '2026-09-11 15:30'
labels: []
dependencies: []
references:
  - 'https://github.com/earendil-works/pi'
  - docs/skillgrid/changes/012-skill-pi-coding-agent/
modified_files:
  - docs/skillgrid/changes/012-skill-pi-coding-agent/change.md
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
SDD plan for skill-pi: a new AI coding agent based on earendil-works/pi. Research the pi architecture, define skill-pi's scope, and produce change.md + tasks.md + acceptance.feature.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 change.md written with Goal, Step Blueprint, Architecture decisions, Threat matrix
- [ ] #2 tasks.md written with NN allocation and per-step tasks
- [ ] #3 acceptance.feature written with @step-NN features
- [ ] #4 Research.md documents pi architecture findings
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 User gate passes (Implement approved)
<!-- DOD:END -->
