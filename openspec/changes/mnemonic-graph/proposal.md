## Why

Mnemonic's `code_*` tools are a "better grep": `code_index` splits files into fixed 80-line
text chunks and `code_search` runs BM25 FTS5 (trigram) over those chunks. That is fast and local,
but it cannot answer the questions agents actually ask about code: *who calls this function*,
*what breaks if I change this type*, *show me the equivalent logic elsewhere*, *is this function
even used?*. Exact/substring matching also misses semantic intent ("handle the auth token" vs a
function literally named `RefreshSessions`), and the trigram tokenizer does not treat identifiers
as the units they are.

Three 2025/26 code-intelligence systems confirm the same shape of solution and refine it, so this
design is grounded in working implementations, not speculation:

- **codegraph** (colbymchenry, 69k★, TS + Rust kernel): SQLite symbol graph + call edges +
  impact analysis; watcher with staleness banners; per-language extractor + per-file fallback.
- **codebase-memory-mcp** (DeusData, 41.5k★, **pure C single binary, no LLM — the agent is the
  query translator**): 158 tree-sitter languages + Hybrid LSP type resolution for Go/TS/Py/C#/Java/
  Kotlin/Rust/C++; **11-signal hybrid semantic scorer** (TF-IDF, random-indexing, MinHash near-clone,
  API/Type/Decorator signature vectors, AST profile, approximate data flow, Halstead-lite, module
  proximity, graph diffusion) on top of a **bundled** `nomic-embed-code` 768d int8 embedding (no API
  key, no Ollama); a **camelCase/snake_case-aware FTS5 tokenizer**; **qualified names** as the
  canonical node key; a **rich edge taxonomy** (`CALLS` vs `CALL_REFERENCE` vs `USAGE`, `INHERITS`,
  `DATA_FLOWS`, `SIMILAR_TO`, `SEMANTICALLY_RELATED`, `MEMBER_OF`); **Louvain community detection**
  (`get_architecture`), **dead-code detection** (zero callers), **git-diff impact mapping**
  (`detect_changes`); Cypher-like queries; and a **team-shared compressed graph artifact**
  (`graph.db.zst`, `merge=ours`, two-tier fast/best export, bootstrap-import + incremental fill).
- **Graphify** (already analysed): confirms "symbols + edges + community + no vector store required
  is enough" for structural recall.

The fix is an **addition on top of Mnemonic's existing store**, not a rewrite. Mnemonic already owns
the right substrate — per-project SQLite, incremental content-hash syncing, and a **CGo-free**
(`modernc.org/sqlite`) driver. We add, in priority order:

1. **Identifier-aware search** — a code/tokenized FTS layer that understands camelCase/snake_case.
2. **A call-graph layer** — symbols (with a qualified-name key) + a rich relation taxonomy, with
   `callers` / `callees` / `impact`.
3. **Hybrid + multi-signal semantic search** — deterministic similarity signals (no backend required)
   plus an optional bundled/Ollama dense vector, fused with FTS, so `code_search` and `code_similar`
   return conceptually-related code even when names do not match.
4. **Code analysis** — community detection, hotspots, dead code, and git-diff impact.
5. **Freshness & sharing** — a debounced watcher, staleness banners, and an optional shareable
   compressed graph artifact.

Every layer keeps the 100% local, dependency-light promise: the **core graph needs no LLM, no API key,
and no new native service**; dense embeddings are opt-in (Ollama default, optional bundled model), and
the deterministic similarity signals provide "find similar code" even with no backend configured.

## What Changes

### A. Identifier-aware code search (highest value, lowest cost)
- Replace the bare `trigram` FTS over chunks with a **code-identifier tokenizer** that splits
  `camelCase`, `snake_case`, `Kebab-Case`, and dotted qualified names into identifier tokens before
  building the FTS index (the codebase-memory-mcp `cbm_camel_split` behaviour). `code_search` keeps
  its name/signature but now matches `RefreshSessions` against the tokens `refresh` + `sessions` and
  ranks partial identifier hits. This alone materially improves "find the thing" recall over trigram.
- `code_search` results continue to return path + line range + snippet (existing fields); a
  `match` discriminator and identifier-level score are added (additive).

### B. Call graph (symbols + qualified names + rich relation taxonomy)
- A new **symbols + edges** layer in the per-project store:
  - `symbols`(path, **qualified_name**, name, kind [`function` | `method` | `class` | `interface` |
    `enum` | `type` | `constant` | `route` | `module` | `file`], start/end line, signature, and a
    `profile` BLOB holding per-symbol computed signals — see §C). The **qualified name** is the
    canonical, disambiguated identity (codebase-memory-mcp QN: `project.path.pkg.Entity`, with
    suffix lookup for short names).
  - `edges`(src_symbol, dst_symbol, **relation**, **confidence** `EXTRACTED`|`INFERRED`|`AMBIGUOUS`,
    and a `properties` BLOB for relation-specific data such as arg-to-param mapping).
- **Relation taxonomy** (adopt the subset we can resolve with confidence; extend over time):
  `CALLS` (invocation site), `CALL_REFERENCE` (used as a value, single proven target),
  `USAGE` (reference without a proven unique target — kept provisional so "might be" is legible),
  `IMPORTS`, `DEFINES`, `IMPLEMENTS`, `INHERITS`, `DEFINED_IN`, `HANDLES` (route → handler),
  `DATA_FLOWS` (property, arg→param), `EMITS` / `LISTENS_ON` (channels/event names), and the
  semantic edges `SIMILAR_TO` and `SEMANTICALLY_RELATED` (see §C) plus community `MEMBER_OF`
  (see §D).
- Per-language **extractor + reference-resolver behind one interface** with a **per-file fallback**
  so one malformed file never aborts the run (the codegraph/CBM insight we adopt **without their
  Rust/WASM runtime**). **First wave: Go**, then TypeScript/TSX; Markdown link edges reuse existing
  chunk text. Every edge is tagged `EXTRACTED` / `INFERRED` / `AMBIGUOUS` so precision is always
  legible — this is the single most-trusted part of these tools and the thing to copy first.
- **Traversal tools (cheap SQL over edges):** `code_callers(symbol, direction?, depth?)`,
  `code_callees(symbol, depth?)`, `code_impact(symbol, depth?)` = transitive dependency closure
  (the blast radius of changing a symbol), and `code_trace(symbol, direction)` = a bounded call path.
  All accept a named symbol **or** `path:line`. Optional `code_cypher` (a small
  `MATCH (a)-[:CALLS*1..5]->(b)` engine) is a Tier-3 add if traversal tools feel limiting.
- The `skillgrid index` pipeline gains a symbol+edge pass running alongside chunking, **incremental**
  (re-extract only changed files; prune symbols/edges owned by deleted files) using the existing
  `mtime_ns` + `content_hash` guard. Extraction is parallelized and buffered in memory (codebase-
  memory-mcp's RAM-first idea) before a single write transaction.

### C. Hybrid + multi-signal semantic search (tiered)
- **Deterministic similarity signals — no backend required (Tier 1):** per symbol, at index time,
  compute and store in `symbols.profile`: a **MinHash fingerprint** (+ LSH buckets) for near-clone
  detection, an **AST structural profile** (control-flow shape, expression mix, literal density),
  an **approximate data-flow** vector (param→return / param→condition), and **Halstead-lite**
  operator/operand metrics. This is how codebase-memory-mcp scores "similar code" without any model.
- **Dense embeddings (Tier 1, opt-in):** one int8 vector per indexed unit, stored as a compact `BLOB`
  with stored dimension. `code_search`/`code_similar` run an **in-process top-k cosine scan** over
  the per-project corpus (codebase-memory-mcp does exactly this — no kNN index; `sqlite-vec` remains
  an optional opt-in for very large corpora). Provider is pluggable: **default Ollama**
  (`OLLAMA_BASE_URL`, reusing Mnemonic's backend awareness), optional **bundled** local model
  (e.g. ONNX `nomic-embed-code`) for the zero-dependency path, and optional remote
  (`openai`/`gemini`/`bedrock`) by `--embedding-backend`. Provider + model + dimension are recorded
  per store (self-describing, drift-safe: a model/dimension switch triggers a vector re-index).
- **Fusion = multi-signal scoring (Tier 2):** `code_search` and `code_similar` combine FTS (BM25),
  dense-vector cosine, and the deterministic signals weighted together — codebase-memory-mcp's
  11-signal `cbm_sem_combined_score`, with **graph diffusion** (blend a node's score with its
  neighbours') and **module proximity** (boost code in the same directory/package) added at query
  time. Ships in two tiers: **v1** = FTS + dense (RRF, no per-corpus tuning); **v2** = full weighted
  multi-signal scorer with graph diffusion. Configurable weights; sensible defaults; per-hit provenance
  (`match`, each signal's sub-score) for transparency.
- Semantic edges materialised from search signals: strong `SIMILAR_TO` (MinHash+LSH, Jaccard-scored)
  and `SEMANTICALLY_RELATED` (vocabulary-mismatch, same-language, high combined score) edges are
  optionally persisted so `code_similar` can traverse them instead of re-scanning.

### D. Code analysis (architecture, health, risk)
- **Community detection** over call edges (Louvain/Leiden) → `MEMBER_OF` edges → a `code_architecture`
  tool returning languages, packages/modules, entry points, routes, **hotspots** (high-degree/hub
  symbols), **boundaries/layers**, and clusters in one call (codebase-memory-mcp `get_architecture`;
  Graphify "god nodes + communities").
- **Dead-code detection** — symbols with zero inbound `CALLS`/`USAGE`, excluding entry points
  (main, exported routes, framework entry hooks). New tool `code_dead_code`.
- **Git-diff impact mapping** — given the working-tree diff, map changed files → affected symbols →
  their callers, with a **risk classification** (direct hit / transitive / isolated) (codebase-
  memory-mcp `detect_changes`). New tool `code_impact_diff` (or folded into `code_impact`).
- All three are cheap SQL + a graph walk over the layers above; no new dependency.

### E. Freshness & sharing
- **Staleness banner** on `code_search` / `code_read` / `code_callers`: name any file whose
  `content_hash` changed on disk since the last index and instruct the agent to re-read (the
  staleness half is free — the hash is already stored).
- **Debounced `fsnotify` watcher** inside `skillgrid serve` auto-reindexes changed files (Go-first,
  incremental). Codebase-memory-mcp's model of a **shared coordination daemon** owning the watcher +
  long-lived indexing jobs across client sessions is the pattern to follow if sessions multiply; start
  with one watcher per server.
- **Optional shareable graph artifact** — a compact, `VACUUM INTO`-ed, zstd-compressed export of the
  graph store (`.skillgrid/graph.sqlite.zst`) with a `merge=ours` gitattributes line, two-tier
  (fast for watcher writes, best for explicit `skillgrid export-graph`), and bootstrap-import-then-
  incremental-fill on a fresh clone (codebase-memory-mcp's team artifact; mirrors graphify's committed
  `graphify-out/`). Aligns with Mnemonic's existing "commit the store so the team shares it" stance.
  Off by default; `.gitignore` guidance for teams who prefer everyone rebuilding.

### Not in scope (deliberate)
- A Rust/WASM/C extraction kernel — we adopt its *strategy* (per-language extractor, parallel
  resolver, per-file fallback, byte-identical outputs) in Go, using tree-sitter-Go bindings or a
  lightweight per-language AST parser. No second native runtime.
- Cross-language iOS / React-Native / Expo bridging and gRPC/GraphQL/tRPC cross-service linking —
  add only if those stacks appear.
- Full Cypher (Tier 3 — only if the traversal tools feel limited).
- A hosted multi-user graph service — MCP + HTTP transports already exist.

### Tiered rollout (what "done" means per rung)
- **Tier 1 (core):** identifier tokenizer (§A) + symbols/edges for Go (§B) + `code_callers` /
  `callees` / `impact` + FTS+dense hybrid `code_search` & `code_similar` (RRF) (§C v1) + staleness
  banner + watcher (§E).
- **Tier 2:** TS/TSX resolvers, multi-signal weighted scorer with graph diffusion + module proximity
  (§C v2), deterministic MinHash/AST/data-flow signals feeding `SIMILAR_TO`, `code_architecture`,
  dead code, git-diff impact (§D).
- **Tier 3 (optional):** `code_cypher`, embedded/bundled embedding model, shareable graph artifact,
  gRPC/GraphQL/HTTP cross-service edges.

## Capabilities

### New Capabilities
- `code-call-graph`: symbol graph with qualified-name identity + a rich, confidence-tagged relation
  taxonomy, per-language extraction/resolution with per-file fallback, and `code_callers` /
  `code_callees` / `code_impact` / `code_trace` traversal tools.
- `hybrid-code-search`: identifier-aware FTS + dense embeddings **and** deterministic similarity
  signals (MinHash, AST profile, data-flow, Halstead-lite), fused multi-signal ranking with query-time
  graph diffusion + module proximity; a natural-language / anchored `code_similar`; optional
  `SIMILAR_TO`/`SEMANTICALLY_RELATED` edges. Recorded as MODIFIED against existing `code-search`
  behaviour (name/signature preserved, fields additive, tiers v1→v2).
- `code-analysis`: community detection (`MEMBER_OF`) + `code_architecture` (hotspots/boundaries/
  layers/clusters), `code_dead_code`, and `code_impact_diff` (working-tree diff → affected symbols +
  risk classification).
- `embedding-provider`: pluggable local-first (bundled default) + optional Ollama + optional remote
  dense-embedding backends; per-store model/dimension recording; index self-description & drift
  detection (model/dimension switch forces a vector re-index).
- `code-index-freshness`: staleness banners on search/read/traversal, a debounced `fsnotify`
  auto-sync within the running server, and (Tier 3, opt-in) a shareable compressed graph artifact.
  Built on the existing `content_hash` / `mtime_ns` state.

### Modified Capabilities
- `code-layer`: `skillgrid index` additionally extracts symbols + edges + embedding/signal profiles
  (not just chunks); `code_search` changes from pure trigram-FTS to identifier-aware FTS fused with
  semantic signals. Existing `code_read` / `code_status` tools, include/exclude config, chunk window,
  and file-size behaviour are preserved.

## Impact

- **Storage / schema** (all additive, CGo-free — `files` / `chunks` / `chunks_fts` / memory /
  web-cache untouched):
  - `008_call_graph.sql`: `symbols`(…, qualified_name, kind, profile BLOB…), `edges`(src, dst,
    relation, confidence, properties), `symbol_fts` (identifier-aware), `edges_fts`.
  - `009_semantic.sql`: `embeddings`(unit_id, model, dim, vector BLOB), `lsh_buckets`(symbol,
    bucket) for MinHash near-clone, `embed_meta`(model, dim, provider, created_at).
  - `010_analysis.sql`: `communities`(symbol, cluster), community + hotspot materialisation views.
  - `011_freshness.sql` (optional): pending-sync bookkeeping.
  - Replace the `chunks_fts` `trigram` tokenizer with an **identifier-aware** tokenizer (custom FTS5
    udf wrapping camel/snake split; `modernc.org/sqlite` supports FTS5 + udfs).
- **Code (Go, `skillgrid-cli`)**:
  - `internal/mnemonic/search/fts.go` — code-identifier tokenizer + FTS query build; hybrid fusion
    (RRF v1 → weighted multi-signal v2); graph-diffusion + module-proximity rescoring; `CodeSimilar`.
  - `internal/mnemonic/codeindex/` — symbol+edge extraction pass; parallel + in-memory buffer before
    single write tx; reuse `Scan` / hashing for incremental.
  - New `internal/mnemonic/resolve/` (or `codeindex/resolvers.go`) — per-language extractor interface
    + fallback (Go first, then TS/TSX).
  - New `internal/mnemonic/semantic/` — MinHash+LSH, AST profile, data-flow, Halstead-lite, cosine,
    multi-signal scorer, graph diffusion. (Codebase-memory-mcp's `src/semantic/` is the reference for
    the signal set + weighting.)
  - New `internal/mnemonic/embed/` — provider interface (Ollama / bundled / remote), int8 BLOB I/O,
    in-process top-k, model/dimension bookkeeping.
  - New `internal/mnemonic/analysis/` — community detection, hotspots, dead code, git-diff impact.
  - `internal/mnemonic/store/` — migrations + typed accessors.
  - MCP server — register `code_callers`, `code_callees`, `code_impact`, `code_trace`, `code_similar`,
    `code_architecture`, `code_dead_code`, `code_impact_diff` (Tier 3: `code_cypher`); widen
    `code_search` / `code_read` responses (additive).
  - HTTP API — `GET /code/callers`, `/callees`, `/impact`, `/trace`, `/similar`, `/architecture`,
    `/dead-code`, `/impact-diff`; extend `/code/search`, `/code/status` (additive).
  - `skillgrid serve` — `fsnotify` watcher + debounce loop; `code_*` freshness guard; optional
    `skillgrid export-graph` / restore.
- **Dependencies**: pure-Go tree-sitter (per-language), `fsnotify`; Ollama HTTP client only when the
  backend is `ollama`; optional ONNX runtime + `nomic-embed-code` for the bundled path; optional
  `sqlite-vec` for large-corpus kNN. `modernc.org/sqlite` remains the only DB driver. **No Rust, no
  WASM runtime, no required LLM/API key for the core graph.**
- **Compatibility**: existing `code_*` tools keep their names, signatures, and existing result
  fields; new tools + additive JSON fields only. A model/dimension mismatch triggers a vector
  re-index (graceful, non-destructive to memory/web-cache). Tokenizer change re-indexes `chunks_fts`
  (auto, idempotent, content preserved).
- **Agents / plugins**: `AGENTS.md` + plugin nudges updated so that for "who-calls / what-breaks /
  find-similar / is-this-used / architecture" questions agents reach `code_impact` / `code_callers` /
  `code_similar` / `code_dead_code` / `code_architecture` **before** grep, and trust `EXTRACTED` vs
  `INFERRED`/`AMBIGUOUS` + staleness tags before asserting structural facts.
- **Docs / specs**: `openspec/specs/code-layer/spec.md` delta; new/updated section in
  `docs/user-manual/13-mnemonic.md`; new capability spec deltas under
  `openspec/changes/mnemonic-graph/specs/{code-call-graph,hybrid-code-search,code-analysis,
  embedding-provider,code-index-freshness}/spec.md`.

### Reference-system provenance (what came from where)
- codebase-memory-mcp → identifier tokenizer, QN key, edge taxonomy (CALLS/REFERENCE/USAGE, INHERITS,
  DATA_FLOWS, SIMILAR_TO, SEMANTICALLY_RELATED, MEMBER_OF), 11-signal scorer + graph diffusion +
  module proximity, MinHash/AST/data-flow/Halstead-lite signals, bundled-optional embedding,
  community/hotspot/dead-code/git-diff-impact, no-LLM structural core, shareable `.zst` artifact.
- codegraph → per-language extractor + per-file fallback, `EXTRACTED`/`INFERRED/AMBIGUOUS` tags,
  staleness banner, `callers`/`callees`/`impact` tool surface.
- Graphify → "symbol + edge + community, no mandatory vector store" as a sufficient structural core.
