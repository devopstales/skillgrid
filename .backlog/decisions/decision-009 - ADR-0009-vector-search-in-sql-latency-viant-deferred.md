---
id: decision-009
title: "ADR-0009 Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed"
date: "2026-09-24"
status: "accepted"
---

Source: `.skillgrid/artifacts/04-adr-0009-vector-search-in-sql-latency-viant-deferred.md`

# Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed

---
status: "accepted"
supersedes: none
amends: 0006
date: 2026-09-24
---

## Context and Problem Statement

ADR-0006 (in force) chose in-memory brute-force cosine (option A) for the
semantic leg at the current ~20K-vector scale, and named **option G**
(`modernc.org/sqlite/vec`, the modernc-native sqlite-vec subpackage) as the
default revisit path when a single store crosses ~100K vectors. ADR-0006's
revisit analysis cited a **~40 ms at 100K vectors** figure (from a chromem-go
in-memory brute-force benchmark) as the latency expectation at the migration
threshold.

Before committing to the revisit path, a comparison prototype
(`.skillgrid/specs/2026-09-24-mnemonic-vector-db/prototypes/001-cgo-free-vector-db/`)
tested the two cgo-free non-memory candidates head-to-head at 768-dim /
100K vectors: **G** (`modernc.org/sqlite/vec`) and **viant**
(`viant/sqlite-vec`, the only cgo-free non-memory option with a real ANN
index — cover tree — on the same `modernc.org/sqlite` driver).

The prototype produced two findings that change the revisit analysis:

1. **G's in-SQL top-K latency at 100K is ~6.9 s median, not ~40 ms.** The
   ~40 ms figure in ADR-0006's revisit analysis was for **in-memory Go
   cosine** (the chromem-go benchmark: 100K vectors, 2020 i5, 40 ms). The
   in-SQL `ORDER BY vec_distance_cosine(embedding, vec_f32(?)) LIMIT k` path
   is ~170× slower at the same scale. At the current ~20K scale, G's in-SQL
   top-K is ~1.4 s (linear scaling).
2. **viant is not adoptable yet.** All four released versions (v0.1.0–v0.3.0)
   and `main` fail to build (a packaging bug: they pin `viant/vec
   v0.1.1-0.20240628...` but call an exported
   `CosineDistanceWithMagnitude` that only exists in `viant/vec v0.2.3`).
   More importantly, the lazy-index query path **deadlocks**: `ensureIndex`
   re-enters `db.Exec` from inside the vtab `Filter` callback, which blocks
   with `SetMaxOpenConns(1)` (the documented requirement for vtab
   registration to be visible). Insert works (9,933 vectors/sec, 21× faster
   than G's string-based 464/sec), but the query deadlocks before returning
   results. viant's recall@10 is therefore unmeasurable.

This ADR corrects the latency figure, records the viant deferral, and
confirms the revisit path. It does **not** supersede ADR-0006 — the current
decision (in-memory option A at ~20K) is unchanged. It amends ADR-0006's
revisit criteria and measurement appendix.

## Considered Options

Same option space as ADR-0006 (A in-memory, B/B′/C/G sqlite-vec variants, D
Go-native HNSW, E sqlite-vss, F vectorlite). The prototype narrowed the
cgo-free non-memory candidates to:

- **G. `modernc.org/sqlite/vec`** — same driver, mature (sqlite-vec v0.1.9,
  gorse-validated), brute-force in-SQL. **Viable.**
- **viant. `viant/sqlite-vec`** — same driver, cover-tree ANN. **Not
  viable yet** (packaging bug + query deadlock).

The other options (B′ ncruces, D Go-native HNSW, E, F) are unchanged from
ADR-0006 and not re-evaluated here.

## Decision Outcome

1. **ADR-0006's current decision is unchanged.** In-memory brute-force
   (option A) remains the semantic-leg implementation at the current ~20K
   scale. The ~28 ms warm cosine + ~55 ms FTS/signal figure stands.

2. **The ~40 ms at 100K figure in ADR-0006's revisit analysis is corrected
   to ~6.9 s for the in-SQL G path.** The ~40 ms was in-memory Go cosine
   (chromem-go); the in-SQL `ORDER BY vec_distance_cosine` is ~170× slower.
   The revisit-trigger analysis must distinguish the two paths:
   - **In-memory cosine (hot path):** ~40 ms at 100K, ~8 ms at 20K. Used by
     the existing `hybrid/vectorcache.go`. Fast, but the vectors live in
     process RAM (not durable across restarts without re-decode).
   - **In-SQL G (durable path):** ~6.9 s at 100K, ~1.4 s at 20K. Used when
     the vectors must persist in the SQLite file (survive restart, shared
     across processes, no re-decode). Slow, but durable.

   The design question at the revisit threshold is **which path serves which
   query**, not a single latency number. Session-inject's selection
   (locked: BM25 + semantic, RRF-fused, one-shot at resume) is a candidate
   for the durable in-SQL path — ~1.4 s at the current scale is acceptable
   for a one-shot resume, not for a per-tool-call hot path.

3. **viant is deferred, not rejected.** It is the ANN upgrade path (cover
   tree, same driver) but is blocked by (a) the packaging bug (fixable with
   `go get github.com/viant/vec@v0.2.3`) and (b) the query deadlock
   (requires a viant fix or a non-standard connection setup). Re-prototype when
   viant fixes the `ensureIndex` re-entrancy or documents a raw-connection
   usage pattern. At that point, measure recall@10 vs G's brute-force to
   decide whether the ANN index is worth the integration cost.

4. **G remains the default revisit path** (confirmed, not changed). When a
   single store crosses the threshold, adopt G: blank-import
   `modernc.org/sqlite/vec`, bump `modernc.org/sqlite` v1.45.0 → ≥v1.59.0,
   add the `vec0` virtual tables via migration, point the semantic leg at
   `vec_distance_cosine`. B′ (ncruces) remains the fallback only if G proves
   insufficient.

### Consequences

- Good, because the revisit path is now grounded in a measured latency
  (~6.9 s at 100K in-SQL) rather than an extrapolation from an in-memory
  benchmark. The design can make an informed hot-path vs durable-path
  decision at the threshold.
- Good, because viant's deferral is evidence-based (deadlock + packaging
  bug), not a gut rejection. The ANN upgrade path is preserved for when
  viant is ready.
- Bad, because G's in-SQL latency (~6.9 s at 100K) is far higher than the
  ~40 ms originally cited. Any design that assumed in-SQL G would be fast
  at 100K needs to be re-examined. The in-memory path remains the hot-path
  option; G is the durable option.
- Bad, because the modernc v1.45.0 → v1.59.0 bump (required for G) has not
  yet been validated against the existing 39 migrations, WAL-retry logic,
  and store pooling. The prototype used a fresh DB. This validation is a
  prerequisite for the real build.

### Revisit criteria (amended from ADR-0006)

Revisit this ADR (or write a superseding ADR) when **any** of:
1. A single project store crosses ~100K vectors **and** the durable
   in-SQL path's ~6.9 s latency is unacceptable for the serving query —
   at which point viant (if the deadlock is fixed) or B′ (ncruces) becomes
   the path.
2. viant fixes the `ensureIndex` query deadlock — re-prototype for the ANN
   recall@10 measurement.
3. The modernc v1.45.0 → v1.59.0 bump is validated against the existing
   store (39 migrations, WAL-retry, pooling) — at which point G adoption
   is unblocked.
4. Cross-project (`all_projects`) semantic search is implemented (unchanged
   from ADR-0006).

---

## Measurement appendix (2026-09-24, prototype, fresh DB, 768-dim)

| Metric | G (`modernc.org/sqlite/vec`) | viant (`viant/sqlite-vec`) |
|---|---|---|
| Build | ✓ (modernc v1.59.0) | ✓ (viant/vec v0.2.3 bump) |
| cgo-free, same driver | ✓ | ✓ |
| Insert 100K × 768-d | 3 m 35 s (464/s) | 10 s (9,933/s) |
| Top-K=10 query at 100K | **6.9 s median** (min 6.7, max 8.2) | **deadlock** (no result) |
| Recall@10 | exact (brute-force, deterministic) | unmeasurable (deadlock) |
| vec_version | v0.1.9 | n/a (cover tree) |

### Comparison with ADR-0006's in-memory figures

| Path | 20K vectors | 100K vectors |
|---|---|---|
| In-memory Go cosine (ADR-0006 option A, `hybrid/vectorcache.go`) | ~8 ms (warm) | ~40 ms (chromem-go benchmark) |
| In-SQL G `vec_distance_cosine` (this ADR) | ~1.4 s (linear extrapolation) | **~6.9 s (measured)** |
| In-SQL G cold insert (string-based `vec_f32`) | — | 3 m 35 s for 100K |

The in-memory path is the hot path; the in-SQL G path is the durable path.
They serve different query classes and are not directly substitutable.
