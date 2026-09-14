# Hybrid Persistence + Degradation Convention (shared across all Skillgrid skills)

## Hybrid Write (preferred)

Every SDD artifact write does **BOTH**:
1. **Filesystem** — the primary copy in `.skillgrid/specs/<topic>/`
2. **Mnemonic** — `mem_save` under the matching topic_key (see `mnemonic-memory.md`)

Any store-mode token from the orchestrator is honored as `hybrid`.

## Degradation

If one store is unavailable (Mnemonic MCP not wired, filesystem read-only), continue with the other and emit a degradation line in the return envelope:

```
Degraded: mnemonic — MCP not wired, filesystem-only
Degraded: filesystem — read-only workspace, mnemonic-only
```

## Rules

- Never fail silently. A missing write without a `Degraded:` line is a bug.
- The filesystem copy is the recovery source of truth — when both exist and disagree, the file wins.
- A `fail` verdict (qa-report) is **persisted**, not skipped.
- Do not retry a failed Mnemonic save more than once in the same session; note the degradation and continue.
