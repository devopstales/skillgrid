# Hybrid Degradation Convention (artifact store)

Hybrid (filesystem **and** Mnemonic) is the preferred mode for every phase — the filesystem
survives branch switches, Mnemonic survives `/clear`. When one store is unavailable, degrade
explicitly instead of failing the phase. Never degrade silently.

## Degraded modes

| Situation | Mode | Behavior |
|---|---|---|
| Mnemonic tools error or no session possible | `filesystem-only` | Proceed; write all filesystem artifacts; envelope carries `Degraded: filesystem-only (Mnemonic unavailable: {reason})`; retry the Mnemonic save once before giving up |
| Filesystem read-only / outside allowed edit roots | `memory-only` | Proceed with reads + Mnemonic saves where useful; no phase may claim persisted filesystem state; envelope carries `Degraded: memory-only ({reason})` |
| `sdd-archive` without shell access | `blocked` | Archive stays blocked (`shell access required for mechanical archive copy`) — Read→Write copying is never an acceptable fallback |

## Rules

- The envelope's `Degraded:` line is load-bearing — a phase that wrote only one store without
  declaring it is a partial save, not a success.
- Upsert safety still applies in every mode: READ `apply-progress` / `tasks.md` before MERGE-saving,
  or prior batches are silently lost. Filesystem `tasks.md` is the recovery copy.
- A later phase running in full hybrid mode SHOULD backfill the missing store from the available
  one and note it (`Backfilled: {store} from {source}`).
- Fast-track waivers (see `fast-track.md`) are orthogonal: a waived change in degraded mode carries
  both markers.
