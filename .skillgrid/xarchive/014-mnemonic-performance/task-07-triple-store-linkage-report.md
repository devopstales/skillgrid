# Task 07 — Triple-Store Linkage Report

## Summary

Cross-linked the relational (observations, long_term_memories), vector
(embeddings), and graph (symbols) stores of the single per-project SQLite
database. All changes additive; the cross-link query and integrity check are
opt-in methods, not part of the default search path. No `mem_*`/`code_*`/`web_*`
tool contract changed; `modernc.org/sqlite` (no CGO) kept.

## Same-store vs separate-store decision (key architectural choice)

**Same store.** `store.Open(dataDir, projectID)` (`store/store.go:99`) opens
exactly one SQLite file per project (`<dataDir>/<projectID>.sqlite`) and runs
all migrations (001-022) into it. The memory tables (`observations`,
`long_term_memories`, `files`) and the codeindex tables (`symbols`, `embeddings`)
all live in that same database — `files` is created by `001_initial.sql:60` in
the "Code index" section of the very same schema that hosts `observations`
(`001_initial.sql:16`), and `symbols`/`embeddings` are added by
`011_hybrid_code_intel.sql` into the same store. The codeindex `Indexer`
(`codeindex/indexer.go:277`) and the memory `Service` (`memory/service.go:234`)
are both handed the same `*store.Store` by the service layer, so the
cross-link JOIN is a plain single-database SQL JOIN — no in-memory join or
bridge-in-a-second-store needed.

## Bridge table: reused `embeddings`, did not create `symbol_embeddings`

Option (b) — **reused the existing `embeddings` table** as the bridge, because
it is in the same store as `observations` and already has exactly the required
shape (from `011_hybrid_code_intel.sql:74`):

```sql
CREATE TABLE IF NOT EXISTS embeddings (
    symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    vector BLOB NOT NULL,
    updated_at TEXT NOT NULL
);
```

The brief's `symbol_embeddings` spec (`symbol_id, embedding_blob, model,
created_at, PRIMARY KEY(symbol_id)`) maps 1:1 onto
(`symbol_id`, `vector`, `model`, `updated_at`, PK `symbol_id`) — a second table
would duplicate every row the indexer already writes.
`codeindex/symbol_embeddings.go` provides the CRUD surface the memory layer
uses (`UpsertSymbolEmbedding`, `GetSymbolEmbedding`,
`DeleteSymbolEmbedding`, `ErrSymbolEmbeddingMissing`), written against
`*sql.DB` so `memory` can reach the bridge without importing `codeindex`
(codeindex already imports memory — import cycle otherwise).

## Migration — `022_triple_store.sql` (next free number after 021)

```sql
-- 022: triple-store cross-linkage (change 014, step 07).
--
-- Cross-links the relational store (observations, long_term_memories), the
-- vector store (embeddings), and the graph store (symbols) inside the single
-- per-project SQLite database. Additive: every new column is nullable, so
-- existing rows and every existing query path are untouched. The cross-link
-- query (memory.CrossLinkQuery) and the integrity check
-- (memory.CheckCrossLinkIntegrity) are opt-in methods, not part of the
-- default search path.
--
--   observations.graph_ref      — codeindex symbols.id when the observation's
--                                 source file has an indexed symbol; NULL
--                                 otherwise (best-effort, never blocks Save).
--   long_term_memories.embedding_blob — optional precomputed embedding BLOB
--                                 (little-endian float32), mirroring the
--                                 observations.embedding layout.
--
-- NOTE: the `embeddings` table (migration 011) already bridges symbols to
-- vectors and lives in the SAME store as observations, so it is reused as
-- the symbol_embeddings bridge instead of creating a duplicate table. The
-- codeindex package exposes it via the symbol_embeddings.go accessors.

ALTER TABLE observations ADD COLUMN graph_ref INTEGER;

CREATE INDEX IF NOT EXISTS idx_obs_graph_ref
    ON observations (graph_ref)
    WHERE graph_ref IS NOT NULL;

ALTER TABLE long_term_memories ADD COLUMN embedding_blob BLOB;
```

## 07.1 — `graph_ref` on observations, best-effort in Save()

- Migration adds nullable `observations.graph_ref` + partial index
  `idx_obs_graph_ref` (non-NULL rows only).
- `Service.Save()` (`memory/service.go:373` region): after the INSERT,
  `graphRefForSource(ctx, source)` resolves the symbol and, when found,
  `UPDATE observations SET graph_ref = ? WHERE id = ?`. Best-effort by
  construction: a missing/empty source, an unindexed file, a file with no
  symbols, or any lookup error returns a NULL `sql.NullInt64` and the
  observation is stored with `graph_ref = NULL` — the save never fails.
- `graphRefForSource` (`memory/service.go`): matches `files.path` by absolute
  path or relative suffix (`f.path = ? OR f.path LIKE '%' || ?`) and picks the
  file's first symbol (`ORDER BY start_line, id LIMIT 1`) — the same
  "first symbol of a file" convention as `txRouteStore.FirstSymbolID`
  (`codeindex/indexer.go:882`).

## 07.2 — symbol_embeddings bridge (codeindex package)

`codeindex/symbol_embeddings.go` (new) over the reused `embeddings` table:

- `UpsertSymbolEmbedding(ctx, db, symbolID, vector, model)` — INSERT ...
  ON CONFLICT(symbol_id) DO UPDATE; dim = len(vector)/4; updated_at = now.
- `GetSymbolEmbedding(ctx, db, symbolID)` — returns (blob, model, err);
  `ErrSymbolEmbeddingMissing` when the symbol has no row.
- `DeleteSymbolEmbedding(ctx, db, symbolID)` — no-op when absent.

## 07.3 — opt-in cross-link query

`memory/crosslink.go` (new):

```sql
SELECT <o2 observation columns>
FROM observations o
JOIN observations o2 ON o2.graph_ref = o.graph_ref
WHERE o.id = ? AND o.graph_ref IS NOT NULL
  AND o2.id != ?
  AND o2.project = ?
  AND o2.deleted_at IS NULL
ORDER BY o2.id
```

`CrossLinkQuery(ctx, observationID)` scans the hits with the shared
`scanObservations` helper (same column list as the default search path, so
result shapes are identical). A NULL `graph_ref` (unlinked observation)
yields no results. **Opt-in**: nothing in `Search`/`SearchWithScope`/
`BlendedSearch`/`SearchByVector` calls it — the default path is byte-for-byte
unchanged. The `embedding_blob` column on `long_term_memories` is added by the
same migration (additive, nullable) as the long-term-memory side of the
triple store.

## 07.4 — orphan detection

`CheckCrossLinkIntegrity(ctx)` (`memory/crosslink.go`) — read-only sweep
scoped to the project:

```sql
SELECT o.id, o.graph_ref, o.title
FROM observations o
WHERE o.project = ?
  AND o.deleted_at IS NULL
  AND o.graph_ref IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM symbols sym WHERE sym.id = o.graph_ref)
ORDER BY o.id
```

Returns `[]CrossLinkOrphan{ID, GraphRef, Title}`. Runs on demand (e.g. a
doctor command); safe to schedule.

## RED → GREEN evidence

| Sub-task | Test | RED | GREEN |
|----------|------|-----|-------|
| 07.2 | `TestSymbolEmbeddingsBridge` (codeindex) | `undefined: GetSymbolEmbedding` / `ErrSymbolEmbeddingMissing` (build failed) | `ok ... 2.805s` |
| 07.1 | `TestGraphRefDefaultsToNull` (memory) | build failed (no graph_ref column / no CrossLinkQuery) | see GREEN run below |
| 07.3 | `TestTripleStoreCrossLinkQuery` (memory) | `fx.svc.CrossLinkQuery undefined` (build failed) | see GREEN run below |
| 07.4 | `TestTripleStoreOrphanDetection` (memory) | `fx.svc.CheckCrossLinkIntegrity undefined` (build failed) | see GREEN run below |

Focused GREEN run:

```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull|TestTripleStoreCrossLinkQuery|TestTripleStoreOrphanDetection' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	1.269s
```

## Full suites

```
$ go test ./internal/mnemonic/memory/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	9.903s

$ go test ./internal/mnemonic/codeindex/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	11.371s
```

Baselines (005/008/010/011/013 packages):

```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/route/... \
          ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	4.782s
--- FAIL: TestReindexStructuralIsEmbedderFree (0.95s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	44.680s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	0.620s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	9.471s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	4.599s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	4.022s
```

## Commits

| SHA | Subject |
|-----|---------|
| `b293b1c` | feat(mnemonic): add symbol_embeddings bridge accessors |
| `c80bc9c` | feat(mnemonic): add graph_ref column to observations |
| `9799a90` | feat(mnemonic): triple-store cross-link query via SQL JOIN |
| `0044ec3` | feat(mnemonic): add cross-link integrity check for orphan detection |
| `b9e516b` | test(mnemonic): move orphan-detection test to crosslink_test.go |

## Concerns

- **Pre-existing failure (not caused by this change)**: `service` package,
  `TestReindexStructuralIsEmbedderFree` — `sql: database is closed`
  (reindex_structural_test.go:107). Verified by `git stash` + re-run on HEAD
  (0044ec3) with none of step 07's changes: it fails identically there, in
  both full-suite and isolated runs. Looks like a pooled-handle/WAL race in
  the reader auto-reopen path (07bec0a `reader auto-reopen of published
  index`), worth a follow-up ticket; does not touch the triple-store path.
- The 07.1 commit includes the shared test helper file `graph_ref_test.go`
  (helpers `seedSymbolFile`, `graphRefOf`, `orphanGraphRefs` used by all three
  memory tests); the orphan-detection test itself lives in
  `crosslink_test.go` under the 07.4 commit.
- `graph_refForSource` matches `files.path` by suffix, so a source like
  `internal/foo/bar.go` binds to a file indexed as `/repo/internal/foo/bar.go`.
  If two indexed files ever share a suffix the first (by row order) wins —
  acceptable for a best-effort advisory link, and the orphan check keeps any
  dangling id visible.
