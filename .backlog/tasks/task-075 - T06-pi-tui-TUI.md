---
id: TASK-075
title: T06 pi-tui TUI
status: needs-triage
assignee: []
created_date: '2026-10-09 13:10'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-073
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/tui/
  - skill-pi/themes/
  - skill-pi/tui/*.test.ts
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Build the TUI on @earendil-works/pi-tui primitives: alt-screen root, header + context gauge, transcript (ScrollView), right rail (agents widget + changes viewer), bottom bar (compact layout). Breakpoint + gauge tone are pure functions. Tracer A, Wave 6.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 At width >=140 the full layout (header + transcript + right rail) is selected
- [ ] #2 At width <140 the compact layout (bottom bar + agents widget) is selected
- [ ] #3 The gauge tone escalates at 0.80 (warning) and 0.95 (critical) usage ratios
- [ ] #4 Theme tokens in skill-pi/themes/
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
