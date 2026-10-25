---
id: TASK-115
title: >-
  Epic: Hermes Agent Integration (MemoryProvider + MCP + Skills + Kanban
  ticketing)
status: in-progress
assignee: []
created_date: '2026-10-25 08:00'
updated_date: '2026-10-25 08:53'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-10-24-hermes-integration/briefing.md
  - .skillgrid/specs/2026-10-24-hermes-integration/blueprint.md
  - .skillgrid/specs/2026-10-24-hermes-integration/tasks.md
  - .skillgrid/specs/2026-10-24-hermes-integration/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make Hermes Agent a first-class install target for skillgrid: native MemoryProvider plugin (Python) for auto-prefetch + sync-turn persistence, MCP server for code intelligence + web search, skill catalog deployment, memory protocol injection, and Hermes Kanban as second ticketing backend. All wired by skillgrid install --agents hermes. Spec: 2026-10-24-hermes-integration.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 MemoryProvider plugin at plugins/hermes/ with full lifecycle
- [ ] #2 MCP server registered in ~/.hermes/config.yaml with MNEMONIC_MCP_EXCLUDE=mem
- [ ] #3 Skills deployed to ~/.hermes/skills/
- [ ] #4 Memory protocol in system_prompt_block + SOUL.md fallback
- [ ] #5 Kanban ticketing backend (type: hermes-kanban)
- [ ] #6 skillgrid install --agents hermes configures all layers
- [ ] #7 No regression to existing agents (opencode, kilocode, cursor, pi)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 cd mnemonic && go test ./... passes
- [ ] #7 cd skillgrid-cli && go test ./... passes
<!-- DOD:END -->
