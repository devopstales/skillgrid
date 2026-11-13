---
id: TASK-123
title: '[FEATURE] Frontmatter vault adapter round-trip (INFRA)'
status: done
assignee: []
created_date: '2026-11-12 16:01'
updated_date: '2026-11-13 10:05'
labels: []
milestone: m-12
dependencies:
  - TASK-121
references:
  - .skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/briefing.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: not yet implemented. Expected State: Parse/serialize frontmatter byte-stable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 round-trip body byte-identical except edited field
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Tests pass
<!-- DOD:END -->
