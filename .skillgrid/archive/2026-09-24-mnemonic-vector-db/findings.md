# Findings — 2026-09-24-mnemonic-vector-db

## Spike: 001-cgo-free-vector-db

- **Verdict:** PARTIAL ⚠
- **What we learned:**
  - **G (`modernc.org/sqlite/vec`) works** — cgo-free, same `modernc.org/sqlite` driver (bump v1.45.0 → v1.59.0), `vec0` virtual tables, `vec_distance_cosine` top-K. Mature (sqlite-vec v0.1.9, gorse-validated).
  - **G's in-SQL top-K latency at 100K × 768-dim is ~6.9s median** — NOT the ~40 ms the ADR-0006 research predicted. That figure was for **in-memory Go cosine** (chromem-go benchmark), not in-SQL `ORDER BY vec_distance_cosine`. At ~20K (our current scale) it's ~1.4s (linear).
  - **viant (`viant/sqlite-vec`) has two blockers:** (1) a packaging bug — all 4 released versions fail to build because they pin `viant/vec v0.1.1-0.2024...` but call an exported `CosineDistanceWithMagnitude` that only exists in v0.2.3 (fix: `go get github.com/viant/vec@v0.2.3`); (2) a **deadlock** in the lazy-index query path — `ensureIndex` re-enters `db.Exec` from inside the vtab `Filter` callback, which blocks with `SetMaxOpenConns(1)` (the required setting for vtab registration). Insert works (9933/s, 21× faster than G's 464/s), but **query deadlocks** before returning results.
  - **viant's recall@10 is unmeasurable** because of the deadlock.
  - **G insert is string-based** (`vec_f32('[...]')`) — 464/s at 768-dim. BLOB-based or batched insert needed for bulk re-index.
- **What's liftable:** the G query pattern (blank-import `modernc.org/sqlite/vec`, `vec0` table, `vec_distance_cosine` top-K) as a `vectorstore/` package; the exactness-check methodology (Go cosine scan as ground truth).
- **Constraints for the build:**
  1. G's in-SQL top-K is ~6.9s at 100K, ~1.4s at 20K — acceptable for session-inject (one-shot at resume), NOT for a per-query hot path (use in-memory cosine for hot paths).
  2. G insert is string-based — batch/BLOB path needed for bulk re-index.
  3. viant is not adoptable until the `ensureIndex` deadlock is fixed (or a raw-connection usage pattern is documented).
  4. The modernc v1.45.0 → v1.59.0 bump must be validated against the existing 39 migrations, WAL-retry, and store pooling (spike used a fresh DB).
  5. ADR-0006's ~40 ms figure is in-memory, not in-SQL — the revisit-trigger analysis must distinguish hot-path (in-memory, fast) from durable (in-SQL G, slow).
