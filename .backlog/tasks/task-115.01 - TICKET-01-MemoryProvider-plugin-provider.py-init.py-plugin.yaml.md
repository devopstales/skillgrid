---
id: TASK-115.01
title: 'TICKET-01: MemoryProvider plugin — provider.py + __init__.py + plugin.yaml'
status: done
assignee: []
created_date: '2026-10-25 08:01'
updated_date: '2026-10-25 08:53'
labels: []
dependencies: []
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
Create the Python MemoryProvider plugin at plugins/hermes/. Full lifecycle class (MnemonicMemoryProvider), register(ctx) entry point, plugin manifest. Verify mnemonic CLI flags (--json). This is the primary integration — the tracer thread that de-risks the Python↔Go subprocess boundary.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 provider.py defines MnemonicMemoryProvider with all 12 methods
- [ ] #2 __init__.py defines register(ctx) calling ctx.register_memory_provider
- [ ] #3 plugin.yaml has name: mnemonic
- [ ] #4 is_available() returns True when mnemonic is on PATH
- [ ] #5 get_tool_schemas() returns 5 valid OpenAI schemas (mem_save, mem_search, mem_context, mem_get, mem_update)
- [ ] #6 handle_tool_call maps tool name to mnemonic CLI subcommand
- [ ] #7 SATISFIES: happy path — MemoryProvider is discovered and activates; happy path — MemoryProvider exposes memory tool schemas
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 plugins/hermes/ imports cleanly (python -c 'import plugins.hermes')
<!-- DOD:END -->
