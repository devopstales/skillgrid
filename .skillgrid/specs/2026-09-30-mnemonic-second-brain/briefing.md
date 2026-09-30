# Briefing — Mnemonic Second Brain (the "gets smarter every conversation" loop)

**Topic:** 2026-09-30-mnemonic-second-brain
**Date:** 2026-09-30
**Classification:** standard (T2)
**Build shape:** Smallest usable whole
**Findings:** `09-claude-os-deep-dive.md` + `08-second-brain-roadmap.md` (roadmap items B, C, D, S4)

## Why

ADR-0012 locked the *engine* (SQLite as the second-brain store). The deep-dive of the closest same-shape product (`brobertsaz/claude-os`) confirmed what makes it *feel* like a second brain rather than a search API is **three capabilities on top of the existing store**, plus a response-shape convention. None of them require new infrastructure — they are thin layers over `mem_save`, the existing FTS5+vec0+graph retrieval, and the existing observation lifecycle (governance/archive).

The loop: **capture what happened (B) → answer questions over it with citations (C) → keep it clean so it stays trustworthy (D)**, all returned in a shape the agent can reason over (S4). That is the "gets smarter every conversation" promise.

## What's already built (do NOT re-plan)

| Capability | Where | Status |
|---|---|---|
| FTS5 keyword search (porter/trigram) | `store/migrations/001_schema.sql`, `fts_drift: 0` | Built |
| Durable vec0 vector search (`float[768]`) | `store/migrations/042_vec0_tables.sql` (ADR-0009 option G) | Built |
| Real dependency graph + blast-radius tiers | code index, `code_impact` | Built |
| RRF fusion (`ReciprocalRankFusion`, k=60) | `memory/embedding.go:74-106` | Built |
| `BlendedSearch` (FTS + vector RRF) | `memory/search_embed.go:117-175` | Built (vector leg unused in MCP path) |
| Intent detection (`ClassifyIntent`, 4 intents) | `memory/retrieval.go:70-80` | Built (not wired into search) |
| Dedup-by-hash on save | `mem_save` | Built |
| Soft-archive / governance (owner, status, versions, visibility) | `mem_governance`, `mem_delete(hard)` | Built |
| `retrieval_usage` + `BumpRetrievalUsage` | `memory/governance.go:237-245` | Built |
| Anti-hallucination extraction prompt (014 step 05 `CapturePassive`/`ExtractWithLLM`) | monitoring spec | In-flight |
| MCP response shape (errors-as-values) | `mcp/tools_memory.go` | Partial |

**Key point:** F (durable sqlite-vec) — the one item previously thought to be missing — is **already done** (migration 042). This change is purely additive capability layers.

## Capabilities to add

### C1: Natural-language capture (roadmap B) — "remember this"

**Problem:** Capture today is 100% explicit `mem_save` with an explicit `type`. When the user says "remember this" or "we decided to use X" mid-conversation, the agent must recognize the intent, infer the `type`, and call `mem_save` with well-formed What/Why/Where/Learned content. Today that is left to the model with no contract — capture is skipped or mis-typed.

**Solution:** Two complementary surfaces, both thin (no regex engine, no new service):

1. **A skill** (the primary trigger). `.agents/skills/mnemonic-second-brain/SKILL.md` whose frontmatter `description` lists trigger phrases — "remember this", "don't forget that", "we decided to…", "note for next time", "important:". Body is a capture contract: extract → infer `type` from the existing taxonomy (decision/bugfix/pattern/discovery/config/correction/learning/…) → assemble `mem_save` content with What/Why/Where(/Learned) → call `mem_save` with a stable `topic_key` so rephrasings upsert. "No questions. No ceremony. Just save it."
2. **A `mem_save` inference helper** (the programmatic path). `mem_save` already accepts `type` + `topic_key`. Add an optional `infer=true` flag (default `false`): when set and `type` is empty, run the existing `ClassifyIntent`/type-heuristic to fill `type`, and `mem_suggest_topic_key` to fill `topic_key`. The agent still owns the content; the tool only fills the metadata it can't infer. This makes the skill's "auto-detect" step deterministic and testable.

**Key detail:** The agent is the classifier (claude-os proved the skill-trigger approach works with zero infra). The `infer` flag is the *deterministic floor* so behavior is testable without a live model.

**Depends on:** `mem_save` (built), `ClassifyIntent` (built), `mem_suggest_topic_key` (built).

### C2: `mem_ask` synthesis (roadmap C) — cited answers over the store

**Problem:** Retrieval returns a ranked list of chunks. The agent must manually synthesize an answer over many calls. There is no single "ask the brain a question and get a cited answer" tool. This is the difference between a search API and a second brain.

**Solution:** A new MCP tool `mem_ask` that **gathers then synthesizes**:

1. **Gather** — run the existing `BlendedSearch` (FTS + vec0 RRF) plus graph expansion over the top hits (resolved edges only, per code-step-01 drop policy) to pull the relationship chains (L4). Project-scoped by default; `all_projects` flag to span stores (cross-project pattern surfacing, roadmap I).
2. **Synthesize** — two modes:
   - **`mode="cited"` (default, no-LLM floor):** return the gathered observations as a structured, deduped, citation-bearing block — each claim annotated with its source observation `id`, `title`, and `type`. Deterministic, token-bounded (`max_tokens`), works with no embedder. This is the *guaranteed* answer.
   - **`mode="llm"` (opt-in):** when an LLM is reachable, produce a prose answer with inline `[obs:<id>]` citations and a `sources` array. Fails open to `mode="cited"` on LLM error/timeout (3s), never breaks.
3. **Response shape:**
   ```json
   {
     "answer": "...(prose when llm mode; omitted when cited mode)...",
     "citations": [{"id": 42, "title": "...", "type": "decision", "snippet": "..."}],
     "matched_via": "hybrid",
     "degraded": false,
     "sources": ["obs:42", "obs:17"]
   }
   ```

**Key detail:** `mem_ask` REUSES the in-memory vec0/FTS retrieval already built (the memory-improvements spec's C1 wires `BlendedSearch` into the MCP path). If that lands first, `mem_ask` consumes it directly; if not, `mem_ask` calls `BlendedSearch` itself. No new retrieval engine.

**Depends on:** `BlendedSearch` (built), code graph expansion (built), LLM seam for `mode="llm"` (optional, fail-open).

### C3: Knowledge lifecycle (roadmap D) — consolidate + archive + health

**Problem:** The store grows. Near-duplicates accumulate, stale observations linger, and nothing tells you the store's health. claude-os's "brain feel" is largely this lifecycle layer. We have dedup-by-hash and soft-archive (governance), but not **consolidate** (merge near-dupes with provenance), **archive** (retire stale with reason), or a **health report** (recommendations).

**Solution:** A `mem_lifecycle` MCP tool with an `action` enum (multiplex one tool, claude-os pattern) + a new audit table:

1. **`action="health"`** — read-only report: counts by type/status, embedding coverage (obs with vec0 row), age distribution (7/30/90d/older), duplicate density (pairwise cosine over top-K, union-find clustering), and rule-based `recommendations` with priority (dedup high if >5 similar pairs; cleanup low if archived >30%; stale medium if >50% older than 90d and total >10; coverage medium if any obs lacks a vector). Caches to a 24h file; **never blocks**.
2. **`action="dedup"`** — `subaction="scan"` (return clusters + density, `dry_run`) / `subaction="merge"` (soft-archive the near-dupes keeping the canonical, recording `consolidated_from` provenance). Union-find over normalized cosine (reuses the vec0 rows; degrade to hash-dedup when no embedder).
3. **`action="consolidate"`** — `obs_ids[]` + `new_title`: merge via the existing LLM extraction/summarization seam (fail-open: without LLM, produce a deterministic concatenated-with-provenance note); creates one observation with `consolidated_from: [ids]` in metadata, soft-archives the sources.
4. **`action="archive"`** — `subaction="archive"|"restore"|"list"|"stale"`, `obs_ids[]`, `reason`, `stale_days` (default 90). Reuses existing governance soft-archive; adds `archive_reason` + `archived_at`.
5. **Audit log** — new migration: `lifecycle_log(id, project, action, subaction, status, input_ids JSON, output_ids JSON, details JSON, created_at, completed_at)`. Two-phase: `pending` → work → `completed|failed` (auto-set `completed_at` on terminal).
6. **Inline staleness warnings** — `mem_search`/`mem_ask` results may carry a `_health_warnings` array (HIGH/CRIT only, 24h file cache, computed lazily, try/except → `[]` so it never breaks search).

**Key detail:** Everything is **additive + soft** (archive via metadata/status flag, reversible via `restore`), mirrors claude-os's verified patterns, and reuses the existing LLM seam and vec0 rows. No hard deletes.

**Depends on:** new migration (additive, idempotent), governance soft-archive (built), LLM seam (optional, fail-open), vec0 rows (built).

### C4: MCP response-shape convention (roadmap S4) — cheap, applies everywhere

**Problem:** Some MCP paths return raw errors; some lack the actionable next-step text; meta-enrichment is inconsistent. claude-os's contract (errors-as-values, `_`-prefixed meta, self-healing error text) is cheap and improves every tool.

**Solution:** Adopt as a convention across the touched tools (`mem_save`, `mem_search`, `mem_ask`, `mem_lifecycle`):
- **Errors are values, never thrown to transport:** `{"error": "…", <empty-payload>}`.
- **Actionable error text with the next step** (e.g. "no embedder active — set MNEMONIC_EMBED=1 to enable the vector leg").
- **`_`-prefixed meta fields** (`_timing`, `_health_warnings`, `_source_project`) the LLM may ignore; never collide with payload fields.
- **`success` bool + `error` string for mutations; `[]` for failed reads.**

**Depends on:** nothing (pure convention + refactor of the touched handlers).

## One-way-door decisions

| # | Decision | Risk | Mitigation |
|---|---|---|---|
| 1 | `mem_ask` is a NEW MCP tool with a `mode` enum; `cited` mode is the no-LLM floor | New surface; agents may reach for it before `mem_search` | `cited` mode is deterministic + token-bounded; `llm` mode fails open to `cited`; ADR records the tool + response contract |
| 2 | `mem_save` gains `infer` (default `false`) — changes metadata when set | Existing `mem_save` callers unaffected (default off); inferred `type` may occasionally mis-type | `infer` only fills EMPTY `type`/`topic_key`; agent-provided values win; deterministic heuristic is unit-tested |
| 3 | New migration `lifecycle_log` (additive table) | Schema change | `CREATE TABLE IF NOT EXISTS`; additive only; no data migration |
| 4 | `_health_warnings` added to `mem_search`/`mem_ask` responses | Additive field; may reorder nothing (warnings are separate from results) | 24h cache; try/except → `[]`; never affects result ordering |

## Trust boundaries

- **MCP tool contract:** `mem_ask` is a new tool (no existing consumers). `mem_save` `infer` is opt-in (default off). `_health_warnings` is additive. Existing strict-schema consumers of `mem_search` are unaffected (new field only).
- **No new trust boundary:** C1–C4 operate within the existing per-project SQLite store + the existing LLM seam. No new external service, no new network boundary. The `llm` mode of `mem_ask`/`consolidate` calls the same LLM already used by the monitoring spec (fail-open, 3s timeout).
- **Audit log:** `lifecycle_log` stores IDs + a JSON `details` blob (action metadata, e.g. density, counts). No PII; wrap any sensitive `reason` in `<private>…</private>` per the mnemonic privacy rule.
- **Soft-only lifecycle:** `archive`/`dedup`/`consolidate` never hard-delete by default; `restore` reverses. Hard delete stays behind `mem_delete(hard=true)`.

## Success criteria

1. A `remember this`-style phrase (via the skill) results in a `mem_save` call with a correctly-inferred `type` and a stable `topic_key` (rephrasings upsert, no duplicate).
2. `mem_save` with `infer=true` and empty `type`/`topic_key` fills them deterministically (unit-tested against a fixture set); agent-provided values are never overwritten.
3. `mem_ask` `mode="cited"` returns a token-bounded, citation-bearing answer over the gathered observations with **no LLM** and **no embedder** (degraded=false for FTS, `degraded=true` flag set when vec0 leg is skipped); `mode="llm"` produces a prose answer with `[obs:<id>]` citations and fails open to `cited` on LLM error.
4. `mem_ask` is project-scoped by default and spans stores with `all_projects=true`.
5. `mem_lifecycle action="health"` returns counts, embedding coverage, age distribution, duplicate density, and rule-based recommendations; it is cached 24h and never throws.
6. `mem_lifecycle action="dedup" subaction="scan"` returns union-find clusters + density (degrades to hash-dedup with no embedder); `subaction="merge"` soft-archives near-dupes keeping the canonical with `consolidated_from` provenance.
7. `mem_lifecycle action="archive"` archives/restores/lists/finds-stale with `reason` + `archived_at`; restore reverses archive.
8. Every mutating lifecycle op writes a `lifecycle_log` row (pending → completed/failed with `completed_at`).
9. `_health_warnings` appear on `mem_search`/`mem_ask` results only when HIGH/CRIT, are cached 24h, and are `[]` on any computation error (search never breaks).
10. Touched tools return errors as values with actionable next-step text and `_`-prefixed meta (no exception to transport).
11. All existing tests pass (no regression). New tests cover: infer heuristic, `mem_ask` cited/llm + degradation + scoping, lifecycle health/dedup/archive + audit log, `_health_warnings` non-breaking.

## ADR needed

ADR-0016: "Mnemonic second-brain capability layer: `mem_ask` synthesis tool (cited floor + fail-open LLM), `mem_save.infer` metadata inference, `mem_lifecycle` (health/dedup/consolidate/archive + `lifecycle_log` audit), and the MCP response-shape convention (errors-as-values, `_`-prefixed meta, actionable error text)."

## Out of scope (listed but not in this change)

- **A. Tool-call capture hook (default-on, observe-mode)** — already designed in the monitoring spec (07); separate change, depends on hook wiring.
- **E. Session-inject** — its own spec (2026-09-24-mnemonic-session-inject); depends on A.
- **G. Human browse view** — `serve` dashboard work; separate change.
- **H. Passive learning (session-end auto-extract)** — depends on the monitoring spec's LLM extraction + hook trigger; the anti-hallucination prompt (S3) is the reference for that change, not this one.
- **I. Cross-project pattern synthesis** — `mem_ask all_projects` (C2) provides the retrieval; the *synthesis* across projects is a follow-up.
- **RRF wiring into `mem_search` (memory-improvements C1)** — `mem_ask` reuses `BlendedSearch` directly; wiring it into `mem_search` ranking is that spec's job.
- **L5 always-on autonomous brain** — north star, not a change.
