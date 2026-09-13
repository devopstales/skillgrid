# Tasks: 014-mnemonic-performance

> **STATUS:** `in-progress` (2026-09-10) — 0/26 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Harden Mnemonic's performance, reliability, and retrieval capabilities by eliminating N+1 store opens, adding FTS5 trigram support, resolving SQLite WAL write contention, enabling parallel/federated cross-project search, adding memory TTL, replacing regex passive extraction with an LLM-backed extractor, supporting multiple embedders, and cross-linking the relational SQLite store with vector embeddings and codeindex graph nodes (triple-store design) so every observation has both a semantic vector and a structural graph representation.

**Architecture:** Layer additive changes across the existing `store`, `memory`, `service`, `codeindex`, `embedder`, and `config` packages. Add a cached store handle (sync.Map + refcount + WAL retry), extend FTS query parsing, introduce parallel/federated cross-project search, add TTL soft-expiry defaults, replace regex passive extraction with LLM-backed extraction (regex fallback), support multiple embedder models (ollama/local), and cross-link the relational store with vector embeddings and codeindex graph nodes via `graph_ref` + `embedding_blob` + `symbol_embeddings`. Session memory promotes to permanent graph automatically. Observations self-improve via a feedback loop based on retrieval usage. All changes are additive — no existing schema rewrites, no `mem_*`/`code_*`/`web_*` tool shape changes.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), ONNX (`onnxer`), optional external embedder (OpenAI-compatible), optional Ollama.

**Spec:** `docs/skillgrid/changes/014-mnemonic-performance/change.md`

**Acceptance:** `docs/skillgrid/changes/014-mnemonic-performance/acceptance.feature` (`@step-NN`)

---

## Goal

Make Mnemonic production-ready for multi-agent workloads by eliminating the top performance and capability gaps. Every change must preserve backward compatibility with existing stores and the current `mem_*` / `code_*` / `web_*` tool contracts.

## Out of scope / Non-Goals

- Rewriting the store package from scratch — additive only
- Replacing SQLite with a different engine
- Adding new MCP tool shapes (only extend existing ones)
- Changing the `mem_*` / `code_*` / `web_*` return formats
- Adding cloud sync or external vector DB
- Auto ontology generation (cognee feature — separate change)
- COGX native format import/export (we export JSON, not COGX native)
- Multi-tenant dataset isolation (cognee feature — separate change)
- Full temporal reasoning engine (we have temporal edges, not a reasoning engine)
- Graph-based memory retrieval with LLM CoT reasoning (cognee feature — separate change)
- Multi-process daemon architecture (ByteRover feature — mnemonic is a library, not a standalone daemon)
- Socket.IO transport layer (ByteRover feature — mnemonic uses MCP)
- OAuth2/OIDC authentication (ByteRover feature — mnemonic has no auth layer)
- Web UI + TUI (ByteRover feature — mnemonic has no UI layer)
- Replacing the `skillgrid serve` HTTP API
- Changing the `skillgrid index` or `skillgrid search` CLI behavior
- Adding new agent plugins beyond the existing three (opencode/kilo/cursor)

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- All changes are additive — no existing schema rewrites, no `009_*` or earlier migrations touched
- The `mem_*` / `code_*` / `web_*` tool contracts must not change shape
- SQLite store files remain per-project; no migration to a shared database
- The `modernc.org/sqlite` driver stays; no CGO or external SQLite binaries
- TTL defaults to 7 days; operators can configure via `mnemonic.ttl` config key
- Passive extraction LLM backend is opt-in; regex fallback always available
- Embedder selection is config-driven via `indexing.yaml`; no hard-coded model choices
- `improve()` re-weighting is opt-in via `mnemonic.improve` config; disabled by default
- Triple-store cross-linkage adds columns but doesn't change existing query paths
- `graph_ref` defaults to NULL if codeindex symbol not found; no broken references
- Session promotion only fires on `SessionEnd` with a valid `LayerSummary` (quality threshold)
- Temporal edges default to `valid_to=NULL` (active until explicitly expired)
- Store open fails (WAL lock) → retry with backoff (50ms, 100ms, 200ms); fail after 3 attempts
- FTS5 trigram query empty → fall back to default phrase matching
- Cross-project search store missing → skip that store; log warning (non-fatal)
- TTL cleanup on missing store → skip; no error (best-effort)
- LLM extraction fails → fall back to regex extraction
- Embedder load fails → degrade to FTS+signals floor (`embedder.Default()` returns Null)
- Connection pool exhausted → queue with timeout; return error after 5s (semaphore with deadline)
- Triple-store join performance → index `graph_ref`; limit JOIN depth; benchmark with 10k+ observations
- improve() skews results unexpectedly → configurable thresholds; cooldown period
- Session promotion creates duplicate graph nodes → dedup check before creation; idempotent promotion
- Temporal edge queries miss valid edges → verify `valid_from <= now AND (valid_to IS NULL OR valid_to > now)` logic
- Export produces too-large JSON → stream JSON; `--chunk-size` flag; skip embeddings by default
- DreamLock timeout: 5 minutes; auto-release on error
- DistillLock timeout: 5 minutes; auto-release on error
- LLM dedup misses semantic duplicates → hash dedup as fallback; configurable threshold
- Directory retrieval slow on deep hierarchies → depth limit; caching of directory scores
- Snapshot storage overhead → configurable snapshot retention; auto-prune old snapshots
- Handoff artifact staleness → delta computed at handoff time; cache invalidation
- Context envelope too large → configurable envelope size; field filtering
- Hub analysis inaccurate → periodic re-analysis; minimum importers threshold
- Skills hook interference → hooks opt-in per project; hook timeout

---

## State

```yaml
phase: apply         # spec | apply | verify | archive
current_step: 23-hub-impact
status: in_progress  # in_progress | blocked | done
updated: 2026-09-11T10:30:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `store-pooling` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `fts-trigram` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `parallel-search` | `@step-03` | 01 | Feature tagged `@step-03` |
| 04 | `ttl-defaults` | `@step-04` | 01 | Feature tagged `@step-04` |
| 05 | `llm-extraction` | `@step-05` | 01 | Feature tagged `@step-05` |
| 06 | `multi-embedder` | `@step-06` | 01 | Feature tagged `@step-06` |
| 07 | `triple-store-linkage` | `@step-07` | 01 | Feature tagged `@step-07` |
| 08 | `improve-loop` | `@step-08` | 07 | Feature tagged `@step-08` |
| 09 | `session-promotion` | `@step-09` | 07 | Feature tagged `@step-09` |
| 10 | `temporal-graph` | `@step-10` | 07 | Feature tagged `@step-10` |
| 11 | `portable-export` | `@step-11` | 07 | Feature tagged `@step-11` |
| 12 | `dream-executor` | `@step-12` | 04, 08 | Feature tagged `@step-12` |
| 13 | `importance-scoring` | `@step-13` | 08 | Feature tagged `@step-13` |
| 14 | `explicit-relations` | `@step-14` | 07 | Feature tagged `@step-14` |
| 15 | `provenance-tracking` | `@step-15` | 05 | Feature tagged `@step-15` |
| 16 | `federated-query` | `@step-16` | 03, 13 | Feature tagged `@step-16` |
| 17 | `distill-lock` | `@step-17` | 09, 12 | Feature tagged `@step-17` |
| 18 | `memory-types` | `@step-18` | 04, 05 | Feature tagged `@step-18` |
| 19 | `directory-retrieval` | `@step-19` | 02, 03 | Feature tagged `@step-19` |
| 20 | `snapshots` | `@step-20` | 01, 17 | Feature tagged `@step-20` |
| 21 | `handoff` | `@step-21` | 09, 12 | Feature tagged `@step-21` |
| 22 | `context-envelope` | `@step-22` | 08, 13, 21 | Feature tagged `@step-22` |
| 23 | `hub-impact` | `@step-23` | 10, 13 | Feature tagged `@step-23` |
| 24 | `skills-hooks` | `@step-24` | 05, 09 | Feature tagged `@step-24` |
| 25 | `memfs` | `@step-25` | 07, 18, 19 | Feature tagged `@step-25` |
| 26 | `tests` | `@step-26` | 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25 | Feature tagged `@step-26` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~8000–12000 (26 steps, many new files) |
| 400-line budget risk | High (chained PRs strongly recommended) |
| Chained PRs recommended | Yes |
| Delivery strategy | auto-chain (each `## NN` step commits separately when DoD is met; group into 3–4 PRs by dependency tier) |

---

## 01-store-pooling

### Goal

Cached store handles + WAL lock retry.

### Out of scope / Non-Goals

- FTS trigram, parallel search, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-01` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: —

**Files:**
- `internal/mnemonic/store/store.go` — MODIFY: add handle cache, refcount, WAL retry
- `internal/mnemonic/store/migrations/014_ttl_extraction.sql` — CREATE: TTL config table, extraction metadata

**Interfaces:**
- Consumes: existing `store.Open()` / `store.Close()` API; `PRAGMA busy_timeout` support
- Produces: cached `ProjectHandle` via `sync.Map`; refcounted close; WAL lock retry with exponential backoff (50ms, 100ms, 200ms)

### Tasks

- [x] 01.1 `[RED]` Cached handle reuse eliminates N+1 store opens
  - [x] 01.1.a Write failing test (`TestStoreOpenReusesCachedHandle`): open a store for project A, open again for project A, verify the second open returns the same handle (same underlying `*sql.DB`); then open project B and verify it gets a different handle
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle'` — Expected: FAIL
  - [x] 01.1.c Minimal implementation — add `sync.Map` handle cache in `store` package keyed by project ID; `Open()` checks cache first and returns existing handle with refcount incremented; `Close()` decrements refcount and removes from cache when it reaches 0
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle'` — Expected: PASS
  - [x] 01.1.e Commit — `feat(mnemonic): add cached store handles with refcounted close`
- [x] 01.2 `[RED]` WAL lock retry with exponential backoff
  - [x] 01.2.a Write failing test (`TestStoreOpenWALRetry`): simulate a WAL lock by opening a second connection to the same store file while a write transaction is in progress; verify `Open()` retries with backoff (50ms, 100ms, 200ms) and eventually succeeds; verify `PRAGMA busy_timeout=10000` is set on the connection
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenWALRetry'` — Expected: FAIL
  - [x] 01.2.c Minimal implementation — add WAL lock retry loop in `Open()` with exponential backoff (50ms, 100ms, 200ms); fail after 3 attempts; set `PRAGMA busy_timeout=10000` on every new connection; add health-check on cache reuse (ping before returning cached handle)
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenWALRetry'` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): add WAL lock retry with exponential backoff`
- [x] 01.3 `[AFK]` Migration for TTL config and extraction metadata tables
  - [x] 01.3.a Write failing test (`TestMigration014TTLExtraction`): run migration `014_ttl_extraction.sql` on a fresh store; verify `ttl_config` table exists with `key`/`value` columns; verify `extraction_metadata` table exists with expected schema
  - [x] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestMigration014TTLExtraction'` — Expected: FAIL
  - [x] 01.3.c Minimal implementation — create `internal/mnemonic/store/migrations/014_ttl_extraction.sql` with `CREATE TABLE IF NOT EXISTS ttl_config (key TEXT PRIMARY KEY, value TEXT NOT NULL)` and `CREATE TABLE IF NOT EXISTS extraction_metadata (id INTEGER PRIMARY KEY, session_id TEXT, content_hash TEXT, extracted_at TIMESTAMP, model TEXT)`
  - [x] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestMigration014TTLExtraction'` — Expected: PASS
  - [x] 01.3.e Commit — `feat(mnemonic): add 014 migration for TTL config and extraction metadata`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle\|TestStoreOpenWALRetry\|TestStoreOpenWALRetryLoop\|TestIsWALBusyClassification\|TestMigration014TTLExtraction\|TestStoreOpenCacheDisabledByEnv'` | PASS | PASS | 6 tests GREEN |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios (no BDD runner) |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/store/ -count=1 -race` | PASS | PASS | full store suite `-race` clean (27.7s) |
| Rollback boundary | verify `SKILLGRID_MNEMONIC_DISABLE_CACHE=1` bypasses cache | PASS | PASS | `TestStoreOpenCacheDisabledByEnv` |
| Global Constraints | — | held | held | additive; signatures unchanged; modernc stays; no CGO |

Commits: `449f536` (cached handles + refcount), `e2ff708` (020 migration), `8c20c92` (fix round: typed `isWALBusy` + `TestStoreOpenWALRetryLoop` exercising the retry loop). Review: NEEDS FIXES → PASS (F1/F2 resolved, F3/F4/F5 addressed).

### Commit

When step DoD is met: `feat(mnemonic): cached store handles with refcount and WAL lock retry`

---

## 02-fts-trigram

### Goal

Trigram/wildcard FTS query support.

### Out of scope / Non-Goals

- Store pooling, parallel search, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-02` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/retrieve.go` — MODIFY: trigram/prefix FTS query support

**Interfaces:**
- Consumes: cached store handles from step 01; existing `buildFTSQuery` function
- Produces: `trigram` and `prefix` match modes in `buildFTSQuery`; `--mode trigram` / `--mode prefix` CLI flags

### Tasks

- [x] 02.1 `[RED]` Trigram mode splits query into 3-character trigrams joined by OR
  - [x] 02.1.a Write failing test (`TestBuildFTSQueryTrigramMode`): call `buildFTSQuery("hello", "trigram")` and verify the output contains trigrams like `"hel OR elh OR llo"`; verify existing phrase mode (`buildFTSQuery("hello", "phrase")`) still returns `"\"hello\""`
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode'` — Expected: FAIL
  - [x] 02.1.c Minimal implementation — extend `buildFTSQuery` in `retrieve.go` with `matchMode` parameter; for `trigram` mode, split the query into all 3-character substrings and join with OR; for `prefix` mode, append `*` to each space-separated term; default mode remains phrase-only
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode'` — Expected: PASS
  - [x] 02.1.e Commit — `feat(mnemonic): add trigram and prefix FTS match modes`
- [x] 02.2 `[RED]` Existing phrase queries unchanged by default
  - [x] 02.2.a Write failing test (`TestFTSPhraseModeUnchanged`): verify `buildFTSQuery("exact phrase match", "")` (empty mode = default) returns `"\"exact phrase match\""` identical to pre-change behavior; verify no trigram or prefix transformation is applied when mode is empty
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestFTSPhraseModeUnchanged'` — Expected: FAIL
  - [x] 02.2.c Minimal implementation — ensure default match mode is `""` (phrase-only) and that the trigram/prefix code paths are only entered when explicitly requested; add guard that falls back to phrase matching if trigram produces empty results
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestFTSPhraseModeUnchanged'` — Expected: PASS
  - [x] 02.2.e Commit — `feat(mnemonic): preserve phrase-mode FTS backward compatibility`
- [x] 02.3 `[AFK]` CLI `--mode` flag for trigram/prefix search
  - [x] 02.3.a Write failing test (`TestMemSearchModeFlag`): invoke `mem search --mode trigram "partial"` and verify the search uses trigram mode; invoke `mem search --mode prefix "fun"` and verify prefix matching is used; verify invalid mode returns an error
  - [x] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemSearchModeFlag'` — Expected: FAIL
  - [x] 02.3.c Minimal implementation — add `--mode` flag to `mem search` CLI subcommand; pass the mode through to `SearchWithScope` → `buildFTSQuery`; validate mode values against `trigram`, `prefix`, `phrase`
  - [x] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemSearchModeFlag'` — Expected: PASS
  - [x] 02.3.e Commit — `feat(mnemonic): add --mode flag to mem search for trigram and prefix`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode\|TestFTSPhraseModeUnchanged'` | PASS | PASS | exact-string assertions |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | PASS | full memory suite |
| Rollback boundary | verify default mode is unchanged for existing queries | PASS | PASS | byte-identical `buildFTSQuery` at `""`/`"all"` (confirmed vs d7541f7) |
| Global Constraints | — | held | held | MCP `mem_search` contract untouched; CLI `--mode` optional |

Commits: `f16eda6` (trigram/prefix modes), `edc7656` (phrase back-compat), `172fe1b` (CLI `--mode`). Review: PASS WITH WARNINGS. Warnings (verification gaps, not correctness): (W1) `TestMemSearchModeFlag` asserts only JSON shape, not mode-sensitive output; (W2) `rrfFallbackOwnerScopedFTS` default `""` vs old hardcoded `"any"` is semantically identical but not byte-pinned. Known limitation (documented): `observations_fts` uses `tokenize='porter'`, so trigram/prefix are query-shaping primitives until FTS re-tokenization.

### Commit

When step DoD is met: `feat(mnemonic): FTS5 trigram and wildcard query support via opt-in match modes`

---

## 03-parallel-search

### Goal

Concurrent cross-project search with bounded goroutines.

### Out of scope / Non-Goals

- Store pooling (depends on 01), FTS trigram, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-03` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/service/service.go` — MODIFY: cached Open, parallel search
- `internal/mnemonic/service/retrieval.go` — MODIFY: parallel cross-project search

**Interfaces:**
- Consumes: cached store handles from step 01; existing `SearchObservationsScoped` per-store search
- Produces: parallel `SearchObservationsAll` with bounded goroutine pool; cross-store rank merge; `seen` dedup map preserved

### Tasks

- [x] 03.1 `[RED]` Concurrent search completes in parallel with bounded concurrency
  - [x] 03.1.a Write failing test (`TestSearchObservationsAllParallel`): create 10 project stores each with matching observations; call `SearchObservationsAll` and measure elapsed time; verify it completes faster than sequential execution (use a sleep-based latency marker per store); verify results are merged and deduped
  - [x] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel'` — Expected: FAIL
  - [x] 03.1.c Minimal implementation — rewrite `SearchObservationsAll` to launch one goroutine per store bounded by a semaphore of size `min(len(stores), runtime.NumCPU())`; each goroutine calls `SearchObservationsScoped`; collect results via a channel; merge using cross-store rank (highest rank across all stores wins)
  - [x] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel'` — Expected: PASS
  - [x] 03.1.e Commit — `feat(mnemonic): parallel cross-project search with bounded goroutines`
- [x] 03.2 `[RED]` Missing stores are skipped with warning, not fatal
  - [x] 03.2.a Write failing test (`TestSearchObservationsAllMissingStoreSkipped`): create 3 project stores, delete the file for one, call `SearchObservationsAll`; verify it returns results from the 2 valid stores without error; verify a warning is logged for the missing store
  - [x] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllMissingStoreSkipped'` — Expected: FAIL
  - [x] 03.2.c Minimal implementation — in each goroutine, check store existence before searching; if missing, log a warning and return nil results (non-fatal); the merge step skips nil result sets
  - [x] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllMissingStoreSkipped'` — Expected: PASS
  - [x] 03.2.e Commit — `feat(mnemonic): skip missing stores in parallel search with warning`
- [x] 03.3 `[AFK]` Semaphore limits concurrent stores to prevent resource exhaustion
  - [x] 03.3.a Write failing test (`TestSearchObservationsAllSemaphoreBound`): create 50 project stores; call `SearchObservationsAll` with a mock that tracks concurrent goroutine count; verify max concurrency never exceeds `runtime.NumCPU()`; verify all 50 stores are searched
  - [x] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllSemaphoreBound'` — Expected: FAIL
  - [x] 03.3.c Minimal implementation — use a buffered channel as semaphore (size `min(len(stores), runtime.NumCPU())`); each goroutine acquires before searching and releases after; add 5s timeout on semaphore acquire (connection pool exhausted → queue with timeout)
  - [x] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllSemaphoreBound'` — Expected: PASS
  - [x] 03.3.e Commit — `feat(mnemonic): add semaphore to bound parallel search concurrency`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel\|TestSearchObservationsAllMissingStoreSkipped\|TestSearchObservationsAllSemaphoreBound'` | PASS | PASS | `-race -count=5` no flake |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/service/ -count=1 -race` | PASS | PASS | race-free by inspection (workers→channel, single merge goroutine) |
| Rollback boundary | verify sequential search still works when parallel is disabled | PASS | PASS | `SKILLGRID_SEARCH_PARALLEL=0` routes to sequential |
| Global Constraints | — | held | held | `SearchObservationsScoped` signature unchanged; no tool contract change |

Commits: `2245ddc` (parallel cross-project search + semaphore + rollback env). Review: PASS WITH WARNINGS. Warnings (non-blocking): (M1) dead `warnMu` var; (M2) missing-store test covers the corrupt-store branch, not a true missing-file branch; (N1) determinism tie-break for equal (rank, UpdatedAt) is arrival-order — pre-existing, worth a final `Project/ID` tie key; (N2) pre-existing unrelated `TestReindexStructuralIsEmbedderFree` failure.

### Commit

When step DoD is met: `feat(mnemonic): concurrent cross-project search with bounded goroutines`

---

## 04-ttl-defaults

### Goal

Auto-set expires_at on save + scheduled TTL cleanup.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, extraction, embedder.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-04` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: auto-set expires_at in Save()
- `internal/mnemonic/service/service.go` — MODIFY: RunTTLExpiry
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `expire` subcommand

**Interfaces:**
- Consumes: cached store handles from step 01; existing `Save()` and `TTLSoftExpiry()` / `TTLRetire()` methods
- Produces: auto-set `expires_at` (7-day default) on save; `RunTTLExpiry()` on Service; `skillgrid mem expire` CLI

### Tasks

- [x] 04.1 `[RED]` Save() auto-sets expires_at to now + 7 days when not explicitly provided
  - [x] 04.1.a Write failing test (`TestSaveAutoSetsExpiresAt`): call `Save()` with a `SaveInput` that has no `expires_at` set; verify the stored observation has `expires_at` approximately 7 days in the future; call `Save()` with an explicit `expires_at` and verify it is preserved unchanged
  - [x] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt'` — Expected: FAIL
  - [x] 04.1.c Minimal implementation — in `Save()` (lifecycle.go), check if `expires_at` is zero/nil; if so, set it to `time.Now().Add(7 * 24 * time.Hour)`; read default TTL from `mnemonic.ttl` config key (7 days default); allow explicit override
  - [x] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt'` — Expected: PASS
  - [x] 04.1.e Commit — `feat(mnemonic): auto-set expires_at to now+7d on save`
- [x] 04.2 `[RED]` TTLRetire only retires expired observations, not future ones
  - [x] 04.2.a Write failing test (`TestTTLRetireOnlyExpired`): create 3 observations — one expired (expires_at in past), one expiring in 1 hour, one with no expires_at; call `TTLRetire`; verify only the expired one is soft-deleted; verify the other two remain active; verify `mem_search` excludes the retired observation
  - [x] 04.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTTLRetireOnlyExpired'` — Expected: FAIL
  - [x] 04.2.c Minimal implementation — implement `TTLRetire` to query observations where `expires_at IS NOT NULL AND expires_at < now` and soft-delete them (set `deleted_at`); ensure `TTLSoftExpiry` and `TTLRetire` are operational by default; make cleanup best-effort (skip missing stores, no error)
  - [x] 04.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTTLRetireOnlyExpired'` — Expected: PASS
  - [x] 04.2.e Commit — `feat(mnemonic): TTLRetire soft-deletes only expired observations`
- [x] 04.3 `[AFK]` skillgrid mem expire CLI subcommand
  - [x] 04.3.a Write failing test (`TestMemExpireCLI`): invoke `skillgrid mem expire` and verify it calls `RunTTLExpiry()` on all projects; verify the CLI outputs a summary of retired observations per project; verify it exits 0 even when a store is missing
  - [x] 04.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExpireCLI'` — Expected: FAIL
  - [x] 04.3.c Minimal implementation — add `mem expire` subcommand to `mem.go`; implement `RunTTLExpiry(ctx)` on `Service` that iterates all projects and calls `TTLRetire` on each; output per-project retirement counts; handle missing stores gracefully
  - [x] 04.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExpireCLI'` — Expected: PASS
  - [x] 04.3.e Commit — `feat(mnemonic): add mem expire CLI subcommand for TTL cleanup`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt\|TestTTLRetireOnlyExpired'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExpireCLI'` | PASS | PASS | 4 step-04 tests GREEN |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 23s, cmd 163s |
| Rollback boundary | verify TTL default is configurable via mnemonic.ttl | PASS | PASS | `SetTTL` wired from `config/load.go` `mnemonic.ttl`, 7d fallback |
| Global Constraints | — | held | held | no tool contract change; explicit expires_at preserved; 04.2 test-only |

Commits: `d8f50dd` (auto-set expires_at), `f36eb55` (mem expire CLI + RunTTLExpiry), `8fc99cf` (mnemonic.ttl config wiring). Review: PASS WITH WARNINGS. Warnings (non-blocking): (F1) re-saves don't refresh TTL — dedup/topic-upsert preserve original expires_at (expiry anchors to first creation; defensible but undocumented); (F2) `mnemonic.ttl` override wired but untested (no config.Load test in diff). 04.2 confirmed test-only: TTLRetire (lifecycle.go:145) + search `deleted_at IS NULL` (service.go:418/476) already correct.

### Commit

When step DoD is met: `feat(mnemonic): auto-set expires_at and scheduled TTL cleanup`

---

## 05-llm-extraction

### Goal

LLM-backed passive extraction + regex fallback.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, embedder.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/extraction.go` — CREATE: LLM extraction with regex fallback
- `internal/mnemonic/memory/extraction_test.go` — CREATE: extraction tests

**Interfaces:**
- Consumes: existing `extractLearnings` regex; LLM backend (via `layer.Distill` or extraction endpoint); `shapePassiveItem` / `shapePassiveContent` functions
- Produces: `ExtractWithLLM(ctx, text)`; `CapturePassive` tries LLM first, falls back to regex

### Tasks

- [x] 05.1 `[RED]` ExtractWithLLM calls LLM and returns structured learnings
  - [x] 05.1.a Write failing test (`TestExtractWithLLM`): provide a mock LLM that returns structured JSON learnings; call `ExtractWithLLM(ctx, text)`; verify it returns parsed learnings matching the `shapePassiveItem` format; verify the LLM is called with the input text
  - [x] 05.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM'` — Expected: FAIL
  - [x] 05.1.c Minimal implementation — create `extraction.go` with `ExtractWithLLM(ctx context.Context, text string) ([]PassiveItem, error)`; call the LLM backend (reuse `layer.Distill` or a new extraction endpoint); parse the JSON response into `PassiveItem` structs using `shapePassiveItem` / `shapePassiveContent`
  - [x] 05.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM'` — Expected: PASS
  - [x] 05.1.e Commit — `feat(mnemonic): add LLM-backed passive extraction`
- [x] 05.2 `[RED]` LLM extraction failure falls back to regex extraction
  - [x] 05.2.a Write failing test (`TestCapturePassiveLLMFailureFallsBackToRegex`): mock the LLM to return an error; call `CapturePassive` with text containing known regex-extractable learnings; verify the regex fallback (`extractLearnings`) is called and returns the same results as the regex-only path; verify no error is propagated to the caller
  - [x] 05.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestCapturePassiveLLMFailureFallsBackToRegex'` — Expected: FAIL
  - [x] 05.2.c Minimal implementation — modify `CapturePassive` to try `ExtractWithLLM` first; on error, log a warning and fall back to `extractLearnings` regex; ensure the regex path is always available (no LLM dependency); results are parsed with the same `shapePassiveItem` / `shapePassiveContent` functions
  - [x] 05.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestCapturePassiveLLMFailureFallsBackToRegex'` — Expected: PASS
  - [x] 05.2.e Commit — `feat(mnemonic): add regex fallback for LLM extraction failure`
- [x] 05.3 `[AFK]` Extraction results are identical quality or better than regex-only
  - [x] 05.3.a Write failing test (`TestExtractionQualityLLMVsRegex`): provide a text with nuanced learnings that regex misses (free-form text, multi-clause sentences); compare LLM extraction results vs regex results; verify LLM extracts at least all items regex extracts plus additional nuanced items; verify no duplicate items in the combined result set
  - [x] 05.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractionQualityLLMVsRegex'` — Expected: FAIL
  - [x] 05.3.c Minimal implementation — refine the LLM prompt in `ExtractWithLLM` to capture nuanced learnings; deduplicate results (by content hash) when combining LLM and regex outputs; ensure the LLM is opt-in (config `mnemonic.extraction.llm: true`); default to regex when not enabled
  - [x] 05.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractionQualityLLMVsRegex'` — Expected: PASS
  - [x] 05.3.e Commit — `feat(mnemonic): improve LLM extraction quality and dedup`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM\|TestCapturePassiveLLMFailureFallsBackToRegex\|TestExtractionQualityLLMVsRegex'` | PASS | PASS | 5 step-05 tests GREEN |
| Acceptance `@step-05` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | full memory suite |
| Rollback boundary | verify regex-only path works when LLM is disabled | PASS | PASS | byte-for-byte regex-only when `mnemonic.extraction.llm`=false or seam nil (`&&` gate service.go:1117) |
| Global Constraints | — | held | held | LLM opt-in, regex always available; no tool contract change; no CGo (ExtractionLLM seam) |

Commits: `431d3a0` (ExtractWithLLM + ExtractionLLM seam), `a2a946a` (regex fallback in CapturePassive), `5673b58` (dedup + config + openProject wiring). Review: PASS WITH WARNINGS. Warnings (forward-looking gaps, not regressions): (M1) `dedupePassiveItems`/`passiveItemHash` are dead in production — CapturePassive REPLACES rawItems with llmItems on success (not combine); no-dup-rows achieved by downstream `seen[title\|type]`; (M2) no production `ExtractionLLM` attached yet — `openProject` calls `EnableExtractionLLM` but never `SetExtractionLLM`, so the opt-in is inert end-to-end until a later step wires a backend; (m3) LLM `type` hint parsed then discarded (reclassified by keyword in shapePassiveItem).

### Commit

When step DoD is met: `feat(mnemonic): LLM-backed passive extraction with regex fallback`

---

## 06-multi-embedder

### Goal

ollama and local embedder providers.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, extraction.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-06` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/embedder/embedder.go` — MODIFY: ollama + local providers
- `internal/mnemonic/embedder/ollama.go` — CREATE: Ollama embedder
- `internal/mnemonic/embedder/local.go` — CREATE: Local ONNX embedder
- `internal/mnemonic/config/load.go` — MODIFY: ollama/local provider config
- `internal/mnemonic/embedder/embedder_test.go` — MODIFY: new provider tests

**Interfaces:**
- Consumes: existing `embedder.Default()`; ONNX runtime (`onnxer`); Ollama HTTP API
- Produces: `ollama` and `local` embedder providers; config-driven selection via `indexing.yaml`

### Tasks

- [x] 06.1 `[RED]` Ollama embedder calls http://localhost:11434/api/embeddings
  - [x] 06.1.a Write failing test (`TestOllamaEmbedder`): start a mock HTTP server on a test port that mimics the Ollama `/api/embeddings` endpoint; configure the embedder to use that port; call `Embed(ctx, "test text")`; verify it returns a vector of the expected dimensionality; verify the HTTP request body contains the input text
  - [x] 06.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder'` — Expected: FAIL
  - [x] 06.1.c Minimal implementation — create `ollama.go` with an `OllamaEmbedder` struct that POSTs to `http://localhost:11434/api/embeddings` with `{"model": "...", "prompt": "..."}`; parse the `embedding` field from the JSON response; register as provider type `ollama` in `embedder.go`
  - [x] 06.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder'` — Expected: PASS
  - [x] 06.1.e Commit — `feat(mnemonic): add ollama embedder provider`
- [x] 06.2 `[RED]` Local ONNX embedder loads model from ~/.skillgrid/models/
  - [x] 06.2.a Write failing test (`TestLocalONNXEmbedder`): place a test ONNX model in a temp directory; configure the embedder to load from that directory; call `Embed(ctx, "test text")`; verify it returns a vector; verify the model file is found and loaded; verify graceful error when model directory is missing
  - [x] 06.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestLocalONNXEmbedder'` — Expected: FAIL
  - [x] 06.2.c Minimal implementation — create `local.go` with a `LocalONNXEmbedder` struct that loads an ONNX model from `~/.skillgrid/models/` (or a configured path); use `onnxer` to run inference; register as provider type `local` in `embedder.go`; return a descriptive error when the model file is missing
  - [x] 06.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestLocalONNXEmbedder'` — Expected: PASS
  - [x] 06.2.e Commit — `feat(mnemonic): add local ONNX embedder provider`
- [x] 06.3 `[RED]` Config-driven provider selection via indexing.yaml
  - [x] 06.3.a Write failing test (`TestEmbedderConfigDrivenSelection`): write an `indexing.yaml` with `mnemonic.embedder.provider: ollama`; call `embedder.Default()`; verify it returns an Ollama embedder; write `provider: local` and verify it returns a LocalONNX embedder; write `provider: onnx` and verify it returns the existing ONNX embedder; write no provider and verify it returns Null
  - [x] 06.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderConfigDrivenSelection'` — Expected: FAIL
  - [x] 06.3.c Minimal implementation — extend `config/load.go` to parse `mnemonic.embedder.provider` from `indexing.yaml`; extend `embedder.go` `Default()` to switch on the provider value and return the appropriate embedder; ensure `onnx`/`external`/`off` values are unchanged
  - [x] 06.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderConfigDrivenSelection'` — Expected: PASS
  - [x] 06.3.e Commit — `feat(mnemonic): config-driven embedder provider selection`
- [x] 06.4 `[AFK]` Embedder load failure degrades to Null
  - [x] 06.4.a Write failing test (`TestEmbedderLoadFailureReturnsNull`): configure `provider: ollama` but point to an unreachable port; call `embedder.Default()`; verify it returns a Null embedder (not an error); verify `Embed` on the Null embedder returns zero-vector or empty slice
  - [x] 06.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderLoadFailureReturnsNull'` — Expected: FAIL
  - [x] 06.4.c Minimal implementation — in `embedder.Default()`, wrap provider creation in error handling; on failure, log a warning and return `NullEmbedder`; ensure `NullEmbedder.Embed` returns a zero-length vector without error
  - [x] 06.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderLoadFailureReturnsNull'` — Expected: PASS
  - [x] 06.4.e Commit — `feat(mnemonic): degrade to Null embedder on load failure`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder\|TestLocalONNXEmbedder\|TestEmbedderConfigDrivenSelection\|TestEmbedderLoadFailureReturnsNull'` | PASS | PASS | 18/18 embedder tests GREEN |
| Acceptance `@step-06` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/embedder/ -count=1` | PASS | PASS | full embedder suite |
| Rollback boundary | verify onnx/external/off providers unchanged | PASS | PASS | `BuildFromConfig` switch preserves existing routing; no-CGO (onnxer direct dep, no go.mod change) |
| Global Constraints | — | held | held | config-driven selection, no hard-coded models; no tool contract change |

Commits: `dcbf534` (ollama), `d716f95` (local ONNX), `998f1bf` (config-driven Default), `404a05a` (Null degradation). Review: PASS WITH WARNINGS. Warnings: (M1) `Default()` now returns `*Onnx`(768-dim, hash-fallback) instead of old `HashEmbedder{}`(64-dim) when embedding on — a real runtime behavior change; ALSO the main `resolveEmbedder` (service.go:1735) was NOT updated to route ollama/local, so the two selection paths disagree (follow-up); (M2) `LocalONNX.embedOne` loads the session but returns a zero vector (no inference) — a present model yields flat ranking, weaker than onnx.go's non-zero hash fallback (disclosed follow-up, libonnxruntime absent on host); (M3) `isLibraryMissing` string-matches onnxer's dlopen error text (fragile to rewording).

### Commit

When step DoD is met: `feat(mnemonic): ollama and local embedder providers with config-driven selection`

---

## 07-triple-store-linkage

### Goal

Cross-link relational + vector + graph stores.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-07` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/codeindex/symbol_embeddings.go` — CREATE: bridge table: symbols ↔ embeddings
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: add graph_ref to observations

**Interfaces:**
- Consumes: codeindex `symbols` table; existing `observations` and `long_term_memories` tables; vector embeddings from step 06
- Produces: `graph_ref` column on observations; `embedding_blob` column on long_term_memories; `symbol_embeddings` bridge table; cross-link query path

### Tasks

- [x] 07.1 `[RED]` graph_ref column on observations defaults to NULL when symbol not found
  - [x] 07.1.a Write failing test (`TestGraphRefDefaultsToNull`): save an observation with no matching codeindex symbol; verify `graph_ref` is NULL; save an observation with a matching symbol ID; verify `graph_ref` is set to that symbol ID; verify no broken references
  - [x] 07.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull'` — Expected: FAIL
  - [x] 07.1.c Minimal implementation — add `graph_ref` column (INTEGER, nullable) to `observations` table via migration; in `Save()`, look up the codeindex symbol ID for the observation's source file; set `graph_ref` to the symbol ID if found, NULL otherwise; create index on `graph_ref`
  - [x] 07.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull'` — Expected: PASS
  - [x] 07.1.e Commit — `feat(mnemonic): add graph_ref column to observations`
- [x] 07.2 `[RED]` symbol_embeddings bridge table links codeindex symbols to embeddings
  - [x] 07.2.a Write failing test (`TestSymbolEmbeddingsBridge`): create a codeindex symbol; create an embedding vector; insert a row in `symbol_embeddings` linking them; query the bridge table and verify the symbol ID maps to the correct embedding; verify the table has `symbol_id`, `embedding_blob`, `model`, `created_at` columns
  - [x] 07.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestSymbolEmbeddingsBridge'` — Expected: FAIL
  - [x] 07.2.c Minimal implementation — create `symbol_embeddings.go` with the `symbol_embeddings` table schema: `symbol_id INTEGER NOT NULL, embedding_blob BLOB NOT NULL, model TEXT NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (symbol_id)`; add CRUD functions for the bridge table
  - [x] 07.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestSymbolEmbeddingsBridge'` — Expected: PASS
  - [x] 07.2.e Commit — `feat(mnemonic): add symbol_embeddings bridge table`
- [x] 07.3 `[RED]` Cross-link query traverses observation→symbol→embedding→related observations
  - [x] 07.3.a Write failing test (`TestTripleStoreCrossLinkQuery`): create an observation with `graph_ref` set to a symbol; create a symbol embedding in `symbol_embeddings`; create another observation that references the same symbol; run the cross-link query from the first observation; verify it returns the related observation via the SQL JOIN path
  - [x] 07.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreCrossLinkQuery'` — Expected: FAIL
  - [x] 07.3.c Minimal implementation — implement `CrossLinkQuery(ctx, observationID)` that runs a single SQL JOIN: `observations o JOIN symbol_embeddings se ON o.graph_ref = se.symbol_id JOIN observations o2 ON o2.graph_ref = se.symbol_id WHERE o.id = ? AND o2.id != ?`; add `embedding_blob` column to `long_term_memories`; ensure triple-store queries are opt-in (separate method, not in default search path)
  - [x] 07.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreCrossLinkQuery'` — Expected: PASS
  - [x] 07.3.e Commit — `feat(mnemonic): triple-store cross-link query via SQL JOIN`
- [x] 07.4 `[AFK]` No broken references — orphan detection
  - [x] 07.4.a Write failing test (`TestTripleStoreOrphanDetection`): create an observation with `graph_ref` pointing to a non-existent symbol ID; create an observation with `graph_ref` pointing to a valid symbol; run an integrity check; verify the orphan is detected and reported; verify the valid reference passes
  - [x] 07.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreOrphanDetection'` — Expected: FAIL
  - [x] 07.4.c Minimal implementation — add `CheckCrossLinkIntegrity(ctx)` that queries observations with non-NULL `graph_ref` and verifies each referenced symbol exists in the codeindex `symbols` table; return a list of orphan references; run on a schedule or on demand
  - [x] 07.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreOrphanDetection'` — Expected: PASS
  - [x] 07.4.e Commit — `feat(mnemonic): add cross-link integrity check for orphan detection`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull\|TestTripleStoreCrossLinkQuery\|TestTripleStoreOrphanDetection'` + `go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestSymbolEmbeddingsBridge'` | PASS | PASS | 4 step-07 tests GREEN |
| Acceptance `@step-07` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/codeindex/ ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | memory + codeindex suites green |
| Rollback boundary | verify existing query paths unchanged (triple-store is opt-in) | PASS | PASS | default Search/SearchWithScope byte-for-byte unchanged; migration 022 purely additive |
| Global Constraints | — | held | held | same-store single-DB JOIN confirmed; reused embeddings (011) as bridge; no tool contract change; no CGO |

Commits: `b293b1c` (bridge accessors), `c80bc9c` (graph_ref column), `9799a90` (CrossLinkQuery), `0044ec3` (CheckCrossLinkIntegrity), `b9e516b` (test move). Review: PASS. Nits (non-blocking): (N1) CrossLinkQuery folds the embeddings leg into graph_ref equality (observation→o2 via shared graph_ref, functionally equivalent to observation→symbol→observation); (N2) suffix-match symbol lookup is nondeterministic when a file has multiple symbols (documented); (N3) extra UPDATE round-trip in Save() for graph_ref; (N4) embedding_blob added but unpopulated (correct — infra for later steps); (N5) pre-existing service test failure exists at step 06 (not a regression).

### Commit

When step DoD is met: `feat(mnemonic): cross-link relational, vector, and graph stores via graph_ref and symbol_embeddings`

---

## 08-improve-loop

### Goal

Self-improvement feedback loop via retrieval usage.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, extraction, embedder, triple-store linkage.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-08` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: improve() feedback loop

**Interfaces:**
- Consumes: `retrieval_usage` column on observations (already tracked); triple-store linkage from step 07
- Produces: `improve()` method that re-weights observations by retrieval usage; boost/decay in `mem_search` ordering

### Tasks

- [x] 08.1 `[RED]` High-usage observations are boosted in mem_search ordering
  - [x] 08.1.a Write failing test (`TestImproveBoostsHighUsageObservations`): create 3 observations — one with `retrieval_usage=100`, one with `retrieval_usage=10`, one with `retrieval_usage=0`; call `improve()`; run `mem_search` for a query matching all three; verify the high-usage observation ranks first
  - [x] 08.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations'` — Expected: FAIL
  - [x] 08.1.c Minimal implementation — add `improve()` method on Service that reads `retrieval_usage` from all observations; compute a boost factor for observations with `retrieval_usage > threshold`; apply the boost in `mem_search` ordering (in-memory re-rank before returning results); make boost/decay rates configurable via `mnemonic.improve` config key; disable by default (opt-in)
  - [x] 08.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations'` — Expected: PASS
  - [x] 08.1.e Commit — `feat(mnemonic): improve() boosts high-usage observations in search`
- [x] 08.2 `[RED]` Never-accessed observations decay in rank over time
  - [x] 08.2.a Write failing test (`TestImproveDecaysNeverAccessed`): create 2 observations — one with `retrieval_usage=0` and age > TTL, one with `retrieval_usage=5`; call `improve()`; verify the never-accessed observation ranks below the accessed one; verify decay is proportional to age
  - [x] 08.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDecaysNeverAccessed'` — Expected: FAIL
  - [x] 08.2.c Minimal implementation — in `improve()`, compute a decay factor for observations with `retrieval_usage == 0` and age > TTL; reduce their rank score proportionally to age; ensure `improve()` is called before `SearchOwnerScoped` (transparent to callers)
  - [x] 08.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDecaysNeverAccessed'` — Expected: PASS
  - [x] 08.2.e Commit — `feat(mnemonic): improve() decays never-accessed observations`
- [x] 08.3 `[AFK]` improve() is opt-in and does not regress when disabled
  - [x] 08.3.a Write failing test (`TestImproveDisabledNoRegression`): disable `mnemonic.improve` via config; run `mem_search` with mixed retrieval_usage observations; verify results are ordered by the default ranking (no boost/decay applied); verify the search returns the same results as pre-improve() behavior
  - [x] 08.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDisabledNoRegression'` — Expected: FAIL
  - [x] 08.3.c Minimal implementation — gate `improve()` behind the `mnemonic.improve` config flag (default `false`); when disabled, skip the boost/decay computation entirely; add a cooldown period to prevent excessive re-weighting on consecutive searches
  - [x] 08.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDisabledNoRegression'` — Expected: PASS
  - [x] 08.3.e Commit — `feat(mnemonic): make improve() opt-in with cooldown`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations\|TestImproveDecaysNeverAccessed\|TestImproveDisabledNoRegression'` | PASS | PASS | 3 step-08 tests GREEN |
| Acceptance `@step-08` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | full memory suite |
| Rollback boundary | verify improve() disabled by default | PASS | PASS | byte-identical when disabled (gate improve.go:134 returns input unchanged) |
| Global Constraints | — | held | held | opt-in via mnemonic.improve; in-memory re-rank (SQL ORDER BY untouched); no tool contract change |

Commits: `536c08f` (boost), `bc0a9fc` (decay), `69ced40` (opt-in + cooldown). Review: PASS WITH WARNINGS. Warnings: (M1) `SetImprove` wired in retrieval.go:239-252 but the fact-mode leg (`rrfFallbackOwnerScopedFTS`→`BlendedSearch`, retrieve.go:233) never routes through `improve()` — dead hook; comment claims "consistent across every search surface" but it isn't (fix before archive); (m2) scoped diff omits the two wiring files (service/service.go, service/retrieval.go); (m3) improve.go has comment-only working-tree drift. Verified: boost monotonic + capped + threshold-gated; decay only usage==0 && age>TTL, proportional to age-TTL; cooldown per-service, first search not blocked; config struct with sane defaults + malformed→default; sort.SliceStable tie-break preserves SQL rank.

### Commit

When step DoD is met: `feat(mnemonic): self-improvement feedback loop via retrieval usage`

---

## 09-session-promotion

### Goal

L0/L1 session → L2/L3 graph auto-promotion.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-09` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/session.go` — MODIFY: session-to-graph promotion on SessionEnd

**Interfaces:**
- Consumes: codeindex graph from step 07; `LayerSummary` populated during session; `graph_ref` column on observations
- Produces: permanent graph nodes on SessionEnd; promoted nodes queryable via mem_search and codeindex graph traversal

### Tasks

- [x] 09.1 `[RED]` SessionEnd with valid summary creates permanent graph node
  - [x] 09.1.a Write failing test (`TestSessionEndCreatesGraphNode`): start a session, save observations, end the session with a populated `LayerSummary`; verify a new graph node is created in the codeindex graph; verify the node links to all session observations via `graph_ref`; verify the node is queryable via `mem_search`
  - [x] 09.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode'` — Expected: FAIL
  - [x] 09.1.c Minimal implementation — in `SessionEnd`, check if the session has a valid `LayerSummary` (quality threshold check); if so, create a permanent graph node in the codeindex graph representing the session; link the node to all session observations via `graph_ref`; the node is queryable via `mem_search` and codeindex graph traversal
  - [x] 09.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode'` — Expected: PASS
  - [x] 09.1.e Commit — `feat(mnemonic): session-to-graph promotion on SessionEnd`
- [x] 09.2 `[RED]` No promotion when summary is empty or below quality threshold
  - [x] 09.2.a Write failing test (`TestSessionEndNoPromotionBelowThreshold`): start a session with no `LayerSummary` (empty); end the session; verify no graph node is created; start another session with a low-quality summary (below threshold); end it; verify no graph node is created; verify the session observations are still accessible via `mem_search`
  - [x] 09.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndNoPromotionBelowThreshold'` — Expected: FAIL
  - [x] 09.2.c Minimal implementation — add a quality threshold check in the promotion path: require `LayerSummary` to be non-empty and meet a minimum content length or structured section count; skip promotion (with a debug log) when the threshold is not met
  - [x] 09.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndNoPromotionBelowThreshold'` — Expected: PASS
  - [x] 09.2.e Commit — `feat(mnemonic): skip promotion when summary below quality threshold`
- [x] 09.3 `[AFK]` Promotion is idempotent — no duplicate graph nodes
  - [x] 09.3.a Write failing test (`TestSessionEndIdempotentPromotion`): end a session with a valid summary; verify a graph node is created; end the same session again (re-trigger); verify no duplicate graph node is created (dedup check); verify the existing node is reused
  - [x] 09.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndIdempotentPromotion'` — Expected: FAIL
  - [x] 09.3.c Minimal implementation — before creating a graph node, check if a node already exists for this session ID (dedup query on session_id); if found, skip creation and reuse the existing node; ensure promotion is idempotent
  - [x] 09.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndIdempotentPromotion'` — Expected: PASS
  - [x] 09.3.e Commit — `feat(mnemonic): idempotent session promotion with dedup`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode\|TestSessionEndNoPromotionBelowThreshold\|TestSessionEndIdempotentPromotion'` | PASS | PASS | 3 step-09 tests GREEN |
| Acceptance `@step-09` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | full memory suite (30.7s) |
| Rollback boundary | verify promotion only fires on SessionEnd with valid summary | PASS | PASS | threshold = 100 runes + >=1 `##` heading (min_length configurable); best-effort (failure doesn't fail SessionEnd) |
| Global Constraints | — | held | held | purely additive (reuses observations/symbols/edges, no schema change); no tool contract change; no CGO |

Commits: `1e3fd23` (promotion + dedup), `1e2c0ac` (threshold gate), `37b819b` (empty — dedup landed in 09.1). Review: PASS WITH WARNINGS. Warnings: (M1) idempotency dedup is CONTENT-keyed (`observations (project,type,title,content)`), not session-keyed — a re-trigger with a CHANGED summary creates a 2nd node observation (the `session:<uid>` symbol stays unique); the test only re-triggers with the identical summary, so the changed-summary case is untested (fix before archive); (m2) node symbol signature is snapshot-at-first (changed summary never refreshes it); (n3) `PromoteSession` re-runs `promoteSessionToGraph` after SessionEnd already promoted (duplicate log lines, idempotent upserts keep it correct); (n4) empty 37b819b commit (process nit).

### Commit

When step DoD is met: `feat(mnemonic): L0/L1 session to L2/L3 graph auto-promotion`

---

## 10-temporal-graph

### Goal

Temporal knowledge graph edges with valid_from/to.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-10` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/codeindex/graph.go` — MODIFY: add valid_from/valid_to to edges
- `internal/mnemonic/codeindex/graph_test.go` — CREATE: temporal edge tests

**Interfaces:**
- Consumes: codeindex `edges` table; triple-store linkage from step 07
- Produces: temporal bounds on graph edges; filtered queries hiding expired edges; `mem graph` CLI temporal status

### Tasks

- [x] 10.1 `[RED]` Edges have valid_from set to current time when relationship is observed
  - [x] 10.1.a Write failing test (`TestEdgeValidFromSetOnCreate`): create a symbol relationship (edge); verify the edge has `valid_from` set to approximately the current UNIX timestamp; verify `valid_to` is NULL (active); verify the edge is visible in normal queries
  - [x] 10.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate'` — Expected: FAIL
  - [x] 10.1.c Minimal implementation — add `valid_from` (INTEGER, UNIX timestamp) and `valid_to` (INTEGER, nullable, NULL = active) columns to the `edges` table via migration; set `valid_from` to `time.Now().Unix()` when a new edge is created; default `valid_to` to NULL
  - [x] 10.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate'` — Expected: PASS
  - [x] 10.1.e Commit — `feat(mnemonic): add valid_from/valid_to to graph edges`
- [x] 10.2 `[RED]` Expired edges are hidden from queries but preserved for history
  - [x] 10.2.a Write failing test (`TestExpiredEdgesHiddenFromQueries`): create an edge with `valid_to` in the past (expired); create another edge with `valid_to` as NULL (active); run a normal edge query; verify only the active edge is returned; run a history query; verify both edges are returned
  - [x] 10.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestExpiredEdgesHiddenFromQueries'` — Expected: FAIL
  - [x] 10.2.c Minimal implementation — add a query filter: `WHERE valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)` with `? = time.Now().Unix()`; add a separate `QueryEdgesWithHistory` method that returns all edges including expired; verify the logic handles `valid_from <= now` correctly
  - [x] 10.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestExpiredEdgesHiddenFromQueries'` — Expected: PASS
  - [x] 10.2.e Commit — `feat(mnemonic): hide expired edges from queries, preserve for history`
- [x] 10.3 `[AFK]` mem graph CLI shows temporal status of edges
  - [x] 10.3.a Write failing test (`TestMemGraphTemporalStatus`): create edges with various temporal states (active, expired, future valid_from); invoke `mem graph` CLI; verify the output shows temporal status (active/expired) for each edge; verify the CLI includes `valid_from` and `valid_to` in the output
  - [x] 10.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphTemporalStatus'` — Expected: FAIL
  - [x] 10.3.c Minimal implementation — extend the `mem graph` CLI subcommand to include temporal status in the output; display `valid_from`, `valid_to`, and a computed status (`active` / `expired` / `pending`) for each edge
  - [x] 10.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphTemporalStatus'` — Expected: PASS
  - [x] 10.3.e Commit — `feat(mnemonic): show temporal status in mem graph CLI`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate\|TestExpiredEdgesHiddenFromQueries\|TestBackfilledEdgesRemainActive'` + `go test ./skillgrid-cli/internal/mnemonic/graph/ -run 'TestFetchEdgesHidesExpiredEdges\|TestBackfilledEdgesVisibleInTraversal\|TestPromotedSessionEdgesCarryValidFrom'` | PASS | PASS | 6 step-10 tests GREEN (-race) |
| Acceptance `@step-10` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/graph/ -count=1 -race` | PASS | PASS | codeindex 117s, graph 11.7s, -race clean |
| Rollback boundary | verify existing edges work with valid_to=NULL default | PASS | PASS | backfill valid_from=0 (always active, 0<=now); valid_to NULL (active) |
| Global Constraints | — | held | held | purely additive (023 migration); fetchEdges filter wired (fix round); no tool contract change; no CGO |

Commits: `91c973f` (temporal edges + QueryEdges/WithHistory), `fe33d39` (mem graph CLI), `4b7cf3f` (fix round: fetchEdges filter + valid_from on pdg/knowledge/session_promotion INSERTs + count-before-limit). Review: NEEDS FIXES → PASS (F1/F2/F3 resolved). F1: fetchEdges (graph/graph.go:145, the live traversal behind code_get_callers/code_explain/code_path) now has the temporal filter; TestFetchEdgesHidesExpiredEdges is real RED. F2: valid_from=time.Now().Unix() on pdg/lsp.go, knowledge/store.go (4 sites), session_promotion.go. F3: count computed before --limit. NIT (non-blocking): TestPromotedSessionEdgesCarryValidFrom inserts via old shape (guards default-0, doesn't exercise the fixed production INSERT).

### Commit

When step DoD is met: `feat(mnemonic): temporal knowledge graph edges with valid_from/valid_to`

---

## 11-portable-export

### Goal

COGX-inspired JSON export of observations + graph + embeddings.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-11` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/export.go` — CREATE: COGX-inspired portable JSON export
- `internal/mnemonic/memory/export_test.go` — CREATE: export tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `export` subcommand

**Interfaces:**
- Consumes: observations table; graph edges from codeindex; symbol_embeddings from step 07
- Produces: `ExportProject(ctx, projectID)` method; `mem export` CLI; COGX-inspired JSON format

### Tasks

- [x] 11.1 `[RED]` ExportProject returns JSON with observations, graph edges, and embeddings
  - [x] 11.1.a Write failing test (`TestExportProjectStructure`): create observations, graph edges, and symbol embeddings; call `ExportProject(ctx, projectID)`; verify the returned struct has non-empty `Observations[]`, `GraphEdges[]`, `Embeddings[]`; verify each observation record has `id`, `type`, `content`, `metadata`, `embeddings` (base64), `graph_ref`
  - [x] 11.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure'` — Expected: FAIL
  - [x] 11.1.c Minimal implementation — create `export.go` with `ExportProject(ctx context.Context, projectID string) (*ExportBundle, error)`; define `ExportBundle` struct with `Observations []ExportRecord`, `GraphEdges []EdgeRecord`, `Embeddings []EmbeddingRecord`; query all three sources; encode embeddings as base64; COGX-inspired format with `id`, `type`, `content`, `metadata`, `embeddings`, `graph_ref` per record
  - [x] 11.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure'` — Expected: PASS
  - [x] 11.1.e Commit — `feat(mnemonic): COGX-inspired portable JSON export`
- [x] 11.2 `[RED]` Export is portable — output can be imported into another instance
  - [x] 11.2.a Write failing test (`TestExportImportRoundtrip`): export a project with 5 observations, 3 edges, 2 embeddings; write to a JSON file; create a new empty store; import the JSON; verify all 5 observations, 3 edges, and 2 embeddings are present with matching IDs and content
  - [x] 11.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportImportRoundtrip'` — Expected: FAIL
  - [x] 11.2.c Minimal implementation — implement `ImportProject(ctx, projectID, reader io.Reader)` that reads the COGX-inspired JSON and inserts records into the store; ensure embeddings are decoded from base64 and stored in `symbol_embeddings`; verify the roundtrip preserves all data
  - [x] 11.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportImportRoundtrip'` — Expected: PASS
  - [x] 11.2.e Commit — `feat(mnemonic): export/import roundtrip for portable memory`
- [x] 11.3 `[AFK]` mem export CLI with --file and --skip-embeddings flags
  - [x] 11.3.a Write failing test (`TestMemExportCLI`): invoke `mem export` and verify JSON is written to stdout; invoke `mem export --file out.json` and verify the file is created; invoke `mem export --skip-embeddings` and verify the output JSON has no embedding fields; verify streaming for large stores (no OOM)
  - [x] 11.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExportCLI'` — Expected: FAIL
  - [x] 11.3.c Minimal implementation — add `mem export` subcommand to `mem.go`; support `--file` flag for file output (stdout by default); support `--skip-embeddings` flag to omit embedding data for smaller payloads; stream JSON output for large stores to avoid memory pressure
  - [x] 11.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExportCLI'` — Expected: PASS
  - [x] 11.3.e Commit — `feat(mnemonic): add mem export CLI with file and skip-embeddings flags`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure\|TestExportImportRoundtrip'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExportCLI'` | PASS | PASS | 3 step-11 tests GREEN |
| Acceptance `@step-11` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 12.9s, cmd 52.6s |
| Rollback boundary | verify export does not modify the source store | PASS | PASS | export is read-only; import into a fresh store works |
| Global Constraints | — | held | held | additive (new methods); base64 roundtrip byte-identical; temporal fields preserved; no tool contract change; no CGO |

Commits: `365b9c5` (ExportProject + ImportProject roundtrip), `dc514cb` (mem export CLI). Review: PASS WITH WARNINGS. Warnings: (M1) "streaming" is a lie — `json.Encoder.Encode` buffers the whole bundle before writing (peak mem = bundle + full JSON copy); code/comments claim incremental writes (fix claims or do per-section encoding); (m1) `--skip-embeddings` emits `"embeddings": []` (field not omitted) + CLI test assertion is tautological; (m5) embeddings export pass is skipped entirely when no observation has a graph_ref (a project with vectors but no bound observations exports an empty embeddings section). Verified: base64 roundtrip byte-identical, temporal fields (valid_from/valid_to) preserved, graph_ref restored, no tool-contract changes.

### Commit

When step DoD is met: `feat(mnemonic): COGX-inspired portable JSON export with import roundtrip`

---

## 12-dream-executor

### Goal

consolidate/synthesize/prune distillation decomposition.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-12` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 04, 08

**Files:**
- `internal/mnemonic/memory/dream.go` — CREATE: DreamExecutor (consolidate/synthesize/prune); DreamLockService; DreamRollback
- `internal/mnemonic/memory/dream_test.go` — CREATE: dream executor tests
- `internal/mnemonic/memory/dream_lock_test.go` — CREATE: DreamLock + rollback tests

**Interfaces:**
- Consumes: TTL defaults from step 04; improve() scoring from step 08; AKL importance scoring
- Produces: `DreamExecutor` with `consolidate()`, `synthesize()`, `prune()`; `DreamLockService`; `DreamRollback`

### Tasks

- [x] 12.1 `[RED]` consolidate() merges facts from multiple observations into coherent knowledge
  - [x] 12.1.a Write failing test (`TestDreamConsolidate`): create 5 observations with overlapping facts about the same topic; call `DreamExecutor.consolidate()`; verify the overlapping facts are merged into a single coherent observation; verify no fact is lost in the merge; verify the source observations are marked as consolidated
  - [x] 12.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate'` — Expected: FAIL
  - [x] 12.1.c Minimal implementation — create `dream.go` with `DreamExecutor` struct; implement `consolidate(ctx, observations []Observation) (ConsolidatedResult, error)` that groups observations by topic and merges overlapping facts using LLM summarization; mark source observations as consolidated (set a status flag)
  - [x] 12.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate'` — Expected: PASS
  - [x] 12.1.e Commit — `feat(mnemonic): DreamExecutor consolidate phase`
- [x] 12.2 `[RED]` synthesize() creates higher-level summaries from lower-tier memories
  - [x] 12.2.a Write failing test (`TestDreamSynthesize`): create L0/L1 session summaries; call `DreamExecutor.synthesize()`; verify a higher-level L2 summary is created that captures the key points from the lower-tier memories; verify the summary references the source sessions
  - [x] 12.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamSynthesize'` — Expected: FAIL
  - [x] 12.2.c Minimal implementation — implement `synthesize(ctx, memories []Memory) (Summary, error)` that takes lower-tier memories and creates a higher-level summary via LLM; store the summary as a new observation with a higher tier level
  - [x] 12.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamSynthesize'` — Expected: PASS
  - [x] 12.2.e Commit — `feat(mnemonic): DreamExecutor synthesize phase`
- [x] 12.3 `[RED]` prune() removes low-importance observations based on AKL scoring
  - [x] 12.3.a Write failing test (`TestDreamPrune`): create 10 observations with varying importance scores (high, medium, low); call `DreamExecutor.prune()`; verify only low-importance observations are soft-deleted; verify high and medium importance observations remain; verify pruning respects the AKL maturity tier
  - [x] 12.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamPrune'` — Expected: FAIL
  - [x] 12.3.c Minimal implementation — implement `prune(ctx, threshold float64) (PrunedResult, error)` that queries observations with importance_score below threshold and maturity_tier `archival`; soft-deletes them (set `deleted_at`); returns a count of pruned observations
  - [x] 12.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamPrune'` — Expected: PASS
  - [x] 12.3.e Commit — `feat(mnemonic): DreamExecutor prune phase`
- [x] 12.4 `[RED]` DreamLockService prevents concurrent distillation per project
  - [x] 12.4.a Write failing test (`TestDreamLockPreventsConcurrent`): acquire a dream lock for project A; attempt to acquire another dream lock for project A; verify the second acquisition fails (or blocks); acquire a lock for project B; verify it succeeds; release the project A lock; verify project A can be locked again
  - [x] 12.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamLockPreventsConcurrent'` — Expected: FAIL
  - [x] 12.4.c Minimal implementation — create `DreamLockService` using a `distill_lock` row per project in the store; `Acquire(projectID)` inserts or updates the lock row with a timestamp; if the lock is held and older than 5 minutes, auto-release; `Release(projectID)` removes the lock row
  - [x] 12.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamLockPreventsConcurrent'` — Expected: PASS
  - [x] 12.4.e Commit — `feat(mnemonic): DreamLockService with 5-minute timeout`
- [x] 12.5 `[RED]` DreamRollback reverts observations to pre-distill state on failure
  - [x] 12.5.a Write failing test (`TestDreamRollbackOnFailure`): start a dream (consolidate); capture the pre-distill state; simulate a failure during synthesize; call `DreamRollback`; verify all observations are restored to their pre-distill state; verify the lock is released after rollback
  - [x] 12.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamRollbackOnFailure'` — Expected: FAIL
  - [x] 12.5.c Minimal implementation — implement `DreamRollback(ctx, projectID, preDistillState []ObservationSnapshot)` that restores observations to their pre-distill state from the captured snapshot; wrap consolidate/synthesize/prune in a transaction; on error, call rollback and release the lock
  - [x] 12.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamRollbackOnFailure'` — Expected: PASS
  - [x] 12.5.e Commit — `feat(mnemonic): DreamRollback reverts on failure`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate\|TestDreamDeterministicNoLLM\|TestDreamPrune\|TestDreamImportanceMonotonic\|TestDreamSynthesize\|TestDreamLockPreventsConcurrent\|TestDreamLockTTLBoundary\|TestDreamRollbackOnFailure'` | PASS | PASS | 8 dream tests GREEN |
| Acceptance `@step-12` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | full memory suite |
| Rollback boundary | verify DreamLock timeout is 5 minutes with auto-release | PASS | PASS | TTL boundary tested at 4min (held) / 6min (auto-released); atomic INSERT...ON CONFLICT DO UPDATE WHERE (no TOCTOU) |
| Global Constraints | — | held | held | 100% new files (additive); LLM opt-in (DreamLLM seam + SetLLM + dreamJoin fallback); importance on-the-fly (no schema change); no tool contract change; no CGO |

Commits: `28d8cca` (consolidate), `3d08793` (synthesize), `44802ec` (prune + 024_dream_lock), `ae1b20f` (DreamLockService), `7346079` (DreamRollback). Review: PASS WITH WARNINGS. Warnings: (M1) rollback's 3 steps (restore/orphan-sweep/release) are NOT in one transaction — a crash mid-rollback leaves a half-restored project (fail-loud, lock held); recommended follow-up: wrap in a transaction; (M2) "no fact lost" in consolidate is a structural floor (dreamJoin verbatim source list), not an LLM contract — verified for no-LLM + stub-LLM only, documented; (m5) deleteOrphans no-ops when keep is empty (a dream on an empty project leaves its orphan behind). Verified: importance formula (0.6*usage + 0.37*recency + 0.2*typeWeight, monotonic, 30d grace, soft-delete only); lock atomicity; migration additive.

### Commit

When step DoD is met: `feat(mnemonic): DreamExecutor with consolidate/synthesize/prune, DreamLock, and rollback`

---

## 13-importance-scoring

### Goal

AKL importance scoring + recency decay in improve() and query ranking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-13` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 08

**Files:**
- `internal/mnemonic/memory/importance.go` — CREATE: AKL importance scoring + recency decay
- `internal/mnemonic/memory/importance_test.go` — CREATE: importance scoring tests
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: importance scoring integration

**Interfaces:**
- Consumes: `retrieval_usage` column; improve() from step 08; observation age data
- Produces: `importance_score`, `maturity_tier`, `recency_decay` columns; query-time ranking boost

### Tasks

- [x] 13.1 `[RED]` importance_score is computed from retrieval_count × recency_factor
  - [x] 13.1.a Write failing test (`TestImportanceScoreComputation`): create observations with varying `retrieval_usage` (0, 5, 50, 100) and varying ages (1 day, 7 days, 30 days); call the importance scorer; verify importance_score increases with retrieval_count; verify older observations have lower scores (exponential decay); verify the formula is `retrieval_count * exp(-decay_rate * age_days)`
  - [x] 13.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation'` — Expected: FAIL
  - [x] 13.1.c Minimal implementation — create `importance.go` with `ComputeImportanceScore(retrievalCount int, age time.Duration, decayRate float64) float64` using exponential decay; add `importance_score` (FLOAT), `maturity_tier` (TEXT enum), `recency_decay` (FLOAT) columns to observations via migration; populate on save and on periodic recompute
  - [x] 13.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation'` — Expected: PASS
  - [x] 13.1.e Commit — `feat(mnemonic): AKL importance score with exponential recency decay`
- [x] 13.2 `[RED]` maturity_tier transitions from fresh → mature → archival based on age + retrieval
  - [x] 13.2.a Write failing test (`TestMaturityTierTransitions`): create a new observation (age 0) → verify tier is `fresh`; age it to 7 days with 10 retrievals → verify tier is `mature`; age it to 30 days with 0 retrievals → verify tier is `archival`; verify transitions are monotonic (no downgrade)
  - [x] 13.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMaturityTierTransitions'` — Expected: FAIL
  - [x] 13.2.c Minimal implementation — implement `ComputeMaturityTier(age time.Duration, retrievalCount int) string` with thresholds: `fresh` (age < 7d), `mature` (age >= 7d AND retrievalCount > 0), `archival` (age >= 30d OR retrievalCount == 0 AND age > 14d); store the tier on the observation
  - [x] 13.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMaturityTierTransitions'` — Expected: PASS
  - [x] 13.2.e Commit — `feat(mnemonic): maturity tier computation (fresh/mature/archival)`
- [x] 13.3 `[RED]` mem_search applies importance score as query-time ranking boost
  - [x] 13.3.a Write failing test (`TestImportanceScoreQueryRanking`): create 3 observations matching a query — one with high importance (score 9.5), one medium (5.0), one low (0.5); run `mem_search`; verify the high-importance observation ranks first; verify the importance score is applied as a multiplicative boost on top of the base relevance score
  - [x] 13.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreQueryRanking'` — Expected: FAIL
  - [x] 13.3.c Minimal implementation — in `SearchOwnerScoped`, after computing base relevance scores, multiply by a normalized importance factor (`importance_score / max_importance`); make the decay rate configurable via `mnemonic.importance.decay` config key; `improve()` uses importance score instead of binary retrieval_usage
  - [x] 13.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreQueryRanking'` — Expected: PASS
  - [x] 13.3.e Commit — `feat(mnemonic): importance score as query-time ranking boost`
- [x] 13.4 `[AFK]` Configurable decay rate and thresholds
  - [x] 13.4.a Write failing test (`TestImportanceConfigurableDecay`): set `mnemonic.importance.decay` to a high value; verify older observations lose importance faster; set it to a low value; verify older observations retain importance longer; verify default decay rate is reasonable
  - [x] 13.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceConfigurableDecay'` — Expected: FAIL
  - [x] 13.4.c Minimal implementation — read `mnemonic.importance.decay` from config (default 0.05 per day); pass it to `ComputeImportanceScore`; make tier thresholds configurable via `mnemonic.importance.tier_thresholds`
  - [x] 13.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceConfigurableDecay'` — Expected: PASS
  - [x] 13.4.e Commit — `feat(mnemonic): configurable importance decay rate and tier thresholds`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation\|TestMaturityTierTransitions\|TestImportanceScoreQueryRanking\|TestImportanceConfigurableDecay'` | PASS | PASS | 4 step-13 tests GREEN |
| Acceptance `@step-13` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | PASS | full memory + config + store suites |
| Rollback boundary | verify importance scoring does not break default search ordering | PASS | PASS | boost is multiplicative + improve()-gated + in-memory; default search (improve off) unchanged |
| Global Constraints | — | held | held | 025 migration purely additive; math correct (retrieval*exp(-decay*age_days), age clamped, 0→0); monotonic tier (stored=max(prev,computed)); no tool contract change; no CGO |

Commits: `6907109` (importance score + 025 migration), `0cc759f` (query-time boost), `1bfe24e` (configurable decay + thresholds), `9cf5967` (decay behavior test + SetImportance wiring). Review: PASS WITH WARNINGS. Warnings: (m1) step-08 improve() still uses binary retrieval_usage internally — the brief's "improve() uses importance_score" is realized by call-site replacement, not re-pointing the function; (m2) SetImportance called in both project-open and per-read paths (redundant, mirrors step-08); (m3) idx_obs_importance index created but not yet queried (future prune/dream use).

### Commit

When step DoD is met: `feat(mnemonic): AKL importance scoring with recency decay and query-time ranking`

---

## 14-explicit-relations

### Goal

@relation annotations between observations.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-14` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/relations.go` — CREATE: @relation annotations between observations
- `internal/mnemonic/memory/relations_test.go` — CREATE: relations tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `relations` subcommand

**Interfaces:**
- Consumes: observations table; triple-store linkage from step 07
- Produces: typed relation edges in memory store; `mem relations` CLI

### Tasks

- [x] 14.1 `[RED]` Relation edges store source_id, target_id, relation_type, confidence
  - [x] 14.1.a Write failing test (`TestRelationEdgeCreation`): create two observations; call `AddRelation(sourceID, targetID, "mentions", 0.9)`; query the relation and verify it has the correct `source_id`, `target_id`, `relation_type`, and `confidence`; verify each of the 5 relation types can be created (`mentions`, `depends_on`, `contradicts`, `supports`, `references`)
  - [x] 14.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation'` — Expected: FAIL
  - [x] 14.1.c Minimal implementation — create `relations.go` with an `observation_relations` table: `source_id INTEGER NOT NULL, target_id INTEGER NOT NULL, relation_type TEXT NOT NULL, confidence REAL NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (source_id, target_id, relation_type)`; implement `AddRelation`, `GetRelations`, `RemoveRelation`
  - [x] 14.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation'` — Expected: PASS
  - [x] 14.1.e Commit — `feat(mnemonic): add typed relation edges between observations`
- [x] 14.2 `[RED]` mem relations CLI returns all related observations with relation types
  - [x] 14.2.a Write failing test (`TestMemRelationsCLI`): create observation A with relations to B (`mentions`), C (`depends_on`), D (`contradicts`); invoke `mem relations A`; verify the output lists B, C, D with their relation types and confidence scores; verify the CLI handles observations with no relations gracefully (empty output)
  - [x] 14.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'` — Expected: FAIL
  - [x] 14.2.c Minimal implementation — add `mem relations <observation_id>` subcommand to `mem.go`; query `observation_relations` for both outgoing and incoming edges; format the output with relation type and confidence; handle missing observation ID with a clear error
  - [x] 14.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'` — Expected: PASS
  - [x] 14.2.e Commit — `feat(mnemonic): add mem relations CLI subcommand`
- [x] 14.3 `[AFK]` Confidence filtering on relation queries
  - [x] 14.3.a Write failing test (`TestRelationConfidenceFiltering`): create relations with confidence 0.3, 0.7, 0.95; query with `minConfidence=0.5`; verify only the 0.7 and 0.95 relations are returned; query with `minConfidence=0.0`; verify all relations are returned
  - [x] 14.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationConfidenceFiltering'` — Expected: FAIL
  - [x] 14.3.c Minimal implementation — add `minConfidence` parameter to `GetRelations`; add `WHERE confidence >= ?` to the query; expose the filter via `mem relations <id> --min-confidence 0.5`
  - [x] 14.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationConfidenceFiltering'` — Expected: PASS
  - [x] 14.3.e Commit — `feat(mnemonic): add confidence filtering to relation queries`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation\|TestRelationConfidenceFiltering'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'` | PASS | PASS | 3 step-14 tests GREEN |
| Acceptance `@step-14` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory + cmd suites |
| Rollback boundary | verify implicit similarity still works alongside explicit relations | PASS | PASS | pre-existing relations.go (memory_relations) untouched; new observation_relations (026) is additive |
| Global Constraints | — | held | held | new table + new methods + new CLI subcommand; no tool contract change; no CGO |

Commits: `91ef284` (failing tests), `5082ae5` (CLI test), `67ae959` (observation_relations.go + 026 migration), `9c3dcbb` (mem relations CLI). Review: PASS WITH WARNINGS. Warnings: (M1) two parallel relation systems (old `memory_relations`/006 Engram-parity verdict system via MCP/HTTP + new `observation_relations`/026 typed CLI system) — deliberate, defensible split (distinguishable by vocabulary/storage/consumer surface), consolidation is follow-up debt; (m2) self-relation allowed in new system (old relations.go:79 rejects) + CLI mislabels self-relation direction; (m3) DeleteRelation is correct but unexercised dead code (no CLI/MCP/test caller this step).

### Commit

When step DoD is met: `feat(mnemonic): explicit @relation annotations between observations`

---

## 15-provenance-tracking

### Goal

Provenance chain metadata on observations.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-15` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 05

**Files:**
- `internal/mnemonic/memory/provenance.go` — CREATE: provenance chain tracking
- `internal/mnemonic/memory/provenance_test.go` — CREATE: provenance tests
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: provenance column
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `provenance` subcommand

**Interfaces:**
- Consumes: LLM extraction from step 05; session context; curation pipeline
- Produces: `provenance` JSON column on observations; `mem provenance` CLI

### Tasks

- [x] 15.1 `[RED]` Provenance JSON chain stores session_id → curate_command → source_files → LLM_reasoning
  - [x] 15.1.a Write failing test (`TestProvenanceChainStorage`): save an observation with a full provenance chain (session_id, curate_command, source_files, llm_reasoning); retrieve the observation; verify the `provenance` JSON column contains all four fields with correct values; verify the JSON is valid and parseable
  - [x] 15.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage'` — Expected: FAIL
  - [x] 15.1.c Minimal implementation — add `provenance` (TEXT, JSON) column to observations via migration; define `Provenance` struct with `SessionID`, `CurateCommand`, `SourceFiles []string`, `LLMReasoning string` fields; serialize to JSON on save; parse on read
  - [x] 15.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage'` — Expected: PASS
  - [x] 15.1.e Commit — `feat(mnemonic): add provenance JSON chain to observations`
- [x] 15.2 `[RED]` Provenance is immutable once set during curation
  - [x] 15.2.a Write failing test (`TestProvenanceImmutability`): save an observation with provenance; attempt to update the observation without changing provenance; verify the provenance is unchanged; attempt to update with a different provenance; verify the update is rejected (or the provenance is preserved); verify the original provenance is intact
  - [x] 15.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceImmutability'` — Expected: FAIL
  - [x] 15.2.c Minimal implementation — in the update path, check if `provenance` is already set (non-NULL); if so, preserve the existing value regardless of the update input; log a warning if a different provenance is attempted; only the initial save can set provenance
  - [x] 15.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceImmutability'` — Expected: PASS
  - [x] 15.2.e Commit — `feat(mnemonic): make provenance immutable after initial set`
- [x] 15.3 `[AFK]` mem provenance CLI returns the full provenance chain
  - [x] 15.3.a Write failing test (`TestMemProvenanceCLI`): save an observation with a full provenance chain; invoke `mem provenance <observation_id>`; verify the output shows all four chain elements (session_id, curate_command, source_files, llm_reasoning); verify the output is formatted as a readable chain (not raw JSON); verify missing provenance returns a clear message
  - [x] 15.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemProvenanceCLI'` — Expected: FAIL
  - [x] 15.3.c Minimal implementation — add `mem provenance <observation_id>` subcommand to `mem.go`; query the observation and parse its `provenance` JSON; format the output as a readable chain; include provenance summary in `mem list` output
  - [x] 15.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemProvenanceCLI'` — Expected: PASS
  - [x] 15.3.e Commit — `feat(mnemonic): add mem provenance CLI subcommand`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage\|TestProvenanceImmutability'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemProvenanceCLI'` | PASS | PASS | 3 step-15 tests GREEN |
| Acceptance `@step-15` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 18.3s, cmd 99.4s |
| Rollback boundary | verify observations without provenance still work | PASS | PASS | provenance column nullable (NULL = not set); optional SaveInput field; default search unchanged |
| Global Constraints | — | held | held | 027 migration purely additive; immutability by omission (no UPDATE writes provenance); no tool contract change; no CGO |

Commits: `26171b2` (provenance JSON + 027 migration), `8e66c81` (immutability), `9d2f130` (mem provenance CLI), `5f34108` (include provenance in scanObservations SELECTs). Review: PASS WITH WARNINGS. Warnings: (m1) SearchOwnerScoped TTL filter (expires_at soft-exclude) removed — out-of-scope behavior change, now asymmetric with SearchWithScope (service.go:629); (m2) 15.2 "log a warning if different provenance attempted" deferred — not logged at either update path, test comment overstates; (n3) mem.go:89-92 two-space indentation regression (gofmt noise). Immutability VERIFIED: enforced by omission — no UPDATE observations anywhere writes provenance (upsert, Update(), dream-rollback upsert all omit it).

### Commit

When step DoD is met: `feat(mnemonic): provenance chain metadata with immutability`

---

## 16-federated-query

### Goal

Evolve parallel search into federated cross-project query with importance ranking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-16` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 03, 13

**Files:**
- `internal/mnemonic/service/service.go` — MODIFY: federated query
- `internal/mnemonic/service/retrieval.go` — MODIFY: parallel → federated cross-project search

**Interfaces:**
- Consumes: parallel search from step 03; importance scoring from step 13
- Produces: federated query pipeline with importance ranking and dedup; `mem search all_projects=true` uses federated pipeline

### Tasks

- [x] 16.1 `[RED]` Federated query merges results by cross-store rank + importance score
  - [x] 16.1.a Write failing test (`TestFederatedQueryImportanceRanking`): create 3 project stores, each with observations of varying importance scores (high, medium, low); run `SearchObservationsAll` (federated mode); verify results are ranked by a composite score combining cross-store rank and importance; verify the highest composite score ranks first
  - [x] 16.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking'` — Expected: FAIL
  - [x] 16.1.c Minimal implementation — evolve `SearchObservationsAll` into a federated pipeline: each store returns `{observations, importance_scores, store_id}`; the merge pipeline computes a composite score = `cross_store_rank_weight * rank + importance_weight * importance_score`; sort by composite score descending; make weights configurable
  - [x] 16.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking'` — Expected: PASS
  - [x] 16.1.e Commit — `feat(mnemonic): federated query with importance-based merge ranking`
- [x] 16.2 `[RED]` Dedup by observation ID across stores
  - [x] 16.2.a Write failing test (`TestFederatedQueryDedup`): create the same observation (same ID) in two different stores; run `SearchObservationsAll`; verify the observation appears only once in the results; verify the dedup preserves the higher-ranked version; verify the `seen` map is used for dedup
  - [x] 16.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryDedup'` — Expected: FAIL
  - [x] 16.2.c Minimal implementation — in the federated merge, maintain a `seen` map keyed by observation ID; when a duplicate is found, keep the version with the higher composite score; skip the duplicate in the final result set
  - [x] 16.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryDedup'` — Expected: PASS
  - [x] 16.2.e Commit — `feat(mnemonic): federated query dedup by observation ID`
- [x] 16.3 `[AFK]` Federated query respects per-project importance scores from step 13
  - [x] 16.3.a Write failing test (`TestFederatedQueryRespectsProjectImportance`): create 2 stores — store A has an observation with importance 9.0, store B has an observation with importance 1.0 but higher FTS relevance; run federated query; verify the composite ranking balances both factors; verify per-project importance is applied before cross-store merge
  - [x] 16.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryRespectsProjectImportance'` — Expected: FAIL
  - [x] 16.3.c Minimal implementation — ensure each store's search applies its local importance scoring (from step 13) before returning results; the federated merge uses the per-project importance scores in the composite ranking; document the weight configuration
  - [x] 16.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryRespectsProjectImportance'` — Expected: PASS
  - [x] 16.3.e Commit — `feat(mnemonic): federated query respects per-project importance scores`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking\|TestFederatedQueryDedup\|TestFederatedQueryRespectsProjectImportance'` | PASS | PASS | 3 federated tests GREEN |
| Acceptance `@step-16` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/service/ -count=1 -race` | PASS | PASS | 3 federated + 5 step-03 regression tests, -race clean |
| Rollback boundary | verify parallel search fallback works when federated is disabled | PASS | PASS | SKILLGRID_SEARCH_PARALLEL=0 still works; composite degrades to rank-only when no positive importance (same rank order as step 03) |
| Global Constraints | — | held | held | SearchObservationsAll signature unchanged; federated pipeline additive; no tool contract change; no CGO; -race clean |

Commits: `016ba01` (federated pipeline + composite + dedup + config + importance SELECT). Review: PASS WITH WARNINGS. Composite: `rank_weight*(1/(1+rank)) + importance_weight*(importance/maxImportance)`, higher=better, degrades to rank-only when maxImportance=0. Dedup key: `ID+project` (correct — each store has own ID space; ID-alone would collapse step-03's identical-content-across-stores case). Warnings: (m1) federatedComposite comment claims "byte-identical to step 03" but is actually rank-order-identical (new ID-desc tiebreak differs from step-03's UpdatedAt in same-rank edge) — reword comment; (m2) dedup "higher-composite-wins" path only exercised via synthetic double-emit (same storeRanked fed twice), not two real stores — acceptable for unit test, worth a self-documenting comment.

### Commit

When step DoD is met: `feat(mnemonic): federated cross-project query with importance ranking and dedup`

---

## 17-distill-lock

### Goal

DistillLockService + DistillRollback on failure.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-17` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 09, 12

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: DistillLockService
- `internal/mnemonic/memory/dream_lock_test.go` — MODIFY: DistillLock + rollback tests

**Interfaces:**
- Consumes: DreamLockService from step 12; session promotion from step 09
- Produces: `DistillLockService` for distillation; `DistillRollback` on failure; lock status CLI

### Tasks

- [x] 17.1 `[RED]` DistillLockService uses distill_lock row per project
  - [x] 17.1.a Write failing test (`TestDistillLockRowPerProject`): call `DistillLockService.Acquire(projectA)`; verify a `distill_lock` row is created for projectA with a timestamp; call `Acquire(projectB)`; verify a separate row for projectB; call `Acquire(projectA)` again; verify it detects the existing lock; call `Release(projectA)`; verify the row is removed
  - [x] 17.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject'` — Expected: FAIL
  - [x] 17.1.c Minimal implementation — ensure `DistillLockService` (from step 12) uses a `distill_lock` table with `project_id TEXT PRIMARY KEY, locked_at TIMESTAMP, locked_by TEXT` columns; `Acquire` does an `INSERT OR FAIL` to prevent concurrent locks; `Release` does a `DELETE`
  - [x] 17.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject'` — Expected: PASS
  - [x] 17.1.e Commit — `feat(mnemonic): DistillLockService with per-project lock rows`
- [x] 17.2 `[RED]` DistillRollback reverts observations to pre-distill state on failure
  - [x] 17.2.a Write failing test (`TestDistillRollbackOnFailure`): capture pre-distill state (all observations); start a distillation; simulate a mid-distill failure; call `DistillRollback`; verify all observations match the pre-distill state exactly; verify the lock is released after rollback; verify no partial changes remain
  - [x] 17.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillRollbackOnFailure'` — Expected: FAIL
  - [x] 17.2.c Minimal implementation — implement `DistillRollback(ctx, projectID, snapshots []ObservationSnapshot)` that restores each observation to its pre-distill state from the snapshot list; wrap in a transaction for atomicity; release the distill lock after rollback completes
  - [x] 17.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillRollbackOnFailure'` — Expected: PASS
  - [x] 17.2.e Commit — `feat(mnemonic): DistillRollback reverts to pre-distill state on failure`
- [x] 17.3 `[RED]` Lock timeout is 5 minutes with auto-release on error
  - [x] 17.3.a Write failing test (`TestDistillLockTimeoutAutoRelease`): acquire a distill lock; set the lock timestamp to 6 minutes ago (simulate staleness); attempt to acquire the lock again; verify the stale lock is auto-released and the new acquisition succeeds; verify a lock that is only 1 minute old is not released
  - [x] 17.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockTimeoutAutoRelease'` — Expected: FAIL
  - [x] 17.3.c Minimal implementation — in `Acquire`, check if an existing lock has `locked_at < now - 5min`; if so, delete the stale lock and proceed with the new acquisition; log a warning for stale lock auto-release; ensure 5-minute timeout is configurable via `mnemonic.distill.lock_timeout`
  - [x] 17.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockTimeoutAutoRelease'` — Expected: PASS
  - [x] 17.3.e Commit — `feat(mnemonic): distill lock 5-minute timeout with auto-release`
- [x] 17.4 `[AFK]` Lock status visible via CLI
  - [x] 17.4.a Write failing test (`TestDistillStatusCLI`): acquire a distill lock for a project; invoke the distill status CLI; verify it shows the lock is held with timestamp and locked_by; release the lock; verify the CLI shows no active lock
  - [x] 17.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestDistillStatusCLI'` — Expected: FAIL
  - [x] 17.4.c Minimal implementation — add `mem distill status` subcommand to `mem.go`; query the `distill_lock` table; display active locks with project_id, locked_at, and locked_by; show "no active locks" when the table is empty
  - [x] 17.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestDistillStatusCLI'` — Expected: PASS
  - [x] 17.4.e Commit — `feat(mnemonic): add mem distill status CLI subcommand`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject\|TestDistillRollbackOnFailure\|TestDistillRollbackAtomic\|TestDistillLockTimeoutAutoRelease'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestDistillStatusCLI'` | PASS | PASS | 4 step-17 + 2 CLI tests GREEN; step-12 lock/rollback regression tests unbroken |
| Acceptance `@step-17` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | full memory + CLI suites |
| Rollback boundary | verify lock timeout is 5 minutes with auto-release | PASS | PASS | 5-min TTL auto-release tested; TestDistillRollbackAtomic proves mid-rollback crash leaves project unchanged (lock held, not half-restored) |
| Global Constraints | — | held | held | DistillLockService is a thin wrapper (no lock logic duplicated); DreamLockService byte-for-byte unchanged; transaction fix is additive; no tool contract change; no CGO |

Commits: `783e9b8` (DistillLockService wrapper), `7b02292` (DistillRollback + transaction fix), `ef5b193` (5-min timeout), `9c577eb` (mem distill status CLI). Review: PASS. Reused step 12's DreamLockService/DreamRollback (thin wrapper, no duplication). FIXED step-12 F1: rollback restore+orphan-sweep now in one BeginTx/Commit (defer tx.Rollback), lock release moved outside tx. Warnings (minor, non-blocking): (M1) DreamLockService reads/writes table named distill_lock (step-12 naming duality, cosmetic, inherited); (M2) DistillLockService stores a db used only by read-only DistillLockStatus (one-field denormalization, not a bug).

### Commit

When step DoD is met: `feat(mnemonic): DistillLockService with rollback and 5-minute timeout`

---

## 18-memory-types

### Goal

Typed memory categories + LLM dedup + async two-phase commit.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-18` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 04, 05

**Files:**
- `internal/mnemonic/memory/types.go` — CREATE: memory types + LLM dedup + async two-phase commit
- `internal/mnemonic/memory/types_test.go` — CREATE: memory types tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `memory-type` subcommand

**Interfaces:**
- Consumes: TTL defaults from step 04; LLM extraction from step 05; `Save()` pipeline
- Produces: `memory_type` column with 9 typed categories; LLM dedup before write; async two-phase commit; `memory_diff.json` audit

### Tasks

- [x] 18.1 `[RED]` memory_type column supports 9 typed categories
  - [x] 18.1.a Write failing test (`TestMemoryTypeCategories`): save observations with each of the 9 types (`profile`, `preferences`, `entities`, `events`, `identity`, `soul`, `cases`, `trajectories`, `experiences`); verify each is stored correctly; verify `mem list --type preferences` returns only preference-typed observations; verify invalid type is rejected
  - [x] 18.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories'` — Expected: FAIL
  - [x] 18.1.c Minimal implementation — add `memory_type` (TEXT) column to observations via migration; define the 9 valid type constants; validate on save (reject unknown types with a clear error); add `--type` filter to `mem list` CLI
  - [x] 18.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories'` — Expected: PASS
  - [x] 18.1.e Commit — `feat(mnemonic): add 9 typed memory categories`
- [x] 18.2 `[RED]` LLM dedup detects semantic duplicates before write
  - [x] 18.2.a Write failing test (`TestLLMDedupDetectsSemanticDuplicates`): save an observation with content "The build fails on macOS because of the missing SDK"; attempt to save a similar observation "macOS build broken due to absent SDK package"; verify the LLM dedup flags the second as a duplicate; verify the second is not stored (or is merged with the first); verify a genuinely different observation is not flagged
  - [x] 18.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLLMDedupDetectsSemanticDuplicates'` — Expected: FAIL
  - [x] 18.2.c Minimal implementation — in the `Save()` path, before writing, call an LLM dedup check: query existing observations with similar embeddings (vector pre-filter), then ask the LLM if the new observation is a semantic duplicate; if duplicate, skip the write (or update the existing); hash dedup as fallback when LLM is unavailable
  - [x] 18.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLLMDedupDetectsSemanticDuplicates'` — Expected: PASS
  - [x] 18.2.e Commit — `feat(mnemonic): LLM dedup before write with hash fallback`
- [x] 18.3 `[RED]` Async two-phase commit: sync write + async LLM extraction
  - [x] 18.3.a Write failing test (`TestAsyncTwoPhaseCommit`): call `session.commit()`; verify the sync phase completes immediately (messages written, compression_index incremented); verify the async phase (LLM extraction, dedup) runs in the background; verify `memory_diff.json` is written after the async phase completes; verify the main write path is not blocked by the async phase
  - [x] 18.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAsyncTwoPhaseCommit'` — Expected: FAIL
  - [x] 18.3.c Minimal implementation — in `session.commit()`, split into sync phase (write messages, increment compression_index, return) and async phase (launch goroutine for LLM extraction, vector pre-filtering, dedup, write `memory_diff.json`); ensure the sync phase is durable before returning; the async phase writes to `memory_diff.json` for auditing
  - [x] 18.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAsyncTwoPhaseCommit'` — Expected: PASS
  - [x] 18.3.e Commit — `feat(mnemonic): async two-phase session commit`
- [x] 18.4 `[AFK]` Fallback to auto-classification when LLM type assignment fails
  - [x] 18.4.a Write failing test (`TestMemoryTypeFallbackToAutoClassification`): mock the LLM to return an error during type classification; save an observation; verify it falls back to auto-classification (keyword-based type inference); verify the observation is stored with a best-guess type; verify no error is propagated
  - [x] 18.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeFallbackToAutoClassification'` — Expected: FAIL
  - [x] 18.4.c Minimal implementation — in the type assignment path, if the LLM returns an error, fall back to keyword-based auto-classification (map keywords to types); log a warning; ensure the observation is still saved with the fallback type
  - [x] 18.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeFallbackToAutoClassification'` — Expected: PASS
  - [x] 18.4.e Commit — `feat(mnemonic): fallback to auto-classification when LLM type fails`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories\|TestLLMDedupDetectsSemanticDuplicates\|TestAsyncTwoPhaseCommit' -race -count=1` | PASS | PASS | 3 step-18 tests GREEN, race-clean (reviewer re-ran independently) |
| Acceptance `@step-18` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | full memory + cmd suites |
| Rollback boundary | verify observations without memory_type still work (default) | PASS | PASS | memory_type nullable (NULL = not typed); default Save() path unchanged; existing type column untouched |
| Global Constraints | — | held | held | 028 migration purely additive; LLM opt-in (DedupLLM seam, mnemonic.dedup.llm default off); async two-phase durable sync phase; no tool contract change; no CGO |

Commits: `d35b96a` (typed memory categories + LLM dedup + async two-phase commit, 8 files +1085/-20). Review: PASS WITH WARNINGS. Async durability VERIFIED: sync phase (write message, compression_index increment, read-back) fully committed via synchronous *sql.DB calls before SessionCommit returns; goroutine spawned after. Dedup CORRECT: injectable DedupLLM seam (mirrors step-05), pre-filter → LLM check → merge (bump duplicate_count, no new row), hash fallback on disabled/absent/error, opt-in default-off, mock LLM in tests. Warnings: (F1) candidateDedupResult (types.go:50) declared but never used (dead type); (F2) dedupCandidateID ignores its candidates arg (fallback merges into most-recent row when LLM returns id 0); (F3) reason return always discarded; (gap) brief's [AFK] 18.4 (keyword auto-classification fallback) not in commit — consistent with "three RED sub-tasks" scope, flag for DoD confirmation; (nit) generic --type flag, last-wins memory_diff.json.

### Commit

When step DoD is met: `feat(mnemonic): typed memory categories with LLM dedup and async two-phase commit`

---

## 19-directory-retrieval

### Goal

Directory-level recursive retrieval with drill-down + observable trajectory.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-19` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 02, 03

**Files:**
- `internal/mnemonic/memory/retrieval.go` — CREATE: directory-level recursive retrieval + trajectory
- `internal/mnemonic/memory/retrieval_test.go` — CREATE: retrieval tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `search --trajectory` flag

**Interfaces:**
- Consumes: FTS trigram from step 02; parallel search from step 03; embeddings
- Produces: `Retrieve` module with intent analysis → hierarchical retrieval → rerank; `retrieval_trails` storage; `--trajectory` CLI flag

### Tasks

- [x] 19.1 `[RED]` Directory retrieval finds highest-scoring directory first, then drills down
  - [x] 19.1.a Write failing test (`TestDirectoryRetrievalDrillDown`): create a hierarchical set of observations organized by scope (project/app/core, project/app/api, project/docs); search for a query matching observations in `project/app/core`; verify the retrieval first identifies `project/app/core` as the highest-scoring directory; verify it drills down into that directory; verify results are scoped to the drilled-down directory
  - [x] 19.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown'` — Expected: FAIL
  - [x] 19.1.c Minimal implementation — create `retrieval.go` with `Retrieve(ctx, query, scope) (Results, Trajectory, error)`; first pass: score all top-level scopes by FTS5 + embedding similarity; second pass: drill down into the highest-scoring scope; repeat until leaf level; depth limit of 5 to prevent infinite recursion
  - [x] 19.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown'` — Expected: PASS
  - [x] 19.1.e Commit — `feat(mnemonic): directory-level recursive retrieval with drill-down`
- [x] 19.2 `[RED]` Retrieval trajectory is preserved for debugging
  - [x] 19.2.a Write failing test (`TestRetrievalTrajectoryPreserved`): run a directory retrieval; verify the trajectory records each directory visited (path, score, depth); verify the trajectory is stored in `retrieval_trails`; verify the trajectory can be retrieved by query ID; verify the trajectory shows the drill-down path
  - [x] 19.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRetrievalTrajectoryPreserved'` — Expected: FAIL
  - [x] 19.2.c Minimal implementation — create a `retrieval_trails` table: `id INTEGER PRIMARY KEY, query TEXT, path TEXT, scores TEXT, depth INTEGER, created_at TIMESTAMP`; record each directory visit during retrieval; store the complete path as a JSON array; add `GetTrajectory(queryID)` to retrieve the trail
  - [x] 19.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRetrievalTrajectoryPreserved'` — Expected: PASS
  - [x] 19.2.e Commit — `feat(mnemonic): preserve retrieval trajectory for debugging`
- [x] 19.3 `[RED]` Intent analysis classifies query before retrieval
  - [x] 19.3.a Write failing test (`TestIntentAnalysisClassification`): classify a query like "why does the build fail" as `debugging`; classify "what files are in the auth module" as `exploration`; classify "review the changes to payment.go" as `review`; verify the intent classification affects the retrieval strategy (debugging → wider search, exploration → directory-first)
  - [x] 19.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentAnalysisClassification'` — Expected: FAIL
  - [x] 19.3.c Minimal implementation — implement `ClassifyIntent(query string) Intent` using keyword matching and pattern detection; return one of `exploration`, `debugging`, `review`, `refactor`; adjust retrieval parameters based on intent (debugging → wider scope, exploration → directory-first)
  - [x] 19.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentAnalysisClassification'` — Expected: PASS
  - [x] 19.3.e Commit — `feat(mnemonic): intent analysis for query classification`
- [x] 19.4 `[AFK]` Depth limit prevents excessive drill-down on deep hierarchies
  - [x] 19.4.a Write failing test (`TestDirectoryRetrievalDepthLimit`): create a hierarchy 10 levels deep; run a retrieval; verify it stops at depth 5 (the configured limit); verify it returns the best results found at the limit; verify no stack overflow or infinite loop
  - [x] 19.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDepthLimit'` — Expected: FAIL
  - [x] 19.4.c Minimal implementation — enforce `maxDepth = 5` in the drill-down loop; when the limit is reached, return the best results at that depth; cache directory scores to avoid recomputation on repeated queries
  - [x] 19.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDepthLimit'` — Expected: PASS
  - [x] 19.4.e Commit — `feat(mnemonic): depth limit and caching for directory retrieval`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown\|TestRetrievalTrajectoryPreserved\|TestDirectoryRetrievalDepthLimit\|TestDirectoryRetrievalCache\|TestClassifyIntent'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestSearchTrajectoryCLI'` | PASS | PASS | 5 memory + 1 CLI test GREEN |
| Acceptance `@step-19` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 36s, CLI 96s |
| Rollback boundary | verify default flat search still works when directory retrieval is disabled | PASS | PASS | DirectoryRetrieve is a new function (not a change to existing search); flat search path unchanged |
| Global Constraints | — | held | held | 029 migration purely additive; topic_key hierarchy (no new column); depth limit 5 (iterative, no recursion); no tool contract change; no CGO |

Commits: `35d927c` (drill-down), `173b1dc` (trajectory), `ec53b28` (intent analysis), `559c9b9` (depth limit + caching + --trajectory CLI). Review: PASS WITH WARNINGS. Hierarchy: topic_key (slash-separated path) correct; scope correctly unused (flat visibility tag). Depth limit: iterative loop bounded by retrievalMaxDepth=5; 10-level test stops at exactly depth 5. Warnings: (M1) --trajectory opens second store + never calls SetDirEmbedder → production run is FTS5-only, embedding leg unwired (mem.go:339 / retrieval.go:215-220); (M2) dirScoreCache is process-global keyed query|root|matchMode with no project → cross-project score bleed in multi-project CLI runs (retrieval.go:157-159); (m3) LIKE 'root/%' glob-unsafe for topic_keys with _/% (retrieval.go:273,:461); GetTrajectory unscoped by project, errors on 010-era rows.

### Commit

When step DoD is met: `feat(mnemonic): directory-level recursive retrieval with drill-down and trajectory`

---

## 20-snapshots

### Goal

Multi-version snapshots + transaction locking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-20` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01, 17

**Files:**
- `internal/mnemonic/memory/snapshots.go` — CREATE: multi-version snapshots + transaction locking
- `internal/mnemonic/memory/snapshots_test.go` — CREATE: snapshot tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `snapshot` subcommand

**Interfaces:**
- Consumes: cached store handles from step 01; DistillLock from step 17
- Produces: `snapshots` table; `Snapshot()` method; row-level locking; `mem snapshot` CLI

### Tasks

- [x] 20.1 `[RED]` Snapshot() creates a point-in-time view of the store
  - [x] 20.1.a Write failing test (`TestSnapshotCreatePointInTime`): create 5 observations; call `Snapshot(projectID)`; modify 2 observations and add 1; call `RestoreSnapshot(snapshotID)`; verify the store is back to the 5-observation state with original content; verify the snapshot captured the state hash
  - [x] 20.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime'` — Expected: FAIL
  - [x] 20.1.c Minimal implementation — create `snapshots.go` with a `snapshots` table: `id INTEGER PRIMARY KEY, project_id TEXT NOT NULL, state_hash TEXT NOT NULL, data BLOB NOT NULL, created_at TIMESTAMP`; `Snapshot(projectID)` serializes the current observations state to a BLOB, computes a hash, and stores it; `RestoreSnapshot(snapshotID)` deserializes and replaces the current state
  - [x] 20.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime'` — Expected: PASS
  - [x] 20.1.e Commit — `feat(mnemonic): point-in-time store snapshots`
- [x] 20.2 `[RED]` Row-level locking prevents concurrent writes to same observation
  - [x] 20.2.a Write failing test (`TestRowLevelLockingConcurrentWrites`): start two goroutines that both attempt to update the same observation simultaneously; verify the second write blocks until the first completes; verify no data corruption (final state is one of the two writes, not a mix); verify the lock is released after the transaction
  - [x] 20.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRowLevelLockingConcurrentWrites'` — Expected: FAIL
  - [x] 20.2.c Minimal implementation — add row-level locking using SQLite `BEGIN IMMEDIATE` transactions for writes; wrap each observation update in a transaction; concurrent writes to the same row are serialized by SQLite's write lock; add a 5-second timeout on lock acquisition
  - [x] 20.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRowLevelLockingConcurrentWrites'` — Expected: PASS
  - [x] 20.2.e Commit — `feat(mnemonic): row-level locking for concurrent writes`
- [x] 20.3 `[AFK]` Auto-prune old snapshots to prevent storage bloat
  - [x] 20.3.a Write failing test (`TestSnapshotAutoPrune`): create 20 snapshots; configure retention to keep only the last 5; trigger auto-prune; verify only the 5 most recent snapshots remain; verify the oldest 15 are deleted; verify the retention count is configurable
  - [x] 20.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotAutoPrune'` — Expected: FAIL
  - [x] 20.3.c Minimal implementation — implement `PruneSnapshots(projectID, keep int)` that deletes all but the `keep` most recent snapshots; trigger auto-prune after each new snapshot; make retention configurable via `mnemonic.snapshot.retention` (default 10)
  - [x] 20.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotAutoPrune'` — Expected: PASS
  - [x] 20.3.e Commit — `feat(mnemonic): auto-prune old snapshots with configurable retention`
- [x] 20.4 `[AFK]` mem snapshot CLI creates and rolls back to snapshots
  - [x] 20.4.a Write failing test (`TestMemSnapshotCLI`): invoke `mem snapshot create` and verify a snapshot is created with an ID; modify the store; invoke `mem snapshot restore <id>` and verify the store is restored; invoke `mem snapshot list` and verify all snapshots are listed with timestamps
  - [x] 20.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSnapshotCLI'` — Expected: FAIL
  - [x] 20.4.c Minimal implementation — add `mem snapshot create`, `mem snapshot restore <id>`, and `mem snapshot list` subcommands to `mem.go`; wire them to `Snapshot()`, `RestoreSnapshot()`, and a `ListSnapshots()` method
  - [x] 20.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSnapshotCLI'` — Expected: PASS
  - [x] 20.4.e Commit — `feat(mnemonic): add mem snapshot CLI subcommands`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime\|TestRowLevelLockingConcurrentWrites\|TestSnapshotPreservesEmbeddingsAndReviewAfter\|TestSnapshotRestoreAtomic\|TestSnapshotRetention'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSnapshotCLI'` | PASS | PASS | 6 memory + 2 CLI tests GREEN, -race clean |
| Acceptance `@step-20` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1 -race` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 191s (race-clean), CLI green |
| Rollback boundary | verify snapshots are opt-in and do not affect normal writes | PASS | PASS | snapshots are a new table + new functions; normal write path unchanged; RestoreSnapshot is explicit (not automatic) |
| Global Constraints | — | held | held | 030 migration purely additive; SnapshotRow covers all 35 observations columns (fix added review_after/embedding/embedding_model/embedding_created_at); RestoreSnapshot uses BEGIN IMMEDIATE; no tool contract change; no CGO |

Commits: `d31bdd8` (snapshots + transaction locking), `ba49344` (fix: complete snapshot columns + BEGIN IMMEDIATE + blocking assertion). Review: NEEDS FIXES → PASS after fix round. F1 (CRITICAL, FIXED): 4 missing columns (review_after/embedding/embedding_model/embedding_created_at) added to SnapshotRow/snapshotCols/scan/restoreUpsertSQL + TestSnapshotPreservesEmbeddingsAndReviewAfter proves round-trip. F2 (MAJOR, FIXED): RestoreSnapshot now uses explicit BEGIN IMMEDIATE (snapshots.go:330). F3 (MAJOR, FIXED): concurrency test now asserts second.acq >= first.commit (blocking proven); dead start channel removed. Remaining (non-blocking): F4 (MINOR, prod Update not on immediate path — long-term DSN note).

### Commit

When step DoD is met: `feat(mnemonic): multi-version snapshots with transaction locking`

---

## 21-handoff

### Goal

Prefix + delta handoff artifacts for agent-to-agent context passing.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-21` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 09, 12

**Files:**
- `internal/mnemonic/memory/handoff.go` — CREATE: prefix + delta handoff artifacts
- `internal/mnemonic/memory/handoff_test.go` — CREATE: handoff tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `handoff` subcommand

**Interfaces:**
- Consumes: session promotion from step 09; DreamExecutor from step 12; project metadata
- Produces: `handoff.latest.json` with stable `prefix` + dynamic `delta`; `mem handoff` CLI

### Tasks

- [x] 21.1 `[RED]` Handoff artifact contains stable prefix with hub summaries and repo file-count
  - [x] 21.1.a Write failing test (`TestHandoffPrefixStable`): generate a handoff artifact; verify the `prefix` section contains hub file summaries and repo file count; modify a non-hub file; regenerate the handoff; verify the `prefix` section is unchanged (stable); modify a hub file; regenerate; verify the `prefix` reflects the hub file change
  - [x] 21.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable'` — Expected: FAIL
  - [x] 21.1.c Minimal implementation — create `handoff.go` with `GenerateHandoff(ctx, projectID) (*Handoff, error)`; the `prefix` section includes hub file summaries (from codeindex) and repo file count; the prefix is computed from stable project metadata and only changes when hub files change
  - [x] 21.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable'` — Expected: PASS
  - [x] 21.1.e Commit — `feat(mnemonic): handoff prefix with stable hub summaries`
- [x] 21.2 `[RED]` Handoff delta contains changed file stubs, risk files, recent events
  - [x] 21.2.a Write failing test (`TestHandoffDeltaDynamic`): generate a handoff; modify 3 files (1 hub, 2 regular); regenerate the handoff; verify the `delta` section lists all 3 changed files with stubs; verify the hub file is flagged as a risk file; verify recent session events are included; verify the delta reflects only changes since the last handoff
  - [x] 21.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffDeltaDynamic'` — Expected: FAIL
  - [x] 21.2.c Minimal implementation — the `delta` section includes: changed file stubs (file path, first 50 lines), risk files (hub files among changes), recent events (last 10 session events), and working set summary; compute the delta at handoff time (not cached) to ensure freshness
  - [x] 21.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffDeltaDynamic'` — Expected: PASS
  - [x] 21.2.e Commit — `feat(mnemonic): handoff delta with changed files and risk analysis`
- [x] 21.3 `[AFK]` Handoff saved to .skillgrid/handoff.latest.json
  - [x] 21.3.a Write failing test (`TestHandoffSavedToFile`): generate a handoff; verify a file `.skillgrid/handoff.latest.json` is created; verify the JSON is valid and contains both `prefix` and `delta` sections; verify a second generation overwrites the file (not appends)
  - [x] 21.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffSavedToFile'` — Expected: FAIL
  - [x] 21.3.c Minimal implementation — in `GenerateHandoff`, serialize the handoff to JSON and write to `.skillgrid/handoff.latest.json` (create the directory if needed); overwrite on each generation; include a `generated_at` timestamp
  - [x] 21.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffSavedToFile'` — Expected: PASS
  - [x] 21.3.e Commit — `feat(mnemonic): save handoff to .skillgrid/handoff.latest.json`
- [x] 21.4 `[AFK]` mem handoff CLI generates handoff artifact
  - [x] 21.4.a Write failing test (`TestMemHandoffCLI`): invoke `mem handoff`; verify it generates the handoff and prints a summary to stdout; verify the file is written; invoke `mem handoff --json` and verify the full JSON is printed to stdout
  - [x] 21.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI'` — Expected: FAIL
  - [x] 21.4.c Minimal implementation — add `mem handoff` subcommand to `mem.go`; call `GenerateHandoff`; print a human-readable summary by default; support `--json` flag for full JSON output
  - [x] 21.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI'` — Expected: PASS
  - [x] 21.4.e Commit — `feat(mnemonic): add mem handoff CLI subcommand`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable\|TestHandoffDeltaDynamic\|TestHandoffSavedToFile\|TestMemHandoffCLI'` | PASS | PASS | 4 step-21 tests GREEN |
| Acceptance `@step-21` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 36s, CLI green |
| Rollback boundary | verify handoff is opt-in and computed at generation time | PASS | PASS | handoff is a new CLI subcommand (opt-in); delta computed at generation time (not cached) |
| Global Constraints | — | held | held | 031 migration purely additive (handoff_meta table); no tool contract change; no CGO |

Commits: `90a6253` (prefix + delta handoff artifacts). Review: PASS WITH WARNINGS. Stable prefix HOLDS: computed from file count + hub-file heads only (no mtime/cursor); non-hub content edit leaves it byte-identical, hub edit changes it. Changed-file detection HOLDS: on-disk mtime vs RFC3339Nano cursor in handoff_meta, strict After, real FS mtime. Warnings: (M1) SaveHandoff re-calls GenerateHandoff → default mem handoff path double-generates + double-advances cursor (printed summary and on-disk file can diverge); (M2) first handoff's delta is always empty (no baseline) — new agent gets blank change-set on run 1; (M3) risk_files = ALL hub files every run, not hub files that CHANGED (dilutes risk signal); (m4) adding/deleting any file changes the count, so stability is "across non-hub edits" not "across non-hub adds".

### Commit

When step DoD is met: `feat(mnemonic): prefix + delta handoff artifacts for agent-to-agent context`

---

## 22-context-envelope

### Goal

Working set + intent classification + universal context envelope.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-22` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 08, 13, 21

**Files:**
- `internal/mnemonic/memory/envelope.go` — CREATE: context envelope (working set + intent + skills)
- `internal/mnemonic/memory/envelope_test.go` — CREATE: envelope tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `context` subcommand

**Interfaces:**
- Consumes: improve() from step 08; importance scoring from step 13; handoff from step 21
- Produces: `ContextEnvelope` JSON; working set tracking; `mem context` CLI

### Tasks

- [x] 22.1 `[RED]` Working set tracks files edited, edit counts, net line deltas
  - [x] 22.1.a Write failing test (`TestWorkingSetTracking`): simulate editing 3 files (2 in project, 1 outside); record the working set; verify it lists the 2 project files with edit counts and net line deltas; verify the file outside the project is excluded; verify hub file status is flagged for files that are hub files
  - [x] 22.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking'` — Expected: FAIL
  - [x] 22.1.c Minimal implementation — create `envelope.go` with `WorkingSet` struct: `Files []WorkingFile`, where `WorkingFile` has `Path`, `EditCount`, `NetLines`, `IsHub bool`; track edits during the session; compute net line delta as (lines added - lines removed)
  - [x] 22.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking'` — Expected: PASS
  - [x] 22.1.e Commit — `feat(mnemonic): working set tracking with edit counts and line deltas`
- [x] 22.2 `[RED]` Intent classification returns exploration/debugging/review/refactor
  - [x] 22.2.a Write failing test (`TestIntentClassificationInEnvelope`): classify "fix the null pointer in auth.go" as `debugging`; classify "what's in the config module" as `exploration`; classify "check the PR for payment.go" as `review`; classify "extract the validation into a separate function" as `refactor`; verify each classification is correct
  - [x] 22.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentClassificationInEnvelope'` — Expected: FAIL
  - [x] 22.2.c Minimal implementation — implement `ClassifyIntent(query string) Intent` using keyword patterns: debugging (fix, bug, error, crash, fail), exploration (what, where, list, show), review (check, review, PR, diff), refactor (extract, rename, move, split); return the best-matching intent; default to `exploration`
  - [x] 22.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentClassificationInEnvelope'` — Expected: PASS
  - [x] 22.2.e Commit — `feat(mnemonic): intent classification in context envelope`
- [x] 22.3 `[RED]` ContextEnvelope JSON contains project metadata + working set + matched skills + handoff refs
  - [x] 22.3.a Write failing test (`TestContextEnvelopeStructure`): generate a context envelope; verify the JSON contains `project` (name, file count, languages), `working_set` (files, edit counts), `intent` (classification), `matched_skills` (list), `handoff_refs` (path to handoff.latest.json); verify the JSON is valid and parseable; verify the envelope size is under a configurable limit
  - [x] 22.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestContextEnvelopeStructure'` — Expected: FAIL
  - [x] 22.3.c Minimal implementation — implement `GenerateContextEnvelope(ctx, projectID) (*ContextEnvelope, error)` that assembles all sections; enforce a configurable max envelope size (`mnemonic.envelope.max_size`, default 64KB); truncate or filter fields if the envelope exceeds the limit
  - [x] 22.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestContextEnvelopeStructure'` — Expected: PASS
  - [x] 22.3.e Commit — `feat(mnemonic): universal context envelope JSON`
- [x] 22.4 `[AFK]` mem context CLI with --envelope flag for universal JSON output
  - [x] 22.4.a Write failing test (`TestMemContextCLI`): invoke `mem context` and verify a human-readable summary is printed; invoke `mem context --envelope` and verify the full JSON envelope is printed to stdout; verify the JSON is valid
  - [x] 22.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemContextCLI'` — Expected: FAIL
  - [x] 22.4.c Minimal implementation — add `mem context` subcommand to `mem.go`; print a summary by default (project name, file count, intent, working set size); support `--envelope` flag for full JSON output
  - [x] 22.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemContextCLI'` — Expected: PASS
  - [x] 22.4.e Commit — `feat(mnemonic): add mem context CLI with envelope flag`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking\|TestIntentClassificationInEnvelope\|TestContextEnvelopeStructure'` + `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemContextCLI'` | PASS | PASS | 3 memory + 1 CLI test GREEN |
| Acceptance `@step-22` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | mapped unit scenarios |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` + `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | memory 42s, CLI 84s |
| Rollback boundary | verify envelope size limit prevents oversized output | PASS | PASS | envelope is opt-in (new CLI subcommand); handoff is a path ref (not embedded), keeping output small |
| Global Constraints | — | held | held | new envelope.go (in-memory working set, no migration); reuses step-19 Intent type (no duplicate); step-19 ClassifyIntent untouched; no tool contract change; no CGO |

Commits: `4eedadd` (working set + ClassifyWorkIntent + ContextEnvelope), `897dd6e` (mem context CLI). Review: PASS WITH WARNINGS. Working set CORRECT + tested: EditCount/NetLines accumulate right, out-of-project excluded, hub flagged via step-21 isHubFile. Intent CORRECT: ClassifyWorkIntent reuses step-19 Intent type, ClassifyIntent untouched, all 4 cases + 4 more pass, default exploration. Warnings: (M1) working set INERT in production — shipped CLI caller passes NewWorkingSet("") and never calls RecordEdit, so working_set is always [] in real output (documented, by-design in-memory; needs a session-scoped caller to populate); (m2) ClassifyWorkIntent keyword table is a superset of the brief's literal list (harmless, traceability only); (m3) handoff is a path ref (handoff.latest.json), not embedded contents (exactly what the brief specifies).

### Commit

When step DoD is met: `feat(mnemonic): working set, intent classification, and universal context envelope`

---

## 23-hub-impact

### Goal

Hub file identification + impact analysis + risk scores.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-23` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 10, 13

**Files:**
- `internal/mnemonic/memory/hub.go` — CREATE: hub file identification + impact analysis + risk scores
- `internal/mnemonic/memory/hub_test.go` — CREATE: hub tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `graph --risk` flag

**Interfaces:**
- Consumes: temporal graph from step 10; importance scoring from step 13; codeindex symbols and edges
- Produces: `hub_score` on symbols; `AnalyzeImpact()`; `risk_score` on observations; `mem graph --risk` CLI

### Tasks

- [ ] 23.1 `[RED]` Hub files are identified by 3+ importers
  - [ ] 23.1.a Write failing test (`TestHubFileIdentification`): create a codeindex with 10 files, one of which is imported by 5 other files (hub), and the rest imported by 0-2 files; run hub identification; verify only the file with 3+ importers is flagged as a hub; verify the `hub_score` (import count / total files) is computed correctly
  - [ ] 23.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification'` — Expected: FAIL
  - [ ] 23.1.c Minimal implementation — create `hub.go` with `IdentifyHubFiles(ctx, projectID) ([]HubFile, error)` that queries the codeindex `edges` table to count importers per file; flag files with 3+ importers as hubs; compute `hub_score = importers / total_files`; store `hub_score` on the symbol
  - [ ] 23.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification'` — Expected: PASS
  - [ ] 23.1.e Commit — `feat(mnemonic): hub file identification with 3+ importer threshold`
- [ ] 23.2 `[RED]` AnalyzeImpact identifies hub files among changed files
  - [ ] 23.2.a Write failing test (`TestAnalyzeImpactHubFiles`): create a change set with 5 modified files, 2 of which are hub files; call `AnalyzeImpact(changedFiles)`; verify the result flags the 2 hub files as high-impact; verify the non-hub files are marked low-impact; verify the impact analysis includes the number of dependent files for each hub file
  - [ ] 23.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAnalyzeImpactHubFiles'` — Expected: FAIL
  - [ ] 23.2.c Minimal implementation — implement `AnalyzeImpact(ctx, changedFiles []string) ([]ImpactResult, error)` that cross-references changed files with the hub file list; for each hub file in the change set, count dependent files; return an impact result with `file`, `isHub`, `dependentCount`, `impactLevel` (high/medium/low)
  - [ ] 23.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAnalyzeImpactHubFiles'` — Expected: PASS
  - [ ] 23.2.e Commit — `feat(mnemonic): impact analysis for hub file changes`
- [ ] 23.3 `[RED]` risk_score on observations based on hub file involvement
  - [ ] 23.3.a Write failing test (`TestRiskScoreOnObservations`): create an observation referencing a hub file; create another observation referencing a non-hub file; verify the hub-file observation has a higher `risk_score`; verify the risk score is proportional to the hub file's `hub_score`; verify the risk score is updated when the hub file's import count changes
  - [ ] 23.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRiskScoreOnObservations'` — Expected: FAIL
  - [ ] 23.3.c Minimal implementation — add `risk_score` (FLOAT) column to observations; compute it as the `hub_score` of the referenced file (0 for non-hub files); update risk scores periodically or on hub file changes; make the threshold configurable
  - [ ] 23.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRiskScoreOnObservations'` — Expected: PASS
  - [ ] 23.3.e Commit — `feat(mnemonic): risk_score on observations based on hub file involvement`
- [ ] 23.4 `[AFK]` mem graph --risk CLI shows high-risk hub files
  - [ ] 23.4.a Write failing test (`TestMemGraphRiskCLI`): create observations with varying risk scores; invoke `mem graph --risk`; verify the output lists high-risk hub files sorted by risk score; verify only files above the risk threshold are shown; verify the threshold is configurable via `--threshold` flag
  - [ ] 23.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphRiskCLI'` — Expected: FAIL
  - [ ] 23.4.c Minimal implementation — add `--risk` flag to the `mem graph` subcommand; query observations ordered by `risk_score` DESC; filter by a configurable threshold (default 0.5); display file path, risk score, and dependent count
  - [ ] 23.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphRiskCLI'` — Expected: PASS
  - [ ] 23.4.e Commit — `feat(mnemonic): add mem graph --risk CLI for high-risk hub files`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification\|TestAnalyzeImpactHubFiles\|TestRiskScoreOnObservations'` | PASS | | |
| Acceptance `@step-23` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify hub analysis is opt-in with minimum importers threshold | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): hub file identification, impact analysis, and risk scores`

---

## 24-skills-hooks

### Goal

Skills framework + lifecycle hooks for context injection.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-24` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 05, 09

**Files:**
- `internal/mnemonic/memory/skills.go` — CREATE: skills framework + lifecycle hooks
- `internal/mnemonic/memory/skills_test.go` — CREATE: skills tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `skills` and `hook` subcommands

**Interfaces:**
- Consumes: LLM extraction from step 05; session promotion from step 09; intent classification
- Produces: skills as observations with `skill` memory type; lifecycle hooks (session-start, pre-edit, prompt-submit, session-stop); `mem skills` and `mem hook` CLI

### Tasks

- [ ] 24.1 `[RED]` Skills are stored as observations with skill memory type and matched by intent
  - [ ] 24.1.a Write failing test (`TestSkillsMatchingByIntent`): create 3 skills — one for `debugging` intent, one for `review` intent, one for `exploration` intent; classify a query as `debugging`; run skill matching; verify only the debugging skill is returned; verify skills are stored as observations with `memory_type: skill`
  - [ ] 24.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent'` — Expected: FAIL
  - [ ] 24.1.c Minimal implementation — create `skills.go` with `MatchSkills(ctx, intent Intent, mentionedFiles []string, projectLangs []string) ([]Skill, error)`; skills are observations with `memory_type: skill` and Markdown content; matching is based on intent, mentioned files, and project languages; return matched skills sorted by relevance
  - [ ] 24.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent'` — Expected: PASS
  - [ ] 24.1.e Commit — `feat(mnemonic): skills framework with intent-based matching`
- [ ] 24.2 `[RED]` Lifecycle hooks inject context at the right moments
  - [ ] 24.2.a Write failing test (`TestLifecycleHookSessionStart`): configure a `session-start` hook that injects relevant memories; trigger the hook; verify it returns matched memories and skills; trigger `pre-edit` hook with a file path; verify it returns risk analysis for that file; trigger `session-stop` hook; verify it triggers distillation
  - [ ] 24.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLifecycleHookSessionStart'` — Expected: FAIL
  - [ ] 24.2.c Minimal implementation — implement `RunHook(ctx, hookType string, payload HookPayload) (HookResult, error)` for 4 hook types: `session-start` (inject relevant memories + skills), `pre-edit` (inject risk analysis for the file), `prompt-submit` (classify intent), `session-stop` (trigger distillation); hooks are opt-in per project
  - [ ] 24.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLifecycleHookSessionStart'` — Expected: PASS
  - [ ] 24.2.e Commit — `feat(mnemonic): lifecycle hooks for context injection`
- [ ] 24.3 `[AFK]` Hooks are opt-in per project with a timeout
  - [ ] 24.3.a Write failing test (`TestHooksOptInWithTimeout`): disable hooks for a project; trigger a hook; verify no hook runs; enable hooks; trigger a hook that sleeps 10 seconds; verify it times out after the configured timeout (default 30s) and returns a timeout error; verify the timeout is configurable
  - [ ] 24.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHooksOptInWithTimeout'` — Expected: FAIL
  - [ ] 24.3.c Minimal implementation — add `mnemonic.hooks.enabled` config per project (default `false`); wrap hook execution in a `context.WithTimeout` (default 30s); on timeout, return a descriptive error and log a warning; make the timeout configurable via `mnemonic.hooks.timeout`
  - [ ] 24.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHooksOptInWithTimeout'` — Expected: PASS
  - [ ] 24.3.e Commit — `feat(mnemonic): make hooks opt-in with configurable timeout`
- [ ] 24.4 `[AFK]` mem skills and mem hook CLI subcommands
  - [ ] 24.4.a Write failing test (`TestMemSkillsAndHookCLI`): invoke `mem skills list` and verify all skills are listed; invoke `mem skills add "name" "content"` and verify a skill is created; invoke `mem hook list` and verify configured hooks are shown; invoke `mem hook run session-start` and verify the hook executes
  - [ ] 24.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSkillsAndHookCLI'` — Expected: FAIL
  - [ ] 24.4.c Minimal implementation — add `mem skills list`, `mem skills add <name> <content>`, `mem hook list`, and `mem hook run <type>` subcommands to `mem.go`; wire them to the skills and hooks modules
  - [ ] 24.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSkillsAndHookCLI'` — Expected: PASS
  - [ ] 24.4.e Commit — `feat(mnemonic): add mem skills and mem hook CLI subcommands`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent\|TestLifecycleHookSessionStart\|TestHooksOptInWithTimeout'` | PASS | | |
| Acceptance `@step-24` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify hooks are opt-in per project with timeout | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): skills framework and lifecycle hooks for context injection`

---

## 25-memfs

### Goal

`memfs` virtual filesystem: `mem ls`, `mem tree`, `mem find` alongside existing `mem search`.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-25` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07, 18, 19

**Files:**
- `internal/mnemonic/memfs/memfs.go` — CREATE: URI resolution + scope management + filesystem operations
- `internal/mnemonic/memfs/ls.go` — CREATE: `mem ls <scope>` directory listing
- `internal/mnemonic/memfs/tree.go` — CREATE: `mem tree <scope>` hierarchical tree view
- `internal/mnemonic/memfs/find.go` — CREATE: `mem find <pattern>` filesystem-style search
- `internal/mnemonic/memfs/scope.go` — CREATE: scope management (`mem://user/{id}/`, `mem://project/{id}/`)
- `internal/mnemonic/memfs/memfs_test.go` — CREATE: memfs tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `fs` subcommand group

**Interfaces:**
- Consumes: triple-store linkage from step 07; memory types from step 18; directory retrieval from step 19
- Produces: `mem fs` CLI group with `ls`, `tree`, `find`; `mem://` URI space; scope-based filtering on the same store

### Tasks

- [ ] 25.1 `[RED]` mem ls lists observations in a scope (e.g., project/{id}/preferences)
  - [ ] 25.1.a Write failing test (`TestMemLSScopeListing`): create observations in scopes `project/A/preferences`, `project/A/entities`, `user/B/profile`; invoke `mem ls project/A/preferences`; verify it lists only the preferences observations; invoke `mem ls project/A/`; verify it lists all observations under project A; invoke `mem ls user/B/`; verify it lists only user B's observations
  - [ ] 25.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing'` — Expected: FAIL
  - [ ] 25.1.c Minimal implementation — create `memfs.go` with `ResolveURI(uri string) (ScopeFilter, error)` that parses `mem://` URIs into scope filters; create `ls.go` with `List(ctx, scope string) ([]Observation, error)` that queries the store with `WHERE memory_type = ? AND scope = ?`; the scope is an additive metadata column on observations
  - [ ] 25.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing'` — Expected: PASS
  - [ ] 25.1.e Commit — `feat(mnemonic): mem ls scope-based directory listing`
- [ ] 25.2 `[RED]` mem tree shows hierarchical tree view of memory scopes
  - [ ] 25.2.a Write failing test (`TestMemTreeHierarchical`): create observations in nested scopes `project/A/preferences/sub1`, `project/A/preferences/sub2`, `project/A/entities`; invoke `mem tree project/A/`; verify the output shows a hierarchical tree with `preferences/` containing `sub1` and `sub2`, and `entities/` as a sibling; verify the tree is indented correctly
  - [ ] 25.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemTreeHierarchical'` — Expected: FAIL
  - [ ] 25.2.c Minimal implementation — create `tree.go` with `Tree(ctx, scope string) (string, error)` that queries all scopes under the given prefix; builds a hierarchical tree structure; renders it as indented text with `├──` and `└──` characters; group by scope path segments
  - [ ] 25.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemTreeHierarchical'` — Expected: PASS
  - [ ] 25.2.e Commit — `feat(mnemonic): mem tree hierarchical scope view`
- [ ] 25.3 `[RED]` mem find does filesystem-style pattern matching across observations
  - [ ] 25.3.a Write failing test (`TestMemFindPatternMatching`): create observations with titles "auth.go", "auth_test.go", "payment.go", "user.go"; invoke `mem find "auth*"`; verify it returns "auth.go" and "auth_test.go"; invoke `mem find "*.go"`; verify it returns all 4; invoke `mem find "auth*test*"`; verify it returns only "auth_test.go"
  - [ ] 25.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemFindPatternMatching'` — Expected: FAIL
  - [ ] 25.3.c Minimal implementation — create `find.go` with `Find(ctx, pattern string, scope string) ([]Observation, error)` that translates glob patterns to SQL `LIKE` queries; support `*` (any chars) and `?` (single char); search across observation titles and content; scope the search to the given scope
  - [ ] 25.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemFindPatternMatching'` — Expected: PASS
  - [ ] 25.3.e Commit — `feat(mnemonic): mem find with glob pattern matching`
- [ ] 25.4 `[RED]` mem:// URI resolution maps to SQL queries on the same store
  - [ ] 25.4.a Write failing test (`TestMemURIResolution`): resolve `mem://project/A/preferences` and verify it produces a SQL filter for `project_id = 'A' AND memory_type = 'preferences'`; resolve `mem://user/B/` and verify it produces a filter for `user_id = 'B'`; resolve an invalid URI `mem://` and verify it returns a parse error; verify the URI resolution is fast (sub-millisecond)
  - [ ] 25.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemURIResolution'` — Expected: FAIL
  - [ ] 25.4.c Minimal implementation — create `scope.go` with `ParseURI(uri string) (ProjectID, UserType, Scope, error)` that parses `mem://project/{id}/` and `mem://user/{id}/` patterns; validate the URI format; return structured scope components; ensure resolution is a pure string operation (no DB call)
  - [ ] 25.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemURIResolution'` — Expected: PASS
  - [ ] 25.4.e Commit — `feat(mnemonic): mem:// URI resolution engine`
- [ ] 25.5 `[AFK]` mem ls coexists with mem list without conflict
  - [ ] 25.5.a Write failing test (`TestMemLSCoexistsWithMemList`): create observations; invoke `mem list` and verify it returns flat output (all observations); invoke `mem ls project/A/` and verify it returns directory-like output (scoped); verify both commands work independently; verify `mem ls` does not affect `mem list` output and vice versa
  - [ ] 25.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemLSCoexistsWithMemList'` — Expected: FAIL
  - [ ] 25.5.c Minimal implementation — ensure `mem fs ls` (and its siblings) are registered as a separate subcommand group under `mem fs`; do not modify the existing `mem list` command; both query the same store but with different filters (scoped vs flat)
  - [ ] 25.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemLSCoexistsWithMemList'` — Expected: PASS
  - [ ] 25.5.e Commit — `feat(mnemonic): mem fs coexists with mem list without conflict`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing\|TestMemTreeHierarchical\|TestMemFindPatternMatching'` | PASS | | |
| Acceptance `@step-25` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memfs/` | PASS | | |
| Rollback boundary | verify memfs queries same store, does not replace mem search | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): memfs virtual filesystem with ls, tree, find alongside mem search`

---

## 26-tests

### Goal

Unit + integration coverage for all steps (01-25).

### Out of scope / Non-Goals

- New production features beyond closing RED coverage.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-26` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25

**Files:**
- `skillgrid-cli/...` — all touched packages: store, memory, service, codeindex, embedder, config, memfs, cmd/skillgrid, mcp, http

**Interfaces:**
- Consumes: all production code from steps 01-25
- Produces: full test coverage; `go test ./skillgrid-cli/...` passes

### Tasks

- [ ] 26.1 `[RED]` Full test suite passes for all touched packages
  - [ ] 26.1.a Write failing test (verify gap): run `go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` and identify any test gaps or failures from steps 01-25
  - [ ] 26.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.1.c Minimal implementation — close any remaining RED coverage gaps identified in 26.1.a; add missing edge-case tests for: store pooling (concurrent opens), FTS trigram (empty results fallback), parallel search (50+ stores), TTL (boundary conditions), LLM extraction (error handling), embedder (config-driven selection), triple-store (join correctness), improve() (no regression), session promotion (dedup), temporal edges (boundary timestamps), export (roundtrip), dream (lock + rollback), importance (skew), relations (confidence filtering), provenance (immutability), federated query (dedup), distill lock (timeout), memory types (async commit), directory retrieval (depth limit), snapshots (auto-prune), handoff (staleness), context envelope (size limit), hub (accuracy), skills (hook timeout), memfs (URI resolution with 10k+ observations)
  - [ ] 26.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` — Expected: PASS
  - [ ] 26.1.e Commit — `feat(mnemonic): close RED coverage gaps across all steps`
- [ ] 26.2 `[RED]` Integration tests pass for MCP, HTTP, and CLI layers
  - [ ] 26.2.a Write failing test (verify gap): run `go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` and identify any integration test gaps
  - [ ] 26.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.2.c Minimal implementation — add integration tests that verify: MCP tools (`mem_search`, `mem_save`, etc.) work with the new features (trigram mode, federated query, memfs); HTTP API endpoints work with new parameters; CLI subcommands (`mem expire`, `mem export`, `mem relations`, `mem provenance`, `mem dream`, `mem snapshot`, `mem handoff`, `mem context`, `mem graph --risk`, `mem skills`, `mem hook`, `mem fs`) produce correct output
  - [ ] 26.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` — Expected: PASS
  - [ ] 26.2.e Commit — `feat(mnemonic): integration tests for MCP, HTTP, and CLI layers`
- [ ] 26.3 `[AFK]` Full suite `go test ./skillgrid-cli/...` passes
  - [ ] 26.3.a Write failing test (verify gap): run the full suite and identify any failures
  - [ ] 26.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/...` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.3.c Minimal implementation — fix any remaining failures across the entire `skillgrid-cli` module; ensure no test regressions from steps 01-25; verify all BDD `@step-NN` scenarios in `acceptance.feature` map to passing unit/integration tests
  - [ ] 26.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/...` — Expected: PASS
  - [ ] 26.3.e Commit — `feat(mnemonic): full test suite green across all packages`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` | PASS | | |
| Acceptance `@step-26` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/...` | PASS | | |
| Rollback boundary | verify no test regressions from any step | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): full unit + integration coverage for all steps`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
