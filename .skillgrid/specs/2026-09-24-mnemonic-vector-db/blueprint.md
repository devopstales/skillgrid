# Mnemonic Vector DB Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T3 (risky: dependency version bump + new one-way migration; T2 default upgraded because the modernc v1.45.0 to v1.59.0 bump is a high-risk dependency change validated against 39 migrations + WAL-retry + store pooling)

**Build shape:** Tracer thread (one end-to-end path - bump + vec0 migration + indexer dual-write + vectorstore top-K - works first, then thickens with the exactness check and the durable-path flag)

**Goal:** Adopt option G (`modernc.org/sqlite/vec`, cgo-free in-SQL vector store) as the durable semantic-leg path, unblocked by validating the modernc v1.45.0 to v1.59.0 bump against the existing store, and add a `vectorstore/` package that the indexer dual-writes and the durable-path semantic leg reads.

**Architecture:** Reuses the existing `modernc.org/sqlite` driver (bumped v1.45.0 to v1.59.0 for the `vec` subpackage), the existing BLOB embedding tables (`embeddings`, `chunk_embeddings` - source of truth, unchanged), and the existing in-memory cosine cache (`hybrid/vectorcache.go` - stays the hot path). Adds a `vectorstore/` package wrapping two `vec0` virtual tables (one per code-vector table) with an exact top-K query (`ORDER BY vec_distance_cosine`) and a brute-force exactness check. The indexer's `embedPass` dual-writes: after each BLOB upsert it mirrors the row into the vec table, and on model swap it clears both. A new `MNEMONIC_VECTOR_DB=1` opt-in flag routes the durable semantic leg through `vectorstore` (in-SQL) instead of the in-memory cache; the hot path (default) is unchanged.

**Tech Stack:** Go 1.22+, `modernc.org/sqlite` v1.59.0 (bump from v1.45.0), `modernc.org/sqlite/vec` subpackage (cgo-free, blank-import), FTS5 (existing, unchanged), in-memory cosine (`hybrid/vectorcache.go`, unchanged hot path), `database/sql`.

**Spec:** `.skillgrid/artifacts/04-adr-0009-vector-search-in-sql-latency-viant-deferred.md` (G is the confirmed revisit path) + `.skillgrid/artifacts/04-adr-0006-vector-search-in-memory-brute-force.md` (in-memory stays the hot path; this change adds the durable path, does not replace it) + `.skillgrid/artifacts/07-mnemonic-tool-surface.md` (session-inject's durable-path candidate).

**Findings:** `.skillgrid/specs/2026-09-24-mnemonic-vector-db/findings.md` (Spike 001 - G in-SQL top-K ~6.9s at 100K / ~1.4s at 20K, viant deferred, G's string-based insert + batch/BLOB path needed for bulk re-index, modernc bump must be validated against 39 migrations/WAL-retry/pooling).

## Global Constraints

- Go 1.22+ minimum to build (config `context`).
- No new *modules* without an ADR. `modernc.org/sqlite/vec` is a subpackage of the existing `modernc.org/sqlite` module (not a new module), but the v1.45.0 to v1.59.0 version jump is a dependency change - it is covered by ADR-0009's consequence ("the modernc bump has not yet been validated... this validation is a prerequisite for the real build") and re-confirmed by this blueprint's Task 1.
- The BLOB columns (`embeddings.vector`, `chunk_embeddings.vector`) remain the **source of truth**. The vec tables are a **derived index** - they must never be the only copy. If a vec table is absent, empty, or corrupt, the semantic leg degrades to the in-memory path (never fails).
- **SetMaxOpenConns(1) is a hard invariant** of the store (WAL single-writer). The vec0 registration must work under it (the spike's viant deadlock was caused by re-entering `db.Exec` from inside a vtab callback with MaxOpenConns=1; G has no such re-entrancy, but Task 1's validation test must confirm G registers + queries under MaxOpenConns=1).
- Trivy is advisory-only (config `security.trivy.fail_on: ""`) - findings reported, never blocking.
- Conventional commits only; no AI-attribution trailer (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).

## Terms

None beyond the in-force ADRs (0006, 0009) and the spike. The term **durable path** (in-SQL G, ~6.9s at 100K) vs **hot path** (in-memory cosine, ~40ms at 100K) is defined in ADR-0009's decision outcome #2 and reused verbatim.

---

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `go build ./...` succeeds after the modernc v1.45.0 to v1.59.0 bump (Task 1).
- The full existing test suite (`go test ./...`) passes after the bump - the 39 squashed migrations, WAL-retry (`isWALBusy`), and store pooling (`handleCache`) are all still green (Task 1). **backstop** - this is the spike's prerequisite; the spike used a fresh DB, so the bump must be validated against the *existing* store, not a blank one.
- `isWALBusy` still classifies a concurrent-writer busy error correctly after the bump (the typed `*sqlite.Error` / `Code() == SQLITE_BUSY` path AND the text fallback) (Task 1).
- `store.Open` still returns a pooled handle (same `*sql.DB` for repeated opens of the same project, refcounted, cache-evicted on close) after the bump (Task 1).
- The migration `042_vec0_tables.sql` creates two vec0 virtual tables (`vec_symbols`, `vec_chunks`) keyed by `symbol_id` / `chunk_id` respectively, and the store opens an existing DB and finds the tables present (Task 2).
- The migration is idempotent: running `store.Open` twice on the same DB does not error and does not recreate the tables (Task 2).
- The vec tables work under `SetMaxOpenConns(1)` (no deadlock, no "database is locked" on a simple insert+query) (Task 1 - the spike's viant deadlock was exactly this).
- `vectorstore.SearchSymbols(ctx, db, queryVec, limit)` returns the top-K `symbol_id`s by cosine proximity (ascending `vec_distance_cosine`), and the IDs are a subset of the `embeddings` table's `symbol_id`s (Task 3).
- `vectorstore.SearchChunks(ctx, db, queryVec, limit)` returns the top-K `chunk_id`s (Task 3).
- `vectorstore.UpsertSymbol(ctx, tx, symbolID, model, dim, vecBytes)` and `vectorstore.DeleteSymbols(ctx, tx)` mirror the BLOB table's writes (Task 4).
- `vectorstore.ExactnessCheck(ctx, db, table, queryVec, limit)` returns a per-ID agreement report comparing the vec table's top-K against a Go-side brute-force cosine scan over the BLOB table; it reports `agree` (identical ID set) or the set of disagreements (Task 5).
- After the indexer's `embedPass` runs on a fresh store with the embedder active, the vec tables contain exactly the same rows as the BLOB tables (row count + per-ID vector agreement) (Task 4).
- With `MNEMONIC_VECTOR_DB=1` and a populated vec table, the durable semantic leg returns the same top-K IDs as the in-memory leg (exactness agreement, within float tolerance) (Task 6). **backstop** - this is the cross-path equivalence the spike's exactness-check methodology proves; the diff alone cannot confirm the two paths agree at runtime.
- With `MNEMONIC_VECTOR_DB` unset (default), the semantic leg uses the in-memory cache exactly as before (no behavior change, no vec table read) (Task 6).
- With an empty vec table (no embeddings yet) and `MNEMONIC_VECTOR_DB=1`, the durable leg degrades to the in-memory path (no error, no panic) (Task 6).

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/mnemonic/vectorstore/store.go` - the `vectorstore` package: `SearchSymbols`, `SearchChunks`, `UpsertSymbol`, `UpsertChunk`, `DeleteSymbols`, `DeleteChunks`, `TableExists`, `Count`]
- [`skillgrid-cli/internal/mnemonic/vectorstore/exact.go` - `ExactnessCheck`: brute-force Go-cosine ground-truth scan + per-ID agreement report against the vec table's top-K]
- [`skillgrid-cli/internal/mnemonic/vectorstore/store_test.go` - tests for search, upsert, delete, table-exists, count, MaxOpenConns=1, empty-table degrade]
- [`skillgrid-cli/internal/mnemonic/vectorstore/exact_test.go` - tests for the exactness check (agreement + disagreement detection)]
- [`skillgrid-cli/internal/mnemonic/store/migrations/042_vec0_tables.sql` - creates `vec_symbols` + `vec_chunks` vec0 virtual tables (dimension-pinned to 768, the default `DefaultOnnxDim`)]
- [`skillgrid-cli/internal/mnemonic/codeindex/indexer.go` (modified) - `embedPass` dual-writes the vec tables after each BLOB upsert; on model swap, clears both vec tables alongside the BLOB tables]
- [`skillgrid-cli/internal/mnemonic/hybrid/durable.go` (new) - the durable-path leg: when `MNEMONIC_VECTOR_DB=1` and the vec table is non-empty, returns top-K via `vectorstore`; otherwise returns the in-memory cache result (degrade)]
- [`skillgrid-cli/internal/mnemonic/hybrid/rank.go` (modified) - `vectorLeg`/`chunkVectorLeg` consult `durable.go` first, fall back to the in-memory cache (the hot path, unchanged)]
- [`skillgrid-cli/internal/mnemonic/hybrid/durable_test.go` - tests for flag routing, empty-table degrade, cross-path equivalence]

**Key links** (critical connections that must work together):
- `indexer.embedPass` BLOB upsert to `vectorstore.UpsertSymbol/Chunk` (dual-write in the same `*sql.Tx`).
- `indexer.embedPass` model-swap `DELETE FROM embeddings` to `vectorstore.DeleteSymbols/DeleteChunks` (same transaction).
- `store.migrate` runs `042_vec0_tables.sql` to `vectorstore.TableExists` returns true on the opened DB.
- `hybrid.rank.go vectorLeg` to `hybrid.durable.go` to `vectorstore.SearchSymbols` (flag-gated), with the in-memory `vecCache` as the fallback.
- `vectorstore.ExactnessCheck` reads the BLOB table (ground truth) + the vec table (candidate) to the agreement report, the backstop for cross-path equivalence.

**One-way-door decisions:**
- **one-way: the modernc v1.45.0 to v1.59.0 version bump** (Task 1). The driver wraps a newer SQLite (3.46 to 3.49+); the bump is a dependency change that touches every store read/write. It is hard to reverse (downgrade re-validates the same 39 migrations). Tagged on Task 1.
- **one-way: the vec0 virtual tables (migration 042)** (Task 2). Virtual tables are not plain tables; dropping them later is a one-way migration. The tables are additive (no data loss) but the migration is forward-only. Tagged on Task 2.

## Hypothesis

**Claim:** Adopting option G as the durable semantic-leg path - via the modernc v1.45.0 to v1.59.0 bump, a vec0 migration, and an indexer dual-write - produces a durable in-SQL vector index that (a) passes the full existing suite against the real store, (b) works under the store's `SetMaxOpenConns(1)` invariant, and (c) returns top-K IDs that agree with the in-memory cosine leg within float tolerance.

**Right condition:** `go test ./...` is green after the bump; `vectorstore.SearchSymbols` top-K agrees with a Go brute-force cosine scan (exactness `agree`); the durable leg (flag on) returns the same top-K as the hot path for the same query.

**Wrong condition:** The full suite fails after the bump (a migration, WAL-retry, or pooling regression); OR the vec tables deadlock under MaxOpenConns=1; OR the durable leg's top-K diverges from the in-memory leg for a populated store (the two paths disagree, so the "durable path" is not a drop-in for the hot path).

**Thinnest MVP:** Task 1 (bump + suite green) + Task 2 (vec0 migration) - the bump is validated and the tables exist. This proves the spike's prerequisite before any query path is built.

**Door check:** Task 1 - if `go test ./...` is not green after the modernc bump (a regression in the 39 migrations, WAL-retry, or store pooling), the bump is not validated and the blueprint is invalidated (ADR-0009's prerequisite is unmet). Stop and report.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`mem_*` / `code_*` / `web_cache_*`) | N/A: this change adds no new MCP tool and changes no existing tool's contract. `code_semantic_search` / `semantic_search` keep their signatures; the durable path is internal (a new `durable.go` behind a flag) and the hot path is byte-identical. | No tool contract delta. The flag `MNEMONIC_VECTOR_DB` is a process env var, not a tool param. | (N/A - no test) |
| Shared-convention drift | N/A: no `_shared/conventions/*.md`, `_shared/references/*.md`, or `agent-config/*.md` is edited. | - | (N/A - no test) |
| Git repository selection | N/A: no `git -C`, repo path, or worktree logic touched. | - | (N/A - no test) |
| Commit / push / PR state | N/A: no commit/push/PR automation. | - | (N/A - no test) |
| Documentation-like paths | N/A: no executable-file classification or path classification. | - | (N/A - no test) |

No core boundary is Applicable, so no Applicable row needs a covering scenario in `acceptance.feature` beyond the change's own behavioral scenarios (the backstop truths above).

## Tasks

### Task 1: Validate the modernc v1.45.0 to v1.59.0 bump against the existing store

> one-way: the modernc v1.45.0 to v1.59.0 version bump. The driver wraps a newer SQLite (3.46 to 3.49+); every store read/write is affected. Hard to reverse (a downgrade re-validates the same 39 migrations).

**Files:**
- Modify: `skillgrid-cli/go.mod` (bump `modernc.org/sqlite v1.45.0` to `v1.59.0`)
- Modify: `skillgrid-cli/go.sum` (transitive bumps: `modernc.org/libc`, `modernc.org/memory`, `modernc.org/cc/v4`, `modernc.org/ccgo/v4`, `modernc.org/mathutil` - resolved by `go mod tidy`)
- Modify: `skillgrid-cli/internal/mnemonic/store/store.go:19` (verify the `modernc.org/sqlite/lib` import still resolves; the `sqlite3.SQLITE_BUSY` constant is used at `store.go:164`)
- Test: `skillgrid-cli/internal/mnemonic/store/bump_validation_test.go` (new - the targeted regression tests for WAL-retry + pooling under the new driver)

**Interfaces:**
- Consumes: nothing (this is the foundation task).
- Produces: a green `go build ./...` + `go test ./...` on the new driver, so Tasks 2-6 can assume the store layer is intact.

**SATISFIES:** (no acceptance scenario yet - this is the door check / prerequisite. The BDD scenarios land in Task 2-6.)

- [ ] **Step 1: Write the failing bump-validation test**

Create `skillgrid-cli/internal/mnemonic/store/bump_validation_test.go` (the typed-error shim and the three tests - IsWALBusy classification, pooled-handle reuse, migrations-still-apply). The `sqliteErr` shim needs only a `Code()` method matching the driver's `*sqlite.Error` interface so `errors.As` in `isWALBusy` succeeds. Verify the actual `*sqlite.Error` type shape at v1.59.0 before finalizing the shim (it may have a public constructor).

- [ ] **Step 2: Bump the driver and tidy**

Run in `skillgrid-cli/`: `go mod edit -require=modernc.org/sqlite@v1.59.0` then `go mod tidy`.

- [ ] **Step 3: Run the full suite to see the bump's effect**

Run: `go build ./... && go test ./...`
Expected: either PASS (bump is clean) or FAIL with a specific regression (a migration, the `lib` import, the text fallback, or pooling). If FAIL, fix the regression (e.g. update the text fallback strings in `isWALBusy` to match the new driver's rendered messages, or adjust the `lib` import path) and re-run until green. The fix must be minimal - do not restructure the store layer.

- [ ] **Step 4: Run the targeted bump-validation tests**

Run: `go test ./internal/mnemonic/store/ -run 'TestBump_' -v`
Expected: PASS (3 tests: IsWALBusy, OpenPools, MigrationsStillApply).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/go.mod skillgrid-cli/go.sum skillgrid-cli/internal/mnemonic/store/
git commit -m "feat(mnemonic): validate modernc v1.45.0 to v1.59.0 bump against store

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 1
adr: 0009
"
```

---

### Task 2: Add the vec0 virtual tables (migration 042)

> one-way: the vec0 virtual tables. Virtual tables are not plain tables; the migration is forward-only (dropping them later is a one-way migration). Additive (no data loss).

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/042_vec0_tables.sql`
- Test: `skillgrid-cli/internal/mnemonic/store/vec0_migration_test.go` (new)

**Interfaces:**
- Consumes: Task 1's green driver (the `vec` subpackage is available at v1.59.0).
- Produces: `vec_symbols` (keyed by `symbol_id`) and `vec_chunks` (keyed by `chunk_id`) vec0 tables present on every opened store. `vectorstore.TableExists` (Task 3) will check for them.

**SATISFIES:** `vec0-tables-exist` + `vec0-migration-idempotent` (in `acceptance.feature`).

- [ ] **Step 1: Write the failing migration test**

Create `skillgrid-cli/internal/mnemonic/store/vec0_migration_test.go` with two tests: `TestVec0TablesCreatedAfterOpen` (opens a fresh store, asserts `vec_symbols` + `vec_chunks` are present in `sqlite_master`) and `TestVec0MigrationIdempotent` (opens the same store twice, asserts no error and each table count stays exactly 1).

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mnemonic/store/ -run 'TestVec0' -v`
Expected: FAIL - `vec0 table "vec_symbols" not created by migration 042` (the migration doesn't exist yet).

- [ ] **Step 3: Write the migration**

Create `skillgrid-cli/internal/mnemonic/store/migrations/042_vec0_tables.sql`:

```sql
-- 042: vec0 virtual tables for the durable in-SQL vector path (ADR-0009, option G).
--
-- Additive, forward-only. Two vec0 tables mirror the BLOB embedding tables
-- (embeddings, chunk_embeddings) which remain the source of truth. The vec
-- tables are a derived index: if absent/empty/corrupt the semantic leg
-- degrades to the in-memory path (never fails).
--
-- Dimension is pinned to 768 (DefaultOnnxDim, the built-in embedder default).
-- The vec subpackage is cgo-free (modernc.org/sqlite/vec, blank-imported in
-- vectorstore). rowid is keyed to the parent table's PK (symbol_id /
-- chunk_id) so a top-K hit maps 1:1 back to the BLOB row.
CREATE VIRTUAL TABLE IF NOT EXISTS vec_symbols USING vec0(
    rowid     INTEGER PRIMARY KEY,
    embedding float[768]
);
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
    rowid     INTEGER PRIMARY KEY,
    embedding float[768]
);
```

Note: the migration runner (`store.migrate`, `store.go:339-428`) embeds `migrations/*.sql` via `//go:embed` at `store.go:22-23` and tracks applied migrations in `index_meta`. Verify the `//go:embed` pattern picks up `042_vec0_tables.sql` automatically (it should - it's a glob). If the embed is an explicit file list (not a glob), add `042_vec0_tables.sql` to it.

- [ ] **Step 4: Confirm the vec subpackage is importable**

Run: `go build ./internal/mnemonic/store/...`
Expected: PASS. If `modernc.org/sqlite/vec` is not found, the subpackage path may differ at v1.59.0 - check the module's package layout and adjust (it should be `modernc.org/sqlite/vec`).

- [ ] **Step 5: Run the migration tests**

Run: `go test ./internal/mnemonic/store/ -run 'TestVec0' -v`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/store/migrations/042_vec0_tables.sql skillgrid-cli/internal/mnemonic/store/vec0_migration_test.go
git commit -m "feat(mnemonic): add vec0 virtual tables (migration 042)

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 2
adr: 0009
"
```

---

### Task 3: Build the vectorstore package (search + upsert + delete)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/vectorstore/store.go`
- Test: `skillgrid-cli/internal/mnemonic/vectorstore/store_test.go`

**Interfaces:**
- Consumes: Task 2's `vec_symbols` / `vec_chunks` tables; `memory.EncodeVector`/`memory.DecodeVector`/`memory.Vector` (`skillgrid-cli/internal/mnemonic/memory/embedding.go`).
- Produces (exact signatures):
  - `func TableExists(ctx context.Context, db *sql.DB, table string) (bool, error)`
  - `func Count(ctx context.Context, db *sql.DB, table string) (int, error)`
  - `func UpsertSymbol(ctx context.Context, tx *sql.Tx, symbolID int64, model string, dim int, vecBytes []byte) error`
  - `func UpsertChunk(ctx context.Context, tx *sql.Tx, chunkID int64, model string, dim int, vecBytes []byte) error`
  - `func DeleteSymbols(ctx context.Context, tx *sql.Tx) error`
  - `func DeleteChunks(ctx context.Context, tx *sql.Tx) error`
  - `func SearchSymbols(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]int64, error)`
  - `func SearchChunks(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]int64, error)`

**SATISFIES:** `vectorstore-top-k` + `vectorstore-mirrors-blob` + `maxopenconns-one` (in `acceptance.feature`).

- [ ] **Step 1: Write the failing tests**

Create `skillgrid-cli/internal/mnemonic/vectorstore/store_test.go` with five tests:
- `TestTableExists` - asserts `vec_symbols`/`vec_chunks` present, `vec_nonexistent` absent.
- `TestUpsertAndSearchSymbols` - inserts 3 orthogonal 768-d basis vectors (component `hot` = 1) at symbol_id 1,2,3; queries for basis-1 and asserts top-1 is symbol_id 2 (hot index 1); queries top-3 and asserts the matching basis id is first.
- `TestDeleteSymbols` - inserts 3, asserts `Count`==3, deletes, asserts `Count`==0.
- `TestMaxOpenConnsOne` - asserts `db.Stats().MaxOpenConns==1`, then insert + search under that invariant (the spike's viant-deadlock guard).
- A test-only `encodeFloat32s` helper (little-endian float32, same layout as `memory.EncodeVector`).

Use a `mustOpen(t)` helper that calls `store.Open(t.TempDir(), "proj-vstore")` and registers `t.Cleanup(s.Close)`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/mnemonic/vectorstore/ -v`
Expected: FAIL - `undefined: TableExists`, `undefined: UpsertSymbol`, etc. (the package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `skillgrid-cli/internal/mnemonic/vectorstore/store.go`:
- Package doc: durable in-SQL vector path (ADR-0009, option G); wraps the vec0 tables from migration 042.
- Blank import `_ "modernc.org/sqlite/vec"` (registers the vec0 vtab + `vec_distance_cosine` + `vec_f32` with the modernc driver).
- `const dim = 768`.
- `TableExists` / `Count` via `sqlite_master` / `SELECT COUNT(*) FROM <table>`.
- `UpsertSymbol`/`UpsertChunk` delegate to a private `upsert(ctx, tx, table, id, vecBytes)` that runs `INSERT INTO <table>(rowid, embedding) VALUES (?, vec_f32(?)) ON CONFLICT(rowid) DO UPDATE SET embedding = excluded.embedding`.
- `DeleteSymbols`/`DeleteChunks` run `DELETE FROM <table>`.
- `SearchSymbols`/`SearchChunks` delegate to a private `search(ctx, db, table, queryVec, limit)` that runs `SELECT rowid FROM <table> ORDER BY vec_distance_cosine(embedding, vec_f32(?)) LIMIT ?` with the query vector encoded as a little-endian float32 BLOB (via a local `encodeF32` using `encoding/binary`).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/mnemonic/vectorstore/ -v`
Expected: PASS. If `vec_f32(?)` with a BLOB parameter fails (some versions require the string form), switch `upsert` and `search` to the string form: a `vecLiteral([]float32)` helper that formats the float32s as `[f,f,f]` (the spike's `g.go` used the string form for both insert and query). Verify against the spike if needed.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/vectorstore/
git commit -m "feat(mnemonic): add vectorstore package (vec0 search + upsert + delete)

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 3
adr: 0009
"
```

---

### Task 4: Wire the indexer dual-write

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go:1083-1088` (model-swap: add vec table clears alongside the BLOB deletes)
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go:1136-1147` (symbol BLOB upsert: add vec upsert after)
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go:1204-1213` (chunk BLOB upsert: add vec upsert after)
- Test: `skillgrid-cli/internal/mnemonic/codeindex/indexer_vec_dualwrite_test.go` (new)

**Interfaces:**
- Consumes: Task 3's `vectorstore.UpsertSymbol`, `vectorstore.UpsertChunk`, `vectorstore.DeleteSymbols`, `vectorstore.DeleteChunks`.
- Produces: after `embedPass` on a fresh store with the embedder active, `vec_symbols`/`vec_chunks` row counts == `embeddings`/`chunk_embeddings` row counts, and per-ID vectors agree.

**SATISFIES:** `vectorstore-mirrors-blob` + `indexer-dual-write` (in `acceptance.feature`).

- [ ] **Step 1: Write the failing dual-write test**

Create `skillgrid-cli/internal/mnemonic/codeindex/indexer_vec_dualwrite_test.go`. Reuse the existing indexer test harness from `indexer_test.go` (a temp store + a deterministic embedder that returns fixed 768-d vectors). The test seeds symbols + chunks, runs `embedPass` in a transaction, commits, then asserts `vectorstore.Count(ctx, db, "vec_symbols")` == `SELECT COUNT(*) FROM embeddings` and `vectorstore.Count(ctx, db, "vec_chunks")` == `SELECT COUNT(*) FROM chunk_embeddings`. Read `indexer_test.go` first to copy the exact harness constructor + embedPass invocation pattern (do not invent a new harness).

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mnemonic/codeindex/ -run 'TestEmbedPassDualWritesVecTables' -v`
Expected: FAIL - `vec_symbols count=0, embeddings count=N` (the dual-write isn't wired yet).

- [ ] **Step 3: Wire the dual-write**

In `indexer.go`, in the model-swap block (after the two BLOB `DELETE`s at `indexer.go:1083-1088`), add `vectorstore.DeleteSymbols(ctx, tx)` and `vectorstore.DeleteChunks(ctx, tx)`. After the symbol BLOB upsert (`indexer.go:1136-1147`), add `vectorstore.UpsertSymbol(ctx, tx, symID, model, dim, blob)`. After the chunk BLOB upsert (`indexer.go:1204-1213`), add `vectorstore.UpsertChunk(ctx, tx, chunkID, model, dim, blob)`. Add the import `"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/vectorstore"`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/mnemonic/codeindex/ -run 'TestEmbedPassDualWritesVecTables' -v`
Expected: PASS.

- [ ] **Step 5: Run the full codeindex suite (no regression)**

Run: `go test ./internal/mnemonic/codeindex/...`
Expected: PASS (the dual-write must not break the existing indexer tests - they may not have the embedder active, in which case the vec tables stay empty, which is fine).

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/codeindex/
git commit -m "feat(mnemonic): indexer dual-writes vec tables in embedPass

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 4
adr: 0009
"
```

---

### Task 5: Exactness check (brute-force ground truth vs vec top-K)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/vectorstore/exact.go`
- Test: `skillgrid-cli/internal/mnemonic/vectorstore/exact_test.go`

**Interfaces:**
- Consumes: Task 3's `SearchSymbols`/`SearchChunks`; the BLOB tables (`embeddings`, `chunk_embeddings`) via the store's `*sql.DB`; `memory.DecodeVector`/`memory.CosineSimilarity`.
- Produces:
  - `type ExactnessResult struct { Table string; QueryDim int; Limit int; Agree bool; Disagreements []Disagreement }`
  - `type Disagreement struct { Rank int; VecID int64; BlobID int64; VecSim, BlobSim float64 }`
  - `func ExactnessCheck(ctx context.Context, db *sql.DB, table string, queryVec []float32, limit int) (*ExactnessResult, error)` - `table` is `vec_symbols` or `vec_chunks`; the BLOB table is derived (`embeddings` / `chunk_embeddings`).

**SATISFIES:** `exactness-check-agreement` + `exactness-check-disagreement` (in `acceptance.feature`).

- [ ] **Step 1: Write the failing tests**

Create `skillgrid-cli/internal/mnemonic/vectorstore/exact_test.go` with two tests:
- `TestExactnessCheckAgreement` - writes 3 orthogonal basis vectors to BOTH the vec table and the BLOB table (source of truth); queries basis-0; asserts `ExactnessCheck.Agree == true`.
- `TestExactnessCheckDisagreement` - writes basis-0 to the vec table at id 1 but basis-1 to the BLOB table at id 1 (deliberate drift); queries basis-1; asserts `Agree == false` and at least 1 disagreement is reported.

Reuse the `mustOpen(t)` + `encodeFloat32s` + `makeVec(t, hot)` helpers from `store_test.go`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/mnemonic/vectorstore/ -run 'TestExactnessCheck' -v`
Expected: FAIL - `undefined: ExactnessCheck`.

- [ ] **Step 3: Write the implementation**

Create `skillgrid-cli/internal/mnemonic/vectorstore/exact.go`:
- `blobTableFor(vecTable)` maps `vec_symbols` to `embeddings`, `vec_chunks` to `chunk_embeddings`.
- `idColumnFor(vecTable)` maps `vec_symbols` to `symbol_id`, `vec_chunks` to `chunk_id`.
- `ExactnessCheck`: (1) run `search` on the vec table for the top-K (candidate); (2) scan the BLOB table, `memory.DecodeVector` each row, `memory.CosineSimilarity` against the query, sort descending by sim then ascending by id, take top-K (ground truth); (3) compare per rank - if any rank's vec ID differs from the BLOB ID, set `Agree=false` and append a `Disagreement` (best-effort decode of the vec vector for `VecSim`).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/mnemonic/vectorstore/ -run 'TestExactnessCheck' -v`
Expected: PASS (2 tests: Agreement, Disagreement).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/vectorstore/
git commit -m "feat(mnemonic): add vectorstore exactness check (brute-force ground truth)

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 5
adr: 0009
"
```

---

### Task 6: Wire the durable path behind the flag

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/hybrid/durable.go`
- Modify: `skillgrid-cli/internal/mnemonic/hybrid/rank.go:347-359` (`vectorLeg` symbol scoring) and `rank.go:509-519` (`chunkVectorLeg` chunk scoring) - consult the durable path first, fall back to the in-memory cache.
- Test: `skillgrid-cli/internal/mnemonic/hybrid/durable_test.go`

**Interfaces:**
- Consumes: Task 3's `vectorstore.SearchSymbols`/`SearchChunks`/`TableExists`/`Count`; the existing in-memory `vecCache` (the hot path, unchanged); the query vector (`memory.Vector`).
- Produces:
  - `func DurableEnabled() bool` - true when `MNEMONIC_VECTOR_DB` is set to `1`/`true`/`yes`/`on` (mirrors `memory.EmbeddingEnabled`'s parsing).
  - `func durableSymbolHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, bool, error)` - returns the durable top-K as `vectorHit`s (the same shape the in-memory leg produces) plus `ok=true` if the durable path was used, `ok=false` if it degraded to the in-memory path (the caller then runs the existing in-memory scoring).
  - `func durableChunkHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, bool, error)` - analogous for chunks.

**SATISFIES:** `durable-path-equivalence` + `flag-off-unchanged` + `empty-table-degrade` (in `acceptance.feature`).

- [ ] **Step 1: Write the failing tests**

Create `skillgrid-cli/internal/mnemonic/hybrid/durable_test.go` with three tests:
- `TestDurableFlagRouting` - asserts `DurableEnabled()` is false with the env unset, true with `MNEMONIC_VECTOR_DB=1`. Use a `setVectorDBEnv(t, on)` helper that sets/unsets the var and `t.Cleanup` unsets it.
- `TestDurablePathEquivalence` (the backstop) - with the flag on, populate both the vec table and the in-memory cache (via the indexer dual-write or direct upsert) with the same vectors; assert the durable leg's top-K IDs equal the in-memory leg's top-K IDs for the same query.
- `TestDurableEmptyTableDegrade` - with the flag on but an empty vec table, assert `durableSymbolHits` returns `ok=false` (degrade to in-memory), no error.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/mnemonic/hybrid/ -run 'TestDurable' -v`
Expected: FAIL - `undefined: DurableEnabled`, `undefined: durableSymbolHits`.

- [ ] **Step 3: Write the implementation**

Create `skillgrid-cli/internal/mnemonic/hybrid/durable.go`:
- `DurableEnabled()` mirrors `memory.EmbeddingEnabled`'s env parsing for `MNEMONIC_VECTOR_DB`.
- `durableSymbolHits`: if not `DurableEnabled()`, return `nil, false, nil`. Check `vectorstore.Count(ctx, db, "vec_symbols")` - if 0, return `nil, false, nil` (empty-table degrade). Otherwise `vectorstore.SearchSymbols(ctx, db, qVec.Data, limit)`, map each id to a `vectorHit{ID: "sym:<id>", Sim: ...}` (the sim is best-effort: decode the vec vector and cosine, or leave 0 since the caller re-scores the metadata join). Return `hits, true, nil`.
- `durableChunkHits`: analogous for `vec_chunks` / `SearchChunks` / `chunk:<id>` IDs.

In `rank.go`, at the top of `vectorLeg`'s scoring (before the in-memory `loadVectorCache` loop at `rank.go:347-359`), call `durableSymbolHits`; if `ok`, use those hits (then run the existing metadata join at `rank.go:367-379`) and return. If not `ok`, fall through to the existing in-memory path (unchanged). Do the same in `chunkVectorLeg` (before `rank.go:509-519`).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/mnemonic/hybrid/ -run 'TestDurable' -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Run the full hybrid suite (no regression on the hot path)**

Run: `go test ./internal/mnemonic/hybrid/...`
Expected: PASS (with the flag off - the default - the in-memory path is byte-identical, so all existing hybrid tests are unaffected).

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/hybrid/
git commit -m "feat(mnemonic): route durable semantic leg through vectorstore behind flag

[skillgrid-context]
change: 2026-09-24-mnemonic-vector-db
task: 6
adr: 0009
"
```

---

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 0 Important, 2 Minor (deferred: bulk re-index batch/BLOB path for 100K-scale stores is a follow-up per the spike's constraint #2 - the dual-write covers the current ~20K incremental scale; cross-project `all_projects` durable search is out of scope, ADR-0006 revisit criteria #4)
- Reviewed: 2026-09-24

## Execution Handoff

Blueprint has 6 tasks worth of work (3+ tasks) - invoke `skillgrid:slicing` to break it into vertical tracer-bullet tickets with dependency edges and execution waves. Slicing produces `tasks.md` alongside the blueprint.

**Execution options:**

1. **Subagent-Driven (recommended)** - dispatch a fresh subagent per task (or per ticket, if sliced), review between tasks, fast iteration.
2. **Inline Execution** - execute tasks in this session using `skillgrid:simple-execution`, batch execution with checkpoints.

The change is T3 (risky: dependency bump + one-way migration), so the final review should escalate to `skillgrid:parallel-code-review` (multi-reviewer fan-out) rather than the lightweight two-axis pass.
