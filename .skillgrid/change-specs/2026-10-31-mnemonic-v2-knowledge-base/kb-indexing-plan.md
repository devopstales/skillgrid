# KB Indexing — Plan (the derived FTS/index layer of `004-index`)

**Status:** COMMITTED (spec zone — the source-citation doc backing the `004-index.md`
+ `005-query.md` amendments)
**Scope:** the *indexing* half of the `2026-10-31-mnemonic-v2-knowledge-base` change —
the derived FTS cache + the run-state / freshness / migration machinery in
`specs/02-knowledge-base/004-index.md` (and the migration-parity seam it shares with the
memory + code-index changes). This is the *engine* layer; the sibling
`kb-skill-plan.md` covered the *skill* (behavioral) layer.
**References reviewed (2026-11-01, indexing round):**
1. **lucasastorian/llmwiki** `shared/sqlite_migrations/` — a *working* SQLite FTS5
   index: `0001_baseline.sql` (documents/pages/chunks/refs + FTS5 + activity triggers),
   `0007_chunks_highlight_search` (the materialized-annotation chunk split),
   `0010`/`0011` (event-sourcing + the "index is derived state, rebuildable" stance),
   and `api/infra/db/local_migrations.py` (the stdlib migration runner:
   `PRAGMA user_version`, `BEGIN IMMEDIATE`, legacy-reconcile, trigger-aware split).
2. **equationalapplications/expo-llm-wiki (ScopeLab)** — a *serious* TS/SQLite
   retrieval core, with the four load-bearing specs: `2026-10-05-fts5-index-strategy`
   (IndexStrategy seam, FTS5-vs-MiniSearch, the memory-flat-at-any-size fix,
   the parity-gate + threshold lessons), `2026-09-28-incremental-search-index`
   (`syncEntries` — index only the touched rows, conditional vacuum, the stale-entity
   flag, the durable-ledger PR-2), `2026-05-07-pluggable-vector-retrieval`
   (the `VectorRanker` injection point + fallback policy), and the
   `2026-09-22-ingest-grounded-dedup-and-edge-index` design (grounded dedup).

**Sequencing:** the KB change folder currently has **only `specs/`** (no blueprint —
it is queued behind code-index, queued behind memory). This doc amends
`004-index.md` + the shared migration/FTS machinery. It does **not** slice the KB; it
prepares the index spec so the future KB blueprint's index task is unambiguous.

---

## 1. What `004-index.md` already has (validated, not changed)

The spec is sound and the two references **independently confirm** most of it:

- **External-content FTS** (`content='kb_pages'`, body tokenized not duplicated, rowid
  → base row) — *exactly* llmwiki's `chunks_fts` pattern AND the memory change's
  `001-storage` §3. Three implementations, one pattern.
- **File-hash staleness** (`body_hash` vs live file; write-event explicitly rejected in
  favor of file-hash) — llmwiki's `content_hash`/`stale_since` + the "index is derived
  state, deletable and rebuildable" comment confirm this is the right call.
- **Incremental `kb_index` + `kb_reindex`** (rebuild-from-scratch vs body_hash-diff) —
  expo-llm-wiki's `syncEntries` (incremental) + `sync` (full rebuild) is the same
  two-mode split.
- **`kb_run_state`** (the resumable page-job checkpoint, authoritative; the file
  projection is `index/last-update.json`) — openwiki's `openwiki_begin`/
  `submit_plan`/`next_page`/`submit_page`/`finish` lifecycle is the same
  resumable-queue idea.

So the core design is *validated* by two working systems. The gaps below are the
**hard-won details** those systems needed that our spec hasn't spelled out.

## 2. The gaps (all additive to the index spec + shared machinery; none touch the
page model or the one-way doors)

### Gap 1 — the durable FTS pending-ledger (crash-recovery for the index)

**Source:** expo-llm-wiki `2026-09-28-incremental-search-index` §PR-2 — the
**trigger-fed per-entry ledger** (`fts_pending`) + `fts_map` (id→rowid) + all-SQL
chunked drain.

The problem they hit: the in-memory "what's indexed" set is **lost on restart**, and
**nothing healed a crash between the data commit and the FTS write.** If the process
dies after writing `kb_pages` but before writing `kb_pages_fts`, the index silently
drifts from the pages until a full `kb_reindex`.

**Their fix:** a **persistent `fts_pending` ledger** — every FTS write is logged; a
drain (`kb_index` / on-open) replays unapplied entries in SQL. `fts_map` maps
`id → rowid` (because FTS5's `id` column is *not unique* and `entries.rowid` is not
stable across `VACUUM`).

**Fold into our KB:** our `kb_index` is incremental-by-`body_hash`, which is a *read*
of truth (correct), but it has **no crash-recovery for the FTS write itself**. Add to
`004-index`:
- a **`kb_fts_pending`** (or fold into `kb_run_state`) ledger: a page whose `kb_pages`
  row is committed but whose `kb_pages_fts` write has not been confirmed is *pending*;
  `kb_index` (and on-open recovery) drains it. This makes "a crash between the data
  commit and the FTS write" self-healing, not a silent drift.
- the **id→rowid map** discipline: FTS5's `id`/external-content `rowid` is not a stable
  identity across `VACUUM`/rewrite — the index keys off the `kb_pages` row (the
  source of truth), and the FTS row is addressed by an explicit map, not by assuming
  rowid stability.

> **Boundary note:** this is the *strongest* fold — it's a real durability gap our spec
> doesn't currently close. If the KB blueprint wants to keep it simple, the minimum is
> "on open, if `kb_run_state` shows an incomplete `kb_index`, drain it"; the full
> `fts_pending` ledger is the robust version.

### Gap 2 — the trigger-vs-explicit-indexing rule (soft-delete correctness)

**Source:** expo-llm-wiki FTS5 strategy §"Explicit indexing, no triggers" —
*soft-delete + `valid_from`/`valid_to` lifecycle make trigger-based sync **incorrect**:
a soft-delete row must leave the index, and triggers would also fire on every
ALTER/UPDATE that has nothing to do with text.*

llmwiki does the *opposite* — it **does** use triggers (`chunks_fts_insert/delete/update`)
but scopes them to `content` changes and adds a `has_highlight` partial index. Both
are defensible; the *decision* must be explicit and the soft-delete case must be handled.

**Fold into our KB:** our spec says "same triggers as the memory change" but doesn't
state the rule. Make it explicit in `004-index`:
- **the index is rebuilt from a read of truth, not fire-and-forget triggers** — the
  preferred model (matches expo-llm-wiki + our file-hash stance). If triggers are used
  (the memory change's pattern), they fire **only on the content columns**
  (`title`/`body`/`type`/`rel_path`), **never on metadata/lifecycle columns**, and a
  soft-delete / page-removal **must** remove the FTS row (the `chunks_fts_delete`
  trigger pattern).
- **the "stale window is short by construction"** already in our spec (`kb_generate`
  calls `kb_index` as its final step) is the *write-path* guarantee; Gap 1's ledger is
  the *crash-path* guarantee. Together they close the window.

### Gap 3 — the materialized-annotation chunk split (source vs derived text)

**Source:** llmwiki `0007_chunks_highlight_search` — the **`content` / `source_content`
/ `annotations_text`** three-column chunk:
- `source_content` — the **immutable** raw chunk text (what the chunker produced);
- `annotations_text` — the **materialized** footnote/highlight/comment block, rebuilt
  on annotation CRUD;
- `content` = `source_content` + `annotations_text`, the **materialized searchable form**
  written atomically; `has_highlight` is the partial-index target for "search only
  annotated chunks."

The key insight: **the searchable text is a *derived materialization* of an immutable
source + a mutable annotation layer**, and the split lets you (a) re-materialize
without re-chunking, (b) partial-index the small "annotated" subset, (c) keep the
10k cap on `source_content` only (the chunker enforces it; `content` may grow).

**Fold into our KB:** our `kb_pages` stores `body` directly. If the KB ever adds
*annotations* (highlights/comments/pin-notes on a page — the `## Contradiction` marker,
the `verified:` event, a future comment layer), the searchable text becomes a
materialization. Add to `004-index` as a **forward contract** (not a v2 table, but the
shape is fixed now so it's additive later):
- a page's **searchable text** = `body` (the source) **materialized** with any
  annotation layer; the *source* (`body`) stays immutable-in-the-index (re-chunked only
  when the file changes), the *annotations* are a separate materialized column; a
  `has_annotation` partial-index target is available for "search only annotated pages."
- the **10k / size cap applies to the source, not the materialized form** (annotations
  make the searchable form larger; don't drop the chunk-length CHECK off the source).

> **Why fold it now (it's deferred):** it's a *shape* decision, not a table. Fixing it
> now means the KB can add annotations in v2.1 without a migration that re-chunks the
> whole vault. The default (no annotation layer yet) is unchanged.

### Gap 4 — the migration discipline: `PRAGMA user_version` + `BEGIN IMMEDIATE` + legacy-reconcile

**Source:** llmwiki `api/infra/db/local_migrations.py` — a **stdlib-only** migration
runner (so MCP/CLI can load it by file path) with four properties our shared migration
machinery should match:
1. **`PRAGMA user_version`** as the migration cursor (not a `schema_migrations` table) —
   zero extra tables, survives in the DB header.
2. **`BEGIN IMMEDIATE`** per migration — racing api/MCP/CLI processes **serialize** on
   the write lock instead of corrupting a half-applied migration.
3. **Legacy-reconcile** — for a pre-versioning DB (`version == 0` but the tables exist),
   the runner **diffs the live schema against the baseline in `:memory:`** and
   `ALTER TABLE … ADD COLUMN` the *missing columns* (dropping triggers first so
   backfills don't fire), then applies the baseline. This is how you adopt an
   already-existing DB without dropping data.
4. **Trigger-aware statement split** (`sqlite3.complete_statement`) — a naive
   `split(';')` breaks on trigger bodies.

**Fold into the shared machinery (memory + code-index + KB all ride this):** our memory
change's migration layer is the substrate. Add to the shared migration contract
(this is the **migration-parity regression** T-CI-3 already names as load-bearing):
- **`PRAGMA user_version`** as the cursor (confirm the memory change uses it; if it
  uses a `schema_migrations` table, note the choice — both work, but user_version is
  the lighter standard).
- **`BEGIN IMMEDIATE`** per migration (the concurrency-safety guard — matters when the
  MCP server + CLI + watch loop all open the same project DB).
- **legacy-reconcile on adopt** (the `version==0` + tables-exist path) — this is
  *exactly* the v1 bug-class the code-index findings named: v1's `001_schema.sql`
  defined `qualified_name`/`uid` but there was **no `ALTER TABLE symbols ADD COLUMN
  uid` migration**, so a pre-existing DB silently lacked the column and every graph
  read failed `no such column`. The legacy-reconcile runner is the *systemic fix*: it
  diffs live-vs-baseline and adds what's missing, so a drifted pre-existing DB is
  healed, not broken.
- **the trigger-aware split** (don't `split(';')` a migration that contains triggers).

> **This is the fold with the highest cross-change leverage** — it protects the memory,
> code-index, *and* KB migrations, and it is the direct systemic fix for the precise
> v1 bug the code-index change is built to avoid (T-CI-3).

### Gap 5 — the activity/event feed as an *append-only derived projection*

**Source:** llmwiki `0001_baseline` `knowledge_base_events` + `0010` — a **typed
event feed** (`page.created` / `page.updated` / `source.added` / …), written by
**triggers** off the data tables, with a **unique `event_key`** (idempotent — a
re-firing trigger `INSERT OR IGNORE`s), and a feed index `(kb_id, id DESC)`. `0011`
optimizes it: "activity events only depend on these columns; skip the trigger for
highlight/status/version writes" (the `AFTER UPDATE OF content, filename, title, …`
scoping).

**Fold into our KB:** our KB has `kb_run_state` (the *run* checkpoint) but **no
activity feed** — the "what changed and when" history that `kb_status` (the skill
plan's first-verb) and the `log.md` would want. Add to `004-index` (or `001-structure`)
as an **optional derived projection**:
- a **`kb_events`** append-only feed (`event_type` typed, `subject_kind` page/source,
  `event_key` UNIQUE for idempotency, `occurred_at`), written by triggers scoped to
  *content* columns only (the `AFTER UPDATE OF …` discipline).
- it is **derived, rebuildable, and optional** (drop it and re-replay from the data
  tables + file history); it powers `kb_status`'s "activity by kind last 7 days" and
  the `log.md` projection — *not* the source of truth (the pages are).

> **Boundary:** the skill plan's `kb_status` first-verb reads this feed. Keeping it
> derived (not authoritative) matches the INV-1 provenance-immutable stance and the
> "index is derived" invariant. If the KB wants the feed to be the *authoritative*
> event log (openwiki's `events.jsonl` stance), that's a bigger decision — the default
> here is *derived projection*, which is cheaper and matches llmwiki.

### Gap 6 — the vector ranker injection point (the semantic leg's seam)

**Source:** expo-llm-wiki `2026-05-07-pluggable-vector-retrieval` — the **`VectorRanker`**
interface: `rankBySimilarity({entityId, queryVec, candidateIds?, limit})` + optional
`onEmbeddingPersisted`. Core **keeps** embedding-dimension validation, the
`preFilterLimit` semantics (the keyword leg narrows *which ids participate* before
semantic ranking — the ranker **must** respect it), the **`hybridWeight` blend**
(WikiMemory is the *sole* place that computes the final blended score), and the
two-phase `SELECT` (rank top-ids, then `SELECT * WHERE id IN (…)`). The **default**
with no ranker is the portable blob + JS-cosine path (parity-tested). Fallback policy
is explicit: `js-cosine` | `keyword` | `empty` | `throw`.

**Fold into our KB / memory:** this is the **semantic leg's seam** — the same seam the
memory change's `intelligence/embedder` + the code-index's semantic leg (`006-search`)
already have. The KB's `kb_query` (wiki-first retrieval) semantic leg should adopt the
**same injection-point contract**:
- a **`VectorRanker`** (or the memory change's `Embedder`/ranker seam) that the KB's
  semantic leg *injects*; the **default** is the portable hash-64 / cosine path
  (matching the code-index's hash-64 floor, D-CI8).
- the **`preFilterLimit` / candidate-scoping** rule: the keyword (FTS) leg narrows the
  candidate set, the semantic leg ranks *within* it — the two legs compose, they don't
  each scan the whole vault.
- the **`hybridWeight` blend lives in one place** (the query service, not the ranker) —
  matches the code-index's RRF-K=60 fusion being in the `search` package, not the legs.
- the **fallback policy is explicit** (`js-cosine` | `keyword` | `empty` | `throw`),
  not a silent fallback — matches INV-3 (LLM-first, explicit degradation).

> **This is mostly a *confirm-and-name* fold** — the seam already exists in the memory
> change; the KB spec should *reference* it explicitly and adopt the candidate-scoping
> + single-blend-place + explicit-fallback rules so the three functions' semantic legs
> are consistent.

## 3. What does NOT change (resisted pulls)

- **expo-llm-wiki's GraphRAG core / polymorphic edge triples / `schema.org` package
  spec** — that's a *document-KG* ontology (entities/claims/edges over prose). The KB
  already has `kb_claims` + the code-index's `edges`; a full schema.org-tripled document
  graph is a v2.1+ enrichment, not the v2 index. **Not folded.**
- **llmwiki's Supabase/RLS layer** (multi-tenant `auth.uid()` policies) — our KB is
  **single-project** (the KB overview says so explicitly). RLS is a multi-tenant
  deployment concern, not the index design. **Not folded.**
- **expo-llm-wiki's MiniSearch-as-default** — they *default to MiniSearch* (in-RAM) and
  flip to FTS5 via `auto`. We **default to FTS5** (external-content, disk-backed,
  memory-flat) because we're Go/SQLite (the `modernc.org/sqlite` + vec), not a JS
  runtime where FTS5 availability varies by adapter. The **parity-gate + threshold
  lessons** (recall@10 is a *regression net*, not a relevance score; the bar is
  relaxed to observed baseline, not aspirational) are folded; the default choice is
  not.
- **expo-llm-wiki's `onEmbeddingPersisted` / sqlite-vec external index** — the code-index
  already owns the multi-store vector architecture (D-CI8: BLOB + vec0 + HNSW +
  brute-force). The KB *reuses* it; it doesn't fork it. **Confirmed, not re-derived.**

## 4. The spec amendments (to `004-index.md` + `005-query.md`) — APPLIED

| Where | Amendment | Gap |
|---|---|---|
| `004-index` §3.1 | Add the **`kb_fts_pending`** crash-recovery ledger (+ id→rowid map discipline; FTS rowid is not stable across VACUUM). | 1 |
| `004-index` §3.1 | Make **trigger-vs-explicit** explicit: prefer rebuild-from-read-of-truth; if triggers, scope to content columns only + a soft-delete must remove the FTS row. State the write-path (short-by-construction) + crash-path (ledger) guarantees together. | 2 |
| `004-index` §6 | Add the **source/annotation materialization** shape (searchable text = source + annotation layer; size cap on source only; `has_annotation` partial-index target). No v2 table, shape fixed for v2.1. | 3 |
| `004-index` §2.1 (shared migration; T-CI-3) | **`PRAGMA user_version`** cursor + **`BEGIN IMMEDIATE`** per migration + **legacy-reconcile on adopt** (diff live-vs-baseline in `:memory:`, `ADD COLUMN` missing, drop triggers for backfills) + **trigger-aware split**. The systemic fix for the v1 `no such column: uid` bug-class. | 4 |
| `004-index` §3 | Add the **`kb_events`** append-only derived activity feed (typed, `event_key` UNIQUE idempotent, triggers scoped to content columns, rebuildable, optional). Powers `kb_status` + `log.md`. | 5 |
| `005-query` §2.1 | Name the **semantic-leg injection point** (the memory change's `Embedder`/`VectorRanker` seam) + adopt the **candidate-scoping** (FTS narrows, semantic ranks within), **single-blend-place** (the query service owns the RRF/hybrid blend), and **explicit-fallback** (`keyword`\|`empty`\|`throw`) rules. | 6 |
| `004-index` §5.1 | Add the **retrieval-parity regression net**: a fixture corpus + `recall@K` + `meanRankDist` gates, **threshold = observed baseline (regression net), not aspirational** (the expo-llm-wiki lesson: an aspirational 0.85 recall@10 vs a 0.508 observed is a false gate). This is the index half of the migration-parity regression T-CI-3. | 1, 6 |

## 5. The shared-machinery note (cross-change, highest leverage)

Gap 4 (the migration discipline) is the one fold that **belongs to the memory
change's substrate**, not the KB alone — because the memory, code-index, and KB all
ride the same ordered, additive migration set on the same project DB. The plan:

- The **memory change** (the substrate) adopts `PRAGMA user_version` +
  `BEGIN IMMEDIATE` + legacy-reconcile + trigger-aware split in its migration runner.
- The **code-index change** (T-CI-3) already names the migration-parity regression as
  load-bearing (the v1 `no such column: uid` root cause) — the legacy-reconcile runner
  is the *systemic* version of that regression test (it *heals* a drifted pre-existing
  DB, not just detects the mismatch).
- The **KB change** rides it; the `kb_*` tables' migrations get the same guarantees.

This is recorded in the KB `004-index` §2.1 as a **dependency on the memory change's
migration contract** (a forward-ref, like the code-index's forward-refs in `000`
§5), so the KB slice can assert "the memory migration runner has the
user_version/BEGIN-IMMEDIATE/legacy-reconcile discipline" as a T0 precondition.

## 6. The verification questions (defaults recorded, not blocking)

1. **`PRAGMA user_version` vs a `schema_migrations` table** — does the memory change
   use user_version? *(default: user_version — the lighter standard; confirm at the
   KB slice.)*
2. **`kb_fts_pending` full ledger vs minimum "drain on open"** — robust version
   (per-entry ledger) or minimum (on open, if `kb_run_state` shows an incomplete
   `kb_index`, drain it)? *(default: minimum for v2, full ledger if the KB's
   write-path frequency justifies it — the KB re-indexes on every `kb_generate`, so
   the window is short; minimum is likely sufficient.)*
3. **`kb_events` authoritative vs derived** — the feed is a *derived projection*
   (default, matches "index is derived") or the *authoritative* event log (openwiki's
   `events.jsonl` stance)? *(default: derived projection.)*
4. **annotation layer in v2 or v2.1** — is the source/annotation split a v2 table or a
   v2.1 forward contract? *(default: v2.1 forward contract — no v2 table, shape fixed.)*
5. **semantic leg: KB owns a ranker or reuses memory's `Embedder`** — does the KB
   define its own `VectorRanker` or reference the memory change's seam? *(default:
   reference the memory change's seam — one semantic-leg contract across the three
   functions, D-KB's "share the machinery, not the interface" applied to retrieval.)*

## 7. The decision record (what "plan indexing properly" means here)

The plan is **complete** when:
- [x] This plan doc is committed to the KB change folder (spec zone).
- [x] `004-index.md` + `005-query.md` carry the amendments in §4 (the
  `kb_fts_pending` ledger, the trigger-vs-explicit rule, the materialization shape, the
  `kb_events` feed, the semantic-leg seam, the parity regression net).
- [x] The **shared migration discipline** (Gap 4) is recorded as a *dependency on the
  memory change's migration contract* (a forward-ref + a T0 precondition for the
  KB slice).
- [x] The 5 verification questions in §6 have defaults (recorded, not blocking).
- [ ] The future KB blueprint's **index task** references this plan, so the index
  mechanics (the durability + migration + parity details) are unambiguous when
  sliced.

## 8. Out of scope (explicitly)

- The **page model / generate / validate / lint / skill** (separate specs; the skill
  is `kb-skill-plan.md`).
- **GraphRAG / schema.org-tripled document ontology** (v2.1+ enrichment).
- **Supabase/RLS multi-tenancy** (single-project by design).
- **The code-index's own index** (a different function; the KB *reuses* the shared
  machinery but owns the `kb_*` tables).
- **The memory change's migration runner implementation** (this plan *specifies* the
  discipline it should have; the memory change implements it).
