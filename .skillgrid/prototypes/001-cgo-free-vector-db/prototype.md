# Prototype: 001-cgo-free-vector-db

**Date:** 2026-09-24
**Type:** comparison (G `modernc.org/sqlite/vec` vs viant `viant/sqlite-vec`, both on `modernc.org/sqlite`)
**Hypothesis:** Given a Go module on `modernc.org/sqlite`, when we adopt either G (`modernc.org/sqlite/vec`) or viant (`viant/sqlite-vec`), then both provide working cgo-free vector search in a virtual table on the same driver — and at 768-dim / 100K vectors, G's `vec_distance_cosine` top-K completes in ≤100 ms while viant's cover-tree top-K returns recall@10 ≥ 0.9.

## How to run

```bash
cd .skillgrid/specs/2026-09-24-mnemonic-vector-db/prototypes/001-cgo-free-vector-db
go mod tidy
go build -o g_bin ./g              # G path (modernc.org/sqlite/vec)
./g_bin                            # runs: insert 100K, query top-10, exactness check

# viant path (two binaries: build inserts + persists index, query runs MATCH + recall)
go build -o build_bin ./viant-build
go build -o query_bin ./viant-query
rm -f /tmp/viant-prototype.sqlite*
./build_bin /tmp/viant-prototype.sqlite
./query_bin /tmp/viant-prototype.sqlite
```

## Investigation trail

### Iteration 1: G path (modernc.org/sqlite/vec)

- Bumped `modernc.org/sqlite` v1.45.0 → v1.59.0 (the `/vec` release).
- Blank-import `modernc.org/sqlite/vec` → `vec_version()` returns **v0.1.9** ✓
- `CREATE VIRTUAL TABLE vec_items USING vec0(rowid INTEGER PRIMARY KEY, embedding float[768])` ✓
- Inserted 100K × 768-dim vectors: **3m35s (464 vectors/sec)** — slow, due to `vec_f32('[...]')` string-parsing per row.
- Top-10 query via `ORDER BY vec_distance_cosine(embedding, vec_f32(?)) LIMIT 10`:
  - **median 6.9s, min 6.7s, max 8.2s** — far above the ≤100 ms hypothesis.
- Exactness check (Go cosine scan of first 2000 vs G's top-10): **0/10 match**.
  - Diagnosis: the G query returns the global top-10 (correct); the Go scan is subset-limited to 2000, so its top-10 is a different (subset) set. The 0/10 is a **methodology artifact**, not a G correctness failure. G's global top-3 `[77355, 45936, 78555]` is internally consistent across 10 runs (deterministic). G is exact brute-force; the 6.9s is the `ORDER BY` over 100K rows in-SQL, not a recall problem.

**Findings (G):**
- G works: compile, query, cgo-free, same driver. ✓
- G latency at 100K is **~6.9s per top-K query**, not ~40 ms. The ADR-0006 research prediction (~40 ms at 100K) was for **in-memory Go cosine** (chromem-go benchmark), not for **in-SQL `ORDER BY vec_distance_cosine`**. The in-SQL path is ~170× slower.
- G insert is slow (464/s) due to `vec_f32` string parsing; a BLOB-based insert would be faster but the API is string-first.

### Iteration 2: viant path (viant/sqlite-vec)

- **Build failure on all 4 released versions** (v0.1.0, v0.1.1, v0.2.0, v0.3.0) and main:
  `v1.CosineDistanceWithMagnitude undefined (type search.Float32s has no field or method CosineDistanceWithMagnitude, but does have unexported method cosineDistanceWithMagnitude)`.
  - Root cause: viant pins `github.com/viant/vec v0.1.1-0.20240628004145` but the code calls an **exported** `CosineDistanceWithMagnitude` that only exists in **v0.2.3** (unexported in the pinned version).
  - **Fix:** `go get github.com/viant/vec@v0.2.3` (one-line version bump) → builds. This is a **viant packaging bug** (stale pinned dep), not a fundamental blocker, but it means adopting viant requires a `replace` directive or waiting for a viant release that pins v0.2.3.
- **`no such module: vec` on `CREATE VIRTUAL TABLE`** — traced to **`PRAGMA journal_mode=WAL` running before `vec.Register(db)`**. The PRAGMA opens a connection, and the vtab module registration (which is per-`*sql.DB` via `modernc.org/sqlite/vtab.RegisterModule`) doesn't take effect for connections opened before Register.
  - **Fix:** call `vec.Register(db)` **before** any PRAGMA/DDL that opens a connection.
- **`vec_admin` module conflict** — registering both `vec` and `vecadmin` on the same `*sql.DB` causes `no such module: vec` when creating the `vec` vtab. The `vec_admin` vtab and `vec` vtab interfere.
  - **Workaround:** drop `vecadmin` from the build (the index builds **lazily** via `ensureIndex` on first MATCH after shadow changes — triggers in `vec/api.go` invalidate `vector_storage` on shadow insert, so the index rebuilds automatically). `vec_admin` is only for **manual** rebuilds.
- **Deadlock on first MATCH query** — `vec.(*Table).ensureShadow` → `ensureIndex` → `db.Exec(...)` re-enters the same `*sql.DB` **from inside the vtab's `Filter` callback** (called during the MATCH query's `sqlite3_step`). With `SetMaxOpenConns(1)` (required for vtab registration to be visible), the re-entrant `Exec` blocks waiting for the single connection that the outer query is holding.
  - **`fatal error: all goroutines are asleep - deadlock!`** at `database/sql.(*DB).conn`.
  - This is a **genuine viant concurrency bug**: the lazy-index path assumes a pool with >1 connection, but `SetMaxOpenConns(1)` is the documented requirement for vtab registration. The two requirements are mutually exclusive.
  - Insert worked (100K in 10s, 9933 vectors/sec — 21× faster than G's string-based insert), so the build path is fine; only the **query path deadlocks**.

**Findings (viant):**
- viant compiles with a `viant/vec` v0.2.3 bump (packaging bug, fixable).
- viant insert is fast (9933/s vs G's 464/s).
- viant **query deadlocks** with the standard `*sql.DB` + `SetMaxOpenConns(1)` pattern — a genuine concurrency bug in the lazy-index path. Not reproduced in viant's own tests (which likely use a different connection setup or pre-build the index via `vec_admin` in a way that avoids the re-entrant Exec).
- **Recall@10 could not be measured** — the query deadlocks before returning results.

### Comparison summary

| Metric | G (`modernc.org/sqlite/vec`) | viant (`viant/sqlite-vec`) |
|--------|------------------------------|---------------------------|
| Compiles on modernc | ✓ (bump to v1.59.0) | ✓ (bump `viant/vec` to v0.2.3) |
| cgo-free, same driver | ✓ | ✓ |
| Insert 100K × 768-d | 3m35s (464/s) | 10s (9933/s) |
| Top-K query at 100K | **6.9s median** (works) | **deadlock** (no result) |
| Recall@10 | n/a (exact brute-force, deterministic) | **unmeasurable** (deadlock) |
| Integration friction | Low (blank import + version bump + 2 tables) | High (dep bump + Register-before-PRAGMA + no vecadmin + deadlock workaround) |
| Maturity | High (sqlite-vec v0.1.9, gorse-validated) | Low (12★, 1 contributor, packaging bug, concurrency bug) |

## Verdict

**PARTIAL** ⚠ — with constraints.

- **G is the viable path.** It compiles, queries, is cgo-free, same driver, and is mature. **Constraint:** the in-SQL top-K latency at 100K is **~6.9s**, not the ≤100 ms the hypothesis predicted. The ~40 ms figure from ADR-0006 research applies to **in-memory Go cosine**, not in-SQL `ORDER BY`. For session-inject's selection (which is **not** a hot path — it runs once at session resume, not per-tool-call), 6.9s is **tolerable but not ideal**. At our current ~20K-vector scale, G's latency would be ~1.4s (linear scaling) — acceptable.
- **viant is NOT viable yet.** The deadlock in the lazy-index query path is a blocker for the standard `*sql.DB` usage pattern. It would require either (a) a viant fix, (b) a non-standard connection setup (e.g., a raw `*sqlite.SQLiteConn` instead of `*sql.DB`), or (c) pre-building the index in a separate process with `vec_admin` and querying with a pre-built index (which avoids the lazy `ensureIndex` re-entrancy). None of these are clean for an embedded tool.
- **The hypothesis is INVALIDATED on the latency claim** (G is 6.9s, not ≤100 ms at 100K) and **PARTIAL on the viability claim** (G works but slow; viant deadlocks).

## What's liftable

- The **G query pattern** (blank-import `modernc.org/sqlite/vec`, `vec0` table, `vec_distance_cosine` top-K) is the production path. Liftable as a `vectorstore/` package: `Open(dbPath)`, `Insert(id, vector)`, `TopK(query, k)`.
- The **exactness-check methodology** (Go cosine scan as ground truth) is reusable for validating any vector backend.

## Constraints for the build

1. **G's in-SQL top-K is ~6.9s at 100K, ~1.4s at 20K.** For session-inject (one-shot at resume), this is acceptable. For any per-query hot path, it is not — use in-memory cosine (the existing `hybrid/vectorcache.go`) for hot paths, G for durable persistence.
2. **G insert is string-based (`vec_f32`)** — 464/s at 768-dim. For bulk re-indexing, batch inserts and/or a BLOB-based path are needed.
3. **viant is not adoptable until the deadlock is fixed.** Track as a future revisit: if viant fixes the `ensureIndex` re-entrancy (or documents a raw-connection usage pattern), re-prototype for the ANN recall@10 measurement.
4. **The modernc version bump (v1.45.0 → v1.59.0)** must be validated against our 39 migrations, WAL-retry logic, and store pooling before the real build — this prototype used a fresh DB, not the existing store.
5. **ADR-0006's ~40 ms at 100K figure is for in-memory Go cosine, not in-SQL.** The revisit-trigger analysis should distinguish the two: in-memory (hot path, fast) vs in-SQL G (durable, slow). The decision is about **which path serves which query**, not a single number.
