---
id: TASK-115.02
title: >-
  TICKET-02: SetupHermes — Go installer for Hermes (MCP config + plugin copy +
  SOUL.md + Kanban)
status: done
assignee: []
created_date: '2026-10-25 08:01'
updated_date: '2026-10-25 10:44'
labels: []
dependencies:
  - TASK-115.01
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
Implement SetupHermes() in mnemonic/internal/setup/hermes.go. Copies the plugin, writes MCP config to ~/.hermes/config.yaml (YAML), sets MNEMONIC_MCP_EXCLUDE=mem, writes SOUL.md marker block, creates Kanban board, copies skills, updates ticketing config. Add hermes case to RunSetup. Add MNEMONIC_MCP_EXCLUDE handling to mnemonic mcp.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SetupHermes copies plugins/hermes/ to ~/.hermes/plugins/mnemonic/
- [ ] #2 Writes mcp_servers block to config.yaml with all servers from config.d/mcp.yaml
- [ ] #3 Mnemonic entry has env MNEMONIC_MCP_EXCLUDE=mem
- [ ] #4 SOUL.md gets marker-delimited protocol block (idempotent)
- [ ] #5 Kanban board created (non-fatal if exists)
- [ ] #6 Skills copied to ~/.hermes/skills/
- [ ] #7 ticketing config updated to type: hermes-kanban
- [ ] #8 RunSetup hermes dispatches to SetupHermes
- [ ] #9 MNEMONIC_MCP_EXCLUDE=mem suppresses mem_* tools from mnemonic mcp
- [ ] #10 Dry-run writes nothing
- [ ] #11 SATISFIES: happy path — MCP server is registered; happy path — MCP server passes test; edge — all MCP servers registered; failure — malformed config triggers backup
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 cd mnemonic && go test ./internal/setup/ -run TestSetupHermes passes
- [ ] #7 cd mnemonic && go build ./... passes
<!-- DOD:END -->
