# 004 — Index (the derived FTS cache in the shared project DB)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (§2 the layout — where the index lives), the memory
change's `001-storage` (dual-mode dialect abstraction), `002-schema` (the shared DB the
tables join), `003-generate` (the checkpoint that advances it).
**Reference:** the memory change's FTS rendering (`001-storage` §3), openwiki derived
index, pi-llm-wiki derived metadata (rebuilt, never hand-edited).

---

## 1. Capability

The FTS index is the **retrieval backbone** of the KB. It is a **derived cache**
rebuilt from the markdown, living in the **project's shared DB** — the *same* SQLite
file / PG schema as memory and code-index (KB-O2 resolved: **no separate `kb.sqlite`**).
The markdown files in `.skillgrid/knowledge/` are the source of truth (KB-INV-1); the
index is a projection that can always be rebuilt by `kb_index` / `kb_reindex`.

**What is indexed:** page **bodies** (+ the page's frontmatter title/type/aliases for
ranking). Backlinks and pin-checks are **not** stored — they are computed on the fly
from the markdown (`001-structure` §6) and from the code (`006-validate`).

## 2. Why the shared DB (not a separate file)

The KB does **not** open its own database. It **adds tables to the store the memory
change defines** (`001-storage`, `002-schema`). Consequences, all inherited for free:

- **Dual-mode (INV-2):** the `kb_*` tables are defined once, dialect-neutral, and
  rendered for SQLite and PG by the same `Store` abstraction. A feature that cannot be
  expressed in both dialects is not allowed (S-1, S-2).
- **Migration (additive):** the `kb_*` tables ride the same ordered, additive migration
  set as the memory tables. A fresh store applies them; an existing store gets them by
  the next migration.
- **Sync/remote (INV-4):** the index is **local-derived** — it is not a synced entity.
  On a remote (PG) machine, the index is rebuilt locally from the vault's markdown
  (which is git-tracked / synced as files, not as DB rows). The store's sync journal
  (`01-memory/006-sync`) does not journal `kb_*` rows.

> **The vault syncs as files, the index is derived.** The markdown (source of truth) is
> git-tracked and shared like any other project file. The `kb_*` tables are a *local
> cache* of that markdown. Two machines with the same vault rebuild the same index.

## 3. The tables (dialect-neutral, in the shared store)

| Table | Purpose | Key columns |
|---|---|---|
| `kb_pages` | one row per page (the projection) | `id`, `sync_id` (`kbpg-…`, **local row id**, not cross-machine), `project`, `rel_path` (e.g. `decisions/auth-model`), `title`, `type`, `status` (`draft`/`stable`/`deprecated` — projected from frontmatter), `aliases` (json), `in_degree` (int — backlink count, cached; see note), `body_hash`, `anchor_count`, `claim_count`, `source_count`, `updated_at`, `deleted_at` |
| `kb_pages_fts` | the FTS index over page bodies | external-content FTS over `(title, body, type, rel_path)` — the body text is tokenized, not duplicated |
| `kb_claims` | the grounded assertions (for the pin-check's read side) | `id`, `page_id` (FK → kb_pages.id), `subject`, `literal`, `anchor_path`, `anchor_selector` (json — shape below), `created_at` |
| `kb_run_state` | the resumable run checkpoint (**authoritative**; `index/last-update.json` is its file projection) | `project`, `run_id`, `phase` (`idle`/`planning`/`generating`), `source_fingerprint`, `processed` (json — set of `{sync_id, fact_hash}` processed; **no `seq`**, the `SaveEvent` carries none), `plan` (json — the ordered page-job queue), `last_full_run_at`, `updated_at` |
| `kb_page_baselines` | the per-page baseline manifest (the DB half of `index/.page-manifest.json`, `003-generate` §5.2) | `project`, `rel_path` (PK with project), `source_fingerprint`, `page_version` (hash of the verified markdown bytes), `completed_by`, `completed_run_id`, `updated_at` |

**Notes:**

- `kb_pages.rel_path` is the vault-relative path (`decisions/auth-model`), **not** the
  numeric id. It is the **stable cross-machine identity** of a page (a page is a file,
  not a row). `sync_id` (`kbpg-…`) is a **local row id only** — minted for intra-store
  FKs, *not* a cross-machine identity (the KB's sync identity is `rel_path`, `008-
  transport` §4).
- `kb_pages.status` is projected from the frontmatter `status` (`draft`/`stable`/
  `deprecated`). `kb_query` **excludes `deprecated`** pages from results (and excludes
  `draft` unless `include_draft` is set). This is what makes "a `deprecated` page is
  unsearchable" (`001-structure` §5.1) schema-backed.
- `kb_pages.in_degree` is the **backlink count** (how many pages link here), cached by
  `kb_index` for the citation-depth boost in `kb_query` (`005-query` §2). The *listed*
  backlinks in `kb_read` are still computed on the fly (accuracy over speed, `005-query`
  §5); only the *count* is cached, and it is rebuilt by `kb_index` (not `kb_rebuild_meta`
  — see `008-transport` §3).
- `kb_pages.body_hash` is the hash of the page body. It is how `kb_index` detects "this
  page changed, re-index it" (incremental reindex) and how `kb_query` knows the row is
  fresh.
- `kb_claims` is the **read side** of the pin-check: `kb_validate` (`006-validate`)
  reads the claims from here, resolves the anchors against live code, and reports drift.
  The claims are *derived from the frontmatter* — `kb_index` parses `claims:` + the
  page's `anchors:` and populates the table. It is not hand-edited (KB-INV-3).
  `anchor_selector`'s JSON shape (per anchor kind, `006-validate` §2.1):
  - `all_of` → `{ "kind": "token", "all_of": ["capturePayment", "ENTERPRISE"] }`
  - `line`+`version` → `{ "kind": "span", "line": 482, "version": "kb-lines-v1:…" }`
  - `symbol` → `{ "kind": "symbol", "symbol": "capturePayment", "called_from": ["enterprise.go"]? }`
  A claim's `anchor_path` + `anchor_selector` together locate the "anchor site" the
  claim's `literal` must hold at.
- `kb_page_baselines` is the **current-ness ledger** (`003-generate` §5.2): it records,
  per page, the source fingerprint + verified `page_version` the page was derived from.
  A page whose baseline fingerprint no longer matches the live source is re-derived; one
  that still matches is a no-op candidate. It is written by `kb_generate` (on a proven,
  pin-checked page) and read by the no-op proof — never hand-edited (KB-INV-3).
- **FTS is external-content** (`content='kb_pages'` in SQLite): the FTS rowid maps to the
  base `kb_pages` row. The body is not duplicated; FTS stores the token index. This
  matches the memory change's FTS rendering (`001-storage` §3) exactly — same pattern,
  same triggers, same ranking weights shape.

### 3.1 The shared-store boundary

The memory change's `Store` interface (`001-storage` §7) does **not** expose the `kb_*`
tables (the memory function does not know about the KB). The KB has its **own** `Store`
methods (a `KBStore` interface, or an extension of `Store`) that the KB package calls.
Both ride the same dialect abstraction, the same connection, the same schema. This is
the "same store, different query surface" pattern — the KB and memory share the file
but not the interface, so neither imports the other.

## 4. Rebuild: `kb_index` / `kb_reindex`

**`kb_reindex`** (full): walks `.skillgrid/knowledge/`, parses every page (frontmatter +
body), and **rebuilds** `kb_pages` + `kb_pages_fts` + `kb_claims` from scratch. It is the
"delete the index and re-derive" path — guaranteed to converge the index to the
markdown (KB-INV-1). Idempotent: a second `kb_reindex` on an unchanged vault produces an
identical index.

**`kb_index`** (incremental): compares each page's `body_hash` to the stored
`kb_pages.body_hash`. Only changed/added pages are re-tokenized; deleted pages are
removed. It advances the `kb_run_state` checkpoint. This is the cheap, frequent path
(run after a `kb_generate` that wrote pages, or on a scheduled/CI pass).

**Both are local and idempotent.** Neither reads the LLM. They are pure
markdown → index projections. A flaky store (WAL lock) retries (the store's WAL retry,
`001-storage` §5).

## 5. Freshness + `degraded`

`kb_query` reports the index's freshness, defined **file-hash-based** (the cheaper,
deterministic definition — matching `007-lint`'s stale-index check exactly):

- **Fresh:** for every non-deleted page, the live file's body hash equals
  `kb_pages.body_hash`. The index matches the files.
- **Stale:** some page's live file body hash differs from `kb_pages.body_hash` (a page
  was written — by `kb_generate` or a direct edit — and not yet re-indexed). `kb_query`
  still answers (from the last index) but sets `degraded: stale_index` so the agent
  knows the result may miss the newest page (INV-3, explicit — never a silent gap).

> **Why file-hash, not write-event:** a write-event-based "stale" (a page was written
> since the last `kb_index`) diverges from file-hash in the *edited-then-reverted* case
> (file-hash: fresh; write-event: stale). File-hash is what the agent actually cares
> about — "does the index match the files on disk?" — and it is the same rule `kb_lint`
> uses, so `kb_query`'s `degraded: stale_index` and `kb_lint`'s stale-index finding can
> never disagree.

**The stale window is short by construction:** `kb_generate` calls `kb_index` as its
final step (step 5, `003-generate`), so the eager path re-indexes immediately. Staleness
only occurs if a user hand-edits a page without running `kb_index` — the lint
(`007-lint`) surfaces it.

## 6. What is NOT in the index

- **Backlinks.** Computed on the fly from the markdown (grep for `[[…]]`), never stored.
  The index stores *forward* content (what each page says), not the link graph.
- **Anchors' live state.** `kb_claims` stores the *declared* anchors/claims (from
  frontmatter). Whether they *still hold* is computed by `kb_validate` against live code
  — not stored (it would go stale the instant the code moves).
- **Session / trajectory state.** Not a source (`002-sources` §4).

## 7. Invariants

INV-2 (dual-mode — the tables ride the shared dialect abstraction), INV-3 (explicit
`degraded: stale_index`), INV-4 (local truth — the index is derived from local files),
KB-INV-1 (files are truth, index is rebuildable), KB-INV-3 (index never hand-edited).

## 8. Open items

None. The schema, the rebuild semantics, and the freshness model are fixed. The one
tunable (FTS ranking weights for the KB) is inherited from the memory change's
`001-storage` §3 weights and tuned in the review.
