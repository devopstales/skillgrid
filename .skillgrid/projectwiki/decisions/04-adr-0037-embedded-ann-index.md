---
type: TechnicalDecision
title: "HNSW embedded ANN index: pure Go (no new dep), M=16, efCon=64, efSearch=96, persisted in SQLite (065), `source`-namespaced node ids (066, symbol/chunk), degrades to brute force"
status: "accepted"
supersedes: none
date: 2026-10-10
---

# ADR-0037: Embedded HNSW index — pure Go, no new dependency
## Context

TICKET-01's recall/latency bench (`TestRecallLatencyBaseline`) measured brute-force cosine top-K at 100k × 768-dim: warm 168ms. The acceptance gate is cold p95 < 50ms, warm < 10ms (briefing R3). IVF (TICKET-04) is the first ANN path and degrades to brute force on corrupt state; HNSW is the 3rd `VectorIndex` adapter that must meet the latency gate at 100k.

The repo's hard constraint is **no new Go dependencies without an ADR** (AGENTS.md § Rules) and **cgo-free** (ADR-0006/0009: `modernc.org/sqlite` is the only SQLite driver; no CGO). The candidate pure-Go HNSW libraries (`github.com/cespare/hnsw`, `github.com/unum-cloud/usearch`) are both new third-party dependencies. A from-scratch HNSW in the `vectorstore` package (stdlib only) avoids the dependency entirely and keeps the cgo-free invariant intact.

The HNSW graph layout (nodes + adjacency + meta) persists in SQLite (`hnsw_nodes` / `hnsw_links` / `hnsw_meta` tables, migration 065 at landing; created at 057, superseded in-place by 066's source-partitioned re-create) so the index survives process restarts (warm-load). The in-memory graph is the hot path; the BLOB table (`embeddings`) remains the source of truth (ADR-0009). A corrupt or missing HNSW table degrades to brute force — never a hard fail (briefing R3, matching the IVF contract in TICKET-04).

## Decision

**Implement HNSW from scratch in `mnemonic/internal/vectorstore/hnsw.go`** (stdlib only: `context`, `database/sql`, `math`, `math/rand`, `sort`, `sync`). No new Go dependency. The `HnswIndex` struct implements the `VectorIndex` interface (Search/Upsert/Remove/Name) as the 3rd adapter alongside `BruteForce` and `IVFIndex`.

Parameters: `M=16` (max connections per node at layers > 0), `efConstruction=64` (candidate beam during insert), `efSearch=96` (candidate beam during query; the recall/latency knob — lower to 64 for sub-10ms warm at scale, trading a few recall points). Level distribution: geometric with `ml=0.5` (expected max level ~log_2(n)). Neighbor selection uses the standard HNSW heuristic (alpha=1.0 diversity pruning at insert time).

The graph persists as three tables (migration 065 at landing; 057→066 re-created with the `source` partition): `hnsw_nodes (source, node_id, level, vector BLOB)`, `hnsw_links (source, node_id, layer, neighbor_id)`, `hnsw_meta (key, value)` (top node, node count, dim — not yet written or read as of this change's review; see review.md S3). Warm-load reads the tables into the in-memory `map[int64]*node` on first Search (lazy, idempotent). `Upsert` dual-writes the BLOB row + the graph rows in one short tx (MaxOpenConns=1 safe). `Remove` tombstones the node and deletes its links.

The hybrid vector cache (`hybrid/vectorcache.go`) is wired to route the durable leg through `HnswIndex` when `MNEMONIC_VECTOR_DB=1` and the HNSW tables are non-empty (matching the existing `DurableEnabled` flag pattern in `durable.go`). The in-memory cosine cache (default path) is unchanged.

## Consequences

- **Positive:** No new dependency — the cgo-free, single-binary invariant (ADR-0006/0009) is preserved. The HNSW graph is inspectable in SQLite (debuggable with `sqlite3`). The 3rd `VectorIndex` adapter justifies the interface seam (blueprint Task 4 deletion test). Warm-load from SQLite means the index survives restarts without a re-embed.
- **Negative:** The from-scratch HNSW is ~400 lines of Go with a subtle insert/search algorithm (neighbor pruning, level distribution, descent). Build time at 100k is ~100s (1ms/insert × 100k) — acceptable for an offline index pass, not for interactive use. The recall@10 gate (>= 0.95 at 100k) depends on the neighbor-selection heuristic being correct; the test corpus is synthetic gaussian (same as IVF), not real embeddings.
- **Migration:** none beyond the additive graph tables (065 at landing; 066 re-creates them with the `source` partition — derived data, lossless drop), all `CREATE TABLE IF NOT EXISTS`. The compat VIEW `observations` + INSTEAD-OF triggers (TICKET-02/03) are untouched.
- **One-way gate:** This ADR is accepted before the HNSW code lands. If the from-scratch HNSW proves insufficient (recall < 0.95 or warm > 10ms at 100k), the decision is to ship IVF-only (the door check in the blueprint) and revisit HNSW with a library in a superseding ADR.
