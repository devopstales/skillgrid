# 006 — Search (hybrid: FTS + structural + semantic)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `001-files-chunks` (`chunks_fts` — the FTS leg), `002-symbols-edges`
(`symbol_fts` — the symbol FTS leg), memory `00-foundation/003-intelligence` (the
embedder seam — the semantic leg).
**Consumed by:** `008-transport` (`code_search` / `code_semantic` / `code_grep`), the
KB (the `code_search` read the KB can use to find a code span to cite).

---

## 1. Capability

`code_search` is the **"prefer over grep"** tool — the agent's primary way to find code
in the index. It is a **hybrid** of three legs, fused by RRF (Reciprocal Rank Fusion).
The v1 `code_hybrid_search` (RRF K=60) is the model; v2 keeps the fusion, **simplifies
the deterministic leg**, and **keeps the v1 multi-store vector architecture** (CI-8).

The three legs:

| Leg | Source | What it matches | When it's available |
|---|---|---|---|
| **FTS** (the floor) | `chunks_fts` + `symbol_fts` | identifier / chunk text / signature (BM25) | **always** (deterministic, no model) |
| **structural** | by-example AST grep (`004` §the structural leg) | a code *pattern* (metavariables) | always (index-free, re-parses the tree) |
| **semantic** | the embedding store (one store, CI-8) | meaning (embeddings of symbol name+signature + chunk text) | when the embedder is up (INV-3) |

## 2. The FTS leg (the floor)

BM25 over `chunks_fts` (chunk text) + `symbol_fts` (name/qualified_name/signature/kind/
language). **This is the floor** — it always answers, even when the embedder is down.
A `language` filter is an exact index-level filter (the `files.language` /
`symbols.language` partition). The FTS leg is the **degradation floor** (INV-3): when
the semantic leg is absent, `code_search` is FTS-only + `degraded: no_vector` (explicit,
never a silent gap).

## 3. The structural leg (by-example AST grep)

The v1 `search/structural.go` (index-free, kept): **metavariable** pattern matching over
the gotreesitter syntax trees of a directory. A pattern like `\f(\(ARGS*))` matches any
function call; `\name` is a metavariable, `\(ARGS*)` is a wildcard. It is **index-free**
(re-parses the working tree, so it works on files not yet indexed) and is the **no-
index fallback** (the `code_grep` tool). It is also a *leg* of the hybrid (a pattern
query fuses with the FTS leg). A pattern invalid for a language skips that language
with a note (never a silent skip).

## 4. The semantic leg (the wired 3-index store, D-CI8)

> **Amended 2026-11-02 to D-CI8** (findings.md § D-CI8 — the settled decision): v2 ships
> the **wired, working 3-index** semantic store — BLOB source of truth + `vec0` + `HNSW`
> + a brute-force-over-BLOBs floor. **LSH and IVF are dropped from the v2 claim** (v1's
> `lsh_buckets` was schema-only — no reader/writer — and IVF was an adapter never wired
> into the query path; shipping them "declared, not wired" adds tables + a query-branch
> with no v1 live precedent). Re-adding either in v2.1 is additive (a new derived index +
> a query-branch) — a scope decision, not a migration.

The embedding store is a BLOB source of truth with **three derived indexes** over it
(vec0 + HNSW + brute-force). The query engine picks the best available index for the
query; all are derived (rebuildable from the BLOBs — CI-INV-1).

### 4.1 The BLOB tables (source of truth)

Dual granularity, one BLOB table each (the v1 `embeddings` + `chunk_embeddings`):

```sql
CREATE TABLE embeddings (
    symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    model TEXT NOT NULL, dim INTEGER NOT NULL,
    vector BLOB NOT NULL, language TEXT, updated_at TEXT NOT NULL
);
CREATE TABLE chunk_embeddings (
    chunk_id INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    model TEXT NOT NULL, dim INTEGER NOT NULL,
    vector BLOB NOT NULL, language TEXT, updated_at TEXT NOT NULL
);
CREATE TABLE embed_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);   -- the model-swap guard
```

**Dual-mode (INV-2):** SQLite BLOB; Postgres `vector` (pgvector). The BLOB table is the
*persistent* store — the indexes below are derived over it.

### 4.2 The three derived indexes (vec0 + HNSW + brute-force)

| Index | Backing | What it's for | Status |
|---|---|---|---|
| **vec0** | `modernc.org/sqlite/vec` (cgo-free) | the SQLite vector query path (`MATCH`/`distance`) | v1 `042_vec0_tables.sql` — kept |
| **HNSW** | pure-Go (`vectorstore/hnsw.go`, M=16, efConstruction=64, efSearch=96) | the high-recall ANN (the `hnsw_nodes`/`hnsw_links`/`hnsw_meta` tables) | v1 `057/058_hnsw_store.sql` — kept |
| **brute-force** | scan over the BLOB tables | the floor (always available, any dim) | v1 live path — lifted |
| ~~**IVF**~~ | — (was pure-Go `vectorstore/ivf.go`, `ivf_centroids` + `ivf_assignments`) | — | **dropped from the v2 claim (D-CI8)** — v1 adapter, never wired into the query path; v2.1 re-add is additive |
| ~~**LSH**~~ | — (was `lsh_buckets`) | — | **dropped from the v2 claim (D-CI8)** — v1 schema-only, no reader/writer; v2.1 re-add is additive |

**Remote (Postgres):** the BLOB table is pgvector `vector`; the ANN index is **pgvector
HNSW**. The local indexes (HNSW/vec0) are SQLite-only — the remote uses pgvector's
native index. The *query path* is dialect-agnostic (INV-2); the *index implementation*
is dialect-specific.

**The query engine picks the best available index** (the v1 `vectorstore/store.go`
seam, kept): for a semantic query, it prefers **HNSW** (highest recall) → **vec0** (the
SQLite native path) → **brute-force over the BLOBs** (the floor). A missing index is
*not* a failure — the engine falls to the next, and the last resort is brute-force over
the BLOBs (never a hard fail). The chosen index is reported in the hit's provenance (the
agent can see *which* index answered).

**Dimension gating (the 768 pin):** HNSW/vec0 activate when the embedder's `dim == 768`.
The **default embedder is the deterministic hash-64 floor** (cgo-free, no download,
always works — v1's `HashEmbedder` lifted): a 64-dim embedder leaves the ANN indexes
inactive → the engine runs brute-force over the BLOBs (explicit
`degraded: brute_force`). A real **768-dim** embedder (ONNX nomic-embed-code *with the
pool actually wired* — the v1 follow-through — or an OpenAI-compat provider over the
shared embedder seam) is **opt-in** and activates HNSW/vec0. A **model-swap guard**
(`embed_meta.embedding_model`): a swap clears *all* indexes + vectors (a different
model's vectors are not comparable). The degradation is explicit
(`degraded: no_vector` — no embedder up / `degraded: brute_force` — embedder up, ANN
index not active or not yet built / `degraded: <index>_empty`).

### 4.3 The embedder (the shared seam)

The embedder is the shared `intelligence/embedder` seam (INV-3, memory
`00-foundation/003-intelligence`). **Embedder stance (D-CI8):** the store is
**model-agnostic** (stores whatever `dim` the embedder produces; the `embed_meta`
model-swap guard is lifted from v1). The **default** embedder is the deterministic
**hash-64** floor (cgo-free, no download, always works — v1's `HashEmbedder` lifted, so
the pipeline never hard-fails and semantic search always has *something*). A real
**768-dim** embedder (ONNX nomic-embed-code — ADR-0044, conditional on the pool being
actually wired, the v1 follow-through — or an OpenAI-compat provider) is **opt-in** and
activates HNSW/vec0 when `dim == 768`. **What is embedded:** each symbol (name +
signature) → `embeddings`; each chunk (text) → `chunk_embeddings`. The semantic leg is
**language-scoped** (the `language` filter is exact; `embeddings.language`/
`chunk_embeddings.language` is the partition key, the v1 cocoindex partition).

## 5. The RRF fusion (the hybrid)

`code_search(query, limit?, language?, under?)` → **RRF over the available legs** (K=60,
the v1 constant):
1. Run the **FTS leg** (always) → a ranked list.
2. Run the **structural leg** (when the query is a pattern) → a ranked list.
3. Run the **semantic leg** (when the embedder is up) → a ranked list.
4. **Fuse by RRF:** `score(doc) = Σ 1/(K + rank_i(doc))` over the legs that returned it.
   A doc in more legs ranks higher.
5. **Every hit carries per-signal provenance** (the v1 `hybrid/rank.go` property,
   kept): which legs returned it + each leg's rank/score. The agent can see *why* a hit
   ranked where.

**`code_semantic(query, ...)`** is the semantic leg *alone* (no fusion) — the
"find by meaning" query. **`code_grep(pattern, ...)`** is the structural leg *alone* —
the by-example AST search. **`code_search`** is the fused default.

### 5.1 The `under` subtree scope (progressive-disclosure narrowing)

`under` is an **optional path/package prefix** that narrows *all three legs* to a
subtree (the `ki search --under` pattern: outline narrows the search space, search
picks the slice out of it). It is a **`files.path` prefix match** (`under = 'internal/pay'`
matches `internal/pay/…`) — no new table, no new index. Composes with `language`:
`code_search("retry", under="internal/pay", language="go")`. When `under` is set and
matches nothing, the result is empty (not an error) — the scope was too narrow. This is
the single biggest **context-rot defense for large repos**: the agent orients on a
package (`code_orient`), then searches *within* it, so the search space — and the
context the hits pull — stays bounded as the codebase grows. The `under` prefix is
recorded in the retrieval trail (INV-5) so the scoped query is reproducible.

## 6. The degradation ladder (INV-3 — explicit, never silent)

| State | `code_search` returns | Envelope |
|---|---|---|
| FTS + semantic up (HNSW) | fused (FTS + HNSW semantic) | clean |
| FTS + semantic (HNSW/vec0/brute-force fallback) | fused (FTS + the best available index) | clean (the chosen index is in the provenance) |
| FTS + brute-force (dim mismatch / index empty) | fused (FTS + brute-force over BLOBs) | `degraded: brute_force` |
| FTS only (embedder down) | FTS-only | `degraded: no_vector` |
| stale index | answers from last index + FTS | `degraded: stale_index` (CI-INV-4) |
| structural query, no grammar for a language | that language skipped | `degraded: no_grammar:<lang>` |

**The floor is always FTS** (deterministic, no model) — `code_search` *always* answers.
The semantic leg *improves* the answer; its absence *degrades* it (explicitly).

## 7. `code_read` (the post-search fetch)

`code_read(path, start_line?, end_line?, mode?)` → the source for a path/line-range
(after `code_search` narrows it), with **secret redaction** (the v1 property — a
`token=`/`password=`/`secret=` literal in the fetched slice is masked). It reads the
**working tree** (live), not a snapshot — the same live stance as the graph.

**Two modes** (the `ki get --type full` vs `--type content` pattern — progressive
disclosure, drilling without pulling the subtree):
- **`mode: full`** (default) — the live source span for the range (the
  "search → read the exact span" workflow). A symbol target (`code_read` on a symbol id
  from `code_orient`/`code_search`) expands to that symbol's `start_line`…`end_line`.
- **`mode: content`** — the **drill handle**: the symbol's signature + **pointers to its
  callees/callers as ids** (the `to_id`/`to_name` of its `calls`/`references` edges,
  confidence-labeled), *not* their bodies. The agent sees the shape of the neighborhood
  and copies an id to `code_read` deeper — each read adds one layer, never the whole
  neighborhood. For a file target, `content` mode returns the file TOC (the symbol
  list, line order) instead of the file body. This is the cheap read that keeps the
  context window flat when the agent is still deciding *which* span to pull.

## 8. The retrieval trail (INV-5)

Every `code_search` / `code_semantic` / `code_grep` records a trail: the query, the legs
that ran (FTS/structural/semantic), the top hits, the degradation flag. `mnemonic trail`
(the shared CLI, memory `005-transport-cli`) inspects them. This is the same trail
store the memory + KB use (the shared INV-5 substrate).

## 9. Invariants

INV-2 (dual-mode — FTS5↔tsvector, the local 3-index store↔pgvector), INV-3 (the
degradation ladder, explicit `degraded`), INV-5 (trails), CI-INV-1 (the FTS + all
derived vector indexes (vec0/HNSW) are derived — rebuildable from the BLOBs + the tree;
brute-force is the floor over the BLOBs), CI-INV-4 (stale is a signal, not a failure —
`degraded: stale_index`).

## 10. Open items

- **The RRF K:** 60 (the v1 constant). Configurable. Not a spec constant.
- **The embedding dim:** the store is model-agnostic (D-CI8). The **default** embedder
  is hash-64 (brute-force over the BLOBs — the ANN indexes activate only at `dim == 768`).
  A real 768-dim embedder is opt-in (ONNX per ADR-0044 or OpenAI-compat). A model swap
  clears all vectors + indexes (the `embed_meta` guard).
- **The index-preference order:** HNSW → vec0 → brute-force (D-CI8; the v1 live path,
  lifted). The order is an implementation detail; the contract is "the best available
  index, brute-force is the floor, never a hard fail."
- **The structural-leg pattern syntax:** the v1 metavariable syntax (`\name`,
  `\(ARGS*)`, `\*`, `\_`). Kept verbatim (it is a documented pattern language, not a
  spec detail to redesign).
- **`code_search` default `limit`:** 10 (the v1 default). Configurable.
