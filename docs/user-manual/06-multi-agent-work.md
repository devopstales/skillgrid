# Multi-agent work

How Skillgrid keeps the orchestrator lean while fanning out implementation and research.

## Quick path

1. Main session = **smart side** (route, decide, update `tasks.md` / State).
2. Heavy work = **fresh subagents** or sequential apply with isolated context.
3. Shared truth lives on **disk** (`docs/skillgrid/…`) and in **Mnemonic** — not in chat history alone.

## Smart side vs dumb side

| Side | Role | Budget |
|------|------|--------|
| **Smart** | Orchestrator: classify, unblock DAG, spawn work, record verdicts | Stay in the usable **smart zone** — roughly **≤ ~40%** of the context window for plan/execute load |
| **Dumb** | Raw dumps, long tool output, full-file reads, degraded late-window tokens | Offload to subagents, files, Mnemonic progressive disclosure |

Quality drops before the advertised window is full. Prefer more, smaller slices over one plan that burns the smart zone and finishes degraded (gsd-core smart-zone idea).

## Skills

| Skill | When |
|-------|------|
| `subagent-execution` | Fresh agent per plan slice during apply |
| `dispatching-parallel-agents` | 2+ **independent** domains (no shared files / shared `topic_key`) |
| `simple-execution` | Small slice; inline without spawning |
| `isolated-workspace` | Worktree isolation when branches would collide |
| `handoff` | Peel an out-of-scope side problem into a brief for another agent |

## Parallel dispatch rules

Use parallel only when:

- Root causes / domains are independent
- Agents will not edit the same files
- Agents will not clobber the same Mnemonic `topic_key`

Otherwise run sequential apply along `Depends on:` edges in `tasks.md`.

## Pattern

```
Orchestrator (use-skillgrid / sdd-apply)
    │  reads tasks.md State + Depends
    ├─► Subagent A  (slice 01 — fresh context)
    ├─► Subagent B  (slice 03 — unblocked, parallel)
    └─► Updates checkboxes + mem_save decisions
```

Subagents must receive everything they need **on disk or in the prompt**. They do not inherit the parent chat.

## Checklist

- [ ] Blocking DAG in `tasks.md` is honest
- [ ] Parallel only for unblocked independent work
- [ ] Orchestrator not inlining huge files — points to paths
- [ ] Context still in smart zone; checkpoint to Mnemonic if heavy

## Next step

[Memory and indexing](07-memory-and-indexing.md)
