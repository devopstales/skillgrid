# 001 — Files, Chunks, Freshness (the indexing unit)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `000-overview` (the freshness model, CI-1), memory `00-foundation/002-
schema` §4 (the declared table set), `00-foundation/001-storage` (dual-mode FTS rendering).
**Consumed by:** `002-symbols-edges` (symbols reference `files.id`), `003-extraction`
(the extract tx writes `files`), `006-search` (`chunks_fts` is the FTS leg), `007-watch`
(the fingerprint drives auto-index).

---

## 1. Capability

The `files` table is the **indexing unit** — one row per source file the indexer walked.
It is the **single source of truth for "which files are indexed and when,"** and it is
what the **incremental fingerprint** and the **freshness check** read. The `chunks`
table holds the text the FTS + semantic tiers search over. `index_freshness` is the
**query-path staleness record** (the one table that answers "is the graph current with
the working tree?").

## 2. The `files` table

```sql
CREATE TABLE files (
    id           INTEGER PRIMARY KEY,        -- local (not cross-machine)
    sync_id      TEXT NOT NULL,              -- INV-6 cross-machine identity (ULID)
    project      TEXT NOT NULL,              -- canonical project (local identity)
    path         TEXT NOT NULL UNIQUE,       -- repo-relative path
    mtime_ns     INTEGER NOT NULL,
    size         INTEGER NOT NULL,
    content_hash TEXT NOT NULL,              -- sha256 of file content
    language     TEXT NOT NULL,              -- detected language (extract.go ext map)
    ast_hash     TEXT NOT NULL DEFAULT '',   -- FNV-1a over the extracted structure
    indexed_at   TEXT NOT NULL,
    FOREIGN KEY (project) REFERENCES ...
);
```

**Notes:**

- **`path` is the stable key** the agent sees (a file is a path, not a row). `sync_id`
  is minted for cross-machine identity (INV-6) + the upsert key; the local `id` is
  ephemeral. This mirrors the KB's `rel_path`-is-identity stance (KB `004-index` §3).
- **`content_hash` is the change signal.** The incremental guard (`003-extraction`)
  skips a file iff `mtime_ns AND size AND content_hash` are all unchanged. A content
  change (even a whitespace-only one) forces a re-extract; an mtime-only change (a
  `git checkout` that touches mtime, not content) does not.
- **`ast_hash`** is a deterministic FNV-1a over the *extracted structure* (symbol
  UIDs + edge tuples), not content. A comment-only edit keeps the same `ast_hash`
  (the structure is unchanged) — this is the "re-extract was a no-op on the graph"
  signal. It is **informational** in v2 (a future skip-re-embed hook); it is *not*
  part of the incremental guard (the guard is `content_hash`-based, so a
  structure-preserving content change still re-extracts but produces an identical
  graph — the write is idempotent).
- **`language`** is detected by extension (`003-extraction` §language support) and is
  the partition key for per-language coverage + the semantic-tier language filter.
- **Dual-mode (INV-2):** the `UNIQUE(path)` is per-project (the local file has one
  `path`; the PG schema is `schema_<project>`, so the uniqueness is naturally
  per-schema). No `sync_id` column is needed on the local dialect beyond identity.

## 3. The `chunks` table + `chunks_fts`

```sql
CREATE TABLE chunks (
    id           INTEGER PRIMARY KEY,
    file_id      INTEGER NOT NULL,
    kind         TEXT NOT NULL DEFAULT 'ast',   -- 'ast' | 'lines'
    start_line   INTEGER NOT NULL,
    end_line     INTEGER NOT NULL,
    text         TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

-- FTS5 (SQLite) / tsvector (Postgres) external-content over chunks.text
CREATE VIRTUAL TABLE chunks_fts USING fts5(text, content='chunks', content_rowid='id',
    tokenize='trigram');
```

**Chunking (CI: AST-first, `005` CI-O5 = `ast` default):** one chunk per
function/class/method (≤ 1000 chars target); tiny leading segments merged to ≥ 250
chars; gaps between symbols become line segments; over-long symbols are line-windowed
(80/10 overlap). If the AST chunker produces nothing for a file (no symbols — e.g. a
config), it falls back to plain 80-line chunks (`kind='lines'`).

**`chunks_fts` is the FTS leg** of `code_search` (`006-search`). It is **derived**
(CI-INV-1): rebuilt from `chunks` by `code_index`, never hand-edited. External-content
so the FTS rowid maps to `chunks.id` (the same pattern memory uses for its
`observations_fts`, `00-foundation/002-schema` §5).

**Dual-mode (INV-2):** SQLite FTS5 trigram; Postgres `tsvector` + GIN + triggers. The
`tokenize='trigram'` is the SQLite rendering; the PG rendering is a `tsvector` column
generated from `text`.

## 4. The `index_freshness` table (the query-path contract)

v1 had two tables for freshness (`fingerprint_meta` + `index_meta_kv`). **v2 collapses
them into one** — the single record that answers "is the graph current?":

```sql
CREATE TABLE index_freshness (
    project            TEXT PRIMARY KEY,
    extractor_stamp    TEXT NOT NULL,        -- the extractor version (a change = full re-walk)
    schema_fingerprint TEXT NOT NULL,        -- FNV-12 of the structural schema
    last_indexed_at    TEXT,                 -- wall-clock of the last successful code_index
    files_indexed      INTEGER NOT NULL DEFAULT 0,
    files_deleted      INTEGER NOT NULL DEFAULT 0,
    files_oversized    INTEGER NOT NULL DEFAULT 0,   -- skipped for size (a first-class outcome)
    symbols_count      INTEGER NOT NULL DEFAULT 0,
    edges_count        INTEGER NOT NULL DEFAULT 0,
    chunks_count       INTEGER NOT NULL DEFAULT 0,
    unresolved_count   INTEGER NOT NULL DEFAULT 0,   -- the drop-not-guess total
    coverage_fair      REAL,                 -- per-language fair coverage (004 §coverage)
    capabilities       TEXT NOT NULL,        -- JSON: {graph,fts,vector,lsp,pdg,communities,process}
    updated_at         TEXT NOT NULL
);
```

**How the freshness check works** (`000-overview` §4, the query path):

1. **A metadata-only stat-walk** (size + mtime, ~ms) of the source files (respecting
   the include/exclude globs + `.knowledgeignore`-style excludes) is compared against
   the `files` table rows.
2. **The extractor stamp is checked first.** If `extractor_stamp` ≠ the stored
   `extractor_stamp` (the extractor code changed version), **every file is "changed"**
   (a full re-walk is forced). This is the "the extractor changed, so the whole graph
   is suspect" signal.
3. **Otherwise** only added/modified/deleted files (per the stat-walk vs `files`) are
   reported. A query tool that finds the index **fresh** (no drift) proceeds; one that
   finds it **stale** answers from the last index + sets `degraded: stale_index`
   (CI-INV-4).
4. **`code_status`** reads this table directly (no stat-walk) and reports the stored
   counts + the `capabilities` JSON + the `coverage_fair` + the `last_indexed_at`.

**The staleness definition is file-hash-based** (the same stance as the KB's `004-
index` §5): "does the index match the files on disk?" — not "was a write event
emitted since the last index." An edited-then-reverted file is *fresh* (the hash
matches), not stale. This keeps `code_status`'s stale flag and the query-path
`degraded: stale_index` from ever disagreeing.

**`schema_fingerprint`** is the FNV-12 of the structural schema (the DDL). If the
stored fingerprint differs from the current schema (a migration ran), the index is
**stale-by-schema** (a re-`code_index` is needed). This is the v1 `index_meta_kv`
`schema_fingerprint` role, collapsed into the one table.

## 5. The `unresolved_edges` ledger (the drop-not-guess detail)

v1 scattered drop-not-guess across four tables (`unresolved_refs`, `route_drops`,
`unresolved_members`, `resolution_audit`). **v2 collapses them into one ledger** — the
per-edge record of "we saw this reference but could not resolve it" (CI-INV-3):

```sql
CREATE TABLE unresolved_edges (
    id          INTEGER PRIMARY KEY,
    file_id     INTEGER NOT NULL,
    reference   TEXT NOT NULL,        -- the name we saw (the to_name of a name-only edge)
    kind        TEXT NOT NULL,        -- the edge kind we tried to resolve (calls, references, ...)
    line        INTEGER,
    candidates  TEXT,                 -- JSON: the candidate symbols (empty = no candidates)
    status      TEXT NOT NULL DEFAULT 'open',   -- 'open' | 'resolved' | 'dropped'
    first_seen  TEXT NOT NULL,
    last_seen   TEXT NOT NULL,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);
```

**Target-state per file** (the v1 idempotency stance): on each re-extract of a file,
its `unresolved_edges` rows are deleted + re-inserted from the fresh extract. A
reference that *becomes* resolvable (the target symbol now exists) is dropped from the
ledger (it's now a resolved edge, not unresolved). The **per-file rollup** (count of
open unresolved per file) is computed on demand for `code_status` / `code_unresolved`
(admin), not stored separately.

**This is the quantified health signal** (v1's `resolution_audit` role): how much of
the reference surface the extractor *refused to guess*. A high `unresolved_count` in a
language is the "the extractor is weak here" signal.

## 6. The indexing tx (the write path)

The core graph write is a **single transaction** per `code_index` run (the v1
single-connection constraint is honored): for each changed file, one tx upserts
`files` + `chunks` (delete old + insert new) + `symbols` + `edges` (the
`002-symbols-edges` write) + `unresolved_edges`, and updates `index_freshness` at the
end. The **optional passes** (CI-5: PDG/communities/process, CI-6: LSP) run in a
**separate connection** *after* the core tx commits (they are advisory — they never
roll back the committed graph). The knowledge nodes (route/doc/config/sql) are **not**
separate passes in v2 — they are `symbols` with a kind, written in the core tx
(`003-extraction` §knowledge nodes).

## 7. Invariants

INV-1 (the `files` row carries origin), INV-2 (dual-mode), INV-3 (`degraded:
stale_index` on the query path), INV-4 (local is truth), INV-6 (`sync_id`).
CI-INV-1 (derived — `chunks`/`chunks_fts`/`unresolved_edges`/`index_freshness` are all
rebuildable from the tree), CI-INV-2 (current-state), CI-INV-3 (drop-not-guess),
CI-INV-4 (stale is a signal, not a failure).

## 8. Open items

- **CI-O2 (multi-project):** one project per DB (matches memory). The `project` column
  is the local identity; cross-project is a separate DB. Confirmed.
- **CI-O4 (ledger granularity):** per-edge rows (the detail) + a computed per-file
  rollup. Confirmed above.
- **CI-O5 (chunking):** AST-first, `lines` fallback. Confirmed above.
- **`MaxFileSize`:** v1's 500 KiB size-skip is a **first-class outcome** (counted in
  `files_oversized`, not silently dropped). A file over the cap is not indexed (its
  `files` row is absent) and the count surfaces in `code_status`. The cap is a config
  default (not a spec constant).
