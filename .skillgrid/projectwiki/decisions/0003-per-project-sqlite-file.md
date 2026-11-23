# One SQLite file per project (portable brain)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[entities/mnemonic]] needs a storage layout. Two shapes were on the table:
one **global DB** with a `project` column (single lock, one open per
cross-project query) vs **one file per project** under `~/.mnemonic/`
(`$MNEMONIC_HOME`). Both predecessors
([[sources/skillgrid-mnemonic-v2]], [[sources/engram]]) use per-project
files. The new design also carries a `project` column inside per-project
files so `--all` and export semantics stay uniform.

## Considered Options

- **One global DB, `project` column**: one connection, trivially uniform
  cross-project recall; a corrupted file holds every project; copying one
  project's brain means a SQL extract.
- **One file per project**: the brain is portable (`cp` the file, copy the
  repo), a project's corruption cannot touch another's, and per-project
  backup/prune is a filesystem operation; cross-project recall (`--all`)
  opens every file in the data dir.
- **One file per project + a read-only global index**: uniform recall speed
  at the cost of a second derived store to keep consistent.

## Decision Outcome

Chosen option: **one file per project** — `~/.mnemonic/<project-slug>.sqlite`
(+WAL sidecars), `project` column retained inside each file.

### Consequences

- Good: portability and crash isolation (success criterion 6 — `cp`
  reproduces identical recall); per-project lifecycle is a filesystem op.
- Good: matches both predecessors, so the ecosystem's mental model carries
  over unchanged.
- Bad: `--all` recall = N file opens + N scans; accepted at local project
  counts (10–50 → ms-scale). `trails` records the cost when it ever matters.
- Bad: cross-project writes (a `global`-scope observation) live in one
  project's file — recall widens scope across files instead; no federation
  machinery in v1.

## Not adopted

- **Global DB** — uniformity bought at the cost of portability and isolation;
  revisit only if `--all` latency is measured to hurt.
- **Global read-only index** — a second derived store; the floor (open-all)
  is cheap enough that it is YAGNI.

## Related

- Supersedes: none.
- Depends on: [[sources/skillgrid-mnemonic-v2]] (predecessor precedent),
  [[preferences/build-preferences]] (portability preference).
- Consumed by: [[syntheses/mnemonic-core]],
  [[architecture/mnemonic-memory-core]],
  [[decisions/0009-wal-only-concurrency]].
