# Vectors live inside the SQLite file (BLOB + vec0 + brute-force floor)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The requirement is "sqlite **and vector database**" — the open question was
where the vectors live: an external vector DB (Qdrant/Chroma/Lance), a
loadable C extension, the pure-Go sqlite-vec port, or plain BLOBs. Scale at
agent-memory size: 10³–10⁵ vectors, 384–768 dim.

## Considered Options

- **External vector DB**: a second process with its own lifecycle, network
  or IPC, and backup story — out of proportion at 10⁵ vectors; breaks the
  single-file, single-binary contract.
- **`sqlite-vec` via CGo** (mattn + `LoadExtension`): best SQL-native ANN,
  but a loadable C extension breaks the pure-Go build and ships a `.so` next
  to the binary.
- **`sqlite-vec` pure-Go port** (`asg017/sqlite-vec-go` over
  `modernc.org/sqlite`): vec0 virtual tables in the same file, zero CGo;
  API parity (dim, cosine, multi-table) unverified → spike-gated, build tag
  `sqlitevec`.
- **BLOB + Go brute-force**: exact, zero deps, ~ms at 10⁵ × 384-dim; the
  guaranteed floor.

## Decision Outcome

Chosen option: **vectors in-file, three layers** — (1) `embedding` BLOB +
model + dim on the row is the source of truth; (2) `vec0` virtual tables are
the ANN index, lazily created, rebuilt on dim change; (3) Go brute-force
cosine is always compiled in and active whenever the vec layer is
absent/dim-locked/failed. No external vector database, ever.

### Consequences

- Good: the `.sqlite` file stays the single portable unit; a failed vec
  spike defers speed, not function (the floor owns the result).
- Good: dim-strictness is enforced by storage — a model/dim change is a
  re-embed, never a silent truncate.
- Bad: brute force at 10⁶+ vectors would stop being ~ms; that scale belongs
  to the deferred `code` tool's chunk corpus, and vec0 is exactly the hedge.
- Bad: the pure-Go port is unproven → the spike gate owns it (ADR-0005
  spike culture).

## Not adopted

- **External vector DB** — process + lifecycle + backup surface for a
  problem the floor already solves at local scale.
- **CGo `sqlite-vec`** — rejected while the pure-Go port is plausible;
  revisit only if the pure-Go spike fails *and* brute force measures slow
  (the floor makes both failures tolerable).

## Related

- Supersedes: none.
- Depends on: [[concepts/in-file-vector-layer]] (the full concept),
  [[concepts/degradation-floor]] (the floor discipline),
  [[sources/skillgrid-mnemonic-v2]] (the predecessor's HNSW→vec0→brute
  decision this inherits).
- Consumed by: [[architecture/mnemonic-memory-core]] (the `mem_vec` table).
