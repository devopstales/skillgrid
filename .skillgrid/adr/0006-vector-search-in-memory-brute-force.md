# Vector search: in-memory brute-force cosine, no SQLite vector extension

---
status: "accepted"
supersedes: none
date: 2026-09-16
---

## Context and Problem Statement

The semantic leg of hybrid code search (`code_semantic_search`, the
`semantic` portion of `code_hybrid_search`) ranks stored symbol and chunk
embeddings by cosine similarity. At the time of this decision the live
`aiskillgrid` store held **3,905 symbol + 18,676 chunk vectors** (768-dim,
54.7 MB of BLOBs). The original implementation (`hybrid/rank.go`) scanned the
entire `embeddings` / `chunk_embeddings` tables per query, decoded each BLOB
in Go, computed cosine against the query vector, and sorted. Measured cost:
**~220 ms/query**, of which **~190 ms (87%)** was the SQLite BLOB scan + row
decode through `modernc.org/sqlite`'s pure-Go reader — not the cosine math
(~28 ms).

The question: should we adopt a SQLite vector extension — **sqlite-vec**
(vec0 virtual tables, HNSW ANN index) — to push the KNN into the database?
The cgo-free, single-binary distribution model (`modernc.org/sqlite` +
gotreesitter; `skillgrid doctor` reports `cgo: free`) is an architectural
invariant, and the ncruces WASM binding for sqlite-vec (`ncruces/go-sqlite3`
+ `sqlite-vec-go-bindings/ncruces`) preserves it — but at the cost of
replacing the entire SQLite driver underneath all 39 migrations, the
WAL-retry logic, and the modernc-specific workarounds the codebase depends on.

## Considered Options

- **A. In-memory vector cache** — decode all vectors once per
  (store, embedding-model) into a process-global map; score cosine over RAM;
  invalidate on re-index / model swap. No driver change, no new dependency,
  exact top-K.
- **B. sqlite-vec via ncruces WASM** — replace `modernc.org/sqlite` with
  `ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces`; create vec0
  virtual tables over `embeddings` / `chunk_embeddings`; KNN in-SQL.
  cgo-free (WASM), but a full driver migration.
- **C. sqlite-vec via CGO (mattn)** — `mattn/go-sqlite3` +
  `sqlite-vec-go-bindings/cgo`. Simplest sqlite-vec integration, but breaks
  the cgo-free invariant (per-platform `.so`, CGo build).
- **D. Go-native HNSW (hnswlib-go)** — no SQLite driver change; build an
  ANN index in Go over the cached vectors. Adds a dependency and an in-memory
  index structure; only pays off at 100K+ vectors.

### Considered and rejected after the decision (2026-09-16)

Two further SQLite vector extensions were evaluated post-decision and
rejected. Neither displaces option A at the current scale, and both are
worse revisit triggers than option B (sqlite-vec / ncruces).

- **E. sqlite-vss (alexgarcia.xyz)** — Faiss-backed loadable C extension.
  Rejected: (1) **not in active development** — its own README states the
  author's effort has moved to sqlite-vec; (2) **CGO-only** Go binding, tied
  to `mattn/go-sqlite3` (not `modernc`, not ncruces), with no WASM path —
  breaks the cgo-free invariant; (3) requires supplying per-platform
  pre-compiled static libs (`-L… -Wl,-undefined,dynamic_lookup`); (4) Go
  bindings still in beta with placeholder docs; (5) hard limits that collide
  with the design: **no `UPDATE`** (our embed path upserts in place), **no
  KNN + filter** (the `language` partition key would be post-filtered in Go),
  and a **1 GB Faiss index cap**.

- **F. vectorlite (1yefuwang1)** — hnswlib-backed loadable C extension, the
  strongest *algorithm* of the three (true HNSW ANN). Rejected: (1) **no Go
  binding** — `bindings/` ships only Python wheels and an npm package; the
  Go path is to extract the pre-compiled `vectorlite.[so|dll|dylib]` from a
  Python wheel and load it via `sqlite3_load_extension()`, i.e. hand-rolled
  loader glue, a fragile non-`go.mod` artifact dependency, and no compile-time
  link check; (2) **no transactions** (a stated known limitation) — directly
  conflicts with our incremental transactional embed upserts; (3) the HNSW
  index is held **in memory and lost on connection close** unless explicitly
  `('save', path)`-ed to a side file — the in-memory state option A already
  manages, plus a binary blob and save/load lifecycle; (4) **ANN recall is
  ~50–90%** at our ~20K scale (their own benchmarks: 20K vectors, 512-d,
  `ef=50` → 50% recall), whereas we chose *exact* top-K specifically for clean
  RRF fusion with the FTS leg. The one point in its favor — `modernc.org/sqlite`
  does support `connect_hook`/`load_extension`, so cgo-free *is* preservable —
  does not overcome the missing Go binding, the no-transaction constraint, or
  the ANN recall trade.

  Net: vectorlite is the best *index*, the worst *integration* for a
  Go/cgo-free single binary. It does not change the revisit ladder.

## Decision Outcome

Chosen option: **A. In-memory vector cache**, because at the current dataset
size (~20K vectors) the bottleneck is **BLOB I/O, not the search
algorithm** — 87% of the 220 ms is re-reading 55 MB of vectors out of SQLite
per query. Caching the decoded vectors in RAM eliminates that cost with zero
driver risk, zero new dependencies, and exact (not approximate) top-K
ranking, which the RRF fusion in `hybrid/rank.go` is built around. The
sqlite-vec options (B, C) only become worth the driver-migration / cgo cost
when the dataset crosses ~100K vectors (cross-project `all_projects`
semantic search, or large monorepos) — at which point ANN's asymptotic win
outweighs the migration risk, and the architecture is changing anyway.

### Implementation

- `hybrid/vectorcache.go` — process-global `vecCache` keyed by
  (store path, embedding model). `vecEntry` holds a decoded `memory.Vector`
  + a `degen` flag (zero vectors kept for the warning count).
- `hybrid/rank.go` — `vectorLeg` / `chunkVectorLeg` score cosine over the
  cached map; the language filter and metadata join run over the candidate
  set / top-K only (indexed lookups, not a full-scan join).
- `codeindex/indexer.go` — calls `hybrid.InvalidateVectorCache(path, model)`
  after the embed pass, so a re-index or model swap drops the stale cache.

### Consequences

- Good, because the semantic leg drops from ~220 ms to ~28 ms cosine +
  ~55 ms FTS/signal (the BLOB-scan cost is paid once per re-index, not per
  query), with exact top-K and no new dependency.
- Good, because the cgo-free invariant is untouched — no driver swap, no
  extension lifecycle, no per-platform artifacts.
- Good, because the invalidation signal is the embedding model (already
  tracked in `embed_meta` for the model-swap guard), so a model change
  naturally rebuilds the cache; incremental re-embeds (same model) upsert in
  place and the indexer invalidates explicitly.
- Bad, because the cache is **process-global** (one slot). It is correct for
  the common case (one project per MCP session) but would need per-store
  keying if multi-project concurrent semantic search lands in one process.
- Bad, because the cold-cache build is a one-time ~7 s decode of 55 MB —
  acceptable for a background leg of an MCP tool, but noticeable if a user
  runs the first semantic query immediately after a re-index.
- Bad, because at 100K+ vectors the brute-force cosine (~66 ms for 22K,
  scaling linearly) will eventually exceed the ~5–15 ms an HNSW index
  delivers — the migration to sqlite-vec (option B, ncruces) or a Go-native
  HNSW (option D) becomes the right move. **Migration trigger:** ~100K
  vectors in a single store, or the `all_projects` cross-store semantic
  search feature.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. A single project store crosses ~100K vectors.
2. Cross-project (`all_projects`) semantic search is implemented.
3. The ncruces driver migration is done for an unrelated reason (e.g.
   feature completeness, performance) — at which point sqlite-vec becomes a
   low-risk add-on rather than a driver swap.

When revisiting, option **B (sqlite-vec / ncruces)** remains the default
candidate: it is the only ANN alternative that is cgo-free *and* a real `go.mod`
dependency *and* transactional. sqlite-vss (E) and vectorlite (F) were
evaluated and rejected (see above) and should not be re-explored unless their
binding/transaction/recall constraints change.

---

## Measurement appendix (2026-09-16, `aiskillgrid` store)

| Metric | Value |
|---|---|
| Symbol vectors | 3,905 (768-dim) |
| Chunk vectors | 18,676 (768-dim) |
| Total BLOB size | 54.7 MB |
| Pre-cache full pipeline (scan+decode+cosine+sort) | ~220 ms/query |
| Pre-cache cosine-only (RAM, no I/O) | ~28 ms/query |
| Pre-cache BLOB scan + decode overhead | ~192 ms/query (87%) |
| Post-cache warm semantic leg (cosine only) | ~28 ms/query |
| Post-cache cold build (one-time decode) | ~7 s |
| `COUNT(*)` on `chunk_embeddings` (B-tree) | ~2 ms |
