# Change: 014-mnemonic-performance — Mnemonic Performance & Capability Hardening

> **STATUS:** `draft` (2026-09-10)
>
> **For agentic workers:** REQUIRED: follow `.agents/skills/_shared/conventions/sdd-structure.md`. This file is WHY + HOW (former intent + plan). Spec phase instantiates `tasks.md` + `acceptance.feature` from the Step Blueprint and per-step WHAT below.

**Goal:** Harden Mnemonic's performance, reliability, and retrieval capabilities by addressing the top weaknesses in the current architecture: N+1 store opens, no FTS5 trigram support, SQLite write contention, missing cross-project search, no memory TTL, regex-based passive extraction, and single-embedder dependency. Additionally, adopt the triple-database design (Relational + Vector + Graph cross-linked) proven by cognee (30k+ stars) to unify observation metadata, embeddings, and codeindex graph into a single coherent memory system where every node has both a vector and a graph representation.

**Architecture:** Layer changes across the existing `store`, `memory`, `service`, `codeindex`, and `config` packages. Add a connection-pool-aware store handle, enhance FTS query parsing, introduce a unified cross-project search, add TTL/soft-expiry defaults, replace regex passive extraction with an LLM-backed extractor, support multiple embedder models, and critically — **cross-link the relational SQLite store with vector embeddings and codeindex graph nodes** so every observation has both a semantic vector and a structural graph representation. Session memory promotes to permanent graph automatically. Observations self-improve via a feedback loop based on retrieval usage. All changes are additive — no existing schema rewrites.

**Tech stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), ONNX (`onnxer`), optional external embedder (OpenAI-compatible), optional Ollama.

**Research:** analysis of current mnemonic weaknesses (N+1 opens, FTS limitations, no cross-project search, no TTL, regex extraction, single embedder) + cognee (topoteretes/cognee, 30k★) triple-database architecture, `improve()` self-improvement, temporal knowledge graphs, session-to-graph promotion, and feedback-loop design patterns.

**Ticket:** none

**Depends on:** `003-mnemonic-tiered-storage`, `005-mnemonic-hybrid-code-intelligence`, `013-mnemonic-layered-memory-governance`

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

## Definition of Done

This change is done only when **all** of the following are true:

- [ ] `service.Open()` returns a pooled handle — no N+1 store opens per request
- [ ] FTS5 trigram/wildcard queries work via `mem search` and MCP `mem_search`
- [ ] Cross-project search (`mem search all_projects=true`) runs parallel, not sequential
- [ ] Memory TTL soft-expiry is on by default (configurable, 7-day default)
- [ ] Passive extraction uses LLM backend (regex fallback preserved)
- [ ] Multi-embedder support: `onnx` | `external` | `off` (existing) + `ollama` | `local` options
- [ ] SQLite WAL write contention resolved via connection pooling
- [ ] **Triple-store cross-linkage**: every observation has a vector embedding AND a codeindex graph node reference; queries traverse observation→symbol→embedding→related observations
- [ ] **Self-improvement feedback loop**: `improve()` re-weights observations based on `retrieval_usage` (already tracked, now acted on)
- [ ] **Session-to-graph promotion**: L0/L1 session summaries automatically promote to L2/L3 permanent graph nodes on session end
- [ ] **Temporal knowledge graph**: graph edges carry `valid_from`/`valid_to` timestamps tracking how relationships evolved
- [ ] **Portable export**: `mem export` produces JSON with full observation content, graph edges, and embeddings (COGX-inspired)
- [ ] **Memfs**: `mem ls`, `mem tree`, `mem find` work alongside `mem search`; `mem://` URIs resolve to stored observations
- [ ] `go test ./skillgrid-cli/...` passes
- [ ] Every Step Blueprint entry has a matching section in `tasks.md` with Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing `@happy`, `@edge`, and `@failure` scenarios
- [ ] Applicable threat-matrix rows have RED coverage that passed
- [ ] Testing strategy commands below are green
- [ ] Rollback path below is still valid (or N/A documented)
- [ ] Change archived under `docs/skillgrid/archive/014-mnemonic-performance/`

---

## Problem / why

Mnemonic is the memory backbone for multi-agent workflows, but six weaknesses limit production reliability:

1. **N+1 store opens** — every `service.Open()` creates a new SQLite connection; a single `mem search` call with 20 results opens 20+ connections, causing WAL contention and latency
2. **No FTS5 trigram/wildcard** — `buildFTSQuery` does exact phrase matching only; users cannot search partial identifiers or use wildcards
3. **Sequential cross-project search** — `SearchObservationsAll` iterates stores one-by-one; with 10+ projects, this blocks until all finish
4. **No memory TTL by default** — observations live forever unless explicitly expired; the `expires_at` column exists but is never set automatically
5. **Regex-based passive extraction** — `extractLearnings` uses brittle regex; it misses nuanced learnings and cannot handle free-form text well
6. **Single embedder model** — only `onnx` (nomic-embed-code) or `external` are supported; no `ollama`/`local` options for offline users
7. **No cross-linked store** — observations live in relational SQLite, embeddings live in `path_embeddings`, and codeindex graph nodes live separately in `symbols`/`edges`. No query can traverse from "this observation" → "related code symbols" → "their embeddings" → "related observations". Cognee proved this triple-database design (Relational + Vector + Graph, all cross-linked) works at 30k+ stars.

## Target users

- **Agent orchestrator** — runs many sessions per project; N+1 opens and WAL contention cause latency spikes
- **Operator** — searches across many projects; sequential cross-project search blocks the UI
- **Offline/air-gapped agent** — needs `ollama`/`local` embedder support
- **All users** — benefit from TTL soft-expiry keeping the store clean

## Business rules

- All changes are additive — no existing schema rewrites, no `009_*` or earlier migrations touched
- The `mem_*` / `code_*` / `web_*` tool contracts must not change shape
- SQLite store files remain per-project; no migration to a shared database
- The `modernc.org/sqlite` driver stays; no CGO or external SQLite binaries
- TTL defaults to 7 days; operators can configure via `mnemonic.retrieval_budget` or env var
- Passive extraction LLM backend is opt-in; regex fallback always available
- Embedder selection is config-driven via `indexing.yaml`; no hard-coded model choices
- `improve()` re-weighting is opt-in via `mnemonic.improve` config; disabled by default
- Triple-store cross-linkage adds columns but doesn't change existing query paths
- `graph_ref` defaults to NULL if codeindex symbol not found; no broken references
- Session promotion only fires on `SessionEnd` with a valid `LayerSummary` (quality threshold)
- Temporal edges default to `valid_to=NULL` (active until explicitly expired)

## In scope
- **Connection pooling** — `store.Open()` returns a pooled handle; `ProjectHandle` carries a session-scoped connection; `SetMaxOpenConns` set to 1 per project store (unchanged), but the `service.Service` caches and reuses open handles
- **FTS trigram enhancement** — extend `buildFTSQuery` to support trigram tokenization and wildcard (`*`) suffixes; add `trigram` and `prefix` match modes
- **Parallel cross-project search** — `SearchObservationsAll` uses goroutines to search all stores concurrently; merge results by cross-store rank
- **TTL soft-expiry defaults** — `expires_at` auto-set on save (7-day default); `TTLSoftExpiry` / `TTLRetire` become operational with a scheduled cleanup
- **LLM-backed passive extraction** — add `extraction.go` module with LLM-based learning extraction; keep regex fallback; `extraction.go` lives in `internal/mnemonic/memory/`
- **Multi-embedder** — add `ollama` and `local` embedder providers; `embedder.go` extended with new provider types; `config/load.go` supports new `provider` values
- **WAL write contention** — `store.Open()` caches open handles in a `sync.Map` keyed by project ID; subsequent opens return the cached handle; `Close` decrements refcount
- **Triple-store cross-linkage** — add `graph_ref` column to observations pointing to codeindex symbol ID; add `embedding_blob` column to `long_term_memories`; create `symbol_embeddings` table; queries traverse observation→symbol→embedding→related observations via SQL JOIN; every observation has both a vector and a graph representation
- **Self-improvement feedback loop** — `improve()` reads `retrieval_usage` from observations (already tracked); high-usage observations boosted in `mem_search` ordering; never-accessed observations decay in rank
- **Session-to-graph promotion** — on `SessionEnd`, if session has a summary, auto-create a permanent graph node linking session observations into the codeindex graph
- **Temporal knowledge graph** — add `valid_from`/`valid_to` columns to `edges` table in codeindex; expired edges hidden from queries but preserved for history
- **Portable export** — `ExportProject(ctx, projectID)` produces JSON with observations, graph edges, and embeddings (COGX-inspired); `mem export` CLI subcommand
- **Dream Executor** — decompose `DistillSession` into `consolidate` (merge facts), `synthesize` (create summaries), `prune` (remove low-importance data); `DreamLockService` ensures single dream per project + rollback
- **Importance scoring (AKL)** — Adaptive Knowledge Lifecycle: importance score + maturity tier + recency decay for each observation; feeds query-time ranking
- **Explicit relations** — `@relation` annotations between observations (e.g., `mentions`, `depends_on`, `contradicts`, `supports`); stored as typed edges in the memory store
- **Provenance tracking** — every observation carries a `provenance` chain: `session_id → curate_command → source_files → LLM_reasoning`
- **Federated query** — evolve parallel search into federated cross-project query with cross-project importance ranking and dedup
- **Distill lock + rollback** — `DistillLockService` ensures only one distillation per project; `DistillRollback` reverts on failure
- **Memory types** — typed categories (`profile`, `preferences`, `entities`, `events`, `identity`, `soul`, `cases`, `trajectories`, `experiences`); LLM dedup before write; async two-phase session commit (sync write + async LLM extraction)
- **Directory retrieval** — directory-level recursive retrieval with drill-down; finds highest-scoring directory first then drills down; observable retrieval trajectory preserved for debugging
- **Snapshots** — multi-version snapshots + transaction locking for concurrent access; point-in-time store state
- **Handoff artifacts** — prefix (stable: hub summaries, repo file-count) + delta (dynamic: changed file stubs, risk files, recent events) handoff artifacts for agent-to-agent context passing without re-briefing
- **Context envelope** — working set tracking (files edited during session, edit counts, net line deltas); intent classification (`exploration`, `debugging`, `review`, `refactor`); universal JSON envelope containing project metadata + working set + matched skills + handoff references
- **Hub impact** — hub file identification (3+ importers); impact analysis based on changes to hub files; `risk_score` on observations based on hub file involvement
- **Skills + hooks** — skills framework (Markdown-based guidance matched by intent, mentioned files, project languages); lifecycle hooks (`session-start`, `pre-edit`, `prompt-submit`, `session-stop`) for context injection
- **Memfs** — `mem fs` virtual filesystem with `mem ls`, `mem tree`, `mem find` alongside existing `mem search`; `mem://` URI space; queries same store with scope-based filtering
- **Tests** — unit + integration for each change; `go test ./skillgrid-cli/...` must pass

## Risks & rollback

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Connection pooling introduces stale handles | Med | Health-check on reuse; retry on WAL lock with 50ms backoff |
| FTS5 trigram changes break existing queries | Low | New match modes are opt-in; default mode unchanged |
| Parallel search causes too many open connections | Med | Goroutine pool with semaphore; max concurrent stores configurable |
| TTL default deletes observations too aggressively | Low | 7-day default; operator configurable; soft-delete only |
| LLM extraction adds latency to saves | Med | Async extraction; regex fallback if LLM unavailable |
| New embedder types break existing config | Low | `ollama`/`local` are new enum values; `onnx`/`external`/`off` unchanged |
| Memory type classification errors | Med | LLM extraction quality; fallback to auto-classification |
| Async commit loses LLM extraction | Low | memory_diff.json audit; retry on failure |
| LLM dedup misses semantic duplicates | Low | Hash dedup as fallback; configurable threshold |
| Directory retrieval slow on deep hierarchies | Med | Depth limit; caching of directory scores |
| Snapshot storage overhead | Low | Configurable snapshot retention; auto-prune old snapshots |
| Handoff artifact staleness | Med | Delta computed at handoff time; cache invalidation |
| Context envelope too large | Low | Configurable envelope size; field filtering |
| Hub analysis inaccurate | Med | Periodic re-analysis; minimum importers threshold |
| Skills hook interference | Low | Hooks opt-in per project; hook timeout |

**Rollback:** Drop each migration file (additive `014_*`), remove new modules, revert config changes — all additive. Existing DBs that applied `014_*` keep new tables/columns (harmless if callers stop using them). The `store` handle cache can be bypassed by setting an env var.

## Error handling

| Failure | Behavior | Notes |
|---------|----------|-------|
| Store open fails (WAL lock) | Retry with backoff; fail after 3 attempts | `service.Open` retry loop |
| FTS5 trigram query empty | Fall back to default phrase matching | `buildFTSQuery` graceful degradation |
| Cross-project search store missing | Skip that store; log warning | `SearchObservationsAll` non-fatal |
| TTL cleanup on missing store | Skip; no error | `TTLRetire` best-effort |
| LLM extraction fails | Fall back to regex extraction | `extractLearnings` fallback path |
| Embedder load fails | Degrade to FTS+signals floor | `embedder.Default()` returns Null |
| Connection pool exhausted | Queue with timeout; return error after 5s | Semaphore with deadline |
| Triple-store join performance | Slow queries with many cross-links | Index `graph_ref`; limit JOIN depth; benchmark with 10k+ observations |
| improve() skews results unexpectedly | Over-boosting stale but popular observations | Configurable thresholds; cooldown period |
| Session promotion creates duplicate graph nodes | Memory bloat; duplicate nodes | Dedup check before creation; idempotent promotion |
| Temporal edge queries miss valid edges | Data loss in retrieval | Verify `valid_from <= now AND (valid_to IS NULL OR valid_to > now)` logic |
| Export produces too-large JSON | CLI timeout; memory pressure | Stream JSON; `--chunk-size` flag; skip embeddings by default |

## Testing strategy

- **Unit:** `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` — Expected: PASS
- **Integration / acceptance:** `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` plus BDD `@step-NN` scenarios in `acceptance.feature` — Expected: PASS (`@step-01`…`@step-26` / `@p0`)
- **Full suite:** `Run: go test ./skillgrid-cli/...` — Expected: PASS
- **Green means:** Connection pooling eliminates N+1 opens; FTS trigram queries return correct results; cross-project search completes in parallel; TTL soft-expiry works; LLM extraction returns valid learnings; new embedder types load correctly; triple-store cross-links queryable; improve() re-weights correctly; session promotion creates graph nodes; temporal edges filter correctly; export produces valid JSON; DreamExecutor consolidates/synthesizes/prunes; importance scoring boosts relevant results; @relation annotations queryable; provenance chain tracked; federated query ranks by importance; DistillLock prevents concurrent distillation; `mem ls`/`mem tree`/`mem find` browse the same store structurally

---

## Step Blueprint

Contract for `sdd-spec`. Do not renumber after `tasks.md` exists. Per-step Out of scope / DoD live under Per-step WHAT (table is summary only).

| NN | Step slug | Goal (one line) | Primary package / entry | Depends on |
|----|-----------|-----------------|-------------------------|------------|
| 01 | `store-pooling` | Cached store handles + WAL lock retry | `skillgrid-cli/internal/mnemonic/store` | — |
| 02 | `fts-trigram` | Trigram/wildcard FTS query support | `skillgrid-cli/internal/mnemonic/memory` | 01 |
| 03 | `parallel-search` | Concurrent cross-project search | `skillgrid-cli/internal/mnemonic/service` | 01 |
| 04 | `ttl-defaults` | Auto-set expires_at + scheduled cleanup | `skillgrid-cli/internal/mnemonic/memory/lifecycle` | 01 |
| 05 | `llm-extraction` | LLM-backed passive extraction + regex fallback | `skillgrid-cli/internal/mnemonic/memory/extraction` | 01 |
| 06 | `multi-embedder` | ollama/local embedder providers | `skillgrid-cli/internal/mnemonic/embedder` | 01 |
| 07 | `triple-store-linkage` | Cross-link relational + vector + graph stores | `skillgrid-cli/internal/mnemonic/codeindex` + `memory` | 01 |
| 08 | `improve-loop` | Self-improvement feedback loop via retrieval usage | `skillgrid-cli/internal/mnemonic/memory/lifecycle` | 07 |
| 09 | `session-promotion` | L0/L1 session → L2/L3 graph auto-promotion | `skillgrid-cli/internal/mnemonic/memory/layer` | 07 |
| 10 | `temporal-graph` | Temporal knowledge graph edges with valid_from/to | `skillgrid-cli/internal/mnemonic/codeindex/graph` | 07 |
| 11 | `portable-export` | COGX-inspired JSON export of observations + graph | `skillgrid-cli/internal/mnemonic/memory/export` | 07 |
| 12 | `dream-executor` | consolidate/synthesize/prune distillation decomposition | `skillgrid-cli/internal/mnemonic/memory/dream` | 04, 08 |
| 13 | `importance-scoring` | AKL importance + recency decay in improve() and query ranking | `skillgrid-cli/internal/mnemonic/memory/lifecycle` | 08 |
| 14 | `explicit-relations` | @relation annotations between observations | `skillgrid-cli/internal/mnemonic/memory/relations` | 07 |
| 15 | `provenance-tracking` | Provenance chain metadata on observations | `skillgrid-cli/internal/mnemonic/memory/lifecycle` | 05 |
| 16 | `federated-query` | Evolve parallel search to federated cross-project query | `skillgrid-cli/internal/mnemonic/service` | 03, 13 |
| 17 | `distill-lock` | DistillLockService + Distill rollback on failure | `skillgrid-cli/internal/mnemonic/memory/lifecycle` | 09, 12 |
| 18 | `memory-types` | Typed memory categories (profile, preferences, entities, events, identity, soul, cases, trajectories, experiences) + LLM dedup + async two-phase commit | `skillgrid-cli/internal/mnemonic/memory/types` | 04, 05 |
| 19 | `directory-retrieval` | Directory-level recursive retrieval with drill-down + observable trajectory | `skillgrid-cli/internal/mnemonic/memory/retrieval` | 02, 03 |
| 20 | `snapshots` | Multi-version snapshots + transaction locking | `skillgrid-cli/internal/mnemonic/store/snapshots` | 01, 17 |
| 21 | `handoff` | Prefix + delta handoff artifacts for agent-to-agent context passing | `skillgrid-cli/internal/mnemonic/memory/handoff` | 09, 12 |
| 22 | `context-envelope` | Working set + intent classification + universal context envelope | `skillgrid-cli/internal/mnemonic/memory/envelope` | 08, 13, 21 |
| 23 | `hub-impact` | Hub file identification + impact analysis + risk scores | `skillgrid-cli/internal/mnemonic/codeindex/hub` | 10, 13 |
| 24 | `skills-hooks` | Skills framework + lifecycle hooks for context injection | `skillgrid-cli/internal/mnemonic/memory/skills` | 05, 09 |
| 25 | `memfs` | `memfs` virtual filesystem: `mem ls`, `mem tree`, `mem find` alongside existing `mem search` | `skillgrid-cli/internal/mnemonic/memfs` | 07, 18, 19 |
| 26 | `tests` | Unit + integration coverage for all steps | `skillgrid-cli/...` | 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25 |

---

## Technical approach

1. **Store pooling** — add a `sync.Map` cache in `service.Service` keyed by project ID; `Open()` checks cache first; cached handles get refcounted; `Close` decrements refcount and closes when zero; add WAL lock retry loop with exponential backoff (50ms, 100ms, 200ms)
2. **FTS trigram** — extend `buildFTSQuery` to split query into trigrams and support `*` suffix; add `matchMode` values `trigram` and `prefix`; use `LIKE` fallback for wildcard patterns
3. **Parallel search** — `SearchObservationsAll` splits stores across goroutines (semaphore-limited to `min(len(stores), runtime.NumCPU())`); merge results using cross-store rank; preserve the `seen` dedup map
4. **TTL defaults** — modify `Save()` to set `expires_at` to `now + 7 days` when not explicitly provided; add `RunTTLExpiry()` method to `Service` that calls `TTLRetire` on all projects; expose via `skillgrid mem expire` CLI
5. **LLM extraction** — add `extraction.go` module with `ExtractWithLLM(ctx, text)` that calls the LLM (via the existing `layer.Distill` or a new extraction endpoint); keep `extractLearnings` regex as fallback; `CapturePassive` tries LLM first, falls back to regex
6. **Multi-embedder** — extend `embedder.go` with `ollama` and `local` providers; `ollama` uses `http://localhost:11434/api/embeddings`; `local` uses ONNX model from `~/.skillgrid/models/`; config via `indexing.yaml` `mnemonic.embedder.provider`
7. **Triple-store cross-linkage** — add `graph_ref` column to `observations` pointing to codeindex symbol ID; add `embedding_blob` column to `long_term_memories` for stored vectors; create `symbol_embeddings` table linking codeindex symbols to their embeddings; queries traverse observation→symbol→embedding→related observations via a single SQL JOIN
8. **Self-improvement feedback loop** — add `improve()` method that reads `retrieval_usage` from observations (already tracked in `retrieval_usage` column); observations with high retrieval get boosted in `mem_search` ordering; observations never accessed lose rank over time; implements cognee's `improve()` concept
9. **Session-to-graph promotion** — on `SessionEnd`, if session has a summary, auto-create a permanent graph node linking the session's observations into the codeindex graph; implements cognee's session-to-graph promotion
10. **Temporal knowledge graph** — add `valid_from` and `valid_to` columns to `edges` table in codeindex; when a symbol relationship is observed, record the timestamp; expired edges (`valid_to < now`) are hidden from queries but preserved for history; implements cognee's temporal knowledge graphs
11. **Portable export** — add `ExportProject(ctx, projectID)` method that produces JSON with observations, graph edges, and embeddings; supports COGX-inspired format for data portability; implement `mem export` CLI subcommand
12. **Dream executor** — decompose `DistillSession` into `consolidate()` (merge facts from multiple observations), `synthesize()` (create higher-level summaries), `prune()` (remove low-importance observations); add `DreamLockService` ensuring single dream per project; add `DreamRollback` on error; expose via `brv dream` equivalent
13. **Importance scoring** — add `importance_score` and `maturity_tier` columns to observations; compute via `recency_decay` + `retrieval_count`; feed into `mem_search` ranking at query time; implements ByteRover's Adaptive Knowledge Lifecycle (AKL)
14. **Explicit relations** — add `@relation` annotations between observations; relation types: `mentions`, `depends_on`, `contradicts`, `supports`, `references`; stored as typed edges in memory store; queryable via `mem relations` CLI
15. **Provenance tracking** — add `provenance` JSON column to observations; chain: `session_id → curate_command → source_files → LLM_reasoning`; immutable once set; visible via `mem provenance` CLI
16. **Federated query** — evolve `SearchObservationsAll` into a federated query pipeline; each store returns results with importance scores; merge by cross-store rank + importance score; dedup by observation ID; `mem search all_projects=true` uses this pipeline
17. **Distill lock + rollback** — `DistillLockService` uses a `distill_lock` row per project; `DistillRollback` reverts observations to pre-distill state on failure; `DreamExecutor` wraps `consolidate/synthesize/prune` in a transaction
18. **Memfs** — new `memfs/` package with URI resolution engine mapping `mem://project/{id}/<scope>` to SQL filters on `memory_type` + `scope`; `mem fs ls/tree/find` registered as MCP tools alongside `mem search`; scope metadata additive on observations; no store schema rewrite beyond additive `scope` column

## Architecture decisions

### Decision: Cached store handles with refcount

**Module / Interface / Seam / Adapter / Depth:** `service.Service`; `Open()/Close()`; `sync.Map` cache; deep
**Choice:** Cache open stores by project ID; refcounted close.
**Alternatives considered:** Connection pool at the `sql.DB` level (already done, but no cross-call reuse); lazy open per request (current behavior = N+1)
**Rationale:** Eliminates N+1 opens for multi-request agents; minimal code change; backward compatible.

### Decision: FTS trigram as opt-in match mode

**Module / Interface / Seam / Adapter / Depth:** `memory.buildFTSQuery`; `SearchWithScope`; deep
**Choice:** Add `trigram` and `prefix` match modes; default mode unchanged.
**Alternatives considered:** Rewrite FTS tokenizer (too invasive); always use trigram (breaks existing queries)
**Rationale:** Opt-in preserves backward compatibility; users get the power they need without regression.

### Decision: LLM extraction with regex fallback

**Module / Interface / Seam / Adapter / Depth:** `memory.extraction`; `CapturePassive`; deep
**Choice:** Try LLM first; fall back to regex.
**Alternatives considered:** Replace regex entirely (breaks offline users); add both as parallel paths (duplicate results)
**Rationale:** Graceful degradation; no single point of failure; offline users get regex.

### Decision: Parallel search with semaphore

**Module / Interface / Seam / Adapter / Depth:** `service.SearchObservationsAll`; deep
**Choice:** Goroutines bounded by `min(stores, NumCPU())`.
**Alternatives considered:** Sequential (current = slow); unlimited goroutines (resource exhaustion)
**Rationale:** Bounded parallelism; no resource exhaustion; significant speedup.

### Decision: Triple-store cross-linkage

**Module / Interface / Seam / Adapter / Depth:** `memory` + `codeindex`; `graph_ref`, `embedding_blob`, `symbol_embeddings`; deep
**Choice:** Every observation gets a `graph_ref` to its codeindex symbol; every symbol gets an `embedding_blob`; a `symbol_embeddings` table bridges them.
**Alternatives considered:** Keep stores separate (current — no cross-query); merge into one DB (too invasive; breaks isolation)
**Rationale:** Cognee proved that Relational + Vector + Graph cross-linked triples dramatically improve multi-hop recall. This makes every observation queryable by both semantic similarity AND structural relationship, matching cognee's architecture at the same scale.

### Decision: Self-improvement via retrieval usage

**Module / Interface / Seam / Adapter / Depth:** `memory.lifecycle`; `improve()`; deep
**Choice:** `retrieval_usage` counter (already tracked) drives ranking; high-usage observations rise in `mem_search`; never-accessed observations decay.
**Alternatives considered:** Explicit user feedback (too complex); periodic re-ranking (no signal)
**Rationale:** Cognee's `improve()` re-weights memory from use. Mnemonic already has `retrieval_usage` — just needs to act on it. No new infrastructure needed.

### Decision: Temporal knowledge graph edges

**Module / Interface / Seam / Adapter / Depth:** `codeindex.graph`; `edges.valid_from`/`valid_to`; deep
**Choice:** Add temporal bounds to graph edges; expired edges hidden but preserved.
**Alternatives considered:** Static edges (current — no temporal awareness); full temporal database (too heavy)
**Rationale:** Cognee's temporal knowledge graphs track how entities evolve. For code, relationships between symbols change over time (e.g., a function is called by another only in certain versions). Temporal edges make this queryable.

### Decision: Dream Executor decomposition

**Module / Interface / Seam / Adapter / Depth:** `memory.dream`; `consolidate()`, `synthesize()`, `prune()`, `DreamLockService`, `DreamRollback`; deep
**Choice:** Decompose `DistillSession` into three operations with a lock and rollback.
**Alternatives considered:** Monolithic distillation (current — no phase separation); background-only dreaming (no lock)
**Rationale:** ByteRover's DreamExecutor validated that separating consolidate/synthesize/prune improves quality and enables targeted rollback. A lock prevents concurrent distillation; rollback ensures atomicity on failure.

### Decision: Adaptive Knowledge Lifecycle importance scoring

**Module / Interface / Seam / Adapter / Depth:** `memory.lifecycle`; `importance_score`, `maturity_tier`, `recency_decay`; deep
**Choice:** Compute importance score per observation from retrieval count + recency; feed into query-time ranking.
**Alternatives considered:** Binary TTL only (current — flat expiry); manual importance tagging (too laborious)
**Rationale:** ByteRover's AKL proved that importance + recency decay dramatically outperforms time-only expiry. Mnemonic's `improve()` already tracks retrieval; adding importance scoring makes query results more relevant.

### Decision: Explicit relation annotations

**Module / Interface / Seam / Adapter / Depth:** `memory.relations`; `@relation` typed edges; deep
**Choice:** Add explicit relation types between observations; store as typed edges in memory store.
**Alternatives considered:** Implicit embedding similarity only (no semantics); free-text relations (unstructured)
**Rationale:** ByteRover's `@relation` annotations provide semantic connections that embedding similarity alone cannot capture. Typed relations (mentions, depends_on, contradicts, supports) enable precise multi-hop queries.

### Decision: Provenance tracking

**Module / Interface / Seam / Adapter / Depth:** `memory.lifecycle`; `provenance` column; deep
**Choice:** Every observation carries a full provenance chain.
**Alternatives considered:** Metadata only (current — no chain); provenance as separate table (join overhead)
**Rationale:** ByteRover's provenance tracking enables trust and debugging. Knowing why a memory was stored (which session, which files, which LLM reasoning) is critical for agent trust and debugging.

### Decision: Federated cross-project query

**Module / Interface / Seam / Adapter / Depth:** `service.SearchObservationsAll`; deep
**Choice:** Evolve parallel search into federated query with importance ranking and dedup.
**Alternatives discussed:** Simple parallel search (current — no importance ranking); centralized search (requires shared DB)
**Rationale:** ByteRover's SwarmCoordinator demonstrated that federated memory with importance ranking outperforms simple parallel search. Each store returns importance-scored results; the merge pipeline applies cross-store ranking + dedup.

### Decision: Distill lock + rollback

**Module / Interface / Seam / Adapter / Depth:** `memory.dream`; `DistillLockService`, `DistillRollback`; deep
**Choice:** Lock per project; rollback on failure.
**Alternatives considered:** No lock (concurrent distillation); manual rollback (error-prone)
**Rationale:** ByteRover's DreamLockService prevents data corruption from concurrent distillation. Rollback ensures that partial distillation doesn't leave the store in an inconsistent state.

### Decision: Memory types (typed categories)

**Module / Interface / Seam / Adapter / Depth:** `memory.types`; `memory_type` column; deep
**Choice:** 9 typed categories: profile, preferences, entities, events, identity, soul, cases, trajectories, experiences.
**Alternatives considered:** Flat observations only (current — no categories); user-defined types (too complex)
**Rationale:** OpenViking's typed memory categories enable targeted extraction and retrieval. Mnemonic needs semantic categories, not just tiered depth.

### Decision: Async two-phase session commit

**Module / Interface / Seam / Adapter / Depth:** `memory.types`; `session.commit()`; deep
**Choice:** Synchronous phase (write messages, increment compression_index) + asynchronous phase (LLM extraction, vector pre-filtering, LLM dedup, write memory_diff.json).
**Alternatives considered:** Monolithic commit (current — blocks on LLM); async-only (no durability guarantee)
**Rationale:** OpenViking proved that separating sync and async phases ensures durability while not blocking on LLM. memory_diff.json provides audit trail.

### Decision: LLM dedup before write

**Module / Interface / Seam / Adapter / Depth:** `memory.types`; `dedup()`; deep
**Choice:** Before `Save()`, LLM checks for similar existing observations and deduplicates.
**Alternatives considered:** No dedup (current — memory bloat); hash-based dedup (misses semantic duplicates)
**Rationale:** OpenViking uses LLM dedup to prevent memory bloat. Semantic dedup catches near-duplicates that hash-based approaches miss.

### Decision: Directory-level recursive retrieval

**Module / Interface / Seam / Adapter / Depth:** `memory.retrieval`; `Retrieve` module; deep
**Choice:** First find highest-scoring directory via FTS5 + embeddings, then drill down layer by layer.
**Alternatives discussed:** Flat vector search (current — no directory context); exhaustive search (too expensive)
**Rationale:** OpenViking's hierarchical retrieval preserves surrounding context and provides better results than flat search. Directory-browsing trajectory enables observability.

### Decision: Observable retrieval trajectory

**Module / Interface / Seam / Adapter / Depth:** `memory.retrieval`; `retrieval_trails`; deep
**Choice:** Preserve directory-browsing path for every query in `retrieval_trails`.
**Alternatives considered:** No trajectory (black-box); full trace (too verbose)
**Rationale:** OpenViking proved that observable retrieval trajectories dramatically improve debugging. Agents can see why a result was returned.

### Decision: Multi-version snapshots + transaction locking

**Module / Interface / Seam / Adapter / Depth:** `store.snapshots`; `Snapshot()`; deep
**Choice:** Point-in-time store snapshots; row-level locking for concurrent writes.
**Alternatives considered:** No snapshots (current — no version history); full MVCC (too heavy)
**Rationale:** OpenViking uses snapshots for version management. Point-in-time views enable rollback and auditing without full MVCC overhead.

### Decision: Handoff artifacts (prefix + delta)

**Module / Interface / Seam / Adapter / Depth:** `memory.handoff`; `handoff.latest.json`; deep
**Choice:** Structured handoff with stable `prefix` (hub summaries, repo file-count) + dynamic `delta` (changed files, risk files, working set).
**Alternatives considered:** Free-text handoff (unstructured); no handoff (agents re-brief)
**Rationale:** codemap's handoff artifacts enable agent-to-agent context passing without re-briefing. The prefix/delta split keeps stable context separate from dynamic changes.

### Decision: Context envelope (working set + intent + skills)

**Module / Interface / Seam / Adapter / Depth:** `memory.envelope`; `ContextEnvelope`; deep
**Choice:** Universal JSON envelope containing project metadata, working set, intent classification, matched skills, handoff references.
**Alternatives discussed:** Manual context injection (error-prone); no context envelope
**Rationale:** codemap's ContextEnvelope provides a standardized way to give agents full context. Working set tracking and intent classification adapt retrieval strategy.

### Decision: Hub files + impact analysis + risk scores

**Module / Interface / Seam / Adapter / Depth:** `codeindex.hub`; `AnalyzeImpact()`; deep
**Choice:** Identify hub files (3+ importers); analyze impact of changes; add `risk_score` to observations.
**Alternatives considered:** No hub analysis (current — no risk awareness); full dependency analysis (too expensive)
**Rationale:** codemap's hub file identification enables risk-aware retrieval. Changes to hub files have high impact; observations related to hub files get higher risk scores.

### Decision: Skills framework + lifecycle hooks

**Module / Interface / Seam / Adapter / Depth:** `memory.skills`; `skills` observation type; `hooks`; deep
**Choice:** Markdown-based skills matched by intent, mentioned files, project languages; lifecycle hooks for context injection.
**Alternatives considered:** No skills (current — no guidance); hard-coded rules (inflexible)
**Rationale:** codemap's skills framework provides agent guidance. Lifecycle hooks (`session-start`, `pre-edit`, `prompt-submit`, `session-stop`) inject context at the right moments.

### Decision: Memfs virtual filesystem

**Module / Interface / Seam / Adapter / Depth:** `memfs`; `mem fs` CLI; deep
**Choice:** `mem://` URI space with `mem ls`, `mem tree`, `mem find` alongside existing `mem search`. Queries the same store — filters by scope + memory_type.
**Alternatives considered:** Replace `mem search` with filesystem ops (breaking change); no filesystem access (current — search only)
**Rationale:** OpenViking's VikingFS proved that filesystem-style access (`ls`, `tree`, `find`) complements semantic search (`mem search`). Users can browse structurally (`mem ls project/{id}/preferences`) OR search semantically (`mem search "preferences"`). Both coexist; neither replaces the other.

## Data flow

```
mem search → store.Open() [cached handle] → mem.Budget().Bound(ctx)
  → improve() [re-weight by importance + recency] → SearchOwnerScoped [FTS trigram mode]
  → retrieve layered hits → Budget.Apply [item/char/timeout caps] → return truncated/complete results

observation → graph_ref → codeindex symbol → symbol_embeddings → related observations
  → query traverses observation→symbol→embedding→related observations (triple-store cross-linkage)

observation → provenance → session_id → curate_command → source_files → LLM_reasoning
  → provenance chain tracked on every observation

SessionEnd → DistillSession [L0→L1→L2→L3]
  → DreamLock [ensure single dream per project]
  → consolidate [merge facts from observations]
  → synthesize [create summaries]
  → session-promotion → permanent graph node created
  → prune [remove low-importance data]
  → CapturePassive [LLM extraction first, regex fallback]
  → Save [expires_at auto-set to now+7d]
  → improve() [importance + recency decay scored]
  → DreamUnlock + Rollback on error

observation → @relation → related observations [explicit relations: mentions, depends_on, contradicts, supports]
  → relation edges stored in memory store

mem search all_projects=true → SearchObservationsAll [federated query]
  → each store SearchWithScope [FTS + importance ranking] → merge by cross-store rank + importance score
  → dedup by observation ID → return unified results

mem export → ExportProject → JSON with observations + graph edges + embeddings (COGX-inspired)

DreamExecutor [background] → consolidate + synthesize + prune
  → DreamLockService ensures single dream per project
  → DreamRollback on error

observation → memory_type [profile, preferences, entities, events, identity, soul, cases, trajectories, experiences]
  → LLM dedup before write → async two-phase commit (sync write + async LLM extraction)

mem search → Intent Analysis [classify as exploration/debugging/review/refactor]
  → Directory Retrieval [find highest-scoring directory] → drill-down layer by layer
  → retrieval trajectory preserved for debugging

session-end → handoff.latest.json [prefix (stable: hub summaries, repo file-count) + delta (changed files, risk files, working set)]
  → ContextEnvelope [working set + intent + skills + handoff references]

mem fs ls <scope> → memfs.resolve(<scope>) → SQL query by memory_type + scope
  → returns directory-like listing of observations
mem fs tree <scope> → memfs.resolve(<scope>) → hierarchical tree of scopes
mem fs find <pattern> → memfs.pattern_match(<pattern>) → filesystem-style results
  → queries the same store as mem search; structural browsing alongside semantic search

## File layout

```
skillgrid-cli/internal/mnemonic/
├── store/
│   ├── store.go                      # MODIFY: add handle cache + refcount
│   └── migrations/014_*.sql          # CREATE: TTL config, extraction tables
├── memory/
│   ├── extraction.go                 # CREATE: LLM-backed extraction + regex fallback
│   ├── extraction_test.go            # CREATE
│   ├── lifecycle.go                  # MODIFY: TTL defaults in Save(); improve() feedback loop; importance scoring; provenance
│   ├── retrieve.go                   # MODIFY: trigram FTS support; directory retrieval
│   ├── service.go                    # MODIFY: cached handles, parallel search
│   ├── budget.go                     # NO CHANGE
│   ├── export.go                     # CREATE: COGX-inspired portable JSON export
│   ├── session.go                    # MODIFY: session-to-graph promotion on SessionEnd
│   ├── dream.go                      # CREATE: DreamExecutor (consolidate/synthesize/prune); DreamLockService; DreamRollback
│   ├── dream_test.go                 # CREATE
│   ├── importance.go                 # CREATE: AKL importance scoring + recency decay
│   ├── relations.go                  # CREATE: @relation annotations between observations
│   ├── provenance.go                 # CREATE: provenance chain tracking
│   ├── types.go                      # CREATE: memory types + LLM dedup + async commit
│   ├── types_test.go                 # CREATE
│   ├── retrieval.go                  # CREATE: directory-level recursive retrieval + trajectory
│   ├── retrieval_test.go             # CREATE
│   ├── snapshots.go                  # CREATE: multi-version snapshots + transaction locking
│   ├── snapshots_test.go             # CREATE
│   ├── handoff.go                    # CREATE: prefix + delta handoff artifacts
│   ├── handoff_test.go               # CREATE
│   ├── envelope.go                   # CREATE: working set + intent classification + context envelope
│   ├── envelope_test.go              # CREATE
│   ├── hub.go                        # CREATE: hub file identification + impact analysis + risk scores
│   ├── hub_test.go                   # CREATE
│   ├── skills.go                     # CREATE: skills framework + lifecycle hooks
│   └── skills_test.go                # CREATE
├── embedder/
│   ├── embedder.go                   # MODIFY: ollama + local providers
│   ├── ollama.go                     # CREATE: Ollama embedder
│   └── local.go                      # CREATE: Local ONNX embedder
├── codeindex/
│   ├── graph.go                      # MODIFY: add valid_from/valid_to to edges
│   └── symbol_embeddings.go          # CREATE: bridge table linking symbols to embeddings
├── service/
│   ├── service.go                    # MODIFY: RunTTLExpiry, cached opens, federated query
│   └── retrieval.go                  # MODIFY: parallel → federated cross-project search
└── cmd/skillgrid/
    └── mem.go                        # MODIFY: add expire, export, relations, provenance, dream, memory-type, search --trajectory, snapshot, handoff, context, graph --risk, skills, hook, fs subcommands
├── memfs/
│   ├── memfs.go                   # CREATE: URI resolution + scope management + filesystem operations
│   ├── ls.go                         # CREATE: mem ls <scope> — directory listing
│   ├── tree.go                       # CREATE: mem tree <scope> — hierarchical tree view
│   ├── find.go                       # CREATE: mem find <pattern> — filesystem-style search
│   ├── scope.go                      # CREATE: scope management (mem://user/{id}/, mem://project/{id}/)
│   └── memfs_test.go              # CREATE
```

## Impacted files map

| File | Action | Step | Description |
|------|--------|------|-------------|
| `internal/mnemonic/store/store.go` | Modify | 01 | Add handle cache, refcount, WAL retry |
| `internal/mnemonic/store/migrations/014_ttl_extraction.sql` | Create | 01 | TTL config table, extraction metadata |
| `internal/mnemonic/memory/extraction.go` | Create | 05 | LLM extraction with regex fallback |
| `internal/mnemonic/memory/extraction_test.go` | Create | 05 | Extraction tests |
| `internal/mnemonic/memory/lifecycle.go` | Modify | 04, 08, 13, 15 | Auto-set expires_at; improve() feedback loop; importance scoring; provenance |
| `internal/mnemonic/memory/retrieve.go` | Modify | 02 | Trigram/prefix FTS query support |
| `internal/mnemonic/memory/service.go` | Modify | 02, 03, 16 | Cached handles, parallel → federated search |
| `internal/mnemonic/memory/export.go` | Create | 11 | COGX-inspired portable JSON export |
| `internal/mnemonic/memory/session.go` | Modify | 09 | Session-to-graph promotion on SessionEnd |
| `internal/mnemonic/memory/dream.go` | Create | 12 | DreamExecutor (consolidate/synthesize/prune); DreamLockService; DreamRollback |
| `internal/mnemonic/memory/dream_test.go` | Create | 12 | Dream executor tests |
| `internal/mnemonic/memory/importance.go` | Create | 13 | AKL importance scoring + recency decay |
| `internal/mnemonic/memory/relations.go` | Create | 14 | @relation annotations between observations |
| `internal/mnemonic/memory/provenance.go` | Create | 15 | Provenance chain tracking |
| `internal/mnemonic/service/service.go` | Modify | 03, 04, 16 | RunTTLExpiry, cached Open, federated query |
| `internal/mnemonic/service/retrieval.go` | Modify | 03, 16 | Parallel → federated cross-project search |
| `internal/mnemonic/codeindex/graph.go` | Modify | 10 | Add valid_from/valid_to columns to edges |
| `internal/mnemonic/codeindex/symbol_embeddings.go` | Create | 07 | Bridge table: symbols ↔ embeddings |
| `internal/mnemonic/embedder/embedder.go` | Modify | 06 | ollama + local providers |
| `internal/mnemonic/embedder/ollama.go` | Create | 06 | Ollama embedder implementation |
| `internal/mnemonic/embedder/local.go` | Create | 06 | Local ONNX embedder |
| `internal/mnemonic/cmd/skillgrid/mem.go` | Modify | 04, 11, 14, 15 | Add `expire` + `export` + `relations` + `provenance` + `dream` subcommands |
| `internal/mnemonic/config/load.go` | Modify | 06 | ollama/local provider config |
| `internal/mnemonic/embedder/embedder_test.go` | Modify | 06 | New provider tests |
| `internal/mnemonic/memory/export_test.go` | Create | 11 | Export tests |
| `internal/mnemonic/codeindex/graph_test.go` | Create | 10 | Temporal edge tests |
| `internal/mnemonic/memory/importance_test.go` | Create | 13 | Importance scoring tests |
| `internal/mnemonic/memory/relations_test.go` | Create | 14 | Relations tests |
| `internal/mnemonic/memory/provenance_test.go` | Create | 15 | Provenance tests |
| `internal/mnemonic/memory/dream_lock_test.go` | Create | 12, 17 | DreamLock + rollback tests |
| `internal/mnemonic/memory/types.go` | Create | 18 | Memory types + LLM dedup + async two-phase commit |
| `internal/mnemonic/memory/types_test.go` | Create | 18 | Memory types tests |
| `internal/mnemonic/memory/retrieval.go` | Create | 19 | Directory-level recursive retrieval + trajectory |
| `internal/mnemonic/memory/retrieval_test.go` | Create | 19 | Retrieval tests |
| `internal/mnemonic/memory/snapshots.go` | Create | 20 | Multi-version snapshots + transaction locking |
| `internal/mnemonic/memory/snapshots_test.go` | Create | 20 | Snapshot tests |
| `internal/mnemonic/memory/handoff.go` | Create | 21 | Prefix + delta handoff artifacts |
| `internal/mnemonic/memory/handoff_test.go` | Create | 21 | Handoff tests |
| `internal/mnemonic/memory/envelope.go` | Create | 22 | Context envelope (working set + intent + skills) |
| `internal/mnemonic/memory/envelope_test.go` | Create | 22 | Envelope tests |
| `internal/mnemonic/memory/hub.go` | Create | 23 | Hub file identification + impact analysis + risk scores |
| `internal/mnemonic/memory/hub_test.go` | Create | 23 | Hub tests |
| `internal/mnemonic/memory/skills.go` | Create | 24 | Skills framework + lifecycle hooks |
| `internal/mnemonic/memory/skills_test.go` | Create | 24 | Skills tests |
| `internal/mnemonic/cmd/skillgrid/mem.go` | Modify | 04, 11, 14, 15, 18, 19, 20, 21, 22, 23, 24, 25 | Add `expire` + `export` + `relations` + `provenance` + `dream` + `memory-type` + `search --trajectory` + `snapshot` + `handoff` + `context` + `graph --risk` + `skills` + `hook` + `fs` subcommands |
| `internal/mnemonic/memfs/memfs.go` | Create | 25 | URI resolution + scope management + filesystem operations |
| `internal/mnemonic/memfs/ls.go` | Create | 25 | `mem ls <scope>` directory listing |
| `internal/mnemonic/memfs/tree.go` | Create | 25 | `mem tree <scope>` hierarchical tree view |
| `internal/mnemonic/memfs/find.go` | Create | 25 | `mem find <pattern>` filesystem-style search |
| `internal/mnemonic/memfs/scope.go` | Create | 25 | Scope management (`mem://user/{id}/`, `mem://project/{id}/`) |
| `internal/mnemonic/memfs/memfs_test.go` | Create | 25 | Memfs tests |

## Per-step WHAT

### Step 01 — `store-pooling`

**Goal:** Eliminate N+1 store opens via cached handles + WAL lock retry.
**Out of scope:** FTS trigram, parallel search, TTL, extraction, embedder.
**Definition of Done:** `service.Open()` returns cached handle when available; refcounted close; WAL lock retry with exponential backoff.

- Store open checks `sync.Map` cache first; returns existing handle if found
- Refcount incremented on cache hit; decremented on close; handle closed when refcount reaches 0
- WAL lock retry: 50ms, 100ms, 200ms with `PRAGMA busy_timeout=10000`
- No existing store files are modified or migrated

### Step 02 — `fts-trigram`

**Goal:** FTS5 trigram/wildcard query support via new match modes.
**Out of scope:** Store pooling, parallel search, TTL, extraction, embedder.
**Definition of Done:** `buildFTSQuery` supports `trigram` and `prefix` modes; existing phrase matching unchanged by default.

- `trigram` mode splits query into 3-character trigrams joined by OR
- `prefix` mode appends `*` to each term for prefix matching
- Default mode remains phrase-only for backward compatibility
- `mem search --mode trigram` and `--mode prefix` flags added

### Step 03 — `parallel-search`

**Goal:** Concurrent cross-project search with bounded goroutines.
**Out of scope:** Store pooling (depends on 01), FTS trigram, TTL, extraction, embedder.
**Definition of Done:** `SearchObservationsAll` searches stores in parallel; bounded by `min(stores, NumCPU())`; results merged by cross-store rank.

- Goroutine pool with semaphore (max concurrent = `min(len(stores), runtime.NumCPU())`)
- Each goroutine calls `SearchObservationsScoped` on one store
- Results merged using cross-store rank (highest rank across all stores wins)
- Missing stores skipped with warning; not fatal

### Step 04 — `ttl-defaults`

**Goal:** Auto-set expires_at on save + scheduled TTL cleanup.
**Out of scope:** Store pooling, FTS trigram, parallel search, extraction, embedder.
**Definition of Done:** `Save()` sets `expires_at` to `now + 7 days` when not explicitly provided; `RunTTLExpiry()` cleans up expired observations.

- `Save()` sets `expires_at` when not explicitly provided in `SaveInput`
- `TTLSoftExpiry()` and `TTLRetire()` operational by default
- `skillgrid mem expire` CLI subcommand triggers cleanup on all projects
- Default TTL configurable via `mnemonic.ttl` config key

### Step 05 — `llm-extraction`

**Goal:** LLM-backed passive extraction with regex fallback.
**Out of scope:** Store pooling, FTS trigram, parallel search, TTL, embedder.
**Definition of Done:** `CapturePassive` tries LLM extraction first; falls back to regex; results identical quality or better.

- `extraction.go` adds `ExtractWithLLM(ctx, text)` that calls the LLM
- `CapturePassive` tries LLM first; on error/fallback, uses `extractLearnings` regex
- Regex fallback always available (no dependency on LLM)
- Extraction results parsed with same `shapePassiveItem` / `shapePassiveContent` functions

### Step 06 — `multi-embedder`

**Goal:** ollama and local embedder providers.
**Out of scope:** Store pooling, FTS trigram, parallel search, TTL, extraction.
**Definition of Done:** `ollama` and `local` embedder types work; config-driven via `indexing.yaml`.

- `ollama.go` embeds via `http://localhost:11434/api/embeddings`
- `local.go` embeds via ONNX model from `~/.skillgrid/models/`
- Config: `mnemonic.embedder.provider: ollama|local` in `indexing.yaml`
- `embedder.Default()` returns the configured provider or Null

### Step 07 — `triple-store-linkage`

**Goal:** Cross-link relational + vector + graph stores.
**Out of scope:** All other steps.
**Definition of Done:** Every observation has a `graph_ref`; `symbol_embeddings` bridges symbols and embeddings; queries traverse observation→symbol→embedding→related observations.

- Add `graph_ref` column to `observations` pointing to codeindex symbol ID
- Add `embedding_blob` column to `long_term_memories` for stored vectors
- Create `symbol_embeddings` table linking codeindex symbols to their embeddings
- Cross-link query works: observation → symbol → embedding → related observations via SQL JOIN
- `graph_ref` defaults to NULL if codeindex symbol not found; no broken references
- No existing query paths change; triple-store queries are opt-in

### Step 08 — `improve-loop`

**Goal:** Self-improvement feedback loop based on retrieval usage.
**Out of scope:** Store pooling, FTS trigram, parallel search, TTL, extraction, embedder, triple-store linkage.
**Definition of Done:** `improve()` re-weights observations by `retrieval_usage`; high-usage observations rank higher in `mem_search`; never-accessed observations decay.

- `improve()` method on `Service` reads `retrieval_usage` from all observations
- Observations with `retrieval_usage > threshold` get boosted in `mem_search` ordering
- Observations with `retrieval_usage == 0` and age > TTL lose rank
- `improve()` called before `SearchOwnerScoped` (transparent to callers)
- Configurable boost/decay rates via `mnemonic.improve` config key

### Step 09 — `session-promotion`

**Goal:** L0/L1 session summaries auto-promote to L2/L3 permanent graph nodes.
**Out of scope:** All other steps.
**Definition of Done:** On `SessionEnd`, if session has a summary, a permanent graph node is created linking the session's observations into the codeindex graph.

- `SessionEnd` triggers promotion check
- If session has a summary (`LayerSummary` populated), create a graph node in codeindex
- Graph node links to all observations from the session via `graph_ref`
- Promoted nodes are queryable via `mem search` and codeindex graph traversal
- No promotion if session summary is empty or below quality threshold

### Step 10 — `temporal-graph`

**Goal:** Add temporal bounds to codeindex graph edges.
**Out of scope:** All other steps.
**Definition of Done:** `edges` table has `valid_from` and `valid_to` columns; expired edges hidden from queries but preserved for history.

- Add `valid_from` (UNIX timestamp) and `valid_to` (UNIX timestamp, NULL = active) columns to `edges` table
- When a symbol relationship is observed, record current time as `valid_from`
- Queries filter edges where `valid_from <= now AND (valid_to IS NULL OR valid_to > now)`
- Expired edges preserved in DB for historical analysis; hidden from normal queries
- `mem graph` CLI shows temporal status of edges

### Step 11 — `portable-export`

**Goal:** COGX-inspired portable JSON export of observations + graph + embeddings.
**Out of scope:** All other steps.
**Definition of Done:** `ExportProject(ctx, projectID)` produces JSON with full observations, graph edges, and embeddings; `mem export` CLI subcommand works.

- `ExportProject` returns a struct with `Observations[]`, `GraphEdges[]`, `Embeddings[]`
- JSON format includes all metadata, content, and binary embeddings (base64-encoded)
- Supports export to file (`mem export --file`) or stdout (`mem export`)
- Export is portable: output can be imported into another skillgrid instance
- COGX-inspired format: each record has `id`, `type`, `content`, `metadata`, `embeddings`, `graph_ref`

### Step 12 — `dream-executor`

**Goal:** Decompose DistillSession into consolidate/synthesize/prune with DreamLock + rollback.
**Out of scope:** All other steps.
**Definition of Done:** `DreamExecutor` runs `consolidate` + `synthesize` + `prune`; `DreamLockService` ensures single dream per project; `DreamRollback` reverts on failure.

- `DreamExecutor.consolidate()` merges facts from multiple observations into coherent knowledge
- `DreamExecutor.synthesize()` creates higher-level summaries from lower-tier memories
- `DreamExecutor.prune()` removes low-importance observations based on AKL scoring
- `DreamLockService` uses a `distill_lock` row per project; prevents concurrent distillation
- `DreamRollback` reverts observations to pre-distill state on error
- `brv dream` equivalent CLI subcommand triggers background dream
- DreamLock timeout: 5 minutes; auto-release on error

### Step 13 — `importance-scoring`

**Goal:** AKL importance scoring + recency decay in improve() and query ranking.
**Out of scope:** All other steps.
**Definition of Done:** Every observation has `importance_score`, `maturity_tier`, `recency_decay`; feeds into `mem_search` ranking at query time.

- Add `importance_score` (float), `maturity_tier` (enum), `recency_decay` (float) columns to observations
- `importance_score` computed from: `retrieval_count` × `recency_factor` (exponential decay)
- `maturity_tier`: `fresh` → `mature` → `archival` based on age + retrieval
- `recency_decay`: older observations lose importance exponentially
- `mem_search` applies importance score as query-time ranking boost
- `improve()` uses importance score instead of binary retrieval_usage
- Configurable decay rate via `mnemonic.importance.decay` config key

### Step 14 — `explicit-relations`

**Goal:** @relation annotations between observations.
**Out of scope:** All other steps.
**Definition of Done:** Observations support typed `@relation` edges; queryable via `mem relations` CLI.

- Relation types: `mentions`, `depends_on`, `contradicts`, `supports`, `references`
- Each relation edge stores: `source_id`, `target_id`, `relation_type`, `confidence`
- `mem relations <observation_id>` CLI returns all related observations with relation types
- Relations stored in memory store alongside observations
- No relation types changed by existing queries; implicit similarity still works

### Step 15 — `provenance-tracking`

**Goal:** Provenance chain metadata on observations.
**Out of scope:** All other steps.
**Definition of Done:** Every observation carries a `provenance` JSON chain: `session_id → curate_command → source_files → LLM_reasoning`.

- Add `provenance` JSON column to observations
- Provenance is immutable once set during curation
- `mem provenance <observation_id>` CLI returns the full provenance chain
- Provenance visible in `mem list` output
- Supports debugging and trust: knowing why a memory was stored

### Step 16 — `federated-query`

**Goal:** Evolve parallel search into federated cross-project query with importance ranking.
**Out of scope:** All other steps.
**Definition of Done:** `SearchObservationsAll` uses federated query pipeline; each store returns importance-scored results; merge by cross-store rank + importance score; dedup by observation ID.

- `SearchObservationsAll` evolved from parallel to federated pipeline
- Each store returns `{observations, importance_scores, store_id}`
- Merge pipeline applies cross-store rank + importance score
- Dedup by observation ID (same observation in multiple stores appears once)
- `brv search` equivalent CLI subcommand triggers federated query
- Federated query respects per-project importance scores from step 13

### Step 17 — `distill-lock`

**Goal:** DistillLockService + DistillRollback on failure.
**Out of scope:** All other steps.
**Definition of Done:** `DistillLockService` ensures only one distillation per project; `DistillRollback` reverts on failure.

- `DistillLockService` uses `distill_lock` row per project
- Lock acquired before distillation; released after success or failure
- `DistillRollback` reverts observations to pre-distill state if any step fails
- Lock timeout: 5 minutes; auto-release on error
- Lock visible via `brv distill status` equivalent CLI

### Step 18 — `memory-types`

**Goal:** Typed memory categories + LLM dedup + async two-phase commit.
**Out of scope:** All other steps.
**Definition of Done:** Observations have `memory_type` (profile, preferences, entities, events, identity, soul, cases, trajectories, experiences); LLM dedup before write; async two-phase commit (sync write + async LLM extraction).

- Add `memory_type` column to observations; 9 typed categories
- Before `Save()`, LLM checks for similar existing observations and deduplicates
- `session.commit()` does sync phase (write messages, increment `compression_index`) + async phase (LLM extraction, vector pre-filtering, dedup, write `memory_diff.json`)
- `memory_diff.json` persisted for auditing
- `mem list --type preferences` CLI filters by memory type
- Async extraction does not block the main write path

### Step 19 — `directory-retrieval`

**Goal:** Directory-level recursive retrieval with drill-down + observable trajectory.
**Out of scope:** All other steps.
**Definition of Done:** `mem search` first finds highest-scoring directory, then drills down; retrieval trajectory preserved for debugging.

- `Retrieve` module: Intent Analysis → Hierarchical Retrieval → Rerank
- First locates highest-scoring directory via FTS5 + embeddings
- Drills down layer by layer, preserving surrounding context
- Retrieval trajectory (directory-browsing path) stored in `retrieval_trails`
- `mem search --trajectory` CLI shows the directory-browsing path
- Intent analysis classifies query before retrieval

### Step 20 — `snapshots`

**Goal:** Multi-version snapshots + transaction locking.
**Out of scope:** All other steps.
**Definition of Done:** Point-in-time store snapshots; row-level locking for concurrent access.

- Add `snapshots` table storing point-in-time store state hashes
- `Snapshot()` creates a point-in-time view of the store
- Row-level locking on `observations` table for concurrent writes
- `mem snapshot` CLI creates/rolls back to snapshots
- Transaction locking prevents concurrent writes to same observation

### Step 21 — `handoff`

**Goal:** Prefix + delta handoff artifacts for agent-to-agent context passing.
**Out of scope:** All other steps.
**Definition of Done:** `handoff.latest.json` with stable `prefix` + dynamic `delta`; `mem handoff` CLI generates handoff.

- Stable `prefix`: hub summaries, repo file-count, project metadata
- Dynamic `delta`: changed file stubs, risk files, recent events, working set
- Saved to `.skillgrid/handoff.latest.json`
- `mem handoff` CLI generates handoff artifact
- Handoff allows switching between agents without re-briefing

### Step 22 — `context-envelope`

**Goal:** Working set + intent classification + universal context envelope.
**Out of scope:** All other steps.
**Definition of Done:** `mem context` generates universal JSON envelope containing project metadata + working set + matched skills + handoff references.

- Working set tracking: files edited during session, edit counts, net line deltas, hub status
- Intent classification: `exploration`, `debugging`, `review`, `refactor`
- Skills matching: Markdown-based guidance matched by intent, mentioned files, project languages
- `mem context` CLI generates `ContextEnvelope` JSON
- `mem context --envelope` for universal JSON output

### Step 23 — `hub-impact`

**Goal:** Hub file identification + impact analysis + risk scores.
**Out of scope:** All other steps.
**Definition of Done:** Codeindex identifies hub files (3+ importers); `AnalyzeImpact()` identifies hub files among changes; `risk_score` on observations based on hub file involvement.

- Add `hub_score` (import count / total files) to codeindex symbols
- `AnalyzeImpact()` identifies hub files among changed files
- `risk_score` column on observations: high if related to hub file, low otherwise
- `mem graph --risk` CLI shows high-risk hub files
- Impact analysis based on change to hub files

### Step 24 — `skills-hooks`

**Goal:** Skills framework + lifecycle hooks for context injection.
**Out of scope:** All other steps.
**Definition of Done:** Skills framework (Markdown-based guidance); lifecycle hooks (`session-start`, `pre-edit`, `prompt-submit`, `session-stop`).

- Skills stored as observations with `skill` memory type; Markdown files
- Skills matched by intent, mentioned files, project languages
- Hooks: `session-start` injects relevant memories; `pre-edit` injects risk analysis; `prompt-submit` classifies intent; `session-stop` triggers distillation
- `mem skills` CLI manages skills
- `mem hook` CLI configures hooks

### Step 25 — `memfs`

**Goal:** `memfs` virtual filesystem: `mem ls`, `mem tree`, `mem find` alongside existing `mem search`.
**Out of scope:** All other steps.
**Definition of Done:** `mem fs` subcommand group works; `mem ls`, `mem tree`, `mem find` return directory-like results; `mem://` URI space resolves to stored observations.

- `mem fs` CLI subcommand group with `ls`, `tree`, `find` subcommands
- `mem ls <scope>` lists observations in a scope (e.g., `mem ls project/{id}/preferences`)
- `mem tree <scope>` shows hierarchical tree view of memory scopes
- `mem find <pattern>` does filesystem-style pattern matching across observations
- `mem://` URI space: `mem://project/{id}/` resolves to observations in that project/scope
- Scope management: `mem scope` CLI creates/lists scopes
- `mem fs` queries the **same store** — filters by `memory_type` + `scope`
- Does NOT replace `mem search` — provides structural browsing alongside semantic search
- URI resolution engine in `memfs.go` maps `mem://` URIs to SQL queries
- Scope-based organization is additive metadata on observations

### Step 26 — `tests`

**Goal:** Full test coverage for all steps (01-25).
**Out of scope:** New production features beyond closing RED coverage.
**Definition of Done:** `go test ./skillgrid-cli/...` passes for all touched packages.

- Step 01: test cached handle reuse, refcount, WAL retry
- Step 02: test trigram and prefix mode queries return correct results
- Step 03: test concurrent search completes; verify results match sequential
- Step 04: test auto-set expires_at; test TTLRetire
- Step 05: test LLM extraction; test regex fallback; test error handling
- Step 06: test ollama and local providers; test config-driven selection
- Step 07: test cross-link query; test symbol_embeddings integrity
- Step 08: test high-usage observation ranking boost; test decay for never-accessed
- Step 09: test L0/L1 session creates permanent graph node
- Step 10: test valid_from/valid_to edge filtering
- Step 11: test JSON output contains observations, edges, and embeddings
- Step 12: test consolidate/synthesize/prune; test DreamLock; test DreamRollback
- Step 13: test importance scoring; test recency decay; test query-time ranking boost
- Step 14: test @relation annotations; test relation query
- Step 15: test provenance chain; test provenance immutability
- Step 16: test federated query; test importance ranking; test dedup
- Step 17: test DistillLock; test DistillRollback; test lock timeout
- Step 18: test memory types; test LLM dedup; test async two-phase commit
- Step 19: test directory retrieval; test drill-down; test trajectory preservation
- Step 20: test snapshots; test row-level locking
- Step 21: test handoff artifact (prefix + delta); test handoff generation
- Step 22: test context envelope; test working set; test intent classification
- Step 23: test hub file identification; test impact analysis; test risk_score
- Step 24: test skills framework; test lifecycle hooks
- Step 25: test `mem ls`, `mem tree`, `mem find`; test `mem://` URI resolution; test scope management

## Threat matrix

| Boundary / threat | Applicable? | Owning step | Planned RED coverage |
|-------------------|-------------|-------------|----------------------|
| **Mnemonic tool surface** | Applicable — new `mem expire` subcommand; existing `mem_*` shapes unchanged | 04 | Test `mem expire` CLI; verify `mem_search` returns only non-expired |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` edits | — | — |
| **SQLite WAL lock contention** | Applicable — store pooling + retry | 01 | Test concurrent opens; verify no panic on WAL lock |
| **FTS5 query regression** | Applicable — trigram/prefix modes | 02 | Test existing phrase queries unchanged; test trigram/prefix return correct results |
| **Parallel search resource exhaustion** | Applicable — goroutine pool | 03 | Test with 50+ stores; verify bounded concurrency |
| **TTL premature expiry** | Applicable — auto-set expires_at | 04 | Test `Save()` sets expires_at; test `TTLRetire` only retires expired |
| **LLM extraction failure** | Applicable — regex fallback | 05 | Test LLM error falls back to regex; test extraction quality |
| **New embedder type** | Applicable — ollama/local providers | 06 | Test provider instantiation; test config-driven selection |
| **Triple-store cross-link integrity** | Applicable — graph_ref + embedding_blob + symbol_embeddings | 07 | Test cross-link queries; test orphan detection; test join correctness |
| **Self-improvement correctness** | Applicable — improve() re-weighting | 08 | Test boost for high-usage; test decay for never-accessed; test no regression |
| **Session promotion data loss** | Applicable — L0/L1→graph | 09 | Test summary preserved; test graph node created; test no duplicate promotion |
| **Temporal edge filtering regression** | Applicable — valid_from/valid_to | 10 | Test active edges visible; test expired edges hidden; test history preserved |
| **Portable export completeness** | Applicable — COGX JSON | 11 | Test output contains observations, edges, embeddings; test import/export roundtrip |
| **DreamExecutor data corruption** | Applicable — consolidate/synthesize/prune | 12 | Test DreamLock prevents concurrent; test DreamRollback reverts on error |
| **Importance scoring skew** | Applicable — AKL scoring | 13 | Test importance score computation; test recency decay; test query-time ranking |
| **Explicit relations integrity** | Applicable — @relation edges | 14 | Test relation creation; test query; test confidence filtering |
| **Provenance immutability** | Applicable — provenance chain | 15 | Test provenance chain; test immutability after save |
| **Federated query correctness** | Applicable — federated pipeline | 16 | Test federated results; test importance ranking; test dedup |
| **DistillLock contention** | Applicable — lock per project | 17 | Test concurrent lock blocked; test lock timeout; test rollback |
| **Memory type misclassification** | Applicable — typed categories | 18 | Test type assignment; test fallback to auto-classification |
| **Async commit data loss** | Applicable — sync + async phases | 18 | Test memory_diff.json audit; test retry on failure |
| **LLM dedup false negatives** | Applicable — dedup before write | 18 | Test dedup detects semantic duplicates; test hash fallback |
| **Directory retrieval performance** | Applicable — hierarchical drill-down | 19 | Test with deep hierarchies; test depth limit |
| **Snapshot storage bloat** | Applicable — point-in-time views | 20 | Test snapshot creation; test auto-prune |
| **Handoff staleness** | Applicable — prefix + delta | 21 | Test delta freshness; test handoff generation |
| **Context envelope size** | Applicable — universal JSON | 22 | Test envelope size; test field filtering |
| **Hub analysis accuracy** | Applicable — hub files | 23 | Test hub identification; test impact analysis |
| **Skills hook interference** | Applicable — lifecycle hooks | 24 | Test hooks opt-in; test hook timeout |
| **Memfs scope resolution overhead** | Applicable — URI resolution | 25 | Test URI resolution speed; test with 10k+ observations |
| **Memfs coexists with mem list** | Applicable — structural vs flat | 25 | Verify `mem ls` and `mem list` both work; no conflict |
| **Documentation-like paths** | N/A: all under `.skillgrid/` or `internal/` | — | — |
| **Git repository selection** | N/A: no VCS automation in this change | — | — |
| **Commit state** | N/A: no commit automation | — | — |
| **Push state** | N/A: no push automation | — | — |
| **PR commands** | N/A: no PR automation | — | — |

## Migration / rollout

Additive `014_*` migrations via embed + `index_meta`. No feature flag needed — all changes are backward compatible. Rollback: drop migration files + new modules; leave **013** intact. The `store` handle cache can be disabled by setting `SKILLGRID_MNEMONIC_DISABLE_CACHE=1`. The triple-store linkage (`graph_ref`, `symbol_embeddings`) adds columns/tables that are harmless if callers don't use cross-link queries. The `improve()` feedback loop is opt-in via config. Temporal edges default to `valid_to=NULL` (active). Session promotion fires only on `SessionEnd` with a valid summary. Export is always available regardless of other steps. DreamExecutor operations are opt-in via `brv dream` equivalent. Importance scoring feeds into `mem_search` ranking without changing default ordering. @relation annotations are additive metadata. Provenance is set during curation and immutable. Federated query replaces parallel search seamlessly. DistillLock prevents concurrent distillation; DreamRollback ensures atomicity. Memory types are additive metadata. LLM dedup is opt-in. Directory retrieval is a new mode (default unchanged). Snapshots are opt-in. Handoff artifacts are opt-in. Context envelope is opt-in. Hub analysis is opt-in. Skills and hooks are opt-in per project. Memfs (`mem fs`) is an additional access layer alongside existing `mem search` — `mem ls`, `mem tree`, `mem find` query the same store with scope-based filtering. Research sources: cognee (topoteretes/cognee, 30k★) for triple-store, improve(), session-promotion, temporal-graph, COGX; ByteRover CLI (campfirein/byterover-cli) for DreamExecutor, AKL, @relation, provenance, SwarmCoordinator, DreamLockService; OpenViking (volcengine/OpenViking) for memory types, directory retrieval, async two-phase commit, observable trajectory, snapshots; codemap (JordanCoin/codemap) for handoff artifacts, working set, context envelope, hub impact, skills framework, hooks.

## Open questions

- [ ] Should TTL default be 7 days or 30 days? → 7 days is conservative; can bump to 30
- [ ] Should LLM extraction use the same LLM as distillation or a separate endpoint? → Same endpoint for simplicity; separate if quality differs
- [ ] Should the store cache use `sync.Map` or a `RWMutex` + `map`? → `sync.Map` for simplicity; `RWMutex` if profiling shows contention
- [ ] Should parallel search use a worker pool or one-shot goroutines? → One-shot goroutines with semaphore; simpler for <20 projects
- [ ] Should graph_ref point to codeindex symbol ID or a separate graph node ID? → Symbol ID for simplicity; separate node ID if graph grows complex
- [ ] Should improve() re-weighting happen in-memory or persist to DB? → In-memory (boosted ordering in mem_search); persists via retrieval_usage column
- [ ] Should temporal edge valid_to default to NULL (active) or now+30d? → NULL (active until explicitly expired)
- [ ] Should export include binary embeddings or skip them for smaller payloads? → Include base64-encoded; allow `--skip-embeddings` flag for smaller output
- [ ] Should session promotion require explicit confirmation or happen automatically? → Automatic on SessionEnd; the quality threshold prevents low-quality promotions
- [ ] Should DreamExecutor run on a background timer or only on SessionEnd? → Background timer (e.g., every 5 min) + SessionEnd trigger; ByteRover uses both
- [ ] Should importance scoring be opt-in or default? → Default; configurable off via config
- [ ] Should @relation types be extensible or fixed? → Fixed set initially; extensible via config later
- [ ] Should provenance be stored as JSON or structured columns? → JSON column; structured columns if query patterns demand it
- [ ] Should federated query use a centralized merge or distributed? → Centralized merge in service layer; distributed if scale demands
- [ ] Should graph_ref point to codeindex symbol ID or a separate graph node ID? → Symbol ID for simplicity; separate node ID if graph grows complex
- [ ] Should improve() re-weighting happen in-memory or persist to DB? → In-memory (boosted ordering in mem_search); persists via retrieval_usage column
- [ ] Should temporal edge valid_to default to NULL (active) or now+30d? → NULL (active until explicitly expired)
- [ ] Should export include binary embeddings or skip them for smaller payloads? → Include base64-encoded; allow `--skip-embeddings` flag for smaller output
- [ ] Should session promotion require explicit confirmation or happen automatically? → Automatic on SessionEnd; the quality threshold prevents low-quality promotions
- [ ] Should DreamExecutor run on a background timer or only on SessionEnd? → Background timer (e.g., every 5 min) + SessionEnd trigger; ByteRover uses both
- [ ] Should importance scoring be opt-in or default? → Default; configurable off via config
- [ ] Should @relation types be extensible or fixed? → Fixed set initially; extensible via config later
- [ ] Should provenance be stored as JSON or structured columns? → JSON column; structured columns if query patterns demand it
- [ ] Should memfs scopes be project-scoped or user-scoped? → Project-scoped initially; user-scoped if multi-tenancy added
- [ ] Should memfs replace mem list or coexist? → Coexist; `mem ls` is structural browsing, `mem list` is flat output

## Glossary

| Term | Definition | Glossary file |
|------|------------|---------------|
| **Store Pooling** | Cached `ProjectHandle` by project ID with refcounted close; eliminates N+1 opens | technical |
| **Trigram FTS** | FTS5 query splitting into 3-char trigrams for partial-match recall | technical |
| **TTL Soft-Expiry** | Auto-set `expires_at` on save; `TTLRetire` soft-deletes expired observations | technical |
| **LLM Extraction** | LLM-based passive learning extraction with regex fallback | technical |
| **Multi-Embedder** | Support for `ollama`, `local`, `onnx`, `external`, `off` embedder providers | technical |
| **Parallel Cross-Project Search** | Concurrent `SearchObservationsAll` with bounded goroutine pool | technical |
| **Triple-Store Cross-Linkage** | Relational + Vector + Graph stores cross-linked via `graph_ref` and `symbol_embeddings`; every observation has both a vector and a graph representation | technical |
| **Self-Improvement (`improve()`)** | Re-weight observations by `retrieval_usage`; high-usage observations rank higher, never-accessed decay | technical |
| **Session-to-Graph Promotion** | L0/L1 session summaries auto-create permanent graph nodes on `SessionEnd` | technical |
| **Temporal Knowledge Graph** | Graph edges with `valid_from`/`valid_to` timestamps; expired edges hidden but preserved | technical |
| **Portable Export (COGX-inspired)** | JSON export of observations, graph edges, and embeddings for data portability | technical |
| **Dream Executor** | Decompose distillation into consolidate/synthesize/prune; DreamLockService + DreamRollback. Based on ByteRover's DreamExecutor | technical |
| **Importance Scoring (AKL)** | Adaptive Knowledge Lifecycle: importance_score + maturity_tier + recency_decay; feeds query-time ranking. Based on ByteRover's AKL | technical |
| **Explicit Relations** | @relation typed edges between observations (mentions, depends_on, contradicts, supports, references). Based on ByteRover's Context Tree | technical |
| **Provenance Chain** | Immutable JSON metadata: session_id → curate_command → source_files → LLM_reasoning. Based on ByteRover's provenance tracking | technical |
| **Federated Query** | Cross-project search with importance ranking + dedup; each store returns importance-scored results. Based on ByteRover's SwarmCoordinator | technical |
| **Distill Lock + Rollback** | DistillLockService ensures single dream per project; DistillRollback reverts on failure. Based on ByteRover's DreamLockService | technical |
| **Memory Types (OpenViking)** | Typed observation categories: profile, preferences, entities, events, identity, soul, cases, trajectories, experiences; LLM dedup before write; async two-phase commit. Based on OpenViking's memory types. | `014` memory-types step |
| **Directory Retrieval (OpenViking)** | Hierarchical retrieval: find highest-scoring directory first, drill down layer by layer; observable retrieval trajectory preserved. Based on OpenViking's directory recursive retrieval. | `014` directory-retrieval step |
| **Snapshots + Transaction Locking (OpenViking)** | Point-in-time store snapshots; row-level locking for concurrent access. Based on OpenViking's multi-version management. | `014` snapshots step |
| **Handoff Artifacts (codemap)** | Structured handoff with stable prefix + dynamic delta; enables agent-to-agent context passing without re-briefing. Based on codemap's handoff artifacts. | `014` handoff step |
| **Context Envelope (codemap)** | Universal JSON containing project metadata + working set + intent classification + matched skills + handoff references. Based on codemap's ContextEnvelope. | `014` context-envelope step |
| **Hub Files + Impact Analysis (codemap)** | Hub file identification (3+ importers); impact analysis based on changes to hub files; risk_score on observations. Based on codemap's hub file analysis. | `014` hub-impact step |
| **Skills + Lifecycle Hooks (codemap)** | Markdown-based skills matched by intent, mentioned files, project languages; lifecycle hooks (session-start, pre-edit, prompt-submit, session-stop) for context injection. Based on codemap's skills framework and hooks. | `014` skills-hooks step |
| **Memfs (OpenViking adaptation)** | Virtual filesystem layer with `mem://` URI space; `mem ls`, `mem tree`, `mem find` alongside existing `mem search`. Queries the same store — filters by scope + memory_type. Based on OpenViking's VikingFS. | `014` memfs step; `mem ls` CLI. | "mem list" (flat, not hierarchical); "file system" (too generic). |

## Author self-review

- [x] **Goal**, **Out of scope / Non-Goals**, and **Definition of Done** are filled and testable
- [x] **Error handling** and **Testing strategy** are filled
- [x] Non-goals match Global Constraints that will appear in `tasks.md`
- [x] Rollback plan is present
- [x] Step Blueprint covers a vertical-slice sequence (no horizontal-only layers)
- [x] Every Impacted Files row maps to exactly one step
- [x] Every applicable threat row names an owning step
- [x] Glossary terms reused or defined; no companion reference file
- [x] Size budget: under 450 words for the core proposal
- [x] ByteRover-inspired lessons (DreamExecutor, AKL, @relation, provenance, federated query, DreamLock) documented with rationale
- [x] cognee, ByteRover, OpenViking, codemap sources cited in research notes
- [x] memfs (`mem ls`/`mem tree`/`mem find` alongside `mem search`) documented as additive access layer

<!-- Fold new terms here; also upsert docs/skillgrid/agents/glossary/{business,technical}.md. No companion *-glossary-reference.md. Do not edit glossary files in this migration. -->
