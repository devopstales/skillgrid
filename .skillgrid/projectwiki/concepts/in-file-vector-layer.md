---
type: Concept
title: "In-file vector layer (BLOB + vec0 + brute force)"
status: stable
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md R2 (2026-11-23)"
  - "local: ../skillgrid/mnemonic/internal/codeindex/vectorstore/store.go (predecessor decision)"
---

# In-file vector layer (BLOB + vec0 + brute force)

The "vector database" of [[entities/mnemonic]] is **not a separate system** —
it is three layers inside the same SQLite file. No external vector DB
(Qdrant/Chroma/Lance) is ever in the path: a separate process + lifecycle is
out of proportion at 10³–10⁵ vectors, and it breaks the single-file,
single-binary contract.

## The three layers

1. **`embedding` BLOB + `embedding_model` + `embedding_dim`** on the row —
   the source of truth. Vectors are data, not a side store.
2. **`vec0` virtual tables** (sqlite-vec, CGO-free Go path:
   `asg017/sqlite-vec-go-bindings/ncruces` over `ncruces/go-sqlite3`, per
   findings.md 2026-11-23 — the brainstorm's "asg017/sqlite-vec-go" name was
   never a documented module) — the ANN index. Created lazily once the active
   embedder dim is stable; dropped and rebuilt on dim change. Behind build tag
   `sqlitevec`, spike-gated.
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
