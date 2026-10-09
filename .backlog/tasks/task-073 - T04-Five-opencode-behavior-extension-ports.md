---
id: TASK-073
title: T04 Five opencode-behavior extension ports
status: needs-triage
assignee: []
created_date: '2026-10-09 13:09'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-071
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/mnemonic-memory.ts
  - skill-pi/extensions/mnemonic-codeindex.ts
  - skill-pi/extensions/skillgrid-events.ts
  - skill-pi/extensions/skillgrid-compaction.ts
  - skill-pi/extensions/*.test.ts
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Port `plugins/opencode/{mnemonic-memory,mnemonic-codeindex,skillgrid-events,skillgrid-compaction}.ts` to Pi's `ExtensionAPI.tool()` shape; reserve the squad slot (full squad is T14). Tracer A, Wave 4.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each port registers tools with byte-compatible names/params/behavior vs the opencode plugin
- [ ] #2 A mem_save call lands in the mnemonic store (integration: local store)
- [ ] #3 A mutation failure sets the in-memory failure latch
- [ ] #4 BASE_URL/AGENT env resolution identical to the opencode plugin
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
