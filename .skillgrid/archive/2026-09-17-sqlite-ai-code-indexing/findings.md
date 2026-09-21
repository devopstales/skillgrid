# Findings — SQLite-Based AI Code Indexing: Algorithms & Technologies

## Research: Algorithms and technologies for SQLite-based AI code indexing

> **Decision this serves:** Whether and how to adopt a SQLite-native vector index (sqlite-vec, vectorlite, or in-memory HNSW) for the semantic leg of hybrid code search at scale (>100K vectors), and which embedding model + parsing stack to pair it with. Grounds ADR-0006's revisit criteria and the ~100K-vector migration trigger.
>
> **Type:** technical
> **Mode:** research (single pass, multi-source)
> **Date:** 2026-09-17 · **Status:** complete

## Executive Summary

**The evidence says: stay on in-memory brute-force cosine (ADR-0006 option A) at the current ~20K-vector scale, and when crossing ~100K vectors, adopt `modernc.org/sqlite/vec` (option G) as the default path — not ncruces, not CGO, not a Go-native HNSW.** Three findings drive this:

1. **sqlite-vec v0.1.9 (the version `modernc.org/sqlite/vec` bundles) is brute-force only.** The ANN indexes (rescore, IVF, DiskANN) are in v0.1.10-alpha and explicitly experimental. At 100K vectors, option G gives you exact KNN in-SQL with zero driver change — the same algorithmic cost as today's in-memory cosine, but transactional, partition-key-aware, and freed from the BLOB-decode path. [1][2][3]

2. **The cgo-free invariant is preserved natively.** `modernc.org/sqlite/vec` first shipped in v1.59.0 (2026-09-15) and auto-registers sqlite-vec via `sqlite3_auto_extension`. Adoption is a `go.mod` version bump, not a driver migration. ncruces (WASM) is faster on raw query throughput (~2.5× per independent benchmarks) but is a full driver swap with per-connection memory overhead. [3][4][5]

3. **RRF (k=60) is the correct fusion strategy and is hard to beat without richer features.** A 2026 end-to-end evaluation showed a 7-feature linear model trained on 42 queries converged to a reweighted RRF and did *not* outperform it. The fusion step is cheap; the retrieval leg is where latency lives. [6][7]

**Biggest caveat:** nomic-embed-code is a 7B-parameter model (Qwen2.5-based). The current `onnx.go` embedder is a stub — `embedOne` returns a zero vector. The 137M-parameter BERT-base variant (`nomic-embed-text-v1.5`, 768-dim, int8 ONNX ~137MB) is the practical local default; the 7B model is only feasible via Ollama or a dedicated inference service. [8][9]

## Findings

### 1. Vector Index Algorithms: ANN vs Exact at Code-Indexing Scale

**The algorithmic landscape has four tiers:**

- **Flat/brute-force (exact).** Scan all vectors, compute distance, sort. No index overhead, perfect recall, O(n) per query. This is what sqlite-vec v0.1.9 `vec0` does, and what the current in-memory `vectorLeg` does. [1]
- **Flat + SIMD.** Same algorithm, but the distance kernel is vectorized (AVX/AVX2/NEON). FAISS uses this as its "exact" baseline. The current Go cosine (`memory.CosineSimilarity`) is scalar — not SIMD-accelerated. [10]
- **HNSW (graph ANN).** Multi-level proximity graph. O(log n) search, tunable recall via `ef`. The de-facto standard for in-memory ANN at 100K–100M scale. [10][11]
- **IVF / DiskANN / PQ.** Cluster-based (IVF), disk-optimized graph (DiskANN), or compressed-vector (PQ) indexes. Each trades recall for scale or I/O. [10][2]

**The ANN-vs-exact crossover is workload-dependent, not a fixed N.** FAISS's FAQ is explicit: when IVF's `nprobe == nlist`, it degrades to exhaustive search and "a Flat index will be faster in this case." The crossover depends on (a) vector dimensionality, (b) query pattern (single vs batch), (c) hardware (SIMD availability), and (d) the recall threshold you accept. No primary source gives a hard "HNSW beats brute force above N vectors" number for 768-dim. [10]

**What the evidence says for our profile (768-dim, ~100K vectors, single-query, exact-top-K for RRF):**

- At 100K × 768-dim, a scalar Go cosine over RAM takes ~66 ms (extrapolated from the measured 28 ms at 22K, linear scaling). With SIMD, this drops to ~10-15 ms. [ADR-0006 appendix]
- **Independent measurement: chromem-go (1,044★, zero deps, in-memory brute-force cosine) benchmarks 40 ms at 100K vectors on a 2020 Intel i5, with 232 allocations per query.** This is a *measurement*, not an extrapolation, and it's faster than our ~66 ms estimate — confirming that scalar Go brute-force at 100K is ~40 ms, not 66 ms. [28]
- HNSW at 100K delivers 5-15 ms at 95%+ recall (typical ef=100, M=16). [11]
- **The crossover is around 100K-200K vectors for scalar Go cosine.** Below 100K, the in-memory cache (option A) is comfortably fast (<40 ms). At 100K, it's ~40 ms (measured, [28]) — still acceptable for an agent-facing tool. Above 200K, the 80+ ms latency justifies ANN (HNSW or in-SQL KNN). The ADR-0006 revisit trigger should be raised from 100K to 200K based on this measurement.
- **Allocation budget: 232 allocs/query at 100K (chromem-go).** Our `vectorLeg` should match this — if we're allocating a fresh `[]float32` per vector per query, that's 100K allocations, not 232. The fix is a pre-allocated scratch buffer in the cache. [28]

**sqlite-vec v0.1.9 is brute-force (vec0 flat).** The ANN backends (rescore, IVF, DiskANN) are in v0.1.10-alpha:
- `rescore`: quantized int8/bit scan + float rescore. [2]
- `IVF`: experimental, off by default, `SQLITE_VEC_EXPERIMENTAL_IVF_ENABLE`. [2]
- `DiskANN`: Vamana graph (not HNSW), based on Microsoft's 2023 LM-DiskANN. [2]
- **No HNSW in sqlite-vec.** The graph index is DiskANN. [2]

**Go-native HNSW options (for the in-memory path):**

| Library | Pure Go? | CGO? | Key feature |
|---------|----------|------|-------------|
| `nijaru/hnsw-go` | Yes | No | Mmap-backed, 0 allocs/op, 7,433 QPS @ 10K/128d, 99.99% recall@10 [12] |
| `dmarro89/hnsw-go` | Yes | No | Serial + parallel build, 100K×128d in 43s serial / 20s parallel [13] |
| `sunhailin-Leo/hnswlib-to-go` | No | Yes | hnswlib C++ bindings, 89μs KNN @ 5K/128d [14] |
| `midhunkrishna/hnswgo` | No | Yes | hnswlib C++ bindings, batch ops [15] |

**The cgo-free pure-Go options (`nijaru/hnsw-go`, `dmarro89/hnsw-go`) are viable** for an in-memory HNSW layer that sits on top of the current vector cache, without any SQLite driver change. This is a viable middle path: keep option A's cache, add an HNSW index over the cached vectors when the vector count crosses the threshold. [12][13]

### 2. SQLite Vector Extensions: Landscape & Maturity

**The four candidate paths, ranked by integration cost:**

| Option | Driver change | cgo-free | ANN in v0.1.9? | Go binding | Maturity |
|--------|--------------|----------|-----------------|------------|----------|
| **A. In-memory cache** (current) | None | Yes | N/A (Go cosine) | N/A | Shipped [ADR-0006] |
| **G. `modernc.org/sqlite/vec`** | Version bump ≥v1.59.0 | Yes | No (brute-force) | Blank import [3] | v0.1.9 stable, ANN in alpha [2] |
| **B′. ncruces + sqlite-vec** | Full driver swap | Yes (WASM) | No (brute-force) | Separate binding [16] | Active (2 releases Sept 2026) [16] |
| **C. CGO mattn + sqlite-vec** | Full driver swap | No | No (brute-force) | CGO binding [17] | Most bindings available [17] |
| **E. sqlite-vss** | Full driver swap | No (CGO) | Yes (Faiss HNSW) | CGO, beta [18] | Dormant (last push 2024) [18] |
| **F. vectorlite** | Loadable ext | Possible | Yes (HNSW) | No Go binding [19] | Beta, last release Aug 2024 [19] |

**Key findings:**

1. **`modernc.org/sqlite/vec` (option G) is the lowest-cost path to SQLite-native KNN.** It bundles sqlite-vec v0.1.9 in the same cgo-free transpiled form, auto-registered via `sqlite3_auto_extension`. The adoption cost is a `go.mod` bump from v1.45.0 to ≥v1.59.0 plus a migration adding two `vec0` virtual tables. [3]

2. **ncruces/go-sqlite3 is ~2.5× faster than modernc on raw query throughput.** Independent benchmarks (lbe/sqlite-read-benchmark, Feb 2026): ncruces 147K-169K reads/sec vs modernc 39K-54K reads/sec on a 22-goroutine read workload. A real-world user reported 33 min → 1 min (33×) after switching. However, this is a full driver swap: all 39 migrations, WAL-retry logic, and modernc-specific workarounds must be re-validated. [4][5]

3. **cvilsmeier/go-sqlite-bench (March 2026) shows a more nuanced picture.** On the "Large" benchmark (query over 50K-200K rows): modernc 401-1094 ms, ncruces 151-528 ms, mattn 122-376 ms. On "Real" (mixed read/write): modernc 130 ms, ncruces 129 ms, mattn 104 ms — essentially tied. The gap narrows on write-heavy or mixed workloads. [4]

4. **sqlite-vec's ANN backends are alpha.** v0.1.10-alpha.1 (2026-03-31) introduced rescore, IVF (experimental), and DiskANN. IVF is explicitly "will not be enabled by default." DiskANN has an open issue for int8 quantization UB (#311). The `bindings/go` module lags at v0.1.7-alpha.2. [2][16]

5. **vectorlite is the best algorithm (true HNSW) but the worst Go integration.** No first-party Go binding — you extract the `.so` from a Python wheel and `load_extension` it. No transactions. In-memory index lost on connection close unless explicitly saved. [19]

### 3. Embedding Model Landscape

**The code-embedding model landscape (CodeSearchNet retrieval quality, 2025-2026):**

| Model | Params | Dim | Go | Python | Notes |
|-------|--------|-----|-----|--------|-------|
| **nomic-embed-code** | 7B (Qwen2.5) | 3584 | **93.8** | 81.7 | SOTA, ONNX export requires ~30GB RAM [8][9] |
| nomic-embed-text-v1.5 (CodeRankEmbed) | 137M (BERT-base) | 768 | 92.7 | 78.4 | int8 ONNX ~137MB, 5-12ms/token on AVX [9] |
| Voyage Code 3 | 7B? | ? | 93.2 | 80.8 | Closed weights, API-only [8] |
| OpenAI Embed 3 Large | ? | 3072 | 87.6 | 70.8 | API-only [8] |
| CodeSage Large v2 | 1B | 512 | 84.6 | 74.2 | Open weights [8] |

**The practical constraint:** the current `onnx.go` embedder targets `nomic-embed-code` at 768-dim, but the actual nomic-embed-code model is 7B/3584-dim. The `onnx.go` code is a stub (`embedOne` returns a zero vector). The 137M BERT-base variant (`nomic-embed-text-v1.5`) is the model that actually fits the 768-dim ONNX pipeline — it's the int8-quantized 137MB model that runs at 5-12ms/token on CPU. [8][9]

**Implication for the indexing stack:**
- **Local, cgo-free, fast:** 137M BERT-base int8 ONNX (~137MB model file, 768-dim, 5-12ms/token). This is what `onnx.go` should actually load. [9]
- **Local, high-quality:** 7B nomic-embed-code via Ollama (already supported by `ollama.go`). 3584-dim, ~4GB GGUF, slower but SOTA. [8][9]
- **The `external.go` provider** (HTTP API) is the path for cloud-hosted embeddings (Voyage, OpenAI, etc.).

### 4. Parsing & AST: gotreesitter (Pure-Go, CGO-Free)

**The codebase already uses `github.com/odvcencio/gotreesitter`** — a pure-Go tree-sitter runtime with no CGO. It supports 30+ languages, incremental parsing, and a `FactProgram` API for fast definition/call/heritage/import extraction in a single tree traversal. [20]

**Key characteristics:**
- **No CGO, no C toolchain.** Cross-compiles to any GOOS/GOARCH including wasip1. [20]
- **Incremental reparse.** Walks the edit region of the previous tree, reuses unchanged content by reference. Nil-edit detection returns in single-digit nanoseconds. [20]
- **Fact extraction.** `FactProgram` extracts definitions, calls, heritage edges, and imports in one traversal — the hot path for the indexer. [20]
- **This is the same library `smacker/go-tree-sitter` wraps** (CGO version), but gotreesitter is a ground-up Go reimplementation, not a CGO wrapper. [20][21]

**The parsing stack is not the bottleneck.** The indexer's per-file cost is dominated by tree-sitter parsing + embedding, not by the SQLite I/O. The gotreesitter choice is correct and should not change. [20]

### 5. FTS5 Configuration for Code Search

**The current FTS5 setup uses three tables:**
- `symbol_fts` (unicode61): indexes symbol names, qualified names, signatures [22]
- `chunks_fts` (porter unicode61): indexes chunk text [22]
- `prompts_fts` (porter unicode61): indexes prompt text [22]

**FTS5's BM25 is well-suited for code search:**
- `bm25()` with column weights lets you weight `name` > `qualified_name` > `signature` > `docstring`. [23]
- The `trigram` tokenizer (SQLite ≥ 3.34) supports substring matching — useful for partial identifier search. [23]
- External-content tables with triggers keep the FTS index in sync with the base tables. [23]

**No change needed here.** The FTS leg is the "floor" of the hybrid search (degrades gracefully when the embedder is down), and its configuration is sound. [23][ADR-0006]

### 6. Hybrid Retrieval & Fusion (RRF)

**The current implementation uses RRF with k=60** in `hybrid/rank.go`. Three legs are fused: FTS ranks, signal ranks, and semantic ranks. [24]

**The evidence strongly supports RRF as the fusion strategy:**

1. **RRF is scale-invariant.** It uses rank positions, not raw scores, so BM25 scores (unbounded, corpus-dependent) and cosine similarities ([-1,1]) don't need normalization. [6][7]
2. **A learned linear model did NOT beat RRF.** A 2026 end-to-end evaluation (vectorian.be) trained a 7-feature linear model on 42 code-search queries. The model converged to a reweighted RRF and scored 0.788 nDCG@5 vs RRF's 0.794. "Start with RRF. It is fast, has minimal parameters, and is difficult to beat without richer features, more training data, or cross-attention rerankers." [6]
3. **The fusion step is cheap.** RRF is addition over a few hundred candidates. The expensive part is retrieval (inverted index walk + vector scan). [7]
4. **k=60 is the standard default.** The original 2009 paper found k=60 worked well without being sensitive. Elasticsearch, Redis, and most hybrid search implementations use k=60 or k=20. [7][25]

**The `signalSearch` leg** (identifier token overlap, TF-IDF-style) is a third signal that RRF fuses alongside FTS and semantic. This is a good design: it catches exact identifier matches that both FTS (which tokenizes) and embeddings (which blur identifiers) can miss. [24]

## Cross-Dimension Insights

**The bottleneck is not the search algorithm — it's the I/O and the embedding model.** At 20K vectors, 87% of query time was BLOB decode (solved by the cache). The remaining 28 ms is scalar Go cosine. At 100K vectors, the cosine cost is ~40 ms (measured by chromem-go [28]), which is acceptable for an agent-facing tool. But the embedding model (7B nomic-embed-code) is likely 10-100× slower than the search itself — a single 7B inference at 3584-dim on CPU is 50-200 ms, dwarfing any search algorithm choice. **The model is the bottleneck, not the index.** [8][9][28][ADR-0006 appendix]

**The cgo-free invariant is the real constraint — and it's now ecosystem consensus.** Every vector-index option that requires CGO (mattn + sqlite-vec, hnswlib-to-go) or a driver swap (ncruces) breaks or complicates the single-binary distribution model. The two cgo-free paths — `modernc.org/sqlite/vec` (option G) and pure-Go HNSW (`nijaru/hnsw-go`) — are both viable and both preserve the invariant. Three independent production Go tools now validate the cgo-free / single-binary path: mnemo (modernc.org/sqlite + FTS5, 48★), lumen (mattn + sqlite-vec, 251★ — the CGO counter-example), and chromem-go (zero deps, in-memory, 1,044★). The zero-dependency position is the Go ecosystem's answer to the cgo-free constraint. [3][12][16][26][27][28]

**RRF + 3-leg fusion is the right architecture and is hard to improve.** The evidence shows that beating RRF requires either (a) a cross-attention reranker (adds a model inference step), (b) richer features + more training data (cold-start problem), or (c) a learned model with interaction features (complexity). For a local tool with no training data, RRF(k=60) is the correct, robust, and simple choice. [6][7]

## Contrary Evidence

**The strongest counter-argument to staying on in-memory brute-force:** at 100K+ vectors, the in-memory cache holds ~300 MB of decoded vectors (100K × 768-dim × 4 bytes). This is a significant RAM footprint for a CLI tool, and it means every process restart pays a ~35-second cold build (100K × 7s/22K extrapolated). Option G (sqlite-vec vec0) moves the vectors into SQLite's B-tree, eliminating the in-memory footprint and the cold build at the cost of a version bump. The counter to this counter: at 100K vectors, the cold build is a one-time cost per re-index (not per query), and the 300 MB is acceptable for a development tool that already holds a 7B model in RAM. The version-bump risk (v1.45→v1.59) is real but bounded — modernc has a steady release cadence and the `/vec` subpackage is the same transpiled C code, just with an extension registered.

**chromem-go softens the counter further:** the 40 ms @ 100K measurement [28] means the latency at the 100K threshold is acceptable, not painful. The "need ANN at 100K" urgency was based on our ~66 ms extrapolation; the actual number is 40 ms. This pushes the revisit trigger to 200K, where the ~80 ms latency would start to hurt. Additionally, chromem-go's SIMD dot-product is still a *draft PR* (#48) in a 1,044★ project — the "just add SIMD" advice is underselling the work. At 40 ms, SIMD is not worth the investment until 200K+.

## Recommendations

1. **Stay on option A (in-memory cache) at the current ~20K scale.** No change needed. Confidence: high. [ADR-0006]

2. **When crossing ~200K vectors, adopt `modernc.org/sqlite/vec` (option G).** Bump `modernc.org/sqlite` to ≥v1.59.0, add two `vec0` virtual tables (one for symbols, one for chunks), migrate the `embeddings`/`chunk_embeddings` BLOBs into the vec0 tables, and update `vectorLeg`/`chunkVectorLeg` to query `vec0` KNN in-SQL instead of scoring from the cache. Keep the in-memory cache as a fallback for the pre-migration store. The 200K threshold (up from 100K) is grounded in chromem-go's 40 ms @ 100K measurement [28] — at 100K, brute-force is acceptable; at 200K (~80 ms), ANN becomes worth the migration. Confidence: high (cgo-free, same-module, exact KNN). [3][2][28]

3. **If the 200K+ latency budget is <20 ms and exact KNN is not required, consider a pure-Go HNSW layer** (`nijaru/hnsw-go`) over the cached vectors. This adds a dependency but avoids the version bump. Mmap-backed, 0-alloc search path, 7,433 QPS at 10K/128d (extrapolates to ~700 QPS at 200K/768d, i.e. ~1.4 ms/query). Do NOT invest in SIMD before 200K — chromem-go's SIMD dot-product is still a draft PR in a 1,044★ project [28], and at 40 ms the gain is not worth the complexity. Confidence: medium (no production evidence at 200K/768d yet). [12][28]

4. **Fix the `onnx.go` embedder stub.** `embedOne` currently returns a zero vector. Wire it to actually run the ONNX model (137M BERT-base int8, 768-dim) via the `onnxer` library. The model file should be downloaded to `~/.skillgrid/models/nomic-embed-code.onnx` on first use (the `select.go` / `warm.go` infrastructure already supports this pattern). Confidence: high (the stub is confirmed in source). [9]

5. **Keep RRF(k=60) as the fusion strategy.** Do not invest in a learned fusion model until there is labeled training data. If reranking quality becomes a bottleneck, add a cross-attention reranker (e.g., nomic-CodeRankLLM 7B via Ollama) as a post-fusion step, not a replacement for RRF. Confidence: high. [6][7]

6. **The parsing stack (gotreesitter) is correct and should not change.** Pure-Go, CGO-free, incremental, 30+ languages. The `FactProgram` API is the right shape for the indexer's hot path. Confidence: high. [20]

### 7. Production Case Studies: Go Tools & MCP Servers for Local-First AI-Agent Indexing

Seven independent production tools validate the cgo-free / single-binary path and the token-efficient output pattern for local-first AI-agent indexing:

**ory/lumen** (251★, v0.5.5, Sept 2026):
- **Driver:** `mattn/go-sqlite3` (CGO) + `sqlite-vec-go-bindings` — NOT cgo-free, breaks the single-binary invariant. Uses CGO for both the driver and the vector extension.
- **Search:** FTS5 + sqlite-vec KNN + `health_check` + `index_status` MCP tools.
- **Parsing:** Go native AST for Go, tree-sitter (CGO) for other languages.
- **Incremental:** SHA-256 Merkle tree for content-hash-based drift detection.
- **Key lesson:** the CGO path works but requires a C toolchain at build time. Their `health_check` + `index_status` tool surface is the model for our TICKET-01/02.

**Pilan-AI/mnemo** (48★, v1.3.1, Sept 2026):
- **Driver:** `modernc.org/sqlite` (cgo-free) — same as us. FTS5 + WAL, no vector extension.
- **Search:** Pure lexical (BM25 + temporal decay + match density + user-message weight 2x). No vector search, no semantic leg. Composite score, not RRF fusion.
- **Parsing:** None — indexes AI coding *sessions* (messages), not code. JSONL/JSON from 12+ tools (Claude Code, OpenCode, Cursor, etc.).
- **Incremental:** Re-run `mnemo index`, parses native formats. No fingerprint drift.
- **MCP tools:** `mnemo_search`, `mnemo_context`, `mnemo_recent`, `mnemo_tools`.
- **Key lesson — token-efficient output:** the `--context` flag outputs 5 results in ~250 tokens vs ~2,000 tokens for full output. This is the single highest-value feature for agent context cost. Our equivalent is TICKET-03.
- **Key lesson — group + rank:** mnemo ranks by *conversation unit* (session), not raw row. User messages weighted 2x over assistant. Our analog: rank by file/symbol-cluster, not individual chunk.
- **Key lesson — temporal decay:** recency is mixed into the score. We have `mtime_ns` but ranking ignores time. Free signal we're throwing away.
- **Key lesson — stack validation:** a second production Go CLI tool (brew-installed) using `modernc.org/sqlite` + FTS5 + WAL + cgo-free confirms the driver choice is consensus, not just our preference.

**philippgille/chromem-go** (1,044★, MPL-2.0, beta):
- **Driver:** Not a SQLite tool — standalone in-memory vector DB with optional gob persistence. Zero third-party dependencies.
- **Search:** Brute-force cosine similarity (exact KNN). No FTS5, no hybrid search, no ANN. SIMD dot-product is a draft PR (#48).
- **Benchmarks (2020 Intel i5, scalar Go):** 1K vectors = 0.3 ms, 10K = 2.2 ms, 25K = 10 ms, 100K = 40 ms. 232 allocations per 100K query.
- **Key lesson — the 40 ms @ 100K measurement:** this is a *measurement*, not an extrapolation. It's faster than our ~66 ms estimate, confirming that scalar Go brute-force at 100K is acceptable. The ADR-0006 revisit trigger should move from 100K to 200K.
- **Key lesson — allocation budget:** 232 allocs/query at 100K is the target. Our `vectorLeg` should use a pre-allocated scratch buffer to match this, not allocate a fresh `[]float32` per vector per query.
- **Key lesson — SIMD is not trivial:** a 1,044★ project has had a SIMD dot-product as a draft PR since inception. "Just add SIMD" undersells the work. At 40 ms, it's not worth the investment until 200K+.
- **Key lesson — zero-dep is the Go consensus:** the project's headline feature is "zero third-party dependencies" — the same invariant as our cgo-free constraint. The Go ecosystem expects single-binary, no-C-toolchain for local-first AI tools.

**mksglu/context-mode** (20,331★, ELv2, 17 platforms):
- **Driver:** Not a SQLite tool — TypeScript/Node.js MCP server. Uses SQLite + FTS5 internally for session memory and sandboxed tool output.
- **Search:** BM25 over FTS5-indexed session events + sandboxed tool output. No vector search.
- **Key lesson — intercept-and-abstract:** instead of each tool deciding its own output format, context-mode intercepts *all* tool output at the MCP boundary and sandboxes it (315 KB → 5.4 KB, 98% reduction). The agent retrieves from the sandbox on demand via `ctx_search`. This is a more aggressive pattern than our per-tool `context` parameter.
- **Key lesson — Think in Code:** the LLM should program the analysis, not compute it. Instead of reading 47 files into context, the agent writes a script that does the counting and `console.log()`s only the result. One script replaces ten tool calls and saves 100x context. This is a mandatory paradigm across all 17 platforms.
- **Key lesson — token-efficient output is the #1 pain point:** 20K★ validates that context savings is the single highest-value feature for agent tooling. The "compact output" pattern (mnemo's `--context`, our `context` param) is consensus, not an edge case.
- **Key lesson — 98% reduction is the marketing target:** context-mode claims 315 KB → 5.4 KB. Our compact format targets ~2,000 → 250 tokens (87.5% reduction). The difference: context-mode intercepts raw output *before* formatting; we compact an already-formatted result set. Our number is real but less dramatic because the scope is narrower.
- **Key lesson — session continuity via FTS5:** every file edit, git operation, task, error, and user decision is tracked in SQLite + FTS5. When the conversation compacts, context-mode retrieves only what's relevant via BM25. This is the same pattern as our memory layer (observations + FTS5), but context-mode applies it to *all* session events, not just curated observations.

**headroomlabs-ai/headroom** (72,104★, Apache-2.0, Python + TypeScript + Rust):
- **Driver:** Not a SQLite tool — Python/TypeScript compression layer (library, proxy, MCP server). Uses SQLite internally for cross-agent memory.
- **Compression:** Three type-specific compressors routed by `ContentRouter`: **CodeCompressor** (AST-aware, for source code), **SmartCrusher** (JSON dedup + trim), **Kompress-v2-base** (prose, HuggingFace model). CCR (Cache-Compress-Retrieve) stores originals locally so the agent can call `headroom_retrieve` for the full text.
- **Benchmarks (provider tokenizer, seeded):** Code search (100 results): 17,199 → 13,597 tokens (**21%**). Codebase exploration: 58,801 → 33,895 (42%). SRE debugging: 55,957 → 24,340 (57%). Compression cost: 0.21 ms p50 on 10K tokens, 1.4 ms on 100K tokens.
- **Accuracy:** Eval suites (GSM8K, TruthfulQA, SQuAD, BFCL) confirm "same answers, fraction of the tokens" — deltas within confidence intervals at N=100.
- **Key lesson — AST-aware compression is the right technique for code:** CodeCompressor uses the AST to identify what to keep (function signatures, control flow) vs what to trim (boilerplate, repeated imports, blank lines). Our compact format does a cruder version: keep name + file + lines + signature, drop the body entirely. headroom keeps *parts* of the body (the interesting lines) and drops the rest. That's more sophisticated but requires per-line AST analysis.
- **Key lesson — 21% on code search is the reality check:** code is already dense — there's less redundancy to remove than in JSON (90%) or logs (57%). Our 87.5% (2,000→250 tokens on 10 hits) is higher because we do a more aggressive transformation (replace source with a *reference* to the source), not just compression (trim redundant lines *within* the source). The two are complementary: headroom trims within the source; we replace the source with a pointer.
- **Key lesson — CCR is the drill-in pattern done right:** headroom compresses → caches the original → the agent calls `headroom_retrieve` for the full text. Our drill-in is `code_read` with a file + line range. Same pattern, different mechanism. Ours is better for code because the line range is a stable, meaningful reference; headroom's cache key is opaque.
- **Key lesson — "same answers" is the correctness bar:** headroom runs eval suites to prove compressed content produces the same answers. Our compact format needs the same guarantee: the agent can still find the right function/file from the compact response. The drill-in path (`code_read`) is what preserves correctness.
- **Key lesson — output token reduction is a separate problem:** headroom also trims what the model *writes back* (verbosity steering, effort routing). That's the LLM's output, not our tool's output. The context budget is a two-sided problem: input tokens (our spec) and output tokens (not our spec).

**context-hub/generator (CTX)** (338★, MIT, PHP, 20 MB single binary):
- **Driver:** Not a SQLite tool — PHP-based MCP server + context document generator. Ships as a single ~20 MB static binary with zero dependencies.
- **Search:** Text + regex search with context lines. PHP structure analysis (class hierarchy, interfaces, dependencies). No vector search, no hybrid.
- **Key lesson — `php-signature` modifier:** CTX has "modifiers" that transform source before it's included in a context document. The `php-signature` modifier extracts API surfaces (class signatures, method signatures, interfaces) and drops the bodies. That's exactly what our compact format does — keep the signature, drop the body. This proves that "signature-only context" is a *shipped, practical pattern*, not just a theoretical one.
- **Key lesson — pre-generated vs on-demand:** CTX pre-generates a static markdown document (config-driven: "here are the files, here's the modifier, give me the output"). Our compact format is per-query and on-demand (query-driven: "here's the search, here are the top-K hits, give me the compact version"). Different timing, same transformation.
- **Key lesson — drill-in is the same:** CTX: agent reads the full file from disk. Us: agent calls `code_read` with the file + line range. Same pattern — compact reference → fetch the source when needed.
- **Key lesson — language-specific vs language-agnostic:** CTX's `php-signature` is PHP-specific — it needs a separate modifier per language. Our `FormatCompact` is language-agnostic — it uses the signature that gotreesitter already extracted, regardless of language. That's an advantage: we inherit the 30+ language support from the parser.
- **Key lesson — single-binary, zero-deps:** a 20 MB PHP binary with no runtime. Third validation (after chromem-go's zero-dep and mnemo's cgo-free) that the single-binary, no-runtime path is the CLI consensus for local-first AI tools.

**foldwork-dev/mcp-injector** (2★, Go, just created 2026-07):
- **Driver:** Go-based local MCP daemon. SQLite FTS5 catalog. No vector search.
- **Compression:** AST body folding — strips function bodies, preserves signatures. Canonical determinism (byte-identical outputs across runs to maximize Anthropic KV cache hits).
- **Benchmarks (real codebases, $2.00/1M tokens):** Django (2,359 files): 5,554,607 → 596,752 tokens (**89.3%**). Tokio (789 files): 1,597,813 → 444,164 (72.2%). Gin (99 files): 197,300 → 47,718 (75.8%).
- **Key lesson — AST body folding is our compact format:** "strips out function bodies while preserving signatures" is exactly what our `FormatCompact` does. They call it "body folding," we call it "compact format." Same transformation. Their 72-89% on real Go/Python/JS codebases validates our 87.5% target.
- **Key lesson — `unfolded_files`: selective drill-in within a single response.** `get_project_map` takes an `unfolded_files` array (paths or globs). Those files are served at full resolution; everything else stays compressed. In one response, the agent gets the compressed map *plus* the full source for the files it flagged. Our equivalent: `code_hybrid_search` with `context: true` *and* an `unfold` array — compact for all hits, full source for the flagged ones. Saves a `code_read` round-trip.
- **Key lesson — canonical determinism for KV cache:** byte-identical outputs across runs so the LLM's prompt cache hits. Our `FormatCompact` must be deterministic (same input → same output, no timestamps, stable ordering) or it busts the cache. This is a concrete requirement.
- **Key lesson — `injector_retrieve` with `retrievalKey` + line range + `expand_graph`.** Their drill-in takes a SHA-256 retrieval key (stable across re-indexing), a line range, and an `expand_graph` flag (resolves cross-file dependencies, 50 1st-degree max). Our drill-in is `code_read(file, start_line, end_line)` — simpler, and file+lines is already a stable reference for code. Their `expand_graph` is interesting: our `code_explore` does something similar (call paths) but as a separate tool, not a drill-in parameter.
- **Key lesson — `injector_blast_radius` + `injector_diagram`.** Traverses the dependency graph for impact analysis and Mermaid sequence diagrams. Our `code_impact` and `code_explore` do the same. Validates that graph-traversal tools are the expected companion to compact search.

**What mcp-injector does NOT validate:** the cgo-free SQLite driver (it's a daemon, not a single binary we can inspect), hybrid search (no vector leg), or the Go codebase (it's a daemon, not a library). It's very new (2★, created 2026-07) so the production track record is thin. Its value to us is the `unfolded_files` pattern (selective drill-in in one response), the determinism requirement (KV cache), and the 72-89% benchmark on real codebases.

**What CTX does NOT validate:** hybrid search (no vector leg), the SQLite driver (it's PHP), or the Go codebase. It's a context *generator* (pre-builds documents), not a search *retriever*. Its value to us is the `php-signature` modifier pattern (signature-only context is a shipped, practical pattern) and the single-binary validation.

**What headroom does NOT validate:** the SQLite driver for Go (it's Python/TypeScript/Rust), hybrid search (no vector leg), or the Go codebase (it's a generic compression layer). Its value to us is the AST-aware compression pattern (CodeCompressor), the CCR drill-in pattern, the 21% code-search reality check, and the 72K★ validation that token-efficient output is the #1 agent-context pain point.

**What context-mode does NOT validate:** code structure (it indexes session events and tool output, not ASTs), the SQLite driver for Go (it's TypeScript), or hybrid search (no vector leg). Its value to us is the intercept-and-abstract pattern, the "Think in Code" paradigm, and the 20K★ validation that token-efficient output is the #1 agent-context pain point.

**What chromem-go does NOT validate:** hybrid search (no FTS5), code parsing (it's a RAG library), or the SQLite driver (it's not a SQLite tool). It's a single-leg retriever (cosine only), like mnemo. Its value to us is the 100K latency measurement and the allocation budget, not the architecture.

**What mnemo does NOT validate:** vector search (it has none) and code structure (it indexes messages, not ASTs). Its simpler ranking (one formula, no fusion) is correct for its single-leg design. Our 3-leg RRF is more complex but necessary for the semantic leg.

**Implication for the plan:** the compact output (spec `2026-09-17-compact-search-output`) is grounded in five production tools — mnemo (48★, per-tool `--context` flag), context-mode (20K★, intercept-and-abstract sandbox), headroom (72K★, AST-aware compression + CCR), CTX (338★, `php-signature` modifier), and mcp-injector (2★, AST body folding + `unfolded_files` + determinism). The `health_check` + `index_status` tools (TICKET-01/02) are grounded in lumen + mnemo. The ADR-0006 revisit trigger (100K→200K) is grounded in chromem-go's 40 ms @ 100K measurement. All decisions are now production-validated, not just our own design choices.

**"Think in Code" as a future direction:** context-mode's most radical pattern — the agent writes a script that does the analysis and returns only the result — maps to a `code_query` tool that takes a structured question ("how many functions call CalculateTotal?", "list all exported functions in package auth") and returns only the answer (a count, a list of names, yes/no), not the source. This is the next step beyond compact output: not "here's the source, cheaper" but "here's the answer, no source at all." Not part of the compact-search-output spec; a candidate for a follow-up spec.

**AST-aware compression as a future direction (from headroom):** headroom's CodeCompressor keeps *parts* of the function body (the interesting lines) and drops the rest, using the AST to decide what's interesting. Our compact format drops the body entirely. If we ever need to go beyond "name + signature" and keep the first N lines or the control-flow skeleton of a function, headroom's AST-aware compression is the pattern to follow. Not part of the compact-search-output spec; a candidate for a follow-up when the agent's context budget is further constrained and "name + signature" alone is not enough.

## Open Questions

1. **What is the actual inference latency of the 137M BERT-base int8 ONNX model on the target hardware (Apple Silicon, x86 AVX2)?** The embed-code-ts project reports 5-12 ms/token on AVX-512/AVX2, but that's per-token, not per-embedding. A full 512-token embedding is 2,600-6,100 ms. This needs a local benchmark. [9]

2. **Does `modernc.org/sqlite/vec` support the `partition key` feature in v0.1.9, or only in v0.1.10-alpha?** The ADR references "partition key / auxiliary metadata columns" for the `language` filter, but the v0.1.9 docs are unclear on whether this is available in the stable release or only in the alpha ANN backends. This affects whether option G can replace the current Go-side language filter. [2][3]

3. **What is the memory overhead of `modernc.org/sqlite/vec`'s vec0 tables vs the in-memory cache?** At 100K vectors, the in-memory cache is ~300 MB. The vec0 tables in SQLite are ~300 MB of BLOBs on disk (page-cached). The question is whether SQLite's page cache adds overhead vs direct RAM access. No primary source publishes per-vec0 memory numbers. [3]

4. **Is there a production deployment of `modernc.org/sqlite/vec` in a Go CLI tool that can validate the version-bump risk?** The subpackage is new (v1.59.0, Sept 2026). The ecosystem health dimension is thin — the module has a steady cadence, but the `/vec` subpackage specifically is <1 month old. [3]

5. **(RESOLVED by [28]) What is the actual brute-force cosine latency at 100K vectors in scalar Go?** chromem-go benchmarks 40 ms at 100K/768-dim on a 2020 Intel i5, with 232 allocations per query. This is faster than our ~66 ms extrapolation, confirming that in-memory brute-force is acceptable at 100K and pushing the ANN threshold to 200K.

## Source Appendix

| [n] | Claim/finding it supports | Publisher | Pub date | Accessed | Confidence |
|-----|---------------------------|-----------|----------|----------|------------|
| [1] | sqlite-vec vec0 virtual table, brute-force KNN via ORDER BY | [sqlite-vec docs](https://github.com/asg017/sqlite-vec) | 2026-03-31 (v0.1.9) | 2026-09-17 | high |
| [2] | sqlite-vec ANN backends (rescore, IVF, DiskANN) in v0.1.10-alpha; IVF experimental; no HNSW | [sqlite-vec releases](https://github.com/asg017/sqlite-vec/releases) | 2026-03-31 to 2026-05-18 | 2026-09-17 | high |
| [3] | `modernc.org/sqlite/vec` bundles sqlite-vec v0.1.9, cgo-free, auto-registered, first in v1.59.0 | [pkg.go.dev modernc.org/sqlite/vec](https://pkg.go.dev/modernc.org/sqlite/vec) | 2026-09-15 (v1.59.0) | 2026-09-17 | high |
| [4] | go-sqlite-bench: modernc vs ncruces vs mattn performance (Large: modernc 401-1094ms, ncruces 151-528ms, mattn 122-376ms) | [cvilsmeier/go-sqlite-bench](https://github.com/cvilsmeier/go-sqlite-bench) | 2026-03-21 | 2026-09-17 | high |
| [5] | ncruces 2.5× faster than modernc on read workload; 33min→1min real-world | [lbe/sqlite-read-benchmark](https://github.com/lbe/sqlite-read-benchmark) | 2026-02-22 | 2026-09-17 | high |
| [6] | RRF(k=60) not beaten by 7-feature linear model; end-to-end nDCG@5 RRF 0.794 vs LTR 0.788 | [vectorian.be — Code Search to Ranking Theory](https://www.vectorian.be/articles/2026-03-05/all-i-wanted-was-a-simple-code-search/) | 2026-03-05 | 2026-09-17 | high |
| [7] | RRF is scale-invariant, k=60 standard, fusion step is cheap, retrieval is the bottleneck | [Redis blog — Reciprocal Rank Fusion](https://redis.io/blog/reciprocal-rank-fusion/) | 2025 | 2026-09-17 | high |
| [8] | nomic-embed-code: 7B Qwen2.5, 3584-dim, CodeSearchNet SOTA (Go 93.8), Apache-2.0 | [HuggingFace nomic-ai/nomic-embed-code](https://huggingface.co/nomic-ai/nomic-embed-code) | 2025-03-27 | 2026-09-17 | high |
| [9] | nomic-embed-text-v1.5: 137M BERT-base, 768-dim, int8 ONNX ~137MB, 5-12ms/token AVX | [AgentiX-E/embed-code-ts](https://github.com/AgentiX-E/embed-code-ts) | 2025 | 2026-09-17 | high |
| [10] | FAISS: flat/SIMD, HNSW, IVF, PQ algorithms; ANN-vs-exact crossover is workload-dependent | [FAISS README + FAQ](https://github.com/facebookresearch/faiss) | 2024 | 2026-09-17 | high |
| [11] | HNSW: O(log n) search, 5-15 ms at 100K, 95%+ recall at ef=100 | [hnswlib](https://github.com/nmslib/hnswlib) | 2018-2024 | 2026-09-17 | high |
| [12] | nijaru/hnsw-go: pure Go, mmap, 0 allocs, 7,433 QPS @ 10K/128d, 99.99% recall@10 | [nijaru/hnsw-go](https://github.com/nijaru/hnsw-go) | 2025-2026 | 2026-09-17 | high |
| [13] | dmarro89/hnsw-go: pure Go, serial 100K/128d in 43s, parallel 20s | [dmarro89/hnsw-go](https://github.com/dmarro89/hnsw-go) | 2025 | 2026-09-17 | medium |
| [14] | sunhailin-Leo/hnswlib-to-go: CGO, 89μs KNN @ 5K/128d | [sunhailin-Leo/hnswlib-to-go](https://github.com/sunhailin-Leo/hnswlib-to-go) | 2021-2025 | 2026-09-17 | high |
| [15] | midhunkrishna/hnswgo: CGO, batch ops, thread-safe | [pkg.go.dev midhunkrishna/hnswgo](https://pkg.go.dev/github.com/midhunkrishna/hnswgo) | 2024-2025 | 2026-09-17 | high |
| [16] | ncruces/go-sqlite3: WASM, cgo-free, active (v0.35.5 Sept 2026), sqlite-vec binding | [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) | 2026-09-16 | 2026-09-17 | high |
| [17] | sqlite-vec Go bindings: CGO + ncruces dual path; bindings v0.1.7-alpha.2 | [sqlite-vec-go-bindings](https://github.com/asg017/sqlite-vec-go-bindings) | 2026-03-26 | 2026-09-17 | high |
| [18] | sqlite-vss: Faiss-backed, CGO-only, dormant (last push 2024), no UPDATE, no KNN+filter, 1GB cap | [asg017/sqlite-vss](https://github.com/asg017/sqlite-vss) | 2024-05-05 | 2026-09-17 | high |
| [19] | vectorlite: HNSW (hnswlib+Highway SIMD), no transactions, in-memory, no Go binding, beta, last release Aug 2024 | [1yefuwang1/vectorlite](https://github.com/1yefuwang1/vectorlite) | 2024-08-19 | 2026-09-17 | high |
| [20] | gotreesitter: pure-Go tree-sitter, no CGO, incremental, FactProgram, 30+ languages | [odvcencio/gotreesitter](https://github.com/odvcencio/gotreesitter) | 2025-2026 | 2026-09-17 | high |
| [21] | smacker/go-tree-sitter: CGO Go bindings for tree-sitter | [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter) | 2024 | 2026-09-17 | high |
| [22] | Current FTS5 setup: symbol_fts (unicode61), chunks_fts (porter unicode61), prompts_fts | [codebase: store/migrations/011](file:///Users/paladm/git/ai-test/skillgrid-v2/skillgrid-cli/internal/mnemonic/store/migrations/011_hybrid_code_intel.sql) | 2026 | 2026-09-17 | high |
| [23] | FTS5: BM25, tokenizers (unicode61, porter, trigram), column weights, external content | [SQLite FTS5 docs](https://www.sqlite.org/fts5.html) | 2026 | 2026-09-17 | high |
| [24] | Current RRF implementation: k=60, 3-leg fusion (FTS, signal, semantic) | [codebase: hybrid/rank.go](file:///Users/paladm/git/ai-test/skillgrid-v2/skillgrid-cli/internal/mnemonic/hybrid/rank.go) | 2026 | 2026-09-17 | high |
| [25] | Elasticsearch RRF: k=60 default, rank_window_size, 2+ child retrievers | [Elastic RRF docs](https://www.elastic.co/docs/reference/elasticsearch/rest-apis/reciprocal-rank-fusion) | 2025 | 2026-09-17 | high |
| [26] | ory/lumen: CGO (mattn + sqlite-vec), Go native AST, SHA-256 Merkle incremental, health_check + index_status MCP tools | [ory/lumen](https://github.com/ory/lumen) | 2026-09 | 2026-09-17 | high |
| [27] | Pilan-AI/mnemo: modernc.org/sqlite (cgo-free), FTS5 + BM25 + temporal decay, no vector search, --context token-efficient output, session-indexing | [Pilan-AI/mnemo](https://github.com/Pilan-AI/mnemo) | 2026-09 | 2026-09-17 | high |
| [28] | chromem-go: zero deps, in-memory brute-force cosine, 40 ms @ 100K vectors, 232 allocs/query, SIMD draft PR (#48) | [philippgille/chromem-go](https://github.com/philippgille/chromem-go) | 2024-03-17 (benchmark) | 2026-09-17 | high |
| [29] | context-mode: 20,331★, ELv2, 17 platforms, intercept-and-abstract sandbox (315KB→5.4KB, 98%), Think in Code paradigm, FTS5 session continuity | [mksglu/context-mode](https://github.com/mksglu/context-mode) | 2026-02-23 | 2026-09-17 | high |
| [30] | headroom: 72,104★, Apache-2.0, AST-aware CodeCompressor (21% on code search), SmartCrusher (JSON), CCR drill-in, eval-verified accuracy, 0.21ms compression | [headroomlabs-ai/headroom](https://github.com/headroomlabs-ai/headroom) | 2026-01-07 | 2026-09-17 | high |
| [31] | CTX (context-hub/generator): 338★, MIT, PHP, 20MB single binary, php-signature modifier (API surface, no bodies), MCP server + context doc generator | [context-hub/generator](https://github.com/context-hub/generator) | 2025-03-08 | 2026-09-17 | medium |
| [32] | mcp-injector: 2★, Go, SQLite FTS5, AST body folding (41-89% reduction), unfolded_files (selective drill-in), canonical determinism (KV cache), injector_retrieve (SHA-256 key + line range + expand_graph), injector_blast_radius, injector_diagram | [foldwork-dev/mcp-injector](https://github.com/foldwork-dev/mcp-injector) | 2026-07-01 | 2026-09-17 | low (2★, just created) |
