---
id: TASK-071
title: T03 Global config + system prompt delta
status: done
assignee: []
created_date: '2026-10-09 13:08'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-069
  - TASK-070
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/mcp.json
  - skill-pi/settings.json
  - skill-pi/system/SYSTEM.md
  - skill-pi/system/APPEND_SYSTEM.md
  - skill-pi/version.test.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Write `skill-pi/mcp.json`, `skill-pi/settings.json` (no provider base URL literal; base URL from env), `skill-pi/system/SYSTEM.md` (delta only, no persona/engram blocks), `skill-pi/system/APPEND_SYSTEM.md`. Tracer A, Wave 3.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 grep -rE 'https?://[a-z0-9.-]+/(v1|api)' skill-pi/settings.json skill-pi/mcp.json returns no remote provider base URL (loopback 127.0.0.1:7438 allowed)
- [ ] #2 Setting SKILLGRID_MNEMONIC_HTTP_URL changes the effective base URL
- [ ] #3 grep -c 'gentle-ai:persona|gentle-ai:engram-protocol' skill-pi/system/SYSTEM.md is 0
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
