# Task 12 — Dream Executor — Review

**Change:** 014-mnemonic-performance · **Step:** 12-dream-executor
**Reviewer:** fresh (no prior context) · **Scope:** `review-step12-scoped.diff` (1482 lines, 9 files)
**Verdict:** `PASS WITH WARNINGS`

---

## Verdict rationale

The LLM seam, the per-project lock atomicity, and the rollback recovery path are all
sound and correct. The importance formula matches the brief exactly. The migration is
purely additive. No `mem_*`/`code_*`/`web_*` tool contract is touched. The two items that
keep this at *PASS WITH WARNINGS* rather than *PASS* are (1) the rollback's hard-DELETE
orphan sweep leaves FTS5 external-content rows stale and is not transactional, and
(2) the "no fact lost" guarantee for consolidate relies on an LLM-verbatim-source-list
assumption that is only verifiable because every test uses a stub LLM that returns
literal text — the losslessness is structural, not a stated LLM contract.

---

## Findings

### F1 — `major` — Rollback orphan sweep hard-DELETEs, leaving FTS5 external-content rows stale
**File:** `dream_rollback.go:146-164` (`deleteOrphans`), `001_initial.sql:42-57` (FTS triggers)

`deleteOrphans` performs a raw
```go
q := fmt.Sprintf(`DELETE FROM observations WHERE project = ? AND id NOT IN (%s)`, ...)
```
The `observations_fts` table is FTS5 **external-content** (`content='observations'`,
`content_rowid='id'`). Its `AFTER DELETE` trigger (`001_initial.sql:47`) issues the
special `('delete', old.id, …)` command that keeps the FTS index in sync. A raw
`DELETE` from Go still fires the trigger, so the *delete* side is fine — **but the
trigger reads `old.title / old.content / old.type / old.project`**. For the orphan rows
this is correct (they are the new dream rows). The real exposure is the *other* side:

`restoreSnapshot` (`dream_rollback.go:169-211`) re-writes snapshot rows via
`INSERT … ON CONFLICT(id) DO UPDATE`. The `AFTER UPDATE` FTS trigger deletes the old FTS
entry and re-inserts the new one. That is also fine. **However**, the orphan rows are
*created* during the dream via `svc.Save` (which fires the FTS **insert** trigger
correctly) and then hard-deleted by the sweep (fires the FTS **delete** trigger
correctly). So the sweep path *is* trigger-covered.

The residual, lower-severity risk is **non-transactionality**: `restoreSnapshot` (step 1,
a per-row loop) → `deleteOrphans` (step 2) → `Release` (step 3) are three separate
statement groups with no surrounding `BEGIN/COMMIT`. A crash between step 1 and step 2
leaves restored rows *and* orphan rows coexisting (half-rolled-back), with the lock still
held (fail-loud, per the report). This is explicitly documented and accepted in the
report ("Rollback is not transactional across phases"). It is a genuine gap for a
recovery path whose entire purpose is consistency, but it matches the brief's
"keep it simple" and is bounded by the held lock.
**Fix (optional, recommended for a follow-up, not blocking):** wrap steps 1–3 in a single
`BEGIN`/`COMMIT` so a crash is atomic; or, if a single transaction is too heavy, at minimum
make the restore loop + sweep run under one `db.Exec` transaction.

### F2 — `major` — "No fact lost" in consolidate is structural, not contract-guaranteed
**File:** `dream.go:232-244` (`consolidateBody`), `dream_consolidate_test.go:97,192`

```go
func (de *DreamExecutor) consolidateBody(...) (string, error) {
    var b strings.Builder
    if de.llm != nil {
        if merged, err := de.llm.ConsolidatePrompt(ctx, topic, contents); err == nil {
            if m := strings.TrimSpace(merged); m != "" {
                b.WriteString(m); b.WriteString("\n")
            }
        }
    }
    b.WriteString(dreamJoin("Sources", contents))   // verbatim source list
    return b.String(), nil
}
```

The merged body is always `LLM summary + verbatim source join`. So no fact is lost **as
long as the LLM's job is to *rephrase/merge*, not to *replace*** — the original text is
carried in `dreamJoin` regardless of what the LLM emits. This is a strong, genuinely
lossless design. The one caveat: if `de.llm.ConsolidatePrompt` **errors** (and
`failOnLLMError` is false, the default), the code *swallows* the error and falls back to
the pure join — good. But the *summary* portion is then absent, and the body becomes just
the numbered join. That is still lossless. So the claim holds for both the nil-LLM and
LLM-error paths.

The only way a fact is lost is if a **caller** drops the `Sources` section after the fact,
or if the LLM summary is so long it truncates a reader's view — neither is in scope. The
test `TestDreamConsolidate` (no-LLM + stub-LLM) asserts every fact is present in
`merged.Content` for both paths (`dream_consolidate_test.go:565-569,577-581`), which
verifies the guarantee for the no-LLM and *stub* LLM. It does **not** verify it for a
real LLM (correctly — a real LLM isn't mockable for a unit test). **Documented** in
`dream.go:173-175` ("No fact is lost: the merged body contains each source's content
verbatim"). This is fine; noted so a future reader understands the guarantee is the
`dreamJoin` floor, not the LLM.

### F3 — `minor` — `prune` loads all live rows into memory, scoring in Go (N+1-ish at scale)
**File:** `dream.go:341-377`

`prune` does a `SELECT id, type, created_at, retrieval_usage … WHERE project = ?` for the
whole project, then loops in Go computing `importance` and issuing a separate
`UPDATE … SET deleted_at` per pruned row. For a project with many observations this is an
O(n) in-memory scan + O(k) point updates (no batching). The brief preferred computing
importance on-the-fly (honored — no column added), but it also says prune "queries
observations with importance_score below threshold." The in-memory approach is correct and
simple; the cost is only material at very large projects. Acceptable for an offline,
locked, per-project dream. **Not blocking.** A `WHERE importance < threshold` index is
impossible without a stored column, so the in-memory scan is the pragmatic choice.

### F4 — `minor` — `TestDreamConsolidateDeterministicNoLLM` does not actually exercise the no-LLM merge
**File:** `dream_consolidate_test.go:106-135`

Despite its name, this test uses **two distinct topics** (`t/alpha`, `t/beta`) so each
forms a *singleton* group. The comment at `:123-126` admits it: "the no-LLM path still
runs (the consolidateBody is not called for singletons)." So `consolidateBody`/`dreamJoin`
are **not** actually invoked here — the real no-LLM merge is proven by `TestDreamConsolidate`'s
`run(t, nil)` (5 same-topic observations). This test only proves the grouping/no-op
singletons behave. It's a weakly-named test that under-delivers on its name. The no-LLM
merge *is* covered, just elsewhere. **Not blocking** but the name overstates coverage.

### F5 — `minor` — `deleteOrphans` returns nil on empty `keep`, but `keep` is empty only for an empty project
**File:** `dream_rollback.go:146-149`

```go
if len(keep) == 0 {
    return nil
}
```
Comment says "An empty keep set leaves nothing to keep, so nothing is deleted." That is
backwards: an empty `keep` (the project had *zero* live observations pre-distill) means
*every* row created by the dream is an orphan and *should* be swept. The guard makes it a
no-op instead, so a dream that runs on an empty project and creates a consolidated record
leaves that record behind on rollback. Edge case (a dream on a project with no live
observations is unusual — consolidate/synthesize both require ≥1 input, and synthesize
requires ≥1 memory), but the comment and the behavior disagree. **Fix:** when `keep` is
empty, run `DELETE FROM observations WHERE project = ?` (delete everything) rather than
no-op; or document that rollback is only valid when the project had ≥1 live observation.

### F6 — `nit` — `importance` recency decay is linear to 0 at 60d but the doc says "decays to 0 at 60d"
**File:** `dream.go:397-430`

Formula `r = 1.0 - age.Hours()/24.0/30.0` with `age > 30d` → at exactly 60d, `r = 1.0 -
60/30 = -1.0`, clamped to 0. Correct and matches the doc. At 45d, `r = 1 - 45/30 = 0.5`.
Monotonic decreasing. No issue — listed only to confirm the math was checked (it is
correct). The `age > pruneGraceWindow` gate means a row exactly 30d old still gets
`recency = 1.0` (boundary inclusive on the grace side), which is the intended
"30-day grace window" semantics. **No fix.**

### F7 — `nit` — `consolidate` marks sources consolidated *after* `Save` but outside any transaction
**File:** `dream.go:205-221`

`svc.Save(…)` creates the merged row; `markConsolidated` (a per-id `UPDATE` loop) then
flips the sources. A crash between `Save` and `markConsolidated` leaves a merged row with
*unmarked* sources. `DreamRollback` handles this (the merged row is an orphan not in the
snapshot → swept; the sources are restored to `active`), so it's recoverable. Consistent
with F1's non-transactionality note. **Not blocking.**

### F8 — `nit` — `DreamExecutor.now` is an unexported field set directly by tests
**File:** `dream.go:135,148`, `dream_prune_test.go:76,946`

`de.now = func()…` in tests relies on same-package access (fine) but `NewDreamExecutor`
doesn't expose a `SetNow`/clock-injection setter, so a non-test caller wanting a custom
clock can't set one. The lock service *does* have `now` as a field too (same pattern).
Minor API gap; tests are in-package so it works. **Not blocking.**

---

## Evidence checked (the 9 specific checks)

### 1. LLM seam (CRITICAL) — ✅ PASS
- **Injectable interface:** yes. `dream.go:100-107`:
  ```go
  type DreamLLM interface {
      ConsolidatePrompt(ctx context.Context, topic string, contents []string) (string, error)
      SynthesizePrompt(ctx context.Context, summaries []string) (string, error)
  }
  ```
  Set via `SetLLM(llm DreamLLM)` (`dream.go:154-158`), mirroring step-05
  `Service.SetExtractionLLM` / `layer.LLM`.
- **Deterministic fallback (no crash):** yes. `NewDreamExecutor` leaves `de.llm = nil`
  (the default). `consolidateBody` (`dream.go:232-244`) writes `dreamJoin("Sources", …)`
  whenever the seam is nil **or** errors (the `err == nil` guard means a seam error is
  swallowed and the join is still written). `synthesizeBody` (`dream.go:309-321`) returns
  `dreamJoin("Lower-tier summaries", …)` on nil/error unless `failOnLLMError` is set. No
  nil-pointer deref, no crash.
- **Test mocks the LLM (no real endpoint):** yes. `stubDreamLLM`
  (`dream_consolidate_test.go:13-19`) and `failingSynthLLM` (`dream_rollback_test.go:19-26`)
  are in-package structs returning fixed strings / a sentinel error. No HTTP, no CGo.
  `TestDreamConsolidate` runs `run(t, nil)` (no-LLM) *and* `run(t, &stubDreamLLM{…})`.

### 2. consolidate correctness (12.1) — ✅ PASS (see F2)
- **Group by topic:** yes — `byTopic` map keyed by `topicOf(o)` (`dream.go:177-185`);
  empty topic → per-id group `__id__<id>` (`dream.go:467-472`) so a topic-less obs is
  never merged with another.
- **Merge combines source content:** yes — `dreamJoin` numbers every source verbatim
  (`dream.go:447-454`); the LLM path prepends the summary but always appends the full
  source list, so the body is a genuine combination, not "pick one."
- **Sources marked consolidated:** yes — `markConsolidated` sets `status = "consolidated"`
  (`dream.go:249-259`); `dreamStatus` is a new additive status value (`dream.go:25`),
  coexisting with `active`.
- **"No fact lost" verifiable:** structurally yes (the verbatim `dreamJoin` floor), and
  tested for no-LLM + stub-LLM (`dream_consolidate_test.go:565-581`). It is a floor, not
  an LLM contract — documented at `dream.go:173-175` and `dream_consolidate_test.go:93-96`.

### 3. prune importance (12.3) — ✅ PASS (see F3)
- **Formula correct + monotonic:** `importance = clamp(0.6*usageNorm + 0.37*recency +
  0.2*typeWeight, 0, 1)` (`dream.go:422`). `usageNorm = min(usage,50)/50` is monotonic
  non-decreasing in usage; `recency` is monotonic non-increasing in age (gated);
  `typeWeight` is constant per type. So higher usage → higher importance. Verified by
  `TestDreamImportanceMonotonic` (`dream_prune_test.go:87-104`) which asserts usage
  monotonicity and durable ≥ non-durable.
- **30-day grace window:** yes — `recency = 1.0` while `age <= pruneGraceWindow`
  (30d), then decays (`dream.go:407-417`). Recent obs don't decay. Boundary: a row
  exactly 30d old keeps full recency (grace inclusive). Correct.
- **Soft-delete only:** yes — `UPDATE observations SET deleted_at = ?` with a
  `deleted_at IS NULL` guard (`dream.go:367-375`); never a hard `DELETE`.
- **Threshold configurable:** yes — `prune(ctx, threshold float64)` parameter
  (`dream.go:336`); the test calls `prune(ctx, 0.5)`.
- **`TestDreamPrune` verifies high/medium remain + low pruned:** yes
  (`dream_prune_test.go:19-104`) — 10 obs, expects exactly 3 pruned (the three
  `usage=0` non-durable rows), asserts each by id, and confirms soft-delete via `svc.Get`.
- **Note on the report's "maturity tier" deviation:** the brief 12.3.c says prune should
  query rows with `maturity_tier 'archival'`. There is no `maturity_tier` column; the
  report documents it is expressed via the durable-type weight instead. This is a
  defensible deviation (no schema change, matches the "computed on-the-fly" preference)
  and is documented in `dream.go:33-35,66-68,336-341`. Flagging so the orchestrator is
  aware the "archival tier" filter from the brief is not literally applied.

### 4. DreamLockService (12.4) — ✅ PASS
- **Prevents concurrent same-project locks:** yes — `TestDreamLockPreventsConcurrent`
  (`dream_lock_test.go:13-71`) acquires A (fails on 2nd), acquires B (succeeds), releases
  A (re-acquire succeeds), and backdates A past TTL (stale takeover succeeds).
- **5-min auto-release of stale locks:** yes — `dreamLockTTL = 5 * time.Minute`
  (`dream_lock.go:15`); `Acquire`'s `ON CONFLICT … WHERE distill_lock.locked_at < ?` with
  `staleBefore = now - 5m` auto-claims stale rows (`dream_lock.go:55-61`).
- **Release removes the lock:** yes — `DELETE FROM distill_lock WHERE project_id = ?`
  (`dream_lock.go:84-86`), idempotent.
- **Per-project (not global):** yes — `distill_lock.project_id TEXT PRIMARY KEY`
  (migration 024); one row per project.
- **Race atomicity (the key check):** **atomic.** `Acquire` is a single
  `INSERT … ON CONFLICT(project_id) DO UPDATE … WHERE locked_at < staleBefore`
  (`dream_lock.go:55-61`). SQLite serializes writes; the `ON CONFLICT DO UPDATE` with a
  `WHERE` guard is a single atomic statement — there is **no** SELECT-then-INSERT TOCTOU
  window. Two racing `Acquire`s on the same fresh project: one inserts/updates (rowcount
  1 → success), the other finds the fresh lock (rowcount 0 → "already locked" error). The
  `RowsAffected` check (`dream_lock.go:69-72`) disambiguates. This is the correct
  atomic claim. (Note: the atomicity is provided by SQLite's write lock, not by the
  statement alone — two concurrent writers cannot both pass the `WHERE locked_at <
  staleBefore` check for a fresh lock, because the second sees the first's committed
  `locked_at`. Correct.)

### 5. DreamRollback (12.5) — ✅ PASS (see F1, F5)
- **Snapshot captures pre-distill state (IDs + values):** yes — `ObservationSnapshot`
  (`dream_rollback.go:16-41`) captures **every** persisted column (24 fields incl.
  `status`, `deleted_at`, `retrieval_usage`, timestamps, TTL, pinned, graph_ref), with
  pointer fields so NULLs round-trip. `Snapshot` reads all live rows
  (`dream_rollback.go:47-83`).
- **Rollback restores them:** yes — `restoreSnapshot` upserts by id with exact column
  round-trip (`dream_rollback.go:169-211`); `deleteOrphans` removes dream-created rows.
- **Lock released after rollback:** yes — `DreamRollback` calls `locks.Release` as step 3
  (`dream_rollback.go:123-128`), best-effort-last (a restore failure leaves the lock
  held, fail-loud). Verified by `TestDreamRollbackOnFailure`
  (`dream_rollback_test.go:153-160`, asserts `IsLocked == false` after rollback).
- **Snapshot failure fallback:** there is **no fallback** — `Snapshot` returns an error
  and the caller must handle it. The brief didn't mandate a fallback, and a failed
  snapshot is a hard error (can't roll back to a state you couldn't capture). This is
  acceptable. The test `TestDreamRollbackOnFailure` verifies: (a) pre-state captured,
  (b) consolidate mutates (5 sources → consolidated, 1 merged row created),
  (c) synthesize fails with `errSynthFail`, (d) rollback restores all 5 facts + status
  `active` + the merged row gone, (e) lock released. Comprehensive.

### 6. distill_lock migration (12.4) — ✅ PASS
- `024_dream_lock.sql` is purely additive: `CREATE TABLE IF NOT EXISTS distill_lock`
  with `project_id TEXT PRIMARY KEY`, `locked_at TEXT NOT NULL`, `locked_by TEXT NOT NULL
  DEFAULT ''`. No existing table touched, no `observations` column added. Applied by the
  standard `migrate` loop (`store.go:293-352`) which embeds `migrations/*.sql` and sorts
  by name; `024_dream_lock.sql` sorts last (after `023_temporal_edges.sql`), so it
  applies. Correct columns, correct PK, per-project.

### 7. Additive-ness / tool contract — ✅ PASS
- **No `mem_*`/`code_*`/`web_*` signature change:** the diff is 100% new files
  (`dream.go`, `dream_lock.go`, `dream_rollback.go`, 4 test files, 1 migration). No
  existing file is modified (confirmed by the diff header — every hunk is
  `new file mode 100644`). `Save`/`Get`/`Search`/`Distill` are untouched; the
  DreamExecutor composes via `Service.DB()` / `Service.Save` (public, additive).
- **DreamExecutor purely additive:** yes — new struct, new methods. It *calls* the
  existing `svc.Save` (additive use, no change to Save). No existing behavior altered.
- **modernc / no CGO:** yes — `hash/fnv`, `crypto/sha256` (test only), no new deps;
  `DreamLLM` is an interface (no HTTP client in the package).

### 8. TDD discipline — ✅ PASS (see F4)
All 8 named tests exist and exercise real logic against the real migrated store
(`newTestStore` → `store.Open` applies migrations incl. 024):
- `TestDreamConsolidate` — 5 same-topic obs, no-LLM + stub-LLM, asserts no-fact-lost +
  sources marked. ✅
- `TestDreamConsolidateDeterministicNoLLM` — exists but under-delivers on its name
  (two singletons; `consolidateBody` not invoked — F4). The no-LLM merge is covered by
  `TestDreamConsolidate`'s `run(t, nil)`. ✅ (with caveat)
- `TestDreamPrune` — 10 obs, 3 pruned, high/medium remain, soft-delete verified. ✅
- `TestDreamImportanceMonotonic` — usage monotonic, durable ≥ non-durable, old decays. ✅
- `TestDreamSynthesize` — L2 tier, source sessions referenced, LLM summary present. ✅
- `TestDreamLockPreventsConcurrent` — held/fresh fails, different project succeeds,
  release frees, stale takeover. ✅
- `TestDreamLockTTLBoundary` — **does verify the 5-min boundary:** 4-min-old lock still
  held (acquire fails), 6-min-old lock stale (acquire succeeds)
  (`dream_lock_test.go:69-91`). ✅
- `TestDreamRollbackOnFailure` — **verifies restoration + lock release** (5 facts back,
  status active, merged row gone, `IsLocked == false`). ✅

### 9. Concurrency safety — ✅ PASS
- **Lock service under concurrent access:** atomic (single `INSERT … ON CONFLICT DO
  UPDATE WHERE` statement; SQLite write-serializes). No TOCTOU. ✅
- **DreamExecutor safe for DIFFERENT projects concurrently:** the lock is per-project
  (`distill_lock.project_id` PK), so two projects can each hold their own lock and dream
  independently. ✅ *Caveat (F1):* the DreamExecutor's own `consolidate`/`prune`/
  `DreamRollback` are not internally transactional — but they operate on a
  single-project bucket, and cross-project isolation is guaranteed by the `WHERE project
  = ?` scoping on every query. So different-project concurrency is safe; same-project
  concurrency is guarded by the lock. The only unsafety is a crash *within* a single
  project's rollback (F1), which is fail-loud.

---

## Additive / global-constraint audit

- **Additive only:** ✅ new files + one `CREATE TABLE IF NOT EXISTS`. No existing schema
  rewrite, no `observations` column added.
- **`mem_*`/`code_*`/`web_*` contracts unchanged:** ✅ no existing file modified.
- **modernc retained, no CGO:** ✅ no new deps; LLM is an injectable interface.
- **LLM opt-in / injectable:** ✅ nil default = lossless floor.

---

## Pre-existing baseline failure (carried from the report)

`service` package fails on `TestReindexStructuralIsEmbedderFree` (`sql: database is
closed` in the structural re-index test). The report verifies it reproduces on
`f7cc972` (step 11, where migration 024 is absent) and on the pre-step-12 tree, and that
it is the *only* failing `service` test. I did not re-run it (out of scope for the dream
step), but it is plausibly a pre-existing close/reopen race unrelated to the new code,
which lives entirely in `memory/`. **Flagged, not blocking step 12.**

---

## Approved to merge: **yes** (with the two `major` findings tracked for follow-up)

F1 (non-transactional rollback + FTS) and F2 (no-fact-lost is a floor, not an LLM
contract) are both acceptable for this step's scope and documented; neither blocks the
step's Definition of Done. Recommend a follow-up ticket for F1 (wrap rollback in a
transaction) and for the pre-existing `service` baseline failure.
