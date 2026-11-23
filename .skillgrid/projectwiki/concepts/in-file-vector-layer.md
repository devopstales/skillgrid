---
type: Concept
title: "In-file vector layer (BLOB + vec1 + brute force)"
status: stable
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md R2 (2026-11-23)"
  - "local: ../skillgrid/mnemonic/internal/codeindex/vectorstore/store.go (predecessor decision)"
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/findings.md § Spike Results (2026-11-23)"
---

# In-file vector layer (BLOB + vec1 + brute force)

The "vector database" of [[entities/mnemonic]] is **not a separate system** —
it is three layers inside the same SQLite file. No external vector DB
(Qdrant/Chroma/Lance) is ever in the path: a separate process + lifecycle is
out of proportion at 10³–10⁵ vectors, and it breaks the single-file,
single-binary contract.

## The three layers

1. **`embedding` BLOB + `embedding_model` + `embedding_dim`** on the row —
   the source of truth. Vectors are data, not a side store.
2. **`vec1` virtual tables** (SQLite's own ANN extension, shipped in ncruces'
   official `ext/vec1` on `ncruces/go-sqlite3` v0.38 — CGO-free; the
   asg017 sqlite-vec ncruces binding for the older `vec0` was refuted by
   spike 2, 2026-11-23: unpairable wasm/host drift) — the ANN index,
   `{"index":"flat","distance":"cos"}` in v1 (exact distances). Created
   lazily once the active embedder dim is stable; dropped and rebuilt on dim
   change. Behind build tag `sqlitevec`; spike-gate passed 2026-11-23.
3. **Go brute-force cosine** over decoded BLOBs — the guaranteed floor, always
   compiled in. Active whenever the vec table is absent, dim-locked, or the
   extension fails to load. 10⁵ × 384-dim cosine ≈ a few ms — exact, zero
   deps.

## Invariants

- **Dim-strict:** a dimension change is a re-embed, never a silent truncate or
  rescale (inherited from the predecessor's vectorstore rule).
- **Model-stamped:** every vector row carries model + dim; mixed models in one
  table is a re-embed.
- **Both legs when available:** recall runs FTS5 BM25 **and** vector KNN and
  fuses (RRF); when the vector leg is absent the result carries
  `degraded: true` — never silent ([[concepts/degradation-floor]]).

## Why this shape

BLOB-first keeps the file the single portable unit (`cp` the `.sqlite`, recall
follows). The ANN layer is an optimization the floor can always replace, so a
failed vec spike is a non-event: speed defers, function doesn't.

## Related

- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]], [[architecture/mnemonic-memory-core]].
- Depends on: [[concepts/degradation-floor]] (the floor discipline),
  [[decisions/0004-vectors-in-file]] (the locked decision),
  [[sources/skillgrid-mnemonic-v2]] (predecessor decision provenance).
