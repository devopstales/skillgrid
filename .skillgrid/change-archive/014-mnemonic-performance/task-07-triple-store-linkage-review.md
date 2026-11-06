# Task 07 — Triple-Store Linkage — Review

**Change**: `014-mnemonic-performance` — step 07 (cross-link relational + vector + graph stores)
**Reviewer**: fresh, no prior context
**Commits under review**: `b293b1c`, `c80bc9c`, `9799a90`, `0044ec3`, `b9e516b`
**Verdict**: **PASS**

---

## Verdict

**PASS.** All four sub-tasks (07.1–07.4) are delivered, the same-store
architecture is sound, the single-DB JOIN is valid, the default search path is
byte-for-byte unchanged, the migration is purely additive, and no tool contract
changed. The four focused tests pass and the full `memory` + `codeindex`
suites are green. Two design notes below (the dropped embeddings leg in
`CrossLinkQuery`, and the suffix-match in `graphRefForSource`) are worth
recording but neither is a defect for this step's scope.

---

## The key architectural decision — VERIFIED

**Same-store claim: CONFIRMED.** The claim that `observations`,
`long_term_memories`, `files`, `symbols`, and `embeddings` all live in ONE
per-project SQLite file is correct. Evidence:

- `store/store.go:110` — `dbPath := filepath.Join(dataDir, projectID+".sqlite")`
  opens exactly one file per project.
- `store/store.go:286-344` (`migrate`) — reads `migrationsFS` (all `.sql` files
  under `migrations/`), sorts them, and applies **every** migration (001–022) to
  that single `*sql.DB`. There is no per-domain store split.
- `service/service.go:285-294` — `openProject` calls `store.Open(s.dataDir,
  projectID)` once and passes that **same** `*store.Store` to both
  `memory.New(st, projectID)` and (downstream) `codeindex.New(st)`.
- `codeindex/indexer.go:277` — `func New(st *store.Store) *Indexer` — the
  codeindex `Indexer` is handed the same store handle.

Therefore the cross-link JOIN in `CrossLinkQuery` and the orphan sweep in
`CheckCrossLinkIntegrity` are valid single-database SQL JOINs. There is no
separate codeindex store that could silently return nothing. **This is the
CRITICAL check and it passes.**

**Bridge-reuse decision: SOUND.** The `embeddings` table
(`011_hybrid_code_intel.sql:74-80`) already has `symbol_id` PK, `model`, `dim`,
`vector` BLOB, `updated_at` — a 1:1 map onto the brief's `symbol_embeddings`
shape. Reusing it avoids duplicating every row the indexer already writes. The
`codeindex/symbol_embeddings.go` accessors are `*sql.DB`-based (not
`*store.Store`), so `memory` can reach the bridge **without** importing
`codeindex` (which imports `memory` — `codeindex/indexer.go:796` uses
`memory.EncodeVector`). Import cycle avoided correctly.

---

## Findings

### [nit] `CrossLinkQuery` drops the embeddings leg from the brief's JOIN

- **File**: `skillgrid-cli/internal/mnemonic/memory/crosslink.go:222-234`
- **Brief SQL**: `observations o JOIN symbol_embeddings se ON o.graph_ref =
  se.symbol_id JOIN observations o2 ON o2.graph_ref = se.symbol_id WHERE o.id = ?
  AND o2.id != ?`
- **Actual SQL**:
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
- **Assessment**: Functionally equivalent for the stated goal ("related
  observations"). `graph_ref` already stores the `symbols.id`, so two
  observations share a symbol iff they share `graph_ref`. The intermediate
  `embeddings` table bridges `symbol_id → vector` (for vector-similarity
  retrieval, a separate concern handled in-Go by `SearchByVector` over
  `observations.embedding`), not `symbol → other observations`. Joining
  `embeddings` in the middle would be a no-op for finding related observations
  (it would just add a redundant equi-join on the same key). The implementer's
  simplification is correct and cleaner. **Not a bug.** Note it for the
  acceptance record so the reviewer of a later step that expects a
  vector-similarity leg knows the graph leg is by-symbol, not by-vector.
- **Fix**: none required. Optionally add a one-line comment in
  `crosslink.go` noting the embeddings leg was intentionally folded into the
  `graph_ref` equality join.

### [nit] `graphRefForSource` suffix-match can bind the wrong file

- **File**: `skillgrid-cli/internal/mnemonic/memory/service.go:413-422`
- **SQL**:
  ```sql
  SELECT (SELECT id FROM symbols WHERE file_id = f.id
          ORDER BY start_line, id LIMIT 1)
  FROM files f
  WHERE f.path = ? OR f.path LIKE '%' || ?
  LIMIT 1
  ```
- **Assessment**: The `f.path LIKE '%' || ?` suffix clause means a source like
  `internal/foo/bar.go` binds to *any* indexed file whose path ends in
  `internal/foo/bar.go`. If two indexed files share that suffix (e.g. in
  different module roots), the first by row order wins (no `ORDER BY` on the
  outer `files f` select). The implementer flagged this in the report's
  Concerns and called it acceptable for a best-effort advisory link. Agreed —
  `graph_ref` is advisory (best-effort, never blocks Save), and the orphan
  check keeps any dangling id visible. But the outer `LIMIT 1` without an
  `ORDER BY` is nondeterministic when multiple suffix matches exist.
- **Fix (optional, low priority)**: add `ORDER BY length(f.path) ASC` to the
  outer select so the shortest (most specific) matching path wins
  deterministically. Purely a robustness polish; not a blocker.

### [nit] `graph_ref` is set via a second `UPDATE` after `INSERT` (extra round-trip)

- **File**: `skillgrid-cli/internal/mnemonic/memory/service.go:389-398`
- **Assessment**: `Save()` runs `INSERT INTO observations …` (graph_ref not in
  the column list, so it defaults to NULL), then `UPDATE observations SET
  graph_ref = ? WHERE id = ?` when a symbol is found. Two statements per save
  that has a resolvable symbol. The lookup `graphRefForSource` runs *between*
  them. This is clean and readable, and `files.path` is `UNIQUE` (001
  `CREATE TABLE files … path TEXT NOT NULL UNIQUE`), which SQLite backs with an
  implicit unique index — so the lookup is indexed and fast. Not a
  performance concern at Save frequency. **No fix required**; recording it so a
  future perf pass knows the extra UPDATE is intentional (the INSERT cannot
  carry graph_ref because it depends on the lookup result).

### [nit] `embedding_blob` column added but never populated

- **File**: `skillgrid-cli/internal/mnemonic/store/migrations/022_triple_store.sql:83`
- **Assessment**: `ALTER TABLE long_term_memories ADD COLUMN embedding_blob BLOB`
  is added but no code writes it (grep for `embedding_blob` in `.go` returns
  nothing). This is correct for this step — the brief says it is "infrastructure
  for later steps." Recording it so a later step that consumes
  `long_term_memories.embedding_blob` knows the column exists but is currently
  always NULL.

### [nit] Pre-existing test failure in `service` package (not caused by step 07)

- **File**: `skillgrid-cli/internal/mnemonic/service/reindex_structural_test.go:107`
- **Test**: `TestReindexStructuralIsEmbedderFree` — `sql: database is closed`
- **Assessment**: Verified pre-existing by checking out `4be43b7` (step 06,
  before any step-07 commit) and running the test there — it fails identically
  (`sql: database is closed`). Step 07 touched only `memory`, `codeindex`, and
  `store/migrations` — not `service`. This is a pooled-handle/WAL race in the
  reader auto-reopen path (07bec0a). **Not a step-07 regression.** Worth a
  separate follow-up ticket.

---

## Evidence checked

| # | Check | Verdict | Evidence |
|---|-------|---------|----------|
| 1 | Same-store architecture (CRITICAL) | **PASS** | `store.go:110` single `<dataDir>/<projectID>.sqlite`; `store.go:286-344` applies all 001–022 migrations to one DB; `service.go:285-294` passes the same `*store.Store` to `memory.New` and `codeindex.New`; `indexer.go:277` takes `*store.Store`. The single-DB JOIN is valid. |
| 2 | `graph_ref` in `Save()` (07.1) | **PASS** | `service.go:393-398` — best-effort: `graphRefForSource` returns `sql.NullInt64{}` on any error/miss, and the `UPDATE` runs only when `ref.Valid`. Missing file / no symbol / lookup error → NULL. Sets only when found. Lookup is indexed (`files.path` UNIQUE → implicit unique index). Multiple symbols in a file → picks `ORDER BY start_line, id LIMIT 1` (first by start line), matching the `txRouteStore.FirstSymbolID` convention. No slow unindexed path. |
| 3 | `CrossLinkQuery` correctness (07.3) | **PASS** | SQL joins `observations o JOIN observations o2 ON o2.graph_ref = o.graph_ref WHERE o.id = ? AND o.graph_ref IS NOT NULL AND o2.id != ? AND o2.project = ? AND o2.deleted_at IS NULL`. Traverses observation→symbol(via graph_ref)→related observations. The embeddings leg is folded into the graph_ref equality join (functionally equivalent for the stated goal — see nit #1). NULL `graph_ref` → empty result set (the `WHERE o.graph_ref IS NOT NULL` guards it), not an error. |
| 4 | `embedding_blob` on `long_term_memories` (07.3) | **PASS** | Added as nullable `BLOB` by `022_triple_store.sql:83`. Not yet populated (correct — infrastructure for later steps, see nit #4). |
| 5 | `CheckCrossLinkIntegrity` (07.4) | **PASS** | `crosslink.go:259-289` — `SELECT o.id, o.graph_ref, o.title FROM observations o WHERE o.project = ? AND o.deleted_at IS NULL AND o.graph_ref IS NOT NULL AND NOT EXISTS (SELECT 1 FROM symbols sym WHERE sym.id = o.graph_ref)`. Returns `[]CrossLinkOrphan` (a list, not a count). On-demand (called only from the test; not wired into Save or any default path). Correctly finds orphans. |
| 6 | Additive-ness / migration | **PASS** | `022_triple_store.sql` is purely additive: `ALTER TABLE observations ADD COLUMN graph_ref INTEGER` (nullable), `CREATE INDEX IF NOT EXISTS idx_obs_graph_ref ON observations (graph_ref) WHERE graph_ref IS NOT NULL` (correct partial index), `ALTER TABLE long_term_memories ADD COLUMN embedding_blob BLOB` (nullable). No existing column/table touched. `git diff` of the migrations dir across all step-07 commits shows only `022_triple_store.sql` added (29 insertions). `022` sorts after `021` (confirmed by `ls | sort`). No `009_*` or earlier migration modified. |
| 7 | Opt-in (CRITICAL constraint) | **PASS** | `CrossLinkQuery` and `CheckCrossLinkIntegrity` are separate methods. The default search path (`Search` → `SearchWithScope` at `service.go:460-512`) does NOT call either — its SELECT column list and WHERE clause are byte-for-byte the pre-step-07 form (no `graph_ref`, no cross-link). `BlendedSearch`/`SearchByVector` (`search_embed.go`) also unchanged. The `graph_ref` UPDATE in `Save()` is the only default-path touch, and it is best-effort (never blocks). |
| 8 | Tool contract | **PASS** | No `mem_*`/`code_*`/`web_*` MCP tool signature changed. The diff touches only `codeindex/symbol_embeddings.go` (+test), `memory/crosslink.go` (+test), `memory/graph_ref_test.go`, `memory/service.go` (graph_ref in Save + helper), and the new migration. No tool registration file in the diff. |
| 9 | TDD discipline | **PASS** | All 4 tests run and pass (verified live): `TestGraphRefDefaultsToNull`, `TestSymbolEmbeddingsBridge`, `TestTripleStoreCrossLinkQuery`, `TestTripleStoreOrphanDetection`. The cross-link test (`graph_ref_test.go:492-536`) seeds a REAL file+symbol via `seedSymbolFile`, saves 2 observations bound to that symbol (verifying both carry `graph_ref = symbolID`), then asserts `CrossLinkQuery(obsA)` returns exactly `obsB`. The orphan test seeds a real symbol, creates a valid ref and a dangling `graph_ref = 987654`, and asserts `CheckCrossLinkIntegrity` returns exactly the orphan. Both the JOIN and the orphan detection are genuinely exercised. Full `memory` + `codeindex` suites green. |

---

## Approved to merge

**Yes.** Step 07 delivers the CRITICAL HUB correctly. The same-store
architecture is sound, the single-DB JOIN is valid, the default search path is
unchanged, the migration is purely additive, and no tool contract changed. The
findings are all nits (design notes and one pre-existing unrelated failure).
This step is safe to unblock steps 08-11, 14, and 25.
