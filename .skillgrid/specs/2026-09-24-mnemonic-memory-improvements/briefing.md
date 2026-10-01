# Briefing — Mnemonic Memory Improvements (from Mnemon + mnemonic-ai comparison)

> **STATUS:** `draft` (2026-09-24)

**Topic:** 2026-09-24-mnemonic-memory-improvements
**Date:** 2026-09-24
**Classification:** standard (T2)
**Build shape:** Smallest usable whole
**Findings:** `.skillgrid/specs/2026-09-24-mnemon-comparison/findings.md`

## Why

A comparison against two external memory systems — Mnemon (mnemon-dev/mnemon, Go) and mnemonic-ai (Aamirofficiall/mnemonic, Rust) — surfaced ideas our engine does not have. Three are high-ROI and composable with existing 014 work:

1. **RRF fusion in the MCP search path** — our MCP `mem_search` uses pure BM25; the CLI fact path has RRF infrastructure (`BlendedSearch`) but passes an empty vector, so the vector leg never runs. The code hybrid path (`hybrid/rank.go`) is the only true multi-leg RRF in the engine. Both reference systems converged on RRF independently.
2. **Reinforcement decay in search ranking** — `retrieval_usage` exists and is bumped per hit, but it feeds AKL importance scoring (014 step 13), not `mem_search` ranking directly. Both reference systems use access-frequency in their decay formula. Our 014 step 13 (`importance_score = retrieval_count × exp(-decay_rate × age_days)`) is close but not the same as the Mnemon/mnemonic-ai formula which uses `ln(1 + access_count)` in the denominator to reinforce frequently-accessed facts.
3. **Signal transparency in search output** — neither Mnemon nor mnemonic-ai returns a per-signal score breakdown. Adding `matched_via` + per-signal scores to `mem_search` results lets the agent (and user) see *why* something was recalled, enabling better re-ranking judgments.

## What's already built (do NOT re-plan)

| Capability | Where | Status |
|---|---|---|
| `retrieval_usage` column + `BumpRetrievalUsage` | `memory/governance.go:237-245` | Built |
| AKL importance scoring (014 step 13) | `memory/importance.go:381-416` | Partial (PENDING) |
| Improve loop (014 step 08) | `memory/importance.go` | Partial (PENDING) |
| RRF function `ReciprocalRankFusion` | `memory/embedding.go:74-106` | Built (k=60) |
| `BlendedSearch` (FTS + vector RRF) | `memory/search_embed.go:117-175` | Built (vector leg unused in layered path) |
| Code hybrid RRF (3 legs) | `hybrid/rank.go:70-113` | Built (RRFK=60) |
| Intent detection (`ClassifyIntent`) | `memory/retrieval.go:70-80` | Built (4 intents, not wired into search) |
| `observation_relations` table (014 step 14) | migration 026 | Partial (PENDING) |
| Temporal graph edges (014 step 10) | migration 025 | Partial (PENDING) |
| Session events + monitoring | session-events-layer + monitoring spec | In-flight |
| Session inject (L1 + hybrid retrieval) | session-inject spec | In-flight |

## Capabilities to add

### C1: RRF fusion in MCP `mem_search`

**Problem:** MCP `mem_search` calls `SearchOwnerScoped` which is pure BM25 + optional importance reweight. The vector leg (`SearchByVector`) exists but is never invoked in the MCP path. When `MNEMONIC_EMBED=1`, the vector signal is available but unused.

**Solution:** Wire the existing `BlendedSearch` into the MCP `mem_search` path. When an embedder is active and the query is embedded, run both legs (FTS + vector) and fuse via RRF (k=60, already implemented). When no embedder is active, degrade to BM25-only (existing behavior). This is the same degrade-to-Null pattern used by the code hybrid path.

**Key detail:** The MCP handler must embed the query before calling search. The existing `SearchByVector` expects a `Vector` — the handler needs to call the embedder on the query text. This is a new dependency in the MCP path (currently it goes directly to FTS SQL).

**Depends on:** Nothing new (all infrastructure exists). Can be built independently of 014 steps.

### C2: Reinforcement decay in search ranking

**Problem:** `retrieval_usage` is bumped per hit but doesn't influence `mem_search` ranking. The 014 step 13 AKL formula (`retrieval_count × exp(-decay_rate × age_days)`) multiplies access count by exponential decay — a fact accessed 100× gets 100× the boost of one accessed once. The Mnemon/mnemonic-ai formula (`importance × exp(-λt / (1 + α·ln(1+access_count)))`) divides the decay rate by the access reinforcement — a fact accessed 100× decays ~3× slower. The latter is gentler and more stable for high-access facts.

**Solution:** Add a reinforcement decay factor to the `mem_search` ranking. Two options:
- **Option A (Mnemon-style):** `EI = base_weight(importance) × max(1.0, log(1 + access_count)) × 0.5^(days_since_access/30) × edge_factor`. Compute at query time. 30-day half-life. Immunity at `importance >= 4` OR `access_count >= 3`.
- **Option B (integrate with 014 step 13):** Extend the existing AKL formula to use the reinforcement denominator. `importance_score = retrieval_usage × exp(-decay_rate × age_days / (1 + α × ln(1 + retrieval_usage)))`. This changes 014 step 13's formula — requires coordination.

**Decision:** Option A is independent of 014 step 13 and can ship separately. Option B is more elegant but couples to 014. Ship Option A now; revisit Option B when 014 step 13 lands.

**Depends on:** `retrieval_usage` column (built). `importance_score` column (014 step 13, optional — falls back to `importance` field if absent).

### C3: Signal transparency in search output

**Problem:** `mem_search` returns a flat list of results with no explanation of *how* each result was scored. The agent cannot distinguish a keyword match from a vector match from a recency boost.

**Solution:** Add a `signals` object to each search result:
```json
{
  "id": 42,
  "title": "...",
  "content": "...",
  "score": 0.73,
  "signals": {
    "keyword": 0.82,
    "vector": 0.65,
    "recency": 0.4,
    "entity": 0.0,
    "decay": 0.9,
    "importance": 0.7
  },
  "matched_via": "hybrid"
}
```

`matched_via` is one of: `keyword` (BM25 only), `vector` (cosine only), `hybrid` (both), `recency` (age-sorted fallback), `basic` (SQL LIKE). Per-signal scores are normalized 0-1.

**Depends on:** C1 (RRF produces the per-signal breakdown naturally).

### C4: Entity aliases

**Problem:** The code graph has no alias table. An agent searching "the renderer" won't find `comp_renderer.cpp`. Mnemonic-ai has `entity_aliases` with case-insensitive LIKE matching.

**Solution:** Add an `entity_aliases` table to the code index. Each alias maps to a symbol's qualified name. Populated during indexing (tree-sitter extracts the canonical name; aliases come from comments, docstrings, and a manual `mem alias add` command). Query-time: the entity search path checks aliases before falling back to FTS.

**Depends on:** Code index (built). New migration.

### C5: Query embedding cache

**Problem:** Every `semantic_search` / `hybrid_search` call re-embeds the query. Repeated queries (common in agent loops) recompute the same embedding.

**Solution:** Add a `query_cache` table: `(query_hash TEXT PRIMARY KEY, embedding BLOB, created_at TIMESTAMP)`. Before embedding, check the cache. TTL: 7 days (embeddings are stable; the model rarely changes). Populated on first embed, read on subsequent.

**Depends on:** Nothing new (embedder interface exists).

### C6: Auto-learning observer

**Problem:** Our system is 100% manual `mem_save`. An agent that forgets to call `mem_save` loses the discovery. Mnemonic-ai's `mnemonic observe` ingests tool call JSON from hooks and auto-extracts facts.

**Solution:** This is largely covered by the monitoring spec (session-events-layer → monitoring → session-inject chain). The monitoring spec wires `hookPostToolUse` to capture every tool call. The 014 step 05 LLM extraction (`CapturePassive` with `ExtractWithLLM`) already does the fact extraction. What's missing is the **trigger**: the plugin hook must fire on every tool call (not just Task) and POST to a new route. The monitoring spec covers this.

**Decision:** C6 is NOT a new capability — it's already planned in the monitoring spec. Listed here for completeness; no new tasks.

### C7: Pre-compaction hook

**Problem:** When the context window is about to compact, critical decisions in the conversation can be lost. Mnemon's "Compact" hook phase fires before compaction: "preserve only critical continuity."

**Solution:** Add a `Compact` lifecycle hook to the plugin. Before the runtime compacts the conversation, the hook calls `mem_save` with the session's most important observations (high `importance_score`, recent, not yet persisted). The hook is observe-only (never blocks compaction), fail-open (3s timeout).

**Depends on:** Plugin hook infrastructure (session-events-layer). Monitoring spec's hook wiring.

## One-way-door decisions

| # | Decision | Risk | Mitigation |
|---|---|---|---|
| 1 | C1 changes MCP `mem_search` response shape (adds `signals` object, changes scoring from pure BM25 to RRF) | Existing consumers may not expect `signals` field; scoring changes may reorder results | Additive field (old consumers ignore unknown fields); `matched_via: "keyword"` preserves old behavior when no embedder is active; ADR records the change |
| 2 | C2 changes search ranking (adds decay factor) | Result ordering changes; previously-top results may drop | Config-gated (`mnemonic.decay.enabled`, default `true`); `mnemonic.decay.half_life_days` (default 30); `mnemonic.decay.immunity_min_importance` (default 4) |
| 3 | C4 new migration (`entity_aliases` table) | Schema change; migration must be idempotent | `CREATE TABLE IF NOT EXISTS`; additive only; no data migration needed |

## Trust boundaries

- **MCP tool contract:** C1 changes the `mem_search` MCP tool's response shape. The `signals` field is additive; existing consumers that ignore unknown fields are unaffected. Consumers that do strict schema validation will need updating.
- **No new trust boundary:** C1-C5 all operate within the existing per-project SQLite store. No new external service, no new network boundary.
- **Query cache (C5):** The `query_cache` table stores query text (hashed) + embedding. No PII in the hash input (the query is the agent's search terms). Embedding is a BLOB (numeric vector, not reconstructable to text).

## Success criteria

1. `mem_search` with `MNEMONIC_EMBED=1` returns RRF-fused results with per-signal scores.
2. `mem_search` without an embedder returns BM25-only results (no regression).
3. Frequently-accessed observations rank higher than equally-relevant but rarely-accessed ones (reinforcement decay).
4. Every `mem_search` result includes a `signals` object with per-signal scores.
5. `entity_aliases` table exists and is queryable via the code search path.
6. Repeated identical queries skip re-embedding (query cache hit).
7. All existing tests pass (no regression).
8. New tests cover: RRF fusion, decay ranking, signal transparency, alias lookup, query cache.

## ADR needed

ADR-0017 (next free sequence; 0011 was double-claimed by bitemporal-audn and is now owned by it): "MCP mem_search response includes per-signal score breakdown (additive `signals` field; `matched_via` discriminator; RRF fusion when embedder active)."

## Out of scope (listed but not in this change)

- Four-graph model (temporal/entity/causal/semantic edges between observations) — large design, needs its own brainstorming
- Bi-temporal model (`valid_at`/`invalid_at` + supersede chains) — needs its own design
- AUDN cycle (LLM classification on save) — depends on 014 step 05 LLM extraction landing
- Token-budgeted context (`context --budget 5000`) — session-inject spec covers the retrieval side
- Rolling caps — config-only, can be added as a follow-up
- 13-language intent detection — our `ClassifyIntent` covers the 4 intents we need; multilingual is a stretch goal
