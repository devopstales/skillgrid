# Findings — mnemonic-core

## Research: CGO-free embedding runtime + in-file vector stack for the blueprint

**Decision served:** the Key Decisions section of `blueprint.md` — which
modules implement the default (CGO-free) local ONNX tier, the CGO-free vec0
layer, and the model download. Everything below was fetched 2026-11-23
(access date for all sources).

### 1. The ADR-0005 spike-target module does not exist (counter-evidence)

- `github.com/unum-cloud/onnx` — **404 on both GitHub and pkg.go.dev** [10].
  The pure-Go ONNX runtime named in ADR-0005 / the briefing was a
  hypothesis from the brainstorm, not a verified module. Confidence: **high**
  (two independent 404s).
- The ADR-0004 spike-target `asg017/sqlite-vec-go` (standalone pure-Go
  port) is likewise not the documented Go path — the official docs list two
  Go options, neither named that [2][4]. Confidence: **high**.

→ The spike gates survive; their *targets* change to the modules below.

### 2. Default CGO-free embedding runtime: `knights-analytics/hugot` (Go backend)

- Hugot: "ONNX Transformer Pipelines for Go" — HuggingFace-compatible
  pipelines with pluggable **pure Go / ONNX Runtime / OpenXLA** backends;
  665★, Apache-2.0, 0 open issues, used in production by the authors [3].
- **The Go backend names our model:** "The Go backend is designed for
  simpler workloads, environments that disallow cgo, and for smaller models
  such as **all-MiniLM-L6-v2**. It works best with small batches of
  roughly 32 inputs per call." [3]
- The Go backend is "always available, as it does not require any cgo
  dependencies. It is provided by the **GoMLX** project. Note: currently,
  enabling **GOEXPERIMENT=simd** gives this backend a significant
  performance boost." [3]
- Escalation paths in the *same library*: ORT backend via `-tags ORT`
  ("fastest backend"; needs `libonnxruntime.so` at runtime, default path
  `/usr/lib`, override `WithOnnxLibraryPath`); XLA via `-tags XLA` [3].
- Built-in HF model download: `hugot.DownloadModel(repo, dir, opts)`; the
  `NODOWNLOAD` build tag strips the downloader [3].
- `featureExtraction` pipeline is implemented (= the sentence-embedding
  path: tokenizer + model + pooling) [3].
- HF-compatibility claim: models exported via Optimum "can be used with the
  Hugot pipelines to obtain identical predictions as in the python version"
  [3]. Confidence: **high** (primary source, fresh); that MiniLM runs end-
  to-end on the Go backend is asserted by the upstream README, not yet
  demonstrated by us → **the v0.2 spike must run it** (load
  `onnx/model.onnx`, embed one sentence, compare against a reference
  vector, bench latency).
- Alternatives (weaker for this use): `giraffesyo/ingot` v0.1.1 (general
  pure-Go ONNX interpreter, early, thin docs) [7]; `amikos-tech/pure-onnx`
  (pure Go, WIP per its own README) [9]; `benedoc-inc/onnxer` +
  `shota3506/onnxruntime-purego` (purego bindings — no cgo build, but load
  the **ORT native lib at runtime**, so not self-contained) [9].
- Latency class for pure-Go ONNX interpreters: "10–50× slower inference
  than the normal Python/C++ ONNX runtimes" (author's own report,
  2026-03-26) [8] — consistent with the blueprint's ~10× planning
  assumption; GoMLX+SIMD may do better, spike owns the number.

### 3. CGO-free vector layer: `ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces`

- Official sqlite-vec Go docs: two options — (1) **CGO**:
  `github.com/asg017/sqlite-vec-go-bindings/cgo` with
  `mattn/go-sqlite3` (compiles + statically links the C extension);
  (2) **CGO-free**: `github.com/asg017/sqlite-vec-go-bindings/ncruces` with
  **`github.com/ncruces/go-sqlite3`** — "an alternative SQLite Go driver
  that avoids CGO by using a custom WASM build of SQLite"; the binding
  package "embeds a custom WASM build of SQLite" [2].
- `ncruces/go-sqlite3` (1145★, MIT): "a `cgo`-free SQLite wrapper… wraps a
  Wasm build of SQLite, uses wasm2go to translate it to Go. **Go and
  x/sys are the only required dependencies**." Provides a `database/sql`
  driver (`/driver`), direct C-API surface, **virtual-table support**
  (`CreateModule` — required for vec0), custom functions, and an
  **`ext/fts5`** package (FTS5 in the CGO-free build) [4].
- Caveats (primary source): "replaces the SQLite OS Interface (VFS) with a
  pure Go implementation"; "each database connection executes within a
  Wasm sandboxed environment, **memory usage will be higher** than
  alternatives"; driver performance "competitive with alternatives";
  goroutine-safe per SQLite's multi-thread contract [4].
- `sqlite-vec` itself is **pre-v1 — "expect breaking changes"** [5]; pin
  the version in go.mod and treat API churn as a dependency risk.
  Confidence: **high** for the modules' existence and shape; **medium**
  for FTS5-in-the-WASM-build + vec0 registration working together → spike.
- Consequence for the design: the single CGO-free SQLite driver is
  **ncruces/go-sqlite3** (FTS5 + virtual tables + vec0 all in one driver).
  `modernc.org/sqlite` (named in the briefing) has no loadable-C or WASM
  extension path, so it cannot carry vec0 — it is the driver *only* if the
  vec spike fails and brute-force-only is accepted. Confidence: **high**
  (modernc's extension model is its documented design).

### 4. Model distribution (all-MiniLM-L6-v2)

- Model card: sentence-similarity, **384-dim**, English, **apache-2.0**,
  165M downloads, 6.2K likes; mean-pooling + L2-normalize; input truncated
  at 256 word pieces (training seq 128) [6].
- ONNX exports in the repo [6]: `onnx/model.onnx` = **90,405,214 B**
  (~86.3 MiB, portable fp32); optimized `model_O1/O2/O3.onnx` (~86 MiB);
  `model_O4.onnx` = **45.2 MiB** (quantized); qint8/quint8 variants
  **~23 MiB** but **arch-specific** (arm64 / avx512 / avx2).
  → Default download: `onnx/model.onnx` (portable); `model_O4.onnx` is a
  size-halving candidate whose quality delta is unverified (spike).
  Tokenizer assets: `tokenizer.json` (466 KB) + `vocab.txt` (231 KB) [6].
- Download mechanism: hugot's `DownloadModel` (built-in HF downloader) [3]
  or a direct `https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx` fetch; either way, cache under
  `~/.mnemonic/models/` with size+sha256 verification.
- The exact ONNX **opset** of `model.onnx` is not stated in the repo
  metadata — readable only by loading the file → **spike item** (hugot
  GoMLX opset coverage vs this opset).

### 5. What this changes in the locked record

- ADR-0004 (vectors-in-file) and ADR-0005 (C/CGo-free target) **stand** —
  the decisions are unchanged; only the spike-target module names were
  wrong. The blueprint's Key Decisions + ADR-0013 (stack selection, filed
  at the Key Decisions gate) carry the corrected identities:
  - vec0: `asg017/sqlite-vec-go-bindings/ncruces` over `ncruces/go-sqlite3`
    (was: "asg017/sqlite-vec-go")
  - ONNX runtime: `knights-analytics/hugot` Go backend / GoMLX (was:
    "unum-cloud/onnx")
- ADR-0012 (Go module path) accepted this session: same path as the old
  module — see `decisions/04-adr-0012-go-module-path.md`.

### Sources

- [1] https://github.com/unum-cloud/onnx — HTTP 404 (accessed 2026-11-23)
- [2] https://alexgarcia.xyz/sqlite-vec/go.html — sqlite-vec official docs, "Using sqlite-vec in Go" (accessed 2026-11-23)
- [3] https://github.com/knights-analytics/hugot — README, Apache-2.0, 665★ (accessed 2026-11-23)
- [4] https://github.com/ncruces/go-sqlite3 — README, MIT, 1145★ (accessed 2026-11-23)
- [5] https://github.com/asg017/sqlite-vec — README, "pre-v1, expect breaking changes" (accessed 2026-11-23)
- [6] https://huggingface.co/api/models/sentence-transformers/all-MiniLM-L6-v2 (+ /tree/main, /tree/main/onnx) — model card + file listings (accessed 2026-11-23)
- [7] https://pkg.go.dev/github.com/giraffesyo/ingot — v0.1.1, published 2026-10-06, Apache-2.0 (accessed 2026-11-23)
- [8] https://zenn.dev/kazuhito/articles/f8ddd74e37b850 — "ピュアGo実装でのONNX推論ライブラリ", 2026-03-26 (primary: library author)
- [9] web search "pure Go ONNX runtime inference library no cgo" (2026-11-23): amikos-tech/pure-onnx, benedoc-inc/onnxer, shota3506/onnxruntime-purego, mackross/gonnx, lib-x/go-insightface
- [10] https://pkg.go.dev/github.com/unum-cloud/onnx — HTTP 404 (accessed 2026-11-23)
