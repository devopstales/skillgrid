---
id: TASK-077
title: T07 /memory-sync command + skills sync
status: done
assignee: []
created_date: '2026-10-09 13:10'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-073
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/prompts/memory-sync.md
  - skill-pi/install/sync.sh
  - skill-pi/install/install.md
  - skill-pi/skills/
  - skill-pi/extensions/memory-sync.test.ts
  - .gitattributes
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement /memory-sync (JSONL union merge by observation id, merge=union in .gitattributes, SQLite DB gitignored) and sync.sh (copy ~/.agents/skills/ -> pi skills location, source unchanged). Tracer A, Wave 5.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A store with N observations exports N rows; a second run with the same store exports N rows (not 2N)
- [ ] #2 An empty store exports 0 rows (valid empty JSONL)
- [ ] #3 After sync.sh, a representative ~/.agents/skills/<name>/SKILL.md is present in the pi skills location; ~/.agents/skills/ is unchanged
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
