# Task 12 — Dream Executor (consolidate / synthesize / prune) — Report

**Change:** 014-mnemonic-performance · **Step:** 12-dream-executor · **Verdict:** PASS WITH WARNINGS

The step decomposes the session-close distillation into three phases on a new
`DreamExecutor`, plus a per-project `DreamLockService` and a `DreamRollback`
recovery path. All work is additive on the existing `memory` package (composes
via `Service.DB()`, like `layer`); no `mem_*`/`code_*`/`web_*` tool contract
changes; no CGO; `modernc.org/sqlite` unchanged.

Sub-tasks 12.1 (consolidate) and 12.2 (synthesize) were already implemented and
committed by a prior run (`28d8cca`, `3d08793`). This session: **fixed and
committed 12.3 (prune)**, and **implemented + committed 12.4 (lock) and 12.5
(rollback)**.

---

## Per sub-task

### 12.1 consolidate() — merges same-topic facts into coherent knowledge (commit `28d8cca`)
- `DreamExecutor.consolidate(ctx, observations []Observation) (ConsolidatedResult, error)`
  at `internal/mnemonic/memory/dream.go:157`.
- **Grouping:** by `topic_key` (`topicOf`, dream.go:444). An empty `topic_key`
  becomes its own per-id group (`__id__<id>`) so a topic-less observation is
  never merged with a different one. Input order is preserved within each group.
- **Merge:** a group with >1 member folds all contents into one coherent body
  via the LLM (or the deterministic lossless join), creates ONE new observation
  (type `learning`, `source = "dream"`, topic inherited), and marks every
  source with `status = "consolidated"` (`dreamStatus`, dream.go:25). A singleton
  group is a no-op (`Consolidated: false`).
- **No fact lost:** the merged body always appends the verbatim source list
  (`dreamJoin`) after any LLM summary, so the LLM pass cannot silently drop a
  fact; the no-LLM path is a pure numbered join.
- `ConsolidatedGroup`/`ConsolidatedResult` report topic, new id, source ids,
  count, and the merged body.

### 12.2 synthesize() — lifts lower-tier memories into a higher-tier summary (commit `3d08793`)
- `DreamExecutor.synthesize(ctx, memories []Memory) (Summary, error)` at
  dream.go:252.
- Takes L0/L1 `Memory` records (SessionID + Content), produces a single
  higher-tier summary via the LLM (`SynthesizePrompt`) or the lossless-join
  fallback, and stores it as a NEW observation with `Tier = DreamTierL2`
  (`"L2"`), `source = "synthesis"`, topic `dream/L2/<shortID>`.
- **Provenance:** appends a `Source sessions: <sid>, <sid>` line to the body and
  returns the source session ids in `Summary.SessionIDs` — the L2 record always
  references its source sessions.
- **Failure path (used by 12.5):** `synthesizeBody` honors the new
  `failOnLLMError` flag — when set, an LLM seam error is returned as a hard
  error (no fallback), so a mid-dream synthesize failure is surfaced for
  rollback. Default (false) keeps the lossless floor.

### 12.3 prune() — removes low-importance observations by AKL scoring (commit `44802ec`)
- `DreamExecutor.prune(ctx, threshold float64) (PrunedResult, error)` at
  dream.go:319. Loads the project's live rows, scores each with the on-the-fly
  `importance()`, and soft-deletes (sets `deleted_at`) those below the
  threshold. Never touches an already-deleted row. Returns the pruned count + ids.
- **IMPORTANT: fixed a scoring bug this session.** The pre-session weights
  (`wUsage=0.5, wRecency=0.3, wType=0.2`) with a *raw* recency term
  (`1 - age/30d`, no grace) scored a fresh never-retrieved non-durable row at
  0.38 and, crucially, scored the "keep" rows obs3/obs6/obs9 (usage 10 / 0-durable
  / usage 5) at 0.48/0.48/0.43 — all **below** the 0.5 threshold the test expects
  to keep. `TestDreamPrune` was RED (pruned 6, wanted 3).
- **Fix (two changes, both committed in `44802ec`):**
  1. **Gated recency (minimum age):** recency is now `1.0` while age ≤ a
     30-day grace window (`pruneGraceWindow`), then decays `1 - age/30d` (0 at
     60d). Fresh content is scored on usage + durability alone — age only ever
     *lowers* the score after the grace window. This matches the brief's
     "importance + a minimum age".
  2. **Rebalanced weights:** `wUsage=0.6, wRecency=0.37, wType=0.2` (type
     weights unchanged: durable 0.9 / default 0.4). Chosen so at a 0.5 threshold:
     fresh never-retrieved non-durable = 0.45 (pruned); the least-valued keeper,
     a fresh non-durable note retrieved once (usage 5) = 0.51 (kept); a fresh
     never-retrieved durable note = 0.55 (kept). Usage dominates, durability
     breaks the tie, age only lowers.
- **`TestDreamPrune` is GREEN** (prunes exactly obs 0/1/2) and
  `TestDreamImportanceMonotonic` (usage monotonic, durable ≥ non-durable, old
  lesson decays to ~0.08) stays GREEN.

**Importance approach: computed on-the-fly** from existing columns
(`retrieval_usage`, `created_at`, `type`) — **no `importance_score` or
`maturity_tier` column added** (the brief's preferred option). The maturity tier
is expressed through the durable-type weight (`durableTypes`: decision,
architecture, convention, config). Full formula + weights are documented in
`dream.go` (the `dreamTypeWeights` and `importance` doc comments).

### 12.4 DreamLockService — per-project distillation lock (commit `ae1b20f`)
- New file `internal/mnemonic/memory/dream_lock.go`.
- `NewDreamLockService(db *sql.DB) *DreamLockService` — a standalone service
  (not a `DreamExecutor` method) so a lock can be held across the whole dream
  run and released on success or rollback. Clock is a `now` seam for the TTL.
- **Schema:** the `distill_lock` table (migration `024_dream_lock.sql`,
  committed in `44802ec`): `project_id TEXT PRIMARY KEY, locked_at TEXT NOT
  NULL, locked_by TEXT NOT NULL DEFAULT ''`. One row per project.
- `Acquire(ctx, projectID, holder)`:
  `INSERT ... ON CONFLICT(project_id) DO UPDATE SET locked_at=?, locked_by=?
  WHERE distill_lock.locked_at < <staleBefore>` where
  `staleBefore = now - 5m`. Rowcount 1 → acquired (absent or stale); rowcount 0
  → a fresh lock is held → error. **Auto-release of stale locks is the
  `WHERE ... locked_at < staleBefore` guard** (5-minute TTL, `dreamLockTTL`).
- `Release(ctx, projectID)`: `DELETE FROM distill_lock WHERE project_id = ?`
  (idempotent).
- `IsLocked(ctx, projectID)`: reports whether a *fresh* (non-stale) lock is
  held; a stale lock reports false.

### 12.5 DreamRollback — reverts observations to pre-distill state on failure (commit `7346079`)
- New file `internal/mnemonic/memory/dream_rollback.go`.
- `ObservationSnapshot` captures **every** persisted column of a live
  observation (mirrors the `obsSelectCols` read path + `graph_ref`/`deleted_at`),
  with pointer fields so NULLs round-trip.
- `Snapshot(ctx) ([]ObservationSnapshot, error)` reads all of the project's
  live (non-deleted) observations into an in-memory snapshot — what a dream
  captures **before** it mutates anything.
- `DreamRollback(ctx, projectID, pre, locks)`:
  1. **Restore** each snapshot row to its exact pre-distill values via
     `INSERT ... ON CONFLICT(id) DO UPDATE` (`restoreSnapshot`) — so a source
     row that was marked `consolidated` (status flipped) or soft-deleted is
     brought back to `active` with its original content/timestamps.
  2. **Sweep orphans**: hard-`DELETE` the project's rows whose id is **not** in
     the snapshot (`deleteOrphans`, positional `NOT IN (?, ?, ...)`) — removes
     the consolidated record / partial synthesis the failed dream created.
  3. **Release the lock** (best-effort, after data is restored) so the project
     can be re-dreamed. A restore that fails leaves the lock held (fail loud).

**Rollback scope:** covers the consolidate/synthesize phases (status flips +
new-row creation), which is where the mutations are. Prune is already a
soft-delete and is reversible by clearing `deleted_at`, so it does not need the
snapshot path.

---

## LLM seam (how it's injected / mocked)

- `DreamLLM` interface (dream.go:93) mirrors the step-05 `ExtractionLLM` and
  `layer.LLM` seam pattern: a tiny interface a backend implements, NO CGo LLM
  client in the package.
  ```go
  type DreamLLM interface {
      ConsolidatePrompt(ctx context.Context, topic string, contents []string) (string, error)
      SynthesizePrompt(ctx context.Context, summaries []string) (string, error)
  }
  ```
- **Injection:** `NewDreamExecutor(svc)` builds the executor with a nil seam
  (the deterministic floor). `SetLLM(llm)` attaches it — the same shape as
  `Service.SetExtractionLLM` (step 05). `SetLLM(nil)` / never calling it keeps
  the no-LLM path.
- **Mocking:** tests supply a stub (`stubDreamLLM` in
  `dream_consolidate_test.go`, `failingSynthLLM` in `dream_rollback_test.go`)
  that returns fixed deterministic text or a hard error.
- **No-LLM fallback:** `consolidateBody`/`synthesizeBody` fall back to a
  lossless numbered join (`dreamJoin`) when the seam is nil or errors — unless
  `failOnLLMError` is set (12.5), in which case a synthesize seam error is a
  hard error.

## Commits (this session's contribution)

| SHA | Subject |
|-----|---------|
| `44802ec` | feat(mnemonic): DreamExecutor prune phase (weights + grace-window recency fix, migration 024) |
| `ae1b20f` | feat(mnemonic): DreamLockService with 5-minute timeout |
| `7346079` | feat(mnemonic): DreamRollback reverts on failure |

(12.1/12.2 were already committed: `28d8cca` consolidate, `3d08793` synthesize.)

## RED → GREEN evidence

All five focused tests run against the real migrated store
(`newTestStore` → `store.Open` applies migrations including 024).

| Sub-task | Test | RED (before impl) | GREEN (after impl) |
|----------|------|-------------------|--------------------|
| 12.1 | `TestDreamConsolidate` | (pre-committed; PASS) | PASS |
| 12.2 | `TestDreamSynthesize` | (pre-committed; PASS) | PASS |
| 12.3 | `TestDreamPrune` | **FAIL** — `pruned = 6, want 3` (obs 3/6/9 wrongly pruned) | **PASS** |
| 12.3 | `TestDreamImportanceMonotonic` | PASS | PASS |
| 12.4 | `TestDreamLockPreventsConcurrent` | **build failed** — `undefined: NewDreamLockService` | **PASS** |
| 12.4 | `TestDreamLockTTLBoundary` | build failed | PASS (4-min held, 6-min auto-released) |
| 12.5 | `TestDreamRollbackOnFailure` | **build failed** — `de.failOnLLMError`/`de.Snapshot`/`de.DreamRollback` undefined | **PASS** |

Rollback boundary: `TestDreamLockTTLBoundary` proves the 5-minute TTL — a
4-minute-old lock is still held (acquire fails); a 6-minute-old lock is stale
(auto-released, acquire succeeds).

## Full relevant suite + output

`go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1`

```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory    14.658s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer 2.424s
```

(Focused dream tests, `-v`: `TestDreamConsolidate` PASS,
`TestDreamConsolidateDeterministicNoLLM` PASS, `TestDreamPrune` PASS,
`TestDreamImportanceMonotonic` PASS, `TestDreamSynthesize` PASS,
`TestDreamLockPreventsConcurrent` PASS, `TestDreamLockTTLBoundary` PASS,
`TestDreamRollbackOnFailure` PASS.)

### 005/008/010/011/013 baselines

`go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/...
./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/...
./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/...
./skillgrid-cli/internal/mnemonic/pdg/... -count=1`

```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store      5.538s
FAIL github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service  44.318s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex 14.523s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route      4.477s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected  10.511s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community  0.931s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg        4.135s
```

**One baseline failure — pre-existing, NOT caused by step 12:**
`service` fails on `TestReindexStructuralIsEmbedderFree`
(`reindex_structural_test.go:107: count gamma: sql: database is closed`).
Verified pre-existing: it fails deterministically on `f7cc972` (step 11, where
migration 024 is ABSENT) and on the pre-step-12 tree, and only that one test
fails — every other `service` test passes. It is a close/reopen race in the
structural re-index test (the reopen handle reads after the first store is
closed), unrelated to the dream code, which lives only in `memory/`.

## Global Constraints

- ✅ Additive only — new `DreamExecutor`/`DreamLockService`/`DreamRollback` +
  one `CREATE TABLE IF NOT EXISTS` (migration 024); no existing schema rewrite,
  no `observations` column added.
- ✅ `mem_*`/`code_*`/`web_*` tool contracts unchanged (no `Save`/`Search`/
  `Distill` signature or response-shape changes).
- ✅ `modernc.org/sqlite` driver retained; no CGO (pure-Go `hash/fnv`,
  `crypto/sha256`; the LLM is an injectable interface, not an HTTP call).
- ✅ LLM is OPT-IN via the injectable `DreamLLM` seam (nil default = lossless
  floor), following the step-05 `ExtractionLLM` pattern.

## Concerns

- **Pre-existing `service` baseline failure** (`TestReindexStructuralIsEmbedderFree`):
  deterministic `database is closed` in the structural re-index test. Not
  introduced by step 12 (reproduces on `f7cc972` without migration 024). Flagged
  for the orchestrator / a separate fix — out of scope for the dream step.
- **`Save` topic_key upsert:** saving multiple same-topic observations via
  `Save` collapses them to one row (topic upsert). `consolidate()` is still
  exercised correctly (the existing `TestDreamConsolidate` and the new rollback
  test both rely on the same-topic grouping; the rollback test seeds 5 distinct
  same-topic rows via direct INSERT to bypass the upsert so it can prove a
  5-member merge + full restore). If a future dream input comes through
  `Save`, expect topic collapse before consolidate — a property of the store,
  not the dream.
- **Rollback is not transactional across phases:** it is snapshot-restore +
  orphan sweep + lock release, executed as separate statements (the brief's
  "keep it simple" guidance). A crash *between* restore and sweep can leave the
  lock held (fail loud) rather than half-restored-and-unlocked. Acceptable for
  the current best-effort, offline dream.
