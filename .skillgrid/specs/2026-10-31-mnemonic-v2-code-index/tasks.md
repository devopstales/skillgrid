# Tasks — mnemonic-v2 code-index function

> **STATUS:** `sliced` (2026-11-01)

> Sliced from `.skillgrid/specs/2026-10-31-mnemonic-v2-code-index/blueprint.md`.
> Vertical tracer-bullet tickets, dependency-ordered, sized for one fresh agent context window.
> **This change executes only after the memory change ships** (serial development, AGENTS.md § Rules).

## Epic Summary

Build the code-index function of mnemonic v2 — the live symbol/edge graph of the project's working tree (tree-sitter extraction with an optional LSP tier, dual-dialect storage, graph queries, hybrid search, live watch) — behind the memory change's `Store` and the `code_*` MCP surface the knowledge-base + plugins consume. Tracer-thread: the thin end-to-end `code_index → SQLite graph → code_search` path works first (Task 1's door check), then thickens (graph → hybrid search → semantic store → transport → watch → advanced passes).

The load-bearing requirement is the **live-probe relevance contract (T-CI-1…4, findings.md)**: the v1 live CLI probe on `~/.skillgrid/mnemonic/skillgrid.sqlite` (375 files, 1051 chunks, 1570 symbols, migration 060) showed the four capabilities broken/low-relevance — the FTS leg did not match the queried symbol, a `no such column` leg still shaped the answer, every graph read failed `no such column: s.uid` (no `ALTER TABLE symbols ADD COLUMN uid` bridge), and `code_status` reported `stale:false` while the graph was dead. v2 must give a *good relevant answer*, not just a tool that exists.

Fast-Track: **standard** — high-risk classification (new public `code_*` MCP contract, the shared `Store` extended, the v1 bug-class migration-parity regression, 3 new ADR-gated deps; 8 tasks, ~6k+ changed lines). No fast-track waiver; full tasks.md.

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~6,000–8,000 (8 tasks, ~30–40% test code, 30+ new files) |
| 400-line budget risk | High (Task 1 ~1200 lines: interface extension + 7-table migration + dialect render + door check, held whole) |
| Chained PRs recommended | Yes |
| Suggested split | 7 work units (W1 store-extension+migration+parity → W2 extraction → W3 graph → W4 search → W5 watch → W6 transport → W7 semantic-store+LSP+PG) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main (serial development — one change at a time; waves execute in order; the whole change sits behind the memory change) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | `Store` extension + 7-table migration + migration-parity (T-CI-3) + FTS unit (T21) (Tasks 1–2) | PR 1 | `cd mnemonic-v2 && go test ./internal/core/store/...` | N/A — table-driven tests on temp-dir SQLite | `mnemonic-v2/internal/core/store/` (the code-index additions: `types_code.go`, `fts.go`, `migrations/100_code_index.sql`, the `store.go` method set, `sqlite.go` code-index render) |
| 2 | Extraction ladder + pool + real stamp + chunker + knowledge nodes (Task 3) | PR 2 | `cd mnemonic-v2 && go test ./internal/codeindex/extract/...` | Real gotreesitter grammars (go/python/java/js/ts/tsx) against fixture source files | `mnemonic-v2/internal/codeindex/extract/` |
| 3 | Graph walker (impact/neighbors/resolve/packages/path/affected) (Task 4) | PR 3 | `cd mnemonic-v2 && go test ./internal/codeindex/graph/...` | N/A — walker tested against the test store with seeded graphs | `mnemonic-v2/internal/codeindex/graph/` |
| 4 | Hybrid search (RRF) + structural grep + code_read + T-CI-1/T-CI-2 (Task 5) | PR 4 | `cd mnemonic-v2 && go test ./internal/codeindex/search/...` | N/A — fused legs against the test store + fixture files; the broken-leg test forces a failing leg | `mnemonic-v2/internal/codeindex/search/` |
| 5 | Watch loop (fsnotify + debounce + burst-collapse + re-check + isolation) (Task 6) | PR 5 | `cd mnemonic-v2 && go test ./internal/codeindex/watch/...` | A real temp-dir fsnotify event source (the fake-event seam for the deterministic tests) | `mnemonic-v2/internal/codeindex/watch/` |
| 6 | `code_*` MCP transport + `code_status` + CLI group + T-CI-4 (Task 7) | PR 6 | `cd mnemonic-v2 && go test ./internal/transport/... ./cmd/...` | Real MCP stdio round-trip via the memory server's profile mechanism (the two-profile test) | `mnemonic-v2/internal/transport/handler/handler_code.go` + `mcp/tools_code.go` + `cmd/mnemonic/index.go` |
| 7 | Semantic vector store (D-CI8) + LSP tier + Postgres render (Task 8) | PR 7 | `cd mnemonic-v2 && go test ./internal/codeindex/vectorstore/... ./internal/codeindex/extract/ -run 'Semantic|Vector|ModelSwap|LSP'` | Real `modernc.org/sqlite/vec` + a 768-dim test embedder; the LSP tier against a fake language-server subprocess; PG gated on `MNEMONIC_PG_TEST_DSN` | `mnemonic-v2/internal/codeindex/vectorstore/` + `extract/lsp.go` + `internal/core/store/postgres_code.go` + `migrations_pg/100_code_index.sql` |

> The four plain-text guard lines above are the **contract** — `skillgrid:subagent-execution` matches them literally. Risk High → `Chained PRs recommended` MUST be `Yes` and every work unit names a focused test, a runtime harness (or `N/A` + reason), and a rollback boundary.

## Tickets

> **Precondition (all tickets):** T0 checklist is complete and committed — the **memory change is SHIPPED** (`2026-10-31-mnemonic-v2-memory`: `internal/core/store` + `syncid` + `intelligence` + `config`/`project` + `transport/mcp` exist, `cd mnemonic-v2 && go build ./...` exits 0); `findings.md` committed; ADR-0042/0043/0044/0045 written under `.skillgrid/artifacts/` + indexed in ASSUMPTIONS.md; `state.yaml` repointed to `2026-10-31-mnemonic-v2-code-index`; `acceptance.feature` committed **before** any `mnemonic-v2/` code commit (pre-commit zone guard, AGENTS.md § Rules).

### TICKET-01 — `Store` extension + 7-table migration + door check
- **Tracker ID:** TASK-100  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** Extend the memory change's `Store` interface (additive, ADR-0045) with the code-index write + read methods + the `CreateTable` table-adding seam (T21); `types_code.go` (FileRow/Symbol/Edge/FileGraph/FreshnessRecord/…); `migrations/100_code_index.sql` (the 7 tables, additive, numbered to follow the memory change's 001–060 range); the SQLite dialect render (`sqlite.go`) with the `code_index` write path + the `SymbolByName`/`EdgesBy*` reads. Mint `sync_id` for files + symbols (INV-6). **The door check** — if a `code_index` cannot write `files`+`symbols`+`edges` through the extended Store (SQLite, WAL) and a `code_search` FTS floor cannot find a saved symbol by name, stop: the code-index's foundation does not hold.
- **Acceptance:** `go test ./internal/core/store/ -run 'TestDoorCheckCodeIndexWritesAndSearches|TestStoreExtensionAdditive'` passes — the file+symbol+edge round-trip works, the FTS/name floor finds the symbol by name (the v1 FTS-leg failure fixed), the memory observation methods still work on the same store (additive), and sync ids match `^(file|sym)-[0-9a-f]{16}$`.
- **SATISFIES:** door-check-code, store-extends-additive, single-node-table, sync-id-format-files-symbols
- **Files:** `mnemonic-v2/internal/core/store/{store,types_code,sqlite}.go` + `migrations/100_code_index.sql` + `store_code_test.go`
- **Size:** ~1200 (L) — held whole (the interface extension + the single-node-table schema + the dialect render are the hypothesis being falsified)
- **Blocks:** TICKET-02, TICKET-03, TICKET-04
- **Blocked by:** none (memory change shipped)
- **Reversibility:** one-way (D2 — the `Store` extension; D3 — the migration numbering). A method reshape or a number collision later is a contract break / a rebase.
- **Fails-when:** `TestDoorCheckCodeIndexWritesAndSearches` does not return exactly the seeded symbol for the name query, OR the memory observation methods stop working (the extension is not additive).

### TICKET-02 — Migration-parity regression (T-CI-3) + reusable FTS unit (T21)
- **Tracker ID:** TASK-101  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The migration-parity regression (the v1 bug class, findings.md): a fresh store AND a store opened at each prior migration version run every `code_*` read without a SQL-column error (the v1 live probe: 1570 symbols, migration 060, every graph read failed `no such column: s.uid`); the test iterates a **table-set** (not a hardcoded list) so the KB change extends the same loop over its `kb_*` tables (D-KB). Extract the reusable FTS rendering unit (T21: external-content FTS5 DDL + the four-trigger sync set + the ranking-weights shape) into `store/fts.go` so the KB's `kb_pages_fts` re-uses it verbatim. Add the `OpenUpTo` test seam (apply migrations up to a version).
- **Acceptance:** `go test ./internal/core/store/ -run 'TestMigrationParityAllVersions|TestCodeMigrationRunsFromEmptyAndReopens|TestFtsUnitReusable|TestFTSStaysInSyncCode|TestShortTermLikeFallbackCode'` passes — no read fails `no such column` at any version; a version mismatch is a version error; the FTS unit produces 1 table + 4 triggers and renders identically for `chunks_fts` and `kb_pages_fts`.
- **SATISFIES:** migration-runs-from-empty, migration-parity-all-versions, fts-stays-in-sync-code, short-term-like-fallback-code, fts-unit-reusable
- **Files:** `mnemonic-v2/internal/core/store/{fts,migrate_code_test,fts_test}.go`; modify `migrations/100_code_index.sql` (only the gap the parity test finds)
- **Size:** ~400 (M)
- **Blocks:** TICKET-04 (the FTS leg reads `chunks_fts`)
- **Blocked by:** TICKET-01
- **Reversibility:** reversible (the FTS extraction is a refactor; the parity test is additive)
- **Fails-when:** `TestMigrationParityAllVersions` finds a read that fails `no such column` at a version the memory change shipped (the v1 bug class, not fixed).

### TICKET-03 — Extraction ladder + self-healing pool + real stamp + chunker + knowledge nodes
- **Tracker ID:** TASK-102  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The tree-sitter extraction ladder (6 rich one-pass via gotreesitter + 25 map via the node-type **data table** + regex fallback, ADR-0042); the self-healing pool + per-language circuit breaker (MaxRespawns=3, TripAfter=3); the **real extractor stamp** (D-STAMP — a content hash, not v1's dead `"v1"` const); the AST-first chunker (≤1000 chars, 250 merge, 80/10 window, lines fallback); the knowledge-node extractors (route/doc/config/sql). The per-file write goes through the Store (TICKET-01); the idempotent upsert (same `content_hash` → no-op) + the `MaxFileSize` oversized outcome (counted, not silent).
- **Acceptance:** `go test ./internal/codeindex/extract/...` passes — go is rich tier, python is rich/map, unmapped is regex; a statically-bound call is EXTRACTED and a name-only call is kept with a null target + an unresolved row (CI-INV-3); the circuit breaker trips after 3 deaths; `StampFor` is a content hash (two configs → two stamps); the chunker is AST-first with the lines fallback; `README.md`→doc, `config.yaml`→config, `schema.sql`→table, `.xyz`→skipped; an oversized file is not indexed + is counted.
- **SATISFIES:** extract-ladder-3-tiers, self-healing-pool-circuit-breaker, extractor-stamp-content-hash, chunker-ast-first, knowledge-node-classification, max-file-size-oversized
- **Files:** `mnemonic-v2/internal/codeindex/extract/{ladder,languages,fallback,pool,stamp,chunker,knowledge}.go` + tests
- **Size:** ~1500 (L) — the 25-language data table is the bulk
- **Blocks:** TICKET-04 (the walker reads extracted `FileGraph`s), TICKET-05 (the structural leg re-parses with gotreesitter)
- **Blocked by:** TICKET-01 (the write path)
- **Reversibility:** one-way (D4 — the extractor stamp algorithm; a change changes the stored value once, a one-time re-walk). ADR-0042 (gotreesitter) gates the dep.
- **Fails-when:** `TestExtractLadderRich6` does not emit the EXTRACTED + name-only edge pair, OR `TestExtractorStampIsContentHash` shows two configs producing the same stamp (a const, not a hash).

### TICKET-04 — Graph walker (impact + neighbors + `code_resolve` + `code_packages` + path + affected + `code_orient` map)
- **Tracker ID:** TASK-103  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The dialect-agnostic walker — `graph.Impact` (reverse-BFS, depth-cap 5, confidence filter, WILL BREAK / LIKELY AFFECTED, excluded counted, stored-edges-only, CI-INV-3); the 6 confidence-labeled neighbor views (callers/callees/dependents/implementors/hierarchy/tests_for) with the query-time name-only relabel (0→kept, 1→resolved, >1→AMBIGUOUS each); **`code_resolve`** (NEW, lifts v1 `resolve.go:35-76`; `{exists, name, qualified_name, file, line, sync_id, called_from[] (file paths), ambiguous, candidates[]}`); **`code_packages`** (NEW, the top-level Go package map); `graph.Path` (BFS or the graph-stops, the causal dispatch-kind fix T9); `graph.Affected` (the 3 input modes + the Java test-convention fix T8); **`graph.Orient` (T9b, NEW fold — the `ki outline` navigation map)**: `code_orient(symbol|package|file, depth?)` returns a bounded map (metadata + signature + file TOC + outbound edges with stable ids), `depth`-re-rootable, edge targets not expanded inline at depth 1 — the agent's "find a feature" entry verb; replaces v1's `code_orient`/`code_signature`/`code_file_toc` three tools (spec `004` §8).
- **Acceptance:** `go test ./internal/codeindex/graph/...` passes — `TestImpactBlastRadius` (B WILL BREAK, C LIKELY AFFECTED, unresolved D not in radius, excluded counted); `TestCodeResolveContract` (exists/file/line/called_from, ambiguous→candidates, gone→exists:false); `TestCodePackagesContract` (path/files/symbols, no sub-package unit); `TestPathGraphStops` (the causal dispatch kind); `TestAffectedTestsJava` (the v1 gap fixed); `TestNameOnlyQueryTimeRelabel` (AMBIGUOUS relabel); `TestOrientMapReRootsAndBounded` (T9b — orient on a package → TOC + outbound edges with ids; depth=1 does not expand targets; re-root on an edge-target id → that symbol's map).
- **SATISFIES:** impact-blast-radius, neighbors-confidence-labeled, code-resolve-contract, code-packages-contract, path-graph-stops, affected-tests-java, name-only-query-time-relabel, orient-map-navigable
- **Files:** `mnemonic-v2/internal/codeindex/graph/{walker,neighbors,resolve,path,packages,orient}.go` + tests
- **Size:** ~1100 (L)
- **Blocks:** TICKET-05 (the search's structural context), TICKET-07 (the transport calls the walker)
- **Blocked by:** TICKET-01 (the Store graph-read methods), TICKET-03 (the `FileGraph` shape)
- **Reversibility:** reversible (the walker is pure logic over the Store interface)
- **Fails-when:** `TestCodeResolveContract` returns a silent pick for an ambiguous name (CI-INV-3 — drop-not-guess broken), OR `TestAffectedTestsJava` misses `FooTest.java` (the v1 gap not fixed).

### TICKET-05 — Hybrid search (RRF K=60) + structural grep + `code_read` + T-CI-1/T-CI-2
- **Tracker ID:** TASK-104  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The hybrid search — RRF K=60 fusion of the FTS leg (the floor, identifier-first to fix the v1 FTS-leg miss), the structural leg (index-free by-example AST grep + the **fixed pattern language** T12: the node-type+capture form AND the `\f(ARGS)` shorthand, `degraded: no_grammar:<lang>` for an invalid pattern), and the semantic leg (the embedder seam, `degraded: no_vector`). Per-hit provenance. **The broken-leg-degrades-out (T-CI-2)** — a leg that errors degrades + drops out of the fusion (the v1 emitted a warning but the broken leg still shaped the output). `code_read` (the live source span + secret redaction T13). **Two `ki` folds:** the **`under?` subtree scope** (T10 — a `files.path` package prefix narrows all three legs to a subtree; the context-rot defense for large repos; empty-scope → empty not an error; recorded in the trail) and the **`code_read mode=content` drill handle** (T13 — a symbol in `content` mode returns the signature + callee/caller **ids**, not their bodies; a file returns the TOC, not the body; the `ki get --type content` progressive-disclosure read).
- **Acceptance:** `go test ./internal/codeindex/search/...` passes — `TestSearchRealVsFakeDivergence` (T-CI-1: the real symbol `wireGitHooks` is in the FTS leg; real ≠ fake top hit); `TestBrokenLegDegradesOutOfFusion` (T-CI-2: a failing semantic leg degrades + the FTS floor still answers); `TestStructuralGrepPatternLanguage` (both forms + the `no_grammar` degrade); `TestCodeReadSecretRedaction` (values masked, keys kept); `TestDegradedNoVector` (nil embedder → `no_vector`, FTS floor still answers); `TestSearchUnderScope` (T10 fold: a scoped search returns only hits under the prefix; an empty-scope search returns empty, not an error); `TestCodeReadContentMode` (T13 fold: a symbol in `content` mode returns signature + callee/caller ids but no bodies; a file returns the TOC).
- **SATISFIES:** search-hybrid-rrf, search-real-vs-fake-divergence, broken-leg-degrades-out, structural-grep-pattern-language, code-read-secret-redaction, degraded-no-vector, search-under-scope, code-read-content-mode
- **Files:** `mnemonic-v2/internal/codeindex/search/{hybrid,fts_leg,structural,semantic,read}.go` + tests
- **Size:** ~900 (M-L)
- **Blocks:** TICKET-07 (the transport's `code_search`)
- **Blocked by:** TICKET-02 (the FTS leg reads `chunks_fts`), TICKET-03 (the structural leg re-parses), TICKET-04 (the semantic context)
- **Reversibility:** reversible (the fusion is pure logic; the legs are swappable)
- **Fails-when:** `TestSearchRealVsFakeDivergence` returns the same top hit for a real and a fake symbol (T-CI-1 — the v1 relevance failure, not fixed), OR `TestBrokenLegDegradesOutOfFusion` shows the broken leg's empty result setting the answer (T-CI-2).

### TICKET-06 — Watch loop (fsnotify + debounce + burst-collapse + re-check + isolation)
- **Tracker ID:** TASK-105  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The fsnotify watch loop (ADR-0043) — debounce 2 s (clamped 100 ms–60 s), burst-collapse (a burst → one re-index), the metadata-only stat-walk fingerprint re-check (size+mtime vs `files` + the extractor stamp — a mtime-only change is a no-op), failure isolation (a re-index failure → log + continue, the watcher never stops), and **trigger-not-a-gate** (CI-INV-4 — a query during a re-index sees the last committed index + `degraded: stale_index`, never blocks).
- **Acceptance:** `go test ./internal/codeindex/watch/...` passes — `TestWatchBurstCollapse` (5 edits → 1 onSync); `TestWatchFingerprintRecheck` (mtime-only → no re-index); `TestWatchFailureIsolation` (a failure does not stop the watcher); `TestWatchTriggerNotGate` (a freshness check during a re-index returns immediately, no block).
- **SATISFIES:** watch-debounce-burst-collapse, watch-fingerprint-recheck, watch-failure-isolation, watch-trigger-not-gate
- **Files:** `mnemonic-v2/internal/codeindex/watch/watch.go` + `watch_test.go`
- **Size:** ~350 (M)
- **Blocks:** TICKET-07 (the `code_watch` admin tool + the `mnemonic watch` CLI)
- **Blocked by:** TICKET-01 (the `files` table + the extractor stamp for the re-check), TICKET-03 (`extract.Stamp()`)
- **Reversibility:** reversible (the watcher is a background goroutine; the `onSync` callback is a seam)
- **Fails-when:** `TestWatchFailureIsolation` shows the watcher stopping after a re-index failure, OR `TestWatchTriggerNotGate` shows a query blocking on a re-index (CI-INV-4 broken).

### TICKET-07 — `code_*` MCP transport + `code_status` 4-state + tier + version + CLI + T-CI-4
- **Tracker ID:** TASK-106  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The one shared `handler_code.go` handler set (the MCP + CLI meet here); the 16 agent-profile + the admin `code_*` tools on the memory server's **two-profile mechanism** (D-PROFILE — not v1's `SKILLGRID_MCP_CODE_TOOLS` env filter); the v2.1-deferred tools declared + reporting "run `code_index --<flag>`"; `code_index` as a **queued write** (async `run_id`, 409 on a repeat); `code_status` reporting the **4-state freshness** (`fresh`/`lag`/`empty`/`unknown`) + counts + `capabilities` + the **per-language extractor tier** (D-TIER) + the **schema version** + fair coverage (T-CI-4); the `mnemonic index/status/search/orient/explain/path/impact/affected/grep/read/watch` CLI group (the `--json` matches the MCP contract, byte-identical K5).
- **Acceptance:** `go test ./internal/transport/... ./cmd/...` passes — `TestCodeStatusReportsVersionTierCoverage` (T-CI-4: 4-state freshness `lag` not boolean, per-language tier, schema version, fair coverage); `TestCodeToolContract` (the 16 agent tools + the two NEW tools' exact shapes); `TestCodeProfileMembership` (all ⊇ agent; admin-only in all not agent); `TestCodeIndexQueuedWrite409`; `TestV21DeferredReportRunFlag`; `TestMcpCliParity` (byte-identical JSON).
- **SATISFIES:** code-tool-contract, code-profile-membership, code-status-reports-version-tier-coverage, code-index-queued-write-409, v21-deferred-report-run-flag, mcp-cli-parity
- **Files:** `mnemonic-v2/internal/transport/handler/handler_code.go` + `internal/transport/mcp/tools_code.go` + `cmd/mnemonic/index.go` + tests
- **Size:** ~1000 (L)
- **Blocks:** none (the final transport; TICKET-08's LSP flag surface is declared here)
- **Blocked by:** TICKET-04, TICKET-05, TICKET-06
- **Reversibility:** one-way (D1 — the `code_*` MCP tool contract; the KB + the opencode/pi plugins consume it). A rename / required param / return-shape change after the KB builds breaks them.
- **Fails-when:** `TestCodeStatusReportsVersionTierCoverage` reports a boolean `stale` (T-CI-4 — the v1 report failure, not fixed), OR `TestMcpCliParity` shows divergent JSON (K5 broken).

### TICKET-08 — Semantic vector store (D-CI8) + LSP tier + Postgres render
- **Tracker ID:** TASK-107  (milestone 2026-10-31-mnemonic-v2-code-index)
- **Scope:** The wired 3-index semantic store (D-CI8): BLOB (`embeddings`+`chunk_embeddings`) + vec0 + HNSW + the brute-force-over-BLOBs floor; the preference order **HNSW → vec0 → brute-force** (the v1 *live* path, lifted); the 768-dim dimension gating + the model-swap guard (clear vectors+indexes on `embedding_model` change); **LSH + IVF dropped** (not shipped — v1's `lsh_buckets` is schema-only, IVF is an adapter not wired). The hash-64 default embedder (the deterministic floor, ADR-0044 conditional on the 768-dim path). The optional **LSP tier** (`--lsp`, CI-6, off by default): the language-server subprocess (gopls/pyright/tsserver/rust-analyzer/clangd), per-call 30 s timeout, all-or-nothing per call site (a failing/timing-out server → warn + continue, the static name-only edge stands), the resolved member call → `LSP_RESOLVED` (0.95, the 4th confidence tier). The Postgres dialect render of the 7 tables (tsvector FTS + pgvector BLOB, `lib/pq` ADR-0041) — mirrors `sqlite.go`; the KB's `kb_*` tables ride the same render (T21).
- **Acceptance:** `go test ./internal/codeindex/vectorstore/... ./internal/codeindex/extract/ -run 'Semantic|Vector|ModelSwap|LSP'` passes — `TestSemanticStoreWired3Index` (768-dim → HNSW preferred; no LSH/IVF); `TestVectorPreferenceOrderFallback` (HNSW→vec0→brute-force, never a hard fail); `TestModelSwapGuard` (a swap clears vectors + indexes); `TestLSPAllOrNothingPerCallSite` (resolved → LSP_RESOLVED; failing → the name-only edge stands); `TestPostgresCodeDialectRender` (skips without `MNEMONIC_PG_TEST_DSN`, gate G-15).
- **SATISFIES:** semantic-store-wired-3-index, vector-preference-order, model-swap-guard, lsp-all-or-nothing-per-call-site, postgres-dialect-render (the `--lsp` flag it drives is declared + reported by TICKET-07's `v21-deferred-report-run-flag` scenario)
- **Files:** `mnemonic-v2/internal/codeindex/vectorstore/{store,dim,hash}.go` + `extract/lsp.go` + `internal/core/store/postgres_code.go` + `migrations_pg/100_code_index.sql` + tests
- **Size:** ~800 (M-L)
- **Blocks:** none (the final backstop)
- **Blocked by:** TICKET-05 (the semantic leg calls the vector store), TICKET-01 (the BLOB tables)
- **Reversibility:** reversible (the vector store is an engine behind the embedder seam; the LSP tier is opt-in; the PG render mirrors SQLite). ADR-0044 (conditional onnxer) gates the 768-dim path.
- **Fails-when:** `TestSemanticStoreWired3Index` returns LSH or IVF as the preferred index (D-CI8 — they were never wired in v1), OR `TestLSPAllOrNothingPerCallSite` shows a partial edge set after a failing server (all-or-nothing broken).

## Execution Notes

- **Waves (serial):** W1 (T1–T2) → W2 (T3) → W3 (T4) → W4 (T5) → W5 (T6) → W6 (T7) → W7 (T8). TICKET-03 and TICKET-04 share the W2→W3 boundary but 04's `FileGraph` dependency means 03 must land first. TICKET-06 (watch) can run in parallel with TICKET-05 (search) since both only need T1+T3; the wave order above is the default.
- **The four live-probe relevance truths are the acceptance backstops** (findings.md T-CI-1…4): T-CI-1→TICKET-05 (divergence), T-CI-2→TICKET-05 (broken leg), T-CI-3→TICKET-02 (migration parity), T-CI-4→TICKET-07 (code_status). These are the difference between "the tool exists" (v1, broken) and "the tool gives a good relevant answer" (v2).
- **The KB change's forward dependency (D-KB):** TICKET-02 (the FTS unit + the table-set parity loop) and TICKET-01 (the `CreateTable` seam) are the load-bearing surface the KB change builds on. Do not inline the FTS rendering back into `sqlite.go` (T21 — the KB re-uses `fts.go` verbatim) and do not hardcode the parity table list (D-KB — the KB extends the loop over `kb_*`).
- **One-way doors (flag for explicit user approval at execution):** D1 (the `code_*` contract, TICKET-07), D2 (the `Store` extension, TICKET-01), D3 (the migration numbering, TICKET-01/08), D4 (the extractor stamp, TICKET-03).
- **`ki`-validation folds (post-plan, findings.md "External validation"):** TICKET-04 adds `graph.Orient` (the `ki outline` navigation map, T9b); TICKET-05 adds the `under?` subtree scope (T10) + the `code_read mode=content` drill handle (T13). All three are additive (no one-way doors touched); the `--communities` cohesion metric (conductance, not edge density) + the pinned Leiden seed are fixed in the v2.1 spec `005` §5 now so the v2.1 pass is additive.
- **Spec-zone correction to make at T0 (before any code commit):** `006-search.md` §4/§4.2 still describes the **v1 four-index vector model** (vec0+IVF+HNSW+LSH, "default is ONNX") — that is the pre-D-CI8 spec. Amend `006` §4 to the settled **D-CI8** (3-index: BLOB+vec0+HNSW+brute-force; LSH+IVF dropped; hash-64 default, 768-dim ONNX opt-in) so the spec zone and the blueprint (TICKET-08) agree. Spec-zone changes commit before code-zone changes (AGENTS.md § Rules). See findings.md "Spec-zone inconsistency to correct".
- **10-source review folds (findings.md "10-source review", F-V2-1…F-V2-6):** the retrieval-as-reasoning contract + anti-pattern defense landed in spec `000` §1 / §1.1 (documentation + advisory termination rules — **no new tool, no new task, no one-way door touched**). The ablation (traversal = the biggest term, 12–14 F1) is now the external citation for the T9b/T5 navigation-first-class decisions. **Forward-look for the KB change (NOT this change):** a persistent, learning `kb_error_book` (discover→attribute→constrain→inject→verify&close, the arXiv LLM-Wiki Error Book) — the code-idiomatic *deterministic* half is already covered by `code_status` + CI-INV-1 + the KB's `kb_lint --fix`; only the *learning* half (accumulating constraints across KB ingest batches) is new and belongs to the KB. Flagged here so the KB slice doesn't lose it.
- **ADR gates:** ADR-0042 (gotreesitter) before TICKET-03; ADR-0043 (fsnotify) before TICKET-06; ADR-0044 (conditional onnxer) before the 768-dim path in TICKET-08; ADR-0045 (the `Store` extension) before TICKET-01.
- **Serial development:** the whole change sits behind the memory change. `state.yaml` `pipeline.current_change` is repointed to `2026-10-31-mnemonic-v2-code-index` at T0 (after the memory change ships).
- **QA gate (per skillgrid:qa):** L4 — full suite + build + coverage + go vet; mutation/security advisory (Trivy advisory-only). The four T-CI held-out tests (divergence, broken-leg, migration-parity, code_status report) are the relevance backstops the QA gate must see green.
- **Definition of Done (per AGENTS.md § Rules):** the `code_*` surface is on the two-profile MCP mechanism, the 7 tables are additive on both dialects, the 4 live-probe truths hold, the migration-parity regression covers a table-set the KB extends, the `Store` extension is additive, the extractor stamp is a real content hash, and every degradation is explicit (`degraded: <reason>`, never silent).
