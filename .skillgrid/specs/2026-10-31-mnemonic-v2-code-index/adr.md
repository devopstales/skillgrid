# ADR Review — mnemonic-v2 code-index

> In-force ADR review + the new ADRs for this change. The in-force set comes from the
> `## LOCKED` ADR index in `.skillgrid/ASSUMPTIONS.md` (highest in use: **0042** —
> `04-adr-0042-goldfish-plan-review.md`; the memory change's last was 0041). A blueprint
> that contradicts an in-force ADR must either follow it or record a new superseding ADR.

## In-force ADRs this change relies on (substrate)

| ADR | Decision | How this change uses it |
|---|---|---|
| ADR-0031 | Go 1.22+ minimum to build | `mnemonic-v2/go.mod` declares `go 1.26.0` (the go.work floor); the code-index adds no new Go-version requirement. |
| ADR-0040 | mnemonic-v2 module extraction | The code-index lives in the same fresh `mnemonic-v2/` module the memory change created; no new module. |
| ADR-0041 | Postgres driver (`lib/pq`) | The code-index's Postgres dialect render (Task 8c) uses the same `lib/pq` driver; no new driver. |
| ADR-0008 | The `yaml` package dependency | The code-index's config (`MaxFileSize`, include/exclude, debounce) re-uses the memory change's `yaml.v3` config load; no new config dep. |

## New ADRs introduced by this change (0043–0046)

Next ADR number after 0042 (0042 is already used by `goldfish-plan-review` — this
change's four ADRs were renumbered from 0042–0045 to 0043–0046 at the critic-revision
pass; every blueprint/tasks.md reference follows). Each body is written under
`.skillgrid/artifacts/04-adr-NNNN-slug.md` at execution (T0) and added to the
ASSUMPTIONS.md index.

### ADR-0043 — tree-sitter dependency (`gotreesitter`)

- **Status:** proposed → in-force at T0
- **Context:** the code-index's extraction ladder (Task 3) needs a parser for the 6 rich
  languages (go/python/java/js/ts/tsx) + the map-tier languages (24 distinct maps + the
  `c`→`cpp` alias — 25 data-table rows). v1 uses `gotreesitter` (v0.55.1), the pure-Go
  binding (no CGo — the v1 VERIFIED cgo-free stack). It is in `mnemonic_back/go.mod` but
  new to `mnemonic-v2/go.mod`.
- **Decision:** add `github.com/odvcencio/gotreesitter v0.55.1` to `mnemonic-v2/go.mod`.
- **Alternatives:** (a) `tree-sitter` (the C binding) — rejected: adds CGo, breaks the
  cgo-free invariant (INV: `skillgrid doctor` reports `cgo: free`). (b) hand-rolled parsers
  — rejected: the rich-6 + map-tier coverage is not hand-rollable; v1's ladder is the proven
  mechanism.
- **Consequences:** the rich-6 + map-tier ladder is liftable from v1 (`mnemonic_back/internal/
  extract/`). A gotreesitter upgrade later is a re-test of the extraction ladder (the
  `extractor_stamp` content hash, D-STAMP, catches a behavior change → full re-walk).

### ADR-0044 — fsnotify dependency

- **Status:** proposed → in-force at T0
- **Context:** the watch loop (Task 6) needs filesystem events to keep the graph live. v1
  uses `fsnotify` (v1.10.1). New to `mnemonic-v2/go.mod`.
- **Decision:** add `github.com/fsnotify/fsnotify v1.10.1` to `mnemonic-v2/go.mod`.
- **Alternatives:** (a) polling — rejected: the spec (007-watch) requires a trigger, not a
  poll (debounce + burst-collapse presuppose events). (b) `inotify` directly — rejected:
  `fsnotify` is the cross-platform abstraction v1 uses.
- **Consequences:** the watch loop is a background goroutine (a process-integration threat
  row). `fsnotify` is stable (v1.10.x); the watch is per-process (opt-in, not a daemon).

### ADR-0045 — conditional onnxer dependency (the 768-dim embedder)

- **Status:** proposed → in-force at T0 (conditional)
- **Context:** D-CI8 makes the semantic store model-agnostic; the **default** embedder is
  the deterministic hash-64 floor (no dep). A real 768-dim embedder (ONNX nomic-embed-code,
  or an OpenAI-compat provider) is an **opt-in** that activates HNSW/vec0 when `dim == 768`.
  v1's ONNX embedder (`onnxer` v0.7.0) is not actually wired (it falls through to hash);
  wiring it is the v1 "follow-up."
- **Decision:** add `github.com/benedoc-inc/onnxer v0.7.0` to `mnemonic-v2/go.mod` **only if**
  the ONNX 768-dim embedder is wired in this change (Task 8a's 768-dim path). If the 768-dim
  path is deferred to an OpenAI-compat provider over the memory change's `intelligence/embedder`
  seam (which needs no new dep), this ADR is recorded as *not taken* and `onnxer` is not added.
- **Alternatives:** (a) OpenAI-compat provider only (no `onnxer`) — viable; the hash-64
  default + an OpenAI-compat 768-dim embedder needs no new dep. (b) ONNX local (nomic,
  cgo-free via `onnxer`) — adds the dep but is local-first (no network at index time).
- **Consequences:** if taken, the 768-dim embedder activates HNSW/vec0 (D-CI8); if not
  taken, the semantic leg is hash-64 brute-force by default + an opt-in OpenAI-compat 768-dim
  provider. Either way the store is model-agnostic (the `embed_meta` model-swap guard is
  lifted from v1).

### ADR-0046 — the `Store` interface extension (code-index methods)

- **Status:** proposed → in-force at T0
- **Context:** the code-index adds the 7 content + 2 FTS tables + the graph reads + the
  vector methods to the **same** `Store` the memory change owns (the shared project DB,
  spec `002-schema` §4). The extension is **additive** (no existing memory method
  signature changes) + a **table-adding seam** (T21) the KB change's `KBStore` uses to add
  its `kb_*` tables without forking the dialect code.
- **Decision:** extend the `Store` interface (`mnemonic-v2/internal/core/store/store.go`)
  with the code-index methods (Task 1: `UpsertFile`/`UpsertSymbols`/`ReplaceEdgesForFile`/
  `ReplaceUnresolvedForFile`/`ReplaceChunksForFile`/`DeleteFile`/`PruneOrphanSymbols`/
  `UpdateIndexFreshness`/`EdgesByFrom`/`EdgesByTo`/`EdgesByNameOnlyToName`/`SymbolByName`/
  `SymbolByUID`/`FileByPath`) + the vector methods (`EmbedSymbol`/`EmbedChunk`/
  `ClearVectors` — M2: the `vectorstore.Engine` holds the Store, not a `*sql.DB`; the
  vector SQL is dialect-rendered in the adapters) + the trail seam (`RecordRetrieval`,
  the shared `retrieval_trails` table migration 003 created) + the `ExecDDL` seam (T21).
  The KB change later adds its own methods via the same seam (D-KB) — it does not reshape
  the code-index methods.
- **Alternatives:** (a) a separate `CodeStore` interface — rejected: the code-index tables
  live in the same DB + share the WAL/write-retry + the migration runner; a separate store
  would fork the dialect code (the T21 seam exists to avoid exactly that). (b) the KB forking
  `sqlite.go`/`postgres.go` — rejected: that's the v1 pattern (the KB spec explicitly
  rejects it, `004-index` §3.1 "same store, different query surface").
- **Consequences:** **One-way door (D2)** — adding a method later is additive; reshaping one
  is a contract break for the memory change, the code-index, and the KB. The migration
  numbering (100 to follow the memory change's 001–004, decided in Task 1; Task 8c mirrors
  the same number) is **one-way door (D3)** — a collision is a rebase.

## Supersessions

None. There is no in-force HNSW ADR — the v1 HNSW decision was part of the retired v1 set
(0001–0006, 0009, 0011–0012, 0016–0018, 0020, 0022–0030, 0032–0038; deleted 2026-10-31, numbers
not reissued). The v2 semantic store (D-CI8) lifts the v1 HNSW *mechanism* (M=16/efConstruction=64/
efSearch=96, pure-Go) and records it as a design choice of this change, not a re-issued ADR. If the
768-dim ONNX path is taken, ADR-0045 covers that dependency; the HNSW index itself is pure-Go and
needs no new dep.
