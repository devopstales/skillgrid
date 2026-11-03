---
id: TASK-074
title: T05 Permission + tickets + context + mcp-bridge extensions
status: done
assignee: []
created_date: '2026-10-09 13:09'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-071
  - TASK-073
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/permission.ts
  - skill-pi/extensions/tickets.ts
  - skill-pi/extensions/context.ts
  - skill-pi/extensions/mcp-bridge.ts
  - skill-pi/extensions/*.test.ts
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement `permission.ts` (3-tier merge, tree-sitter bash matcher, edit gate, "Allow always" persists to local tier), `tickets.ts` (ticket tool + /tickets command, 40k truncation), `context.ts` (/skillgrid context [usage|injections|compact]), `mcp-bridge.ts` (register MCP servers as mcp__<server>__<tool> tools). Tracer A, Wave 5. Largest foundation ticket (~700 lines); if it exceeds the 400-line budget in practice, split into 5a (permission+tickets) and 5b (context+mcp-bridge).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A project-tier rule is overridden by the same tool+pattern in the local tier
- [ ] #2 A missing local file does not error; 'Allow always' appends to the local file only
- [ ] #3 A tree-sitter parse of 'git status | grep error' matches both rules; a command with a parse error falls back to wildcard matching
- [ ] #4 A deny rule on 'rm -rf *' blocks 'git status | rm -rf /' via the rm segment
- [ ] #5 ticket ready returns ready-for-agent tasks; ticket create produces .backlog/tasks/task-NNN.md; no .tickets/ dir
- [ ] #6 A 50k-char list is truncated to 40k
- [ ] #7 /skillgrid context usage renders a Mnemonic category; /skillgrid context injections shows hidden parts
- [ ] #8 A pi session exposes mcp__<server>__<tool> entries for mnemonic/backlog/fluxcd/codescene/trivy
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
