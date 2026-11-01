# 002 — Canonical Schema

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture` (INV-1, INV-4, INV-6), `001-storage`
**The keystone.** Every function spec cites tables here rather than re-declaring them.
**Reference:** Engram `internal/store/store.go` (entities, `sync_id`, relations,
sync journal) + mnemonic v1 `internal/store/migrations` (split observations, bitemporal,
lifecycle log).

---

## 1. Principles

- **Dialect-neutral.** Every table below is defined once; `001-storage` renders it for
  SQLite and Postgres. Types shown are the *logical* types.
- **`sync_id` is the identity (INV-6).** Every syncable entity has a `sync_id`
  (ULID, format `obs-`/`rel-`/`ses-`/`prm-` + 16 hex). The local numeric `id` is
  ephemeral and never used cross-machine.
- **Provenance is a column, not an afterthought (INV-1).** Origin is stored on the row.
- **Soft delete by default.** `deleted_at` hides; hard delete is explicit.
- **Additive migrations only.** New tables/columns; never drop in a minor version.

### Type legend (logical → SQLite / Postgres)

| Logical | SQLite | Postgres |
|---|---|---|
| `text` | `TEXT` | `TEXT` |
| `id` | `INTEGER PK AUTOINCREMENT` | `BIGSERIAL PK` |
| `sync_id` | `TEXT` (UNIQUE where syncable) | `TEXT` (UNIQUE where syncable) |
| `ts` | `TEXT` (RFC3339) | `TIMESTAMPTZ` |
| `bool` | `INTEGER` 0/1 | `BOOLEAN` |
| `json` | `TEXT` (JSON) | `JSONB` |
| `vec` | `BLOB` | `vector(…)` (pgvector) |

## 2. Memory domain (the heart of this change)

### 2.1 `sessions`

Groups work from one agent session.

| Column | Type | Notes |
|---|---|---|
| `id` | id | local |
| `sync_id` | sync_id | cross-machine identity |
| `project` | text NOT NULL | |
| `ownership_mode` | text | `shared` (default) \| `project_owned` |
| `directory` | text NOT NULL | canonical worktree path |
| `title` | text | dashboard/session-list display name |
| `started_at` | ts NOT NULL | |
| `ended_at` | ts | set on finalize; terminal once set |
| `summary` | text | free-text (legacy); the structured summary is a `session_summary` observation |
| `runtime_lease_expires_at` | ts | local-only liveness (30 min), never synced |

**Ended sessions are terminal** — never reopened. `resume: true` selects/creates an
`<id>:resume:N` continuation inside one write transaction (Engram behavior).

**`session_finalizations`** (local-only idempotent retry journal) — written by
`mem_session_finalize` (`01-memory/002-sessions` §6); never synced or exported:

| Column | Type | Notes |
|---|---|---|
| `session_id` | text NOT NULL | FK → sessions.id, `ON DELETE CASCADE`, **PK** |
| `summary_id` | id | the `session_summary` observation written at finalize |
| `summary_sync_id` | sync_id | the summary's `sync_id` |
| `created_at` | ts NOT NULL | |

A retry of finalize finds this row and returns `already_finalized: true` instead of
double-closing.

### 2.2 `observations` — the memory unit

Every memory is an observation (Engram's flat model; we keep it flat — no layers).

| Column | Type | Notes |
|---|---|---|
| `id` | id | local |
| `sync_id` | sync_id UNIQUE | `obs-…` |
| `session_id` | text NOT NULL | FK → sessions.id |
| `type` | text NOT NULL | vocabulary §2.2.1 |
| `title` | text NOT NULL | |
| `content` | text NOT NULL | conventionally `**What**/**Why**/**Where**/**Learned**` |
| `tool_name` | text | the agent tool that wrote it |
| `project` | text | **nullable — blank inherits session project**; explicit overrides |
| `scope` | text NOT NULL DEFAULT `'project'` | `project` \| `personal` \| `global` |
| `topic_key` | text | nullable; turns save into upsert (§3.2) |
| `normalized_hash` | text | dedupe hash |
| `revision_count` | int NOT NULL DEFAULT 0 | topic upserts increment |
| `duplicate_count` | int NOT NULL DEFAULT 0 | exact-dedupe hits |
| `last_seen_at` | ts | |
| `pinned` | bool | **local-only**, never synced; search boost + context ordering |
| `review_after` | ts | decay/review clock (type-dependent, §2.2.2) |
| `superseded_by_sync_id` | sync_id | consolidation provenance (nullable) |
| `expires_at` | ts | NULL in v2 — hard expiry deferred |
| `embedding` | vec | reserved → activated (`001-storage` §4) |
| `embedding_model` | text | |
| `embedding_created_at` | ts | |
| `importance` | real NOT NULL DEFAULT 0.5 | **v2 addition** — on-the-fly score, see §2.2.3 |
| `retrieval_usage` | int NOT NULL DEFAULT 0 | **v2 addition (amended 2026-11-01)** — the accumulating retrieval count that feeds the `0.6·usageNorm` term of the §2.2.3 formula (the formula's named source). Incremented by the retrieval trail writer (every `mem_search`/vector hit; `mem_lookup` does not bump — it is the wiki's precise fallback). Syncs with the row. |
| `provenance` | json NOT NULL | **INV-1** — `{session_id, writer, model, created_at}`; immutable |
| `created_at` | ts NOT NULL | |
| `updated_at` | ts NOT NULL | |
| `deleted_at` | ts | soft delete |

Derived virtual state: `state()` → `active` \| `needs_review` from `review_after` vs now.

**Indexes:** `idx_obs_dedupe (normalized_hash, project, scope, type, title)`;
`idx_obs_topic (project, scope, topic_key)`; FTS5 over `(title, content, type, project,
topic_key)` (weights per `001-storage` §3).

#### 2.2.1 `type` vocabulary

`decision`, `architecture`, `bugfix`, `pattern`, `config`, `discovery`, `learning`,
`preference`, `convention`, `session_summary`. (Engram's set + `convention` from v1.)
`policy`/unmapped types are tolerated but unscheduled for review.

#### 2.2.2 Review/decay schedule (per type)

Set at insert from a type → months map (Engram's table):
`config`/`preference` 3 mo · `discovery`/`pattern`/`decision` 6 mo ·
`architecture`/`bugfix`/`policy` 12 mo · `learning`/`session_summary`/unmapped: none.
Overdue → `state=needs_review`, **still searchable, never auto-hidden or deleted**
(hard expiry deferred). `mem_review mark_reviewed` restarts the interval.

#### 2.2.3 `importance` (v2 addition from mnemonic)

A **deterministic, on-the-fly** score (not a stored truth that drifts):
`importance = 0.6·usageNorm + 0.37·recency + 0.2·typeDurability`, clamped [0,1], where
`usageNorm = min(retrieval_usage, 50)/50`. Durable types (decision/architecture/
convention/config) score higher than transient ones. Used by the wiki's graduate-rule
(llm-wiki change, the graduate-rule spec) and by review prioritization — **not** by hard
expiry. Stored column is a *cache*; the formula is the source of truth. It is computed
at save (pipeline step 6d, `01-memory/001-observations` §2) and carried on the
`SaveEvent` (`004-transport-mcp` §5).

### 2.3 `observation_versions`

Immutable snapshots captured **before** a `topic_key` upsert or content-changing
`mem_update` overwrote content.

| Column | Type | Notes |
|---|---|---|
| `id` | id | |
| `observation_id` | id NOT NULL | FK → observations.id |
| `version` | int NOT NULL | |
| `title` | text | |
| `content` | text | |
| `created_at` | ts NOT NULL | |

`UNIQUE(observation_id, version)`. Read via `mem_get_observation(include_history)`,
paged 50/page.

### 2.4 `user_prompts`

Stored user prompts as first-class retrievable context (Engram).

| Column | Type | Notes |
|---|---|---|
| `id` | id | |
| `sync_id` | sync_id UNIQUE | `prm-…` |
| `session_id` | text NOT NULL | FK → sessions.id |
| `content` | text NOT NULL | redacted, truncated |
| `project` | text | blank inherits session project |
| `source_inbox_id` | text | optional replay identity |
| `created_at` | ts NOT NULL | |

`prompt_tombstones` (deleted prompts, local) + FTS over `(content, project)`.
`capture_prompt` (default true) on `mem_save` stores the current turn's prompt here.

### 2.5 `memory_relations` — the edge table

Directed source→target with a conflict/judgment vocabulary. **Keys are `sync_id`
(TEXT), not numeric ids** — cross-machine portable (INV-6).

| Column | Type | Notes |
|---|---|---|
| `id` | id | local |
| `sync_id` | sync_id UNIQUE | `rel-…` |
| `source_id` | sync_id NOT NULL | the source observation's `sync_id` |
| `target_id` | sync_id NOT NULL | the target observation's `sync_id` |
| `relation` | text NOT NULL DEFAULT `'pending'` | vocabulary §2.5.1 |
| `reason` | text | |
| `evidence` | text | |
| `confidence` | real | |
| `judgment_status` | text | `pending` \| `judged` \| `orphaned` \| `ignored` |
| `marked_by_actor` / `marked_by_kind` / `marked_by_model` | text | **INV-1** provenance |
| `session_id` | text | the session that recorded it |
| `superseded_at` | ts | |
| `superseded_by_relation_id` | id | FK → memory_relations.id |
| `created_at` / `updated_at` | ts NOT NULL | |

**Deliberately no UNIQUE on `(source_id, target_id)`** — multi-actor disagreement is
allowed (two agents may judge the same pair differently). No `project` column; project
is derived by JOIN on the endpoints.

#### 2.5.1 Relation vocabulary

`pending` (default, not a verdict) · `related` · `compatible` · `scoped` ·
`conflicts_with` · `supersedes` · `not_conflict`.

**Conflict-candidate detection (write path):** after a `mem_save`, a lexical FTS
pre-filter finds same-project/scope candidates, an opt-in title/topic **relevance gate**
(shared `topic_key`, or ≥2 significant shared title terms) narrows them, and the matches
become `pending` relations returned to the agent as `candidates[]` with
`judgment_required: true`. The agent's LLM then `mem_judge`s them (or `mem_compare`
records a judged relation directly). This is a *lexical heuristic*, not semantic
detection — the LLM is the semantic layer (INV-3, LLM-first).

## 3. Sync domain (full sync in memory — per the decision)

Engram's replication model, adopted. Local SQLite journals every mutation; the
sync/autosync layer ships and applies them.

### 3.1 `sync_mutations` — the local mutation journal

| Column | Type | Notes |
|---|---|---|
| `seq` | id | monotonic, per-store |
| `target_key` | text | `cloud` \| `cloud:inbox` \| `local` |
| `entity` | text | `session` \| `observation` \| `prompt` \| `relation` \| `project` |
| `entity_key` | text | the `sync_id` |
| `op` | text | `upsert` \| `delete` |
| `payload` | json | the canonical entity (project-canonicalized) |
| `source` | text | `local` \| `remote` |
| `project` | text | |
| `acked_at` | ts | |
| `disposition` | text | for conflict/rejection tracking |

### 3.2 `sync_state` — per-target lifecycle

`target_key`, `lifecycle` (`idle`/`pending`/`running`/`healthy`/`degraded`/`inbox`),
`last_enqueued_seq`/`last_acked_seq`/`last_pulled_seq`, backoff, lease, reason codes.

### 3.3 Supporting tables

| Table | Purpose |
|---|---|
| `sync_chunks` | imported chunk tracking (`target_key`, `chunk_id`, `imported_at`) |
| `sync_delete_tombstones` + `…_remote_floors` | delete floors preventing tombstone/late-upsert resurrection races |
| `sync_apply_deferred` | pulled mutations that couldn't apply (missing FK); 5-retry cap then dead |
| `sync_enrolled_projects` | which projects opted into cloud/remote sync |
| `observation_save_operations` | **replay-safe save idempotency ledger** — `operation_id`, `fingerprint` (`v1:` SHA-256 of length-prefixed normalized fields), `observation_id` (nullable, `ON DELETE SET NULL`) |

### 3.4 Conflict model

**Last-write-wins, no CRDT** (Engram's explicit stance). Ordering is handled by
`seq` + tombstone floors + deferred-apply. Distributed atomicity is *disclaimed*: a
receiver may defer a survivor upsert while its parent session is missing. This is a
known, accepted limit.

## 4. Non-memory tables (owned by other changes, listed for completeness)

- **code-index:** `files`, `chunks`, `chunks_fts`, `symbols`, `edges`, `dep_edges`,
  `index_freshness` → `02-code-index/001`, `002`.
- **llm-wiki:** `wiki_pages`, `wiki_links`, `wiki_entities`, `wiki_revisions` →
  `03-llm-wiki/003`.

These live in the *same store* (same project DB) but are specified by their own changes.
The memory change's `Store` interface does not expose them.

## 5. FTS rendering (per dialect)

Per `001-storage` §3: SQLite FTS5 trigram + triggers; Postgres `tsvector` + GIN +
triggers. External-content tables (`content='observations'`) so the FTS rowid maps to
the base row.

## 6. Migration story

Each store ships an ordered migration set for its dialect. `002` is the **target**;
migrations are the additive path. A fresh SQLite file applies the full set; a fresh
Postgres schema applies the Postgres render. **The set must be runnable on both
dialects from an empty DB** (regression-tested). v1 store files are *not* auto-migrated
into v2 (different schema philosophy) — `mnemonic migrate` exports v1 observations as
seed imports.

## 7. Invariants

INV-1 (provenance columns), INV-2 (dual-mode), INV-4 (local truth), INV-6 (sync_id).

## 8. Open items

None. The schema is complete for the memory function; code-index and wiki tables are
deferred to their changes (cross-referenced, not owned here).
