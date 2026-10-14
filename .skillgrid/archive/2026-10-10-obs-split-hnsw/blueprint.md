# Mnemonic obs-split + HNSW — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how*. The *what* lives in `briefing.md` (same folder) + `.skillgrid/artifacts/00-prd.md`. Decisions: `.skillgrid/artifacts/04-adr-0006-vector-search-in-memory-brute-force.md`, `04-adr-0009-vector-search-in-sql-latency-viant-deferred.md`, `04-adr-0011-observations-are-bitemporal.md`, `04-adr-0012-sqlite-as-second-brain.md`, `04-adr-0024-central-pg-shared-memory-state.md`. New dep needs ADR-0037 (not yet written).

**Status:** PROPOSED

**Tier:** T3

**Build shape:** Tracer thread — thin vertical slice (one table split + one index backend) end-to-end first, then thicken. Why: integration risk (migration + ranking) is the unknown.

**Goal:** Split `observations` vertically and add embedded ANN vector search without breaking reads.

**Architecture:** Vertical partition of `observations` into core + 5 side tables behind a compat `VIEW` (rollback window); `VectorIndex` interface in `vectorstore` with `BruteForce` (current), `IVF` (SQL, no dep), `Hnsw` (pure-Go, ADR-0037); per-model dim namespaces; dual-write in indexer `embedPass`.

**Tech Stack:** Go 1.22+ (per AGENTS.md § Rules, ADR-0031), `modernc.org/sqlite` (cgo-free), FTS5, `vec0` (legacy path), pure-Go HNSW lib TBD (ADR-0037 only).

**Spec:** `./briefing.md`

## Hypothesis

**Claim:** Splitting `observations` + embedded ANN cuts `mem_search` p95 with no read regression.
**Right condition:** parity hash equal, recall@10 >= 0.95, cold <50ms / warm <10ms at 100k, `go test ./mnemonic/...` green.
**Wrong condition:** compat view slower than baseline or recall <0.95.
**Thinnest MVP:** Task 1 bench + Task 2 compat-view parity on a copy.
**Door check:** Task 1 — if store <5k rows and <50ms, ship IVF-only, skip HNSW.

## Terms

None beyond spec. Canonical terms: `.skillgrid/artifacts/01-business-terms.md`, `02-technical-terms.md`.

## Must-Haves (goal-backward verification)

**Truths:**
- `SELECT` via compat view returns row-identical results (incl. `valid_at/invalid_at/superseded_by`, `pinned/expires_at`, `embedding*`) vs pre-split DB.
- `mem_save/search/context` pass with `MaxOpenConns=1` + WAL, no `database is locked` regression.
- `SearchSymbols/Chunks` recall@10 >= 0.95 vs brute force at 20k/100k synthetic 768-dim.
- Two models (768 + 1024 dim) coexist; cross-dim query errors explicitly, no silent wipe.

**Artifacts:**
- `mnemonic/internal/store/migrations/054_obs_split.sql` — split tables + compat view + moved indexes
- `mnemonic/internal/store/migrations/055_hnsw_store.sql` — `ivf_centroids/assignments`, `hnsw_nodes/links/meta`
- `mnemonic/internal/vectorstore/index.go` — `VectorIndex` interface + `BruteForce` adapter
- `mnemonic/internal/vectorstore/ivf.go`, `hnsw.go` — ANN paths
- `mnemonic/internal/memory/observations_compat_test.go` — parity test
- `mnemonic/internal/vectorstore/bench_test.go` — recall + latency bench
- `.skillgrid/artifacts/04-adr-0037-embedded-ann-index.md` — HNSW dep decision (if HNSW proceeds)

**Key links:**
- `memory.Service` -> compat view -> new tables; `grep -rn 'SELECT \* FROM observations'` returns only compat-view def + tests.
- `hybrid/memorysearch.go` -> `vectorstore.Search` -> ANN hit (rowid) -> BLOB row 1:1.
- Indexer `embedPass` dual-writes BLOB + ANN tables in one tx; corrupt ANN degrades to brute force, never fails.

**One-way-door decisions:**
- Rebuild of `observations` (COPY/RENAME) — backup + hash verify required.
- New Go dependency (pure-Go HNSW) — ADR-0037 required.
- Dropping the compat view (later release, not this change).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Migration data loss | Applicable | `.dump` backup + count/hash verify before rename; legacy table kept one release | parity test fails on hash mismatch; rollback test restores legacy |
| ANN ranking manipulation | Applicable | `ef_search`/`nprobe` tuned for recall>=0.95; fallback to brute on ANN miss | recall bench RED if <0.95 |
| New dep supply chain | Applicable | pure-Go only, pinned version, `go.sum` verify, no cgo | `go list -m` allowlist test |
| MCP/HTTP contract | N/A: no tool-shape change, ranking only | — | — |

## Global Constraints

- Go 1.22+ to build (per AGENTS.md § Rules / ADR-0031).
- No new dependencies without an ADR (per AGENTS.md § Rules) — HNSW lib needs ADR-0037 before import.
- Trivy advisory-only, never blocking (per AGENTS.md § Rules).
- Serial development: one change at a time (per AGENTS.md § Rules).
- Conventional commits only, no AI trailers (per AGENTS.md § Rules, commit-msg hook).
- Spec-zone changes commit before code-zone changes (per AGENTS.md § Rules, pre-commit zone guard).
- Single binary, cgo-free invariant (ADR-0006/0009, VERIFIED stack).
- SQLite is the store; PG is outbox-only (ADR-0012/0024).

## File Structure

- `mnemonic/internal/store/migrations/054_obs_split.sql` — NEW: 6 tables + compat view + moved partial indexes
- `mnemonic/internal/store/migrations/055_hnsw_store.sql` — NEW: IVF + HNSW meta tables
- `mnemonic/internal/vectorstore/index.go` — NEW: `VectorIndex` interface
- `mnemonic/internal/vectorstore/ivf.go` — NEW: IVF probe search
- `mnemonic/internal/vectorstore/hnsw.go` — NEW: HNSW impl (behind ADR-0037)
- `mnemonic/internal/vectorstore/store.go:22,52-80,119-145` — MODIFY: unpin dim, namespaced tables, dual-write
- `mnemonic/internal/hybrid/vectorcache.go`, `memorysearch.go` — MODIFY: route via `VectorIndex`, warm-load HNSW
- `mnemonic/internal/memory/*.go` — MODIFY: 7 bitemporal read sites + `SaveWithAction` to side tables
- `mnemonic/internal/vectorstore/bench_test.go` — NEW: latency + recall bench
- `mnemonic/internal/memory/observations_compat_test.go` — NEW: parity test

---

### Task 1: Bench + caller audit (door check)

> Checkpoint: none (read-only).

**Files:**
- Create: `mnemonic/internal/vectorstore/bench_test.go`
- Test: `mnemonic/internal/vectorstore/bench_test.go`

**Interfaces:**
- Consumes: current `vectorstore.SearchSymbols/Chunks`, `hybrid/vectorcache.go`
- Produces: baseline numbers (cold/warm ms at 20k/100k, BLOB-scan vs cosine split) + caller list of `FROM observations`
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 (bench only — no seam built)

**SATISFIES:** R3-measure

- [ ] **Step 1: Write the failing test** — bench asserting recall harness exists + `grep FROM observations` caller list checked in as `observations_callers.txt` (test fails: files absent).
- [ ] **Step 2: Run to verify RED** — `go test ./mnemonic/internal/vectorstore/ -run TestBenchHarnessExists -v` FAILs (no file).
- [ ] **Step 3: Implement** — synthetic 768-dim generator (seeded), brute-force top-K timer, caller audit via `rg -n 'FROM observations' mnemonic/internal/memory`.
- [ ] **Step 4: Run GREEN** — bench prints cold/warm; if <50ms at 100k record `DECISION: IVF-only` in bench output.
- [ ] **Step 5: Commit** — `git add .skillgrid/specs/2026-10-10-obs-split-hnsw/ mnemonic/internal/vectorstore/bench_test.go && git commit -m "test: add vector recall bench and observations caller audit"`

### Task 2: Split observations behind compat view

> **Checkpoint:** ⚠ one-way: rebuild of `observations` (COPY/RENAME). STOP for explicit approval before running on a real store; tests run on copies only.

**Files:**
- Create: `mnemonic/internal/store/migrations/054_obs_split.sql`
- Create: `mnemonic/internal/memory/observations_compat_test.go`

**Interfaces:**
- Consumes: `001_schema.sql` observations def + Task 1 caller list
- Produces: `observations_core`, `observation_state/provenance/stats/temporal/vectors` + `VIEW observations` + moved partial indexes
- Seam: compat view (callers unchanged this task)
- Deletion test: dropping view breaks all 7 read sites — view stays one release
- Adapters: 2 (new tables + legacy `_legacy` kept)

**SATISFIES:** R1, R2, R5

- [ ] **Step 1: Failing test** — open fixture DB, apply `054`, assert row-hash parity (`SELECT` via view == pre-split dump) + `invalid_at` filter intact.
- [ ] **Step 2: RED** — `go test ./mnemonic/internal/memory/ -run TestObservationsParity -v` FAILs (no migration).
- [ ] **Step 3: Implement** — `CREATE new tables -> INSERT SELECT copy -> VERIFY count+sha -> ALTER old TO _legacy -> CREATE VIEW`; move `idx_obs_pinned/expires/dedupe/embedding/importance/risk/memory_type` to owning side table.
- [ ] **Step 4: GREEN** — parity test passes on copy; `migrate()` idempotent (re-run no-op via `index_meta`).
- [ ] **Step 5: Commit** — `git add mnemonic/internal/store/migrations/054_obs_split.sql mnemonic/internal/memory/observations_compat_test.go && git commit -m "feat: split observations behind compat view"`

### Task 3: Migrate hot-path callers off the view

**Files:**
- Modify: `mnemonic/internal/memory/*.go` (7 read sites + `SaveWithAction`)
- Test: extend `observations_compat_test.go` with hot-path column-set assertion

**Interfaces:**
- Consumes: Task 2 side tables
- Produces: hot reads hit `core+state` only; vectors lazy via `observation_vectors`
- Seam: none (in-process)
- Deletion test: n/a
- Adapters: 1

**SATISFIES:** R1

- [ ] **Step 1: Failing test** — assert `mem_context` query plan avoids `observation_vectors` (test on `EXPLAIN QUERY PLAN` output).
- [ ] **Step 2: RED** — fails (still hits view).
- [ ] **Step 3: Implement** — rewrite selects; `SaveWithAction` writes core + side rows in one tx.
- [ ] **Step 4: GREEN** — `go test ./mnemonic/...` passes; `go vet ./mnemonic/...`.
- [ ] **Step 5: Commit** — `git commit -m "refactor: read observations hot path from split tables"`

### Task 4: IVF probe search (no new dep)

**Files:**
- Create: `mnemonic/internal/vectorstore/index.go`, `mnemonic/internal/vectorstore/ivf.go`
- Create: part of `055_hnsw_store.sql` (IVF tables only; HNSW tables land in Task 5)

**Interfaces:**
- Consumes: `embeddings`/`chunk_embeddings` BLOBs
- Produces: `VectorIndex` interface + `IVFIndex{Probe(nprobe)}` + `ivf_centroids/assignments`
- Seam: `VectorIndex` (adapters: BruteForce + IVF = 2, seam justified)
- Deletion test: deleting interface collapses search callers back to direct SQL — interface earns its keep at 2+ backends
- Adapters: 2

**SATISFIES:** R3-partial

- [ ] **Step 1: Failing test** — recall@10 IVF vs brute >= 0.90 on 20k synthetic.
- [ ] **Step 2: RED** — no `ivf.go`.
- [ ] **Step 3: Implement** — kmeans-lite centroids build, assignment on upsert, probe-then-rerank search; corrupt-IVF falls back to brute.
- [ ] **Step 4: GREEN** — recall + latency bench green.
- [ ] **Step 5: Commit** — `git commit -m "feat: add IVF probe search behind VectorIndex"`

### Task 5: HNSW (gated by ADR-0037)

> **Checkpoint:** ⚠ one-way: new Go dependency + persisted graph layout. STOP for approval (ADR-0037 must be accepted first).

**Files:**
- Create: `.skillgrid/artifacts/04-adr-0037-embedded-ann-index.md`
- Create: `mnemonic/internal/vectorstore/hnsw.go`
- Modify: `055_hnsw_store.sql` (add `hnsw_nodes/links/meta`)

**Interfaces:**
- Consumes: `VectorIndex`, `hybrid/vectorcache.go`
- Produces: `HnswIndex{M, efConstruction, efSearch}` + warm-load on open + dual-write in `embedPass`
- Seam: same `VectorIndex` (3rd adapter)
- Deletion test: removing HNSW leaves IVF+Brute — callers untouched via interface
- Adapters: 3

**SATISFIES:** R3-full

- [ ] **Step 1: Failing test** — recall@10 >= 0.95 at 100k + warm <10ms.
- [ ] **Step 2: RED** — no `hnsw.go`.
- [ ] **Step 3: Implement** — pure-Go HNSW only (no cgo); persist graph rows; `DELETE+INSERT` on re-embed in same tx as BLOB.
- [ ] **Step 4: GREEN** — bench green; fallback test (drop HNSW tables -> search still returns via brute).
- [ ] **Step 5: Commit** — `git commit -m "feat: add embedded HNSW index with SQL persistence"`

### Task 6: Unpin dim 768, per-model namespaces

**Files:**
- Modify: `mnemonic/internal/vectorstore/store.go:22,26,52-80`
- Test: `mnemonic/internal/vectorstore/store_test.go` (extend)

**Interfaces:**
- Consumes: `embed_meta(model, dim)`, Tasks 4-5 indexes
- Produces: namespaced vec tables per `(model, dim)`; explicit cross-dim error
- Seam: none
- Deletion test: n/a
- Adapters: 1

**SATISFIES:** R4

- [ ] **Step 1: Failing test** — index 768 + 1024 dim vectors, query each, assert both hit; cross-dim asserts typed error.
- [ ] **Step 2: RED** — `vec_f32` dim error / wipe.
- [ ] **Step 3: Implement** — remove `Dim=768` const pin (keep as default); model-swap clears only its own namespace.
- [ ] **Step 4: GREEN** — `go test ./mnemonic/internal/vectorstore/ -v`.
- [ ] **Step 5: Commit** — `git commit -m "feat: namespace vector indexes per model dim"`

### Task 7: L4 gate + rollout doc

**Files:**
- Modify: none (verification only); update `briefing.md` Verdict line on pass

**Interfaces:** n/a

- [ ] **Step 1:** `go build ./... && go test ./...` full green.
- [ ] **Step 2:** `skillgrid doctor --strict` + Trivy advisory (`trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`, reported not blocking) + `gofmt -l`.
- [ ] **Step 3:** Rollback drill on copy: backup `.dump`, migrate, corrupt ANN, verify fallback + legacy restore.
- [ ] **Step 4: Commit** — `git commit -m "docs: record obs-split hnsw verification verdict"` (spec-zone only).
