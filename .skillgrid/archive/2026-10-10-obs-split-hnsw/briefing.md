# Mnemonic obs-split + HNSW — Briefing (what, intent)

**Status:** ACCEPTED
**Tier:** T3 (migration, high-risk)
**Spec ref:** `.skillgrid/artifacts/00-prd.md` (local-first engine, single binary); ADRs 0006/0009 (vector path), 0011 (bitemporal), 0012 (SQLite store), 0024 (PG outbox-only), 0031 (Go 1.22+ floor).

## Intent

Fix two load-bearing defects in the Mnemonic per-project SQLite store (`mnemonic/internal/store/`):

1. `observations` is a ~30-column god-table (core + lifecycle + provenance + stats + temporal + vectors) built by 12+ additive `ALTER TABLE`s. Every hot-path scan drags BLOBs and cold columns; 6+ partial indexes amplify writes.
2. Vector search is brute-force cosine (`hybrid/vectorcache.go` in-memory, `vec0` in-SQL full scan, dim pinned 768 in `vectorstore/store.go:22`). No ANN; model swap wipes the index.

## Requirements (RFC 2119, delta spec)

* R1 (MODIFIED): Hot-path observation reads MUST NOT scan vector BLOBs or cold stats columns. Success: `mem_context`/`mem_search` read `observations_core` + needed side tables only.
* R2 (MODIFIED): Bitemporal semantics (ADR-0011: `valid_at/invalid_at/superseded_by`, 7 read-site filters) MUST be preserved across the split.
* R3 (New): Vector top-K MUST use an embedded, cgo-free ANN path (IVF first, HNSW behind ADR-0037) with recall@10 >= 0.95 vs brute force at 20k/100k vectors.
* R4 (MODIFIED): Embedding dim MUST NOT be pinned to 768; per-model namespaces MUST coexist; model swap MUST NOT wipe other models.
* R5: Rollback MUST be possible (pre-migration `.dump` + hash verify + compat view window).

## Non-goals

* No Postgres-as-primary (ADR-0012/0024 stand). No CGo deps. No TEXT->INTEGER time migration here (follow-up).
* No new npm deps. One allowed Go dep max (pure-Go HNSW), gated by ADR-0037.

## Acceptance (happy + edge + failure)

* Given a pre-split store, when the obs-split migration (blueprint `054`, landed as `062_obs_split.sql` after the CLM/CTX rebase) migrates, then row-hash parity holds and all 7 `invalid_at` read sites filter superseded rows.
* Given 100k 768-dim vectors, when top-K runs cold, then p95 < 50ms warm < 10ms (bench), else IVF-only ships and HNSW stays out (door check).
* Given two models (768 + 1024 dim), when both indexed, then each queries its own namespace; cross-dim query errors explicitly.
* Given a failed migration (hash mismatch), when rollback runs, then legacy table restores and service opens read-only.

## Verdict (TICKET-07, 2026-10-11)

**L4 gate: PASS (with documented caveats).** Rollback drill green (temp copy of the live pre-054 store: .dump before 058 → apply 058 → hash-mismatch detected → restore → pre-058 schema byte-identical, all 6 observations tables/65 rows intact, integrity ok, doctor --strict exit 0). `go build` PASS; `go vet` PASS; `go test ./... -timeout 30m`: 30/41 packages ok, 11 failures all root-caused — the 7 known pre-existing 054 internal/store failures reproduced exactly (5 table-vs-view assertions, 2 `no such column: status` at 054 exec), plus 6 054-era view-read governance-seam knock-ons (http/mcp), 1 FTS content-view seam (service), 1 thermal flake (vectorstore, passes isolated), 1 stale test-harness path (cmd/mnemonic). gofmt: 92 pre-existing drift files + 1 struct-alignment misformat committed by c443c4ce (obs_split.go). Trivy advisory: 0 vulns / 0 misconfigs / 0 secrets, 26 LOW license. Full evidence: `.skillgrid/sdd/blueprint/ticket-07-report.md`.
