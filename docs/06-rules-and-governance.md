# Rules And Governance

This document centralizes where project rules live, how they are applied, and how to update them safely.

## Rule Surfaces

- Core workflow and behavior rules: `.agents/rules/`
- Cursor-local entry and rule wiring: `.cursor/rules/`
- Short rule index for agents: `.configs/AGENTS.md`
- Project context contract and glossary expectations: `.skillgrid/project/CONTEXT.md`

## Rule Categories

- Workflow and phase gates (`skillgrid-sdd-*`, `skillgrid-branch-finish-target.mdc`)
- Memory and persistence (`skillgrid-engram-memory.mdc`)
- Context and terminology (`skillgrid-context-contract.mdc`)
- Multi-agent handoff and orchestration (`skillgrid-multiagent-handoff.mdc`, `skillgrid-gentle-orchestrator-extended.mdc`)
- Coding discipline (`skillgrid-karpathy-coding.mdc`)
- GitNexus graph safety (`skillgrid-gitnexus-nonnegotiables.mdc`, `.cursor/rules/skillgrid-gitnexus-index.mdc`)

## Change Protocol For Rules

1. Keep rules narrowly scoped and composable.
2. Prefer one source of truth per concern; avoid duplicated policy text.
3. When behavior changes, update:
   - relevant rule files
   - `docs/02-workflow-usage.md` (operator-facing flow)
   - this file when category coverage changes
4. Record major policy decisions in durable memory when Engram is enabled.

## Enforcement Notes

- `/sdd-init` is required before normal SDD execution.
- Index refresh is required during init and post-merge paths.
- Engram memory entries use version/timestamp fields with deterministic conflict handling.

## Related Documents

- `02-workflow-usage.md`
- `08-multi-agent-work.md`
- `09-subagent-personas.md`
- `13-memory-and-indexing.md`
- `100-ide-configs.md`
