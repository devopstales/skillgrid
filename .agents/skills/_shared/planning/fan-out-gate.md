# Fan-Out Gate — Cost-Aware Approval

Thresholds and message shape for any skill that fans out parallel subagents
or scouts over a workspace. `execute` is the primary enforcer; `blueprint`
research and `projectwiki` mapping passes apply the same gate before
multi-scout dispatch.

## Thresholds (fixed, stated verbatim)

- **>15 parallel units** (workers or scouts) → MUST pause and ask before
  dispatching.
- **~1,500 source lines per scout chunk** (`rules/scout-contract.md`).
- **≤5 concurrent** workers/scouts per dispatch wave; queue the rest.

The numbers are fixed on purpose: a gate with "an appropriate threshold"
never fires, and a price that isn't stated can't be judged.

## Gate message shape

Ask with the cost attached, never a bare "proceed?":

> "This run fans out **N** parallel units (cap 5 concurrent, ~1,500 source
> lines per scout). Continue? / Abort. / Defer (record in the ledger; fold
> into the next run)."

If the harness has no structured-question tool, ask in plain chat. Wait for
the answer before dispatching anything.

## Direct-execution cutoff

≤5 parallel units → run in the current thread (or a single dispatch); the
gate does not fire.
