# Multi-agent work

How AISkillGrid keeps the orchestrator lean while fanning out implementation and research.

## Quick path

1. Main session = **smart side** (route, decide, update `tasks.md` / State).
2. Heavy work = **fresh subagents** or sequential apply with isolated context.
3. Shared truth lives on **disk** (`.skillgrid/…`, `blueprint.md`, `tasks.md`) and in **Mnemonic** — not in chat history alone.

## Smart side vs dumb side

| Side | Role | Budget |
|------|------|--------|
| **Smart** | Orchestrator: classify, unblock the DAG, spawn work, record verdicts | Stay in the usable **smart zone** — roughly **≤ ~40%** of the context window for plan/execute load |
| **Dumb** | Raw dumps, long tool output, full-file reads, degraded late-window tokens | Offload to subagents, files, Mnemonic progressive disclosure |

Quality drops before the advertised window is full. Prefer more, smaller slices over one plan that burns the smart zone and finishes degraded (gsd-core smart-zone idea).

## Skills

| Skill | When |
|-------|------|
| `subagent-execution` | Fresh implementer subagent per task, per-task spec+quality review, 5-round fix loop, ledger-based |
| `parallel-execution` | 2+ **independent** problem domains in the same message (no shared files / shared `topic_key`) |
| `simple-execution` | Small slice; inline without spawning |
| `isolated-workspace` | Worktree isolation when branches would collide |
| `deep-research` | Parallel researcher subagents (researcher / verifier / red-team) |
| `parallel-code-review` | 6 specialist reviewers in parallel, then triage / dedup / verdict |

## Parallel dispatch rules

Use parallel only when:

- Root causes / domains are independent
- Agents will not edit the same files
- Agents will not clobber the same Mnemonic `topic_key`

Otherwise run sequential apply along the `Depends on:` edges in `tasks.md`.

## Pattern

```
Orchestrator (using-skillgrid / subagent-execution)
    │  reads tasks.md State + Depends
    ├─► Subagent A  (slice 01 — fresh context)
    ├─► Subagent B  (slice 03 — unblocked, parallel)
    └─► Updates checkboxes + mem_save decisions
```

Subagents must receive everything they need **on disk or in the prompt**. They do not inherit the parent chat. `using-skillgrid` carries a `<SUBAGENT-STOP>` so dispatched subagents do not re-enter the router.

## Checklist

- [ ] Blocking DAG in `tasks.md` is honest
- [ ] Parallel only for unblocked independent work
- [ ] Orchestrator not inlining huge files — points to paths
- [ ] Context still in smart zone; checkpoint to Mnemonic if heavy

## Next step

[Ticketing](07-ticketing.md)
