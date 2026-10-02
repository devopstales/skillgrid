# Mnemonic Memory Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Smallest usable whole

**Classification:** risky (migration + MCP contract delta). Verification floor L3. Briefing class was standard; the migration and the `mem_search` shape change raise the floor. Per `ASSUMPTIONS.md` § Locked constraints, serial development stays on this branch (`size:exception`).

**Goal:** Make `mem_search` explain and re-rank its hits, cache query embeddings, resolve code-index aliases, and persist a continuity note before compaction.

**Architecture:** Keep `BlendedSearch` for `mem_ask`. Add `SearchOwnerScopedBlend` beside `SearchOwnerScoped` so both legs honor `visibilityFilter` (ADR-0018). Decay is a pure function applied after fusion. MCP copies `SearchHit` fields onto the existing observation DTO. Migration `045_search_aids.sql` adds `query_cache` and `entity_aliases` only.

**Tech Stack:** Go 1.22+, existing SQLite migrations, `modernc.org/sqlite`, MCP `mcp-go`. No new dependencies.

**Spec:** `.skillgrid/specs/2026-09-24-mnemonic-memory-improvements/briefing.md`

**Findings:** `.skillgrid/specs/2026-09-24-mnemon-comparison/findings.md` is deleted in the working tree; the briefing is the retained comparison record. Do not recreate that findings file.

**One-way approval:** the 2026-10-02 execute request authorizes the three briefing doors (additive `signals`, decay default on, migration 045). Do not stop again for those.

## Hypothesis

**Claim:** Owner-scoped RRF plus query-time reinforcement decay changes `mem_search` order only when an embedder is active or decay factors differ, and every hit then carries `signals`.
**Right condition:** `TestOwnerScopedBlendKeywordFloor` keeps BM25 order with `matched_via=keyword`; `TestReinforcementDecayRanksHotRow` puts the hot row first; `TestOwnerScopedBlendHidesPrivate` still hides the other owner.
**Wrong condition:** a private row leaks, keyword-only order moves when every `LastSeenAt` is empty and every `retrieval_usage` is 0, or `signals` is missing.
**Thinnest MVP:** Task 1 decay function + Task 2 blend and DTO. Tasks 3–6 are growth.
**Door check:** Task 1. If the pure decay tests fail to rank the hot row first while keeping equal factors stable, stop and revise the formula.

## Terms

- Search Signals, Reinforcement Decay, Entity Alias, Query Embedding Cache, Compact Hook — `.skillgrid/artifacts/02-technical-terms.md`
- Visibility stays the ADR-0005 / change-013 owner filter. Do not rename it.

## Must-Haves (goal-backward verification)

**Truths:**
- With `MNEMONIC_EMBED` unset, `mem_search` order matches today's BM25 order when `LastSeenAt` is empty and `retrieval_usage` is 0, and each hit has `matched_via=keyword`, `signals.vector=0`, and the other signal keys present in [0,1]. (`backstop`: needs a live MCP handler test.)
- With an embedder and stored embeddings, fusion uses `ReciprocalRankFusion` k=60 and `matched_via` is `hybrid` when the id is on both legs, `vector` when it is on the vector leg only.
- Reader B does not receive reader A's private observation from the blend.
- An embedder error returns the FTS leg with `matched_via=keyword`.
- Two equal-BM25 rows, `LastSeenAt` 40 days ago, `retrieval_usage` 20 vs 0, decay enabled: the 20-usage row is first. Decay disabled: BM25 order remains.
- Importance ≥ 4 or `retrieval_usage` ≥ 3 sets the half-life term to 1.
- A second `EmbedQuery` of the same text and model within 7 days does not call the inner embedder. Older than 7 days or a different model does.
- Alias `the renderer` resolves to the stored `qualified_name` case-insensitively. A miss returns no alias rows.
- `RunHook(ctx, "compact", payload)` with hooks enabled writes one observation `topic_key=compaction/<sessionID>`. A timeout returns a result to the caller (fail-open) and does not return `hookTimeoutError` for this hook type.
- C6 (auto-learning observer) has no task.

**Artifacts:**
- `skillgrid-cli/internal/mnemonic/memory/decay.go` — pure reinforcement factor
- `skillgrid-cli/internal/mnemonic/memory/search_blend.go` — `SearchOwnerScopedBlend`
- `skillgrid-cli/internal/mnemonic/memory/query_cache.go` — cache get/put
- `skillgrid-cli/internal/mnemonic/store/migrations/045_search_aids.sql` — both tables
- `skillgrid-cli/internal/mnemonic/codeindex/aliases.go` — alias lookup
- tests named in the tasks below

**Key links:**
- `handleMemSearch` calls `SearchOwnerScopedBlend`, not `BlendedSearch` and not `SearchOwnerScoped`, for the single-project branch.
- `budgetedObservationDTOs` copies `SearchHit.Signals` onto each object. The all-projects branch stays on `SearchAllProjects` and stamps `matched_via=keyword` with zero vector (no cross-store embed in this change).
- `rankByDecay` runs only when `DecayConfig.Enabled` is true (zero value means enabled). When false, `SearchOwnerScopedBlend` uses today's `rankByUse` / `importanceRerank` order.
- Alias lookup is called from the code search entry that already queries `symbols` / chunks, before the FTS query string is built. Name that function in the task from the call site you read; do not add a second search stack.

**One-way-door decisions:**
- Additive `signals`, `matched_via`, and `score` on `mem_search` (authorized 2026-10-02).
- `mnemonic.decay.enabled` default true (authorized 2026-10-02).
- Migration `045_search_aids.sql` (authorized 2026-10-02).

## Global Constraints

- Go 1.22+ minimum to build. Per `ASSUMPTIONS.md` § Locked constraints.
- No new dependencies. Per `ASSUMPTIONS.md` § Locked constraints.
- Spec-zone commits before code-zone commits.
- RRF k=60, the constant already used by `ReciprocalRankFusion`.
- `edge_factor` = 1. Per ADR-0018.
- Days-since-access reads `LastSeenAt` only; empty → 0. Per ADR-0018.
- Compact hook budget is 3s, independent of `Hooks.Timeout` (default 30s). Fail-open.
- Query cache TTL is 7 days. Hash is SHA-256 of `model + "\n" + query` hex. Store the hash, not the query text.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`mem_search` return shape) | Applicable | Additive `signals`, `matched_via`, `score`. Existing keys stay. Missing `query` stays an error. Private rows stay hidden. | `TestMemSearchSignalsKeyword`, `TestOwnerScopedBlendHidesPrivate`, `TestMemSearchRequiresQuery` |
| Documentation-like paths | N/A: no path classification | — | — |
| Git repository selection | N/A: no git invocation | — | — |
| Commit state | N/A: no commit automation in the product change | — | — |
| Push state | N/A: no push | — | — |
| PR commands | N/A: no PR automation | — | — |
| Shared-convention drift | N/A: no `_shared/conventions` edit | Terms rows are glossary, not a convention file | — |

## File Structure

- `skillgrid-cli/internal/mnemonic/memory/decay.go` — `DecayConfig`, `Reinforcement`, `rankByDecay`
- `skillgrid-cli/internal/mnemonic/memory/decay_test.go` — pure ranking tests
- `skillgrid-cli/internal/mnemonic/memory/search_blend.go` — `SearchHit`, `SearchSignals`, `SearchOwnerScopedBlend`
- `skillgrid-cli/internal/mnemonic/memory/search_blend_test.go` — keyword floor, hybrid, privacy, embedder error
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` — handler + DTO
- `skillgrid-cli/internal/mnemonic/config/load.go` — `mnemonic.decay`
- `skillgrid-cli/internal/mnemonic/service/service.go` — `SetDecay` from config
- `skillgrid-cli/internal/mnemonic/memory/query_cache.go` — cache
- `skillgrid-cli/internal/mnemonic/store/migrations/045_search_aids.sql`
- `skillgrid-cli/internal/mnemonic/codeindex/aliases.go`
- `skillgrid-cli/internal/mnemonic/memory/skills.go` — `HookCompact`

---

### Task 1: Reinforcement decay

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/decay.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/decay_test.go`

**Interfaces:**
- Consumes: `Observation.RetrievalUsage`, `Observation.LastSeenAt`, `Observation.ImportanceScore`, `Observation.Pinned`
- Produces:
  - `type DecayConfig struct { Enabled bool; HalfLifeDays float64; ImmunityMinImportance float64; ImmunityMinAccess int }`
  - `func DefaultDecayConfig() DecayConfig` — Enabled true, HalfLifeDays 30, ImmunityMinImportance 4, ImmunityMinAccess 3
  - `func (s *Service) SetDecay(cfg DecayConfig)` — non-positive half-life becomes 30; immunity minima below 0 become the defaults; `Enabled` is taken as given. A zero `DecayConfig{}` passed to `SetDecay` becomes `DefaultDecayConfig()` so the zero value means "defaults on".
  - `func Reinforcement(o Observation, now time.Time, cfg DecayConfig) float64`
  - `func rankByDecay(results []Observation, now time.Time, cfg DecayConfig) []Observation`
- Seam: none (in-process)
- Deletion test: ranking policy would be inlined in search
- Adapters: 1 (the function). No port.

**SATISFIES:** high retrieval_usage outranks a cold twin

Formula, natural log:

```
base = 1
if ImportanceScore.Valid && ImportanceScore.Float64 > 0 { base = ImportanceScore.Float64 }
days = 0
if LastSeenAt parses as RFC3339 or SQLite "2006-01-02 15:04:05" { days = now.Sub(t).Hours()/24; if days < 0 { days = 0 } }
half = 0.5 ^ (days / cfg.HalfLifeDays)
if base >= cfg.ImmunityMinImportance || retrieval_usage >= cfg.ImmunityMinAccess { half = 1 }
reinforce = max(1, ln(1+retrieval_usage))
return base * reinforce * half * 1  // edge_factor
```

`rankByDecay` keeps pinned rows first, then sorts the rest by `Reinforcement` descending, stable. Equal factors preserve input order.

Signal helpers (same file, used by Task 2):

```
func SignalRecency(o Observation, now time.Time, cfg DecayConfig) float64  // the half term before immunity, clamped to [0,1]
func SignalDecay(o Observation, now time.Time, cfg DecayConfig) float64    // min(1, reinforce/ln(1+100)) so 100 accesses ≈ 1
func SignalImportance(o Observation) float64                                 // min(1, base/5)
```

- [ ] **Step 1: Write the failing test**

`TestReinforcementDecayRanksHotRow`: two observations, same title rank implied by slice order (cold first), `LastSeenAt` = now−40 days, usage 0 and 20, default config. Expect usage-20 first.

`TestDecayImmunityFreezesHalfLife`: importance 4, `LastSeenAt` 400 days ago, usage 0. `Reinforcement` equals `base * max(1, ln(1)) * 1` (half forced to 1).

`TestEqualDecayKeepsOrder`: both usage 0 and empty `LastSeenAt`. Output order equals input order.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/memory/ -count=1 -run 'TestReinforcementDecayRanksHotRow|TestDecayImmunityFreezesHalfLife|TestEqualDecayKeepsOrder'`
Expected: FAIL (undefined)

- [ ] **Step 3: Write minimal implementation** of the formula above.

- [ ] **Step 4: Run test to verify it passes**

Same command. Expected: PASS

- [ ] **Step 5: Commit** code only (`feat(mnemonic): rank search hits by reinforcement decay`).

### Task 2: Owner-scoped blend and mem_search signals

> ⚠ one-way: additive `mem_search` fields and RRF ordering when an embedder is active. Approved 2026-10-02. Proceed.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/search_blend.go`
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (`SearchOwnerScoped` stays for other callers)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` (`handleMemSearch`, `observationDTO`, `budgetedObservationDTOs`)
- Test: `skillgrid-cli/internal/mnemonic/memory/search_blend_test.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_signals_test.go`

**Interfaces:**
- Consumes: Task 1 `rankByDecay`, `SignalRecency`, `SignalDecay`, `SignalImportance`; existing `SearchOwnerScoped` SQL shape; `ReciprocalRankFusion`; `EmbeddingEnabled`; `SearchByVector`
- Produces:

```go
type SearchSignals struct {
    Keyword, Vector, Recency, Entity, Decay, Importance float64
}
type SearchHit struct {
    Observation Observation
    Score       float64
    MatchedVia  string // keyword | vector | hybrid
    Signals     SearchSignals
}
func (s *Service) SearchOwnerScopedBlend(ctx context.Context, readerOwner, readerAgent, query, matchMode, scope string, limit int, queryVec Vector) ([]SearchHit, error)
```

- Seam: embedder is a `Vector` argument, not a new interface. The handler embeds before the call.
- Deletion test: `handleMemSearch` would call `BlendedSearch` and drop the visibility filter
- Adapters: production handler + tests

**SATISFIES:** mem_search without an embedder is keyword only

Behavior:

1. Run the same SQL as `SearchOwnerScoped` (visibility, scope, invalid_at, project, FTS) but do not call `rankByUse` inside it. Duplicate the query in `SearchOwnerScopedBlend` rather than changing `SearchOwnerScoped`'s return type. Bump retrieval usage the same way, after the scan, on the pre-bump structs used for ranking.
2. If `len(queryVec.Data)==0` or `!EmbeddingEnabled()`, there is no vector leg.
3. Else `SearchByVector`. Drop any vector hit whose id fails `canRead` for this reader (use the existing read check; if the method is unexported, call it from this package). Fetch missing observations with `Get` and drop get errors.
4. Fuse with `ReciprocalRankFusion(..., 60)`. `matched_via`: both maps → `hybrid`; vector only → `vector`; fts only → `keyword`.
5. `signals.keyword` = 0 if absent from FTS else `1/float64(1+ftsRank)`. `signals.vector` = 0 if absent else `max(0, min(1, cosine))`. `signals.entity` = 0. Recency/decay/importance from Task 1. `score` = keyword when vector is 0, else `min(1, rrf / (2.0/61.0))`.
6. If decay enabled, sort by pinned first, then `Reinforcement * score` descending, stable. If disabled, keep the fused order (FTS order when there is no vector leg) and still fill signals with recency 1 and decay 1 only when you skip `rankByDecay` — still report the real signal numbers; just do not reorder. Disabled means: do not call `rankByDecay`.
7. Truncate to limit.
8. Empty FTS query returns `nil, nil` (same as today).

Handler, single-project branch: resolve an embedder only when `memory.EmbeddingEnabled()` is true. Use `embedder.Default().EmbedQuery`. On error, pass an empty `Vector` (keyword floor). Replace the `SearchOwnerScoped` call with `SearchOwnerScopedBlend`. Map hits through a new `budgetedSearchHitDTOs` that applies the same preview/unfold budget to `hit.Observation` and then sets:

```go
m["score"] = hit.Score
m["matched_via"] = hit.MatchedVia
m["signals"] = map[string]any{
  "keyword": hit.Signals.Keyword, "vector": hit.Signals.Vector,
  "recency": hit.Signals.Recency, "entity": hit.Signals.Entity,
  "decay": hit.Signals.Decay, "importance": hit.Signals.Importance,
}
```

All-projects branch: leave `SearchAllProjects`. After `budgetedObservationDTOs`, set `matched_via=keyword`, `score` omitted or 0, and signals with vector 0 and entity 0 so the key exists. Do not embed across stores.

`TestMemSearchContractShape` must still pass: new keys are additive. Update it only if it rejects unknown keys. Do not remove existing keys.

- [ ] **Step 1: Failing tests** `TestOwnerScopedBlendKeywordFloor`, `TestOwnerScopedBlendHidesPrivate`, `TestOwnerScopedBlendHybrid` (hash embedder or a fixed `Vector` stored via `SetEmbedding` and passed in), `TestOwnerScopedBlendEmbedderError` is the handler test with a stub that errors — if the service method cannot see the embedder, the handler test covers the error path and the service test covers an empty vector. `TestMemSearchSignalsKeyword` asserts JSON keys. `TestMemSearchRequiresQuery` already exists in `tools_memory_retrieval_test.go`; do not duplicate if the name differs — point G2 at the existing test name if you find it, and keep a test that fails the tool when `query` is absent.

- [ ] **Step 2: RED** `cd skillgrid-cli && go test ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ -count=1 -run 'TestOwnerScopedBlend|TestMemSearchSignals'`

- [ ] **Step 3: Implement**

- [ ] **Step 4: GREEN** plus `cd skillgrid-cli && go test ./internal/mnemonic/mcp/ ./internal/mnemonic/memory/ -count=1`

- [ ] **Step 5: Commit** `feat(mnemonic): add owner-scoped RRF signals on mem_search`

### Task 3: Decay config

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go` and its test
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` where `SetImprove` is called
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` field `decayCfg`

**Interfaces:**
- Consumes: `DefaultDecayConfig`, `SetDecay`
- Produces: YAML

```yaml
mnemonic:
  decay:
    enabled: true          # default true when the section is absent
    half_life_days: 30
    immunity_min_importance: 4
    immunity_min_access: 3
```

Explicit `enabled: false` opts out. Absent section → defaults on.

- Seam: config file → `SetDecay`
- Deletion test: decay knobs would be constants in the ranker
- Adapters: YAML loader + in-memory `SetDecay` (tests)

**SATISFIES:** decay disabled keeps BM25 order

- [ ] Failing config test: absent section yields Enabled true and half-life 30; `enabled: false` stays false.
- [ ] RED, implement, GREEN.
- [ ] Service open path calls `mem.SetDecay(...)`.
- [ ] `TestDecayDisabledKeepsBM25` on the blend with `SetDecay(DecayConfig{Enabled: false})`.
- [ ] Commit `feat(mnemonic): gate reinforcement decay from config`

### Task 4: Query embedding cache

> Migration file is shared with Task 5. This task creates `045_search_aids.sql` with **only** `query_cache`. Task 5 appends `entity_aliases` in a later commit of the same file. If both land in one commit because Task 5 is next, that is fine; do not create 046.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/045_search_aids.sql`
- Create: `skillgrid-cli/internal/mnemonic/memory/query_cache.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/query_cache_test.go`
- Test: `skillgrid-cli/internal/mnemonic/store/migrations_045_test.go`
- Modify: the `EmbedQuery` call site used by `mem_search` (Task 2 handler) and `semantic_search` / observation embed path if it calls `EmbedQuery` directly. Wrap at one function `CachedEmbedQuery(ctx, db, model string, inner func(context.Context, string) (Vector, error), query string) (Vector, error)`.

```sql
CREATE TABLE IF NOT EXISTS query_cache (
    query_hash TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
```

Hash = hex SHA-256 of `model + "\n" + query`. On hit, if `created_at` is within 168 hours, `DecodeVector` and return. Else embed, `EncodeVector`, `INSERT OR REPLACE`. Do not store the query text.

**SATISFIES:** second identical query is a cache hit

- [ ] RED `TestQueryCacheHit` (counting embedder) and `TestQueryCacheMiss` (8 days old, and a different model). `TestMigration045` asserts the table exists once and a second open does not duplicate the migration row.
- [ ] GREEN. Commit `feat(mnemonic): cache query embeddings for seven days`

### Task 5: Entity aliases

> ⚠ one-way: migration 045 grows `entity_aliases`. Approved 2026-10-02. Proceed.

**Files:**
- Modify: `045_search_aids.sql`
- Create: `skillgrid-cli/internal/mnemonic/codeindex/aliases.go`
- Test: `skillgrid-cli/internal/mnemonic/codeindex/aliases_test.go`
- Modify: the code-index text search function that builds an FTS/LIKE query over symbols. Read it first. Before that query, `LookupAliases(ctx, db, project, text) ([]string, error)` returns `qualified_name` values where `alias = ? COLLATE NOCASE` and `project = ?`. If any names return, search those symbols by `qualified_name` and prepend them to the FTS hits, deduped by symbol id. A miss falls through unchanged.

```sql
CREATE TABLE IF NOT EXISTS entity_aliases (
    project TEXT NOT NULL,
    alias TEXT NOT NULL COLLATE NOCASE,
    qualified_name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'index',
    PRIMARY KEY (project, alias, qualified_name)
);
```

Populate during symbol write: insert one row `alias = symbol name` and one row `alias = qualified_name` with `source='index'`, `INSERT OR IGNORE`. No comment/docstring mining in this change (the briefing's comment source is deferred). Manual insert is just a row; no new CLI verb.

**SATISFIES:** alias lookup finds the symbol

- [ ] RED lookup test with a manually inserted alias `the renderer` → `comp_renderer.cpp`, query `The Renderer`.
- [ ] GREEN. Commit `feat(mnemonic): resolve code search entity aliases`

### Task 6: Compact hook

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/skills.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/compact_hook_test.go`
- Modify: `skillgrid-cli/cmd/skillgrid/mem.go` only if the hook type is a validated allow-list that would reject `compact`.

**Interfaces:**
- Consumes: `CompactionContext`, `Save` or `SaveWithAction`
- Produces: `HookCompact = "compact"` accepted by `RunHook`

**SATISFIES:** compact hook saves continuity

Behavior:

- Add the case next to the other hook constants. Unknown-type error string must include `compact`.
- `hookCompact`: if `SessionID` is empty, return `HookResult{}` and nil. Else call `CompactionContext(ctx, sessionID, 5)`. Build a short body of titles. `Save` one observation type `session_summary`, title `compaction continuity`, `topic_key` `compaction/` + sessionID, so a repeat upserts. Owner = sessionID.
- Timeout: run this hook with `min(configured timeout, 3s)`. On `context.DeadlineExceeded`, return `HookResult{Distilled: false}, nil` — do not return `hookTimeoutError`. Other errors also return `HookResult{}, nil` (fail-open) after the hook function itself returns the error to `RunHook`; catch it in the `compact` arm of `RunHook` before the generic error return.
- Hooks disabled still returns `hooksDisabledError` (the switch stays honest).

- [ ] RED `TestCompactHookSaves` and `TestCompactHookFailOpen` (install `SetHookFunc(HookCompact, ...)` that blocks until ctx.Done(), expect nil error and `Distilled==false`).
- [ ] GREEN. Commit `feat(mnemonic): add fail-open compact continuity hook`

## Self-Review

- C1 → Task 2. C2 → Tasks 1 and 3. C3 → Task 2. C4 → Task 5. C5 → Task 4. C6 → no task. C7 → Task 6.
- Must-have truths map to those tasks. One-way doors are tagged and pre-approved.
- No TBD placeholders. `SearchAllProjects` is intentionally not hybrid.
- Owed-decision gate: score formula, signal clamps, hash, TTL, immunity, edge factor, days clock, compact topic key, and alias sources are named above. No ASSUMED block.

## Plan Review

READY FOR EXECUTION. First wave is Task 1 (door). Task 2 completes the smallest usable whole.

## Execution Handoff

Six tasks. Slice, then subagent-execution on this branch. The user asked to execute on 2026-10-02.
