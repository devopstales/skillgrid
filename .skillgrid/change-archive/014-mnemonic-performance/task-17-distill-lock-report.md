# Step 17 — DistillLockService + DistillRollback on failure

**Change:** 014-mnemonic-performance · **Step:** 17-distill-lock
**Branch:** `feat/v2-structure`
**Depends on:** 09 (session promotion), 12 (dream-executor) — both PASS

## Summary

Step 12 (dream-executor) already built the per-project lock (`DreamLockService`,
`distill_lock` table, migration `024_dream_lock.sql`, 5-min TTL) and the rollback
(`DreamRollback`, snapshot/restore + orphan sweep + lock release). Step 17 asked
for `DistillLockService` + `DistillRollback` under the distillation name, plus a
lock-status CLI, and — critically — to FIX step-12 reviewer finding **F1**: the
rollback's 3 steps (restore / orphan-sweep / release) were NOT in one transaction,
so a crash mid-rollback left a half-restored project.

**Approach: REUSE step 12's implementation, do not duplicate.** `DistillLockService`
is a thin wrapper over `DreamLockService` (same `distill_lock` rows, same TTL, same
atomic acquire — the two names are the same lock). `DistillRollback` reuses
`DreamRollback` (via the `AsDreamLock()` bridge). The F1 fix lives in step 12's
`dream_rollback.go`: the restore + orphan-sweep are now wrapped in a single
`BEGIN…COMMIT` transaction, and the lock release is moved outside the tx so a failed
rollback keeps the lock held (fail loud).

---

## 17.1 [RED] — DistillLockService uses distill_lock row per project

**Reused step 12:** YES. `DistillLockService` (new file
`internal/mnemonic/memory/distill_lock.go`) is a thin wrapper whose
`Acquire`/`Release`/`IsLocked` delegate to `DreamLockService`. No lock logic
duplicated.

- `NewDistillLockService(db)` constructs a `DreamLockService` internally
  (`NewDreamLockService(db)`) and keeps the `*sql.DB` for the status read (17.4).
- `AsDreamLock()` exposes the inner `*DreamLockService` so the step-12
  `DreamRollback` (which releases through that type) can be handed the same lock a
  `DistillLockService.Acquire` created.

**Test** (`distill_lock_test.go`, `TestDistillLockRowPerProject`):
Acquire(A) → 1 row for A; Acquire(B) → a *separate* row for B (A untouched, total 2);
Acquire(A) again → fails (existing fresh lock detected, still 1 A row);
Release(A) → A's row removed, B's row intact (total 1).

**RED→GREEN:**
- RED: `undefined: NewDistillLockService` (build failure) — the wrapper didn't exist.
- GREEN: `--- PASS: TestDistillLockRowPerProject (0.23s)`

**Commit:** `783e9b8` feat(mnemonic): DistillLockService with per-project lock rows

---

## 17.2 [RED] — DistillRollback reverts observations to pre-distill state (ATOMIC)

**Reused step 12:** YES. `TestDistillRollbackOnFailure` drives the existing
`DreamExecutor.consolidate`/`synthesize` (a real mid-distill failure via a
`failOnLLMError` + failing synth LLM seam), captures `de.Snapshot(ctx)`, then calls
`de.DreamRollback(ctx, pid, pre, locks.AsDreamLock())`. No new rollback code — the
distill name reuses the step-12 rollback through the `AsDreamLock()` bridge.

**F1 fix — TRANSACTION WRAPPING.** Step 12's `DreamRollback` ran restore, sweep,
and release as independent auto-committed statements. I refactored
`dream_rollback.go` so restore + sweep run in ONE transaction and release runs
after the commit:

```go
// Steps 1 + 2 in one transaction: restore then sweep, committed atomically.
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin rollback tx: %w", err)
}
committed := false
defer func() {
    if !committed {
        _ = tx.Rollback()
    }
}()

// 1. Restore the captured rows (exact column round-trip) inside the tx.
for i := range pre {
    if err := execSnapshot(ctx, tx, &pre[i], exec); err != nil {
        return fmt.Errorf("restore observation %d: %w", pre[i].ID, err)
    }
}

// 2. Remove any observation created during the failed dream — the orphan sweep,
//    inside the same tx.
if err := deleteOrphansTx(ctx, tx, projectID, idsOf(pre), exec); err != nil {
    return err
}

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit rollback: %w", err)
}
committed = true

// 3. Release the lock (outside the tx, after the data commit succeeded).
if locks != nil {
    if err := locks.Release(ctx, projectID); err != nil {
        return fmt.Errorf("release lock after rollback: %w", err)
    }
}
return nil
```

The restore/sweep SQL is routed through an injected `distillRollbackExec` seam
(`func(ctx, tx *sql.Tx, query, args...) error`). The default `txExec` runs each
statement on the tx (the atomic path); a test swaps it for a failing executor to
simulate a mid-rollback crash. The old standalone `restoreSnapshot`/`deleteOrphans`
methods were replaced by the tx variants `execSnapshot`/`deleteOrphansTx` (same
SQL, now bound to the tx). `DreamRollback` is unchanged in signature — it delegates
to `DreamRollbackWith(ctx, pid, pre, locks, de.txExec)`, so step 12's
`TestDreamRollbackOnFailure` still passes unmodified.

**Tests:**
- `TestDistillRollbackOnFailure` (`distill_rollback_test.go`): capture pre-distill
  state; run consolidate (sources → consolidated + merged body written) then a
  failing synthesize; `DreamRollback` → every row round-trips to its exact
  pre-distill content/status/project/topic_key; live count == pre-distill count; no
  consolidated status remains; every fact present again; lock released.
- `TestDistillRollbackAtomic` (the F1 guard): inject a failing executor (2nd
  statement errors). After the failed rollback the project is UNCHANGED from its
  pre-failure state — all rows still consolidated (the first restore was rolled
  back, not half-applied) and the orphan still present (the sweep never ran) — and
  the lock is STILL held (fail loud). Under step 12's non-transactional code the
  first restore would have committed and the project would be half-restored.

**RED→GREEN:**
- RED: first iteration failed on a test-side bug (orphan detection by
  `source='dream'` collided with pre-existing consolidated sources); after
  correcting the orphan detection and adding the F1 tx fix,
  `--- PASS: TestDistillRollbackOnFailure (0.34s)` and
  `--- PASS: TestDistillRollbackAtomic (0.32s)`.
- Regression check: step 12's `TestDreamRollbackOnFailure` still PASSES
  (backward-compatible refactor).

**Commit:** `7b02292` feat(mnemonic): DistillRollback reverts to pre-distill state on failure

---

## 17.3 [RED] — 5-minute timeout with auto-release on error

**Reused step 12:** YES. The 5-min TTL and stale auto-release already live in
`DreamLockService.Acquire` (`dreamLockTTL = 5 * time.Minute`, atomic
`INSERT … ON CONFLICT DO UPDATE WHERE locked_at < staleBefore`). 17.3 adds no new
logic — the test guards the TTL boundary *through the `DistillLockService` alias*
so the distill name is proven to inherit it.

**Test** (`distill_lock_timeout_test.go`, `TestDistillLockTimeoutAutoRelease`):
acquire a lock, backdate `locked_at` to 6 minutes ago → a new Acquire succeeds and
takes over (auto-release, `locked_by` now the new holder); set `locked_at` to
1 minute ago → a new Acquire FAILS (fresh lock not released, original holder
unchanged).

**RED→GREEN:**
- RED: the test references `NewDistillLockService`'s TTL behavior; before the alias
  existed it was a build failure (part of the 17.1 RED batch).
- GREEN: `--- PASS: TestDistillLockTimeoutAutoRelease (0.14s)`

**Commit:** `ef5b193` feat(mnemonic): distill lock 5-minute timeout with auto-release

---

## 17.4 [AFK] — Lock status visible via CLI (`mem distill status`)

**New (thin):** the CLI subcommand. `DistillLockService.DistillLockStatus(ctx,
projectID)` (new method in `distill_lock.go`) is a read-only diagnostic: it reads the
`distill_lock` row (project_id, locked_at, locked_by) and the freshness flag via the
inner `IsLocked`. It never acquires or releases a lock. `mem.go` gains a `distill`
case → `runMemDistill`, and `distill status` is added to `printMemUsage`.

**Tests** (`cmd/skillgrid/mem_distill_test.go`):
- `TestDistillStatusCLI`: no lock → prints `"no active locks"`; acquire a lock →
  prints the project_id, `locked_by` (the holder), and the `project_id` /
  `locked_at` / `locked_by` labels; release → prints `"no active locks"` again.
- `TestDistillStatusCLITimestamp`: the shown `locked_at` is a real RFC3339
  timestamp (JSON-parsed and `time.Parse(time.RFC3339, …)`).

**RED→GREEN:**
- RED: `go run . mem distill status …` → `error: unknown mem command "distill"`,
  `exit status 2` (subcommand didn't exist).
- GREEN: `--- PASS: TestDistillStatusCLI (9.40s)` and
  `--- PASS: TestDistillStatusCLITimestamp (1.73s)`.

**Commit:** `9c577eb` feat(mnemonic): add mem distill status CLI subcommand

---

## Global Constraints

- **Additive only:** no existing schema rewritten. `distill_lock` (migration 024)
  already exists from step 12; no new migration. `dream_rollback.go` was refactored
  in place (same SQL, now in a tx) — no schema change. `mem.go` gained one subcommand
  case + usage line.
- **No tool contract change:** `mem_*` / `code_*` / `web_*` tool shapes untouched.
  `DistillLockService`, `DistillRollback`, `DistillLockStatus`, and `mem distill
  status` are all NEW additive surfaces.
- **No CGO:** `modernc.org/sqlite` driver unchanged; `BeginTx`/`ExecContext` are
  driver-agnostic `database/sql` calls.

## Test evidence

Focused step-17 set:
```
go test ./internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject|TestDistillRollbackOnFailure|TestDistillRollbackAtomic|TestDistillLockTimeoutAutoRelease' -count=1
--- PASS: TestDistillLockRowPerProject (0.14s)
--- PASS: TestDistillRollbackOnFailure (0.35s)
--- PASS: TestDistillRollbackAtomic (0.35s)
--- PASS: TestDistillLockTimeoutAutoRelease (0.14s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory
```

Full relevant suite:
```
go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  20.363s
```

Baselines (005/008/010/011/013):
```
go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/codeindex/... \
        ./internal/mnemonic/route/... ./internal/mnemonic/affected/... \
        ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1
ok  .../mnemonic/store         6.498s
FAIL .../mnemonic/service      (1 test — see below)
ok  .../mnemonic/codeindex     23.241s
ok  .../mnemonic/route         4.316s
ok  .../mnemonic/affected      11.425s
ok  .../mnemonic/community     3.626s
ok  .../mnemonic/pdg           7.521s
```

CLI package (where 17.4 lives):
```
go test ./cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  78.934s
```

### Pre-existing failure (NOT caused by step 17)

`TestReindexStructuralIsEmbedderFree` (in `internal/mnemonic/service/`) fails with
`sql: database is closed`. It has **no** dream/distill/rollback references, and it
fails identically at the pre-step-17 commit `3edcafe` (step 16):
```
git checkout 3edcafe && go test ./internal/mnemonic/service/ -run TestReindexStructuralIsEmbedderFree
--- FAIL: TestReindexStructuralIsEmbedderFree (1.42s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
```
With that one test skipped, the rest of the service package passes:
```
go test ./internal/mnemonic/service/ -count=1 -skip TestReindexStructuralIsEmbedderFree
ok  .../mnemonic/service  60.669s
```
Characterized as a pre-existing, out-of-scope flake/closed-db issue in the service
package's reindex path; it does not touch the distill lock or rollback.

## Commits

| SHA | Subject |
|-----|---------|
| `783e9b8` | feat(mnemonic): DistillLockService with per-project lock rows |
| `7b02292` | feat(mnemonic): DistillRollback reverts to pre-distill state on failure |
| `ef5b193` | feat(mnemonic): distill lock 5-minute timeout with auto-release |
| `9c577eb` | feat(mnemonic): add mem distill status CLI subcommand |

## Concerns

- The `TestReindexStructuralIsEmbedderFree` service-package failure is pre-existing
  (fails at step 16) and unrelated to step 17; flagged for a separate ticket.
- `mem distill status` shows a stale-but-not-yet-released lock with `held: false`
  (the row is visible, just not "held") — intentional: the diagnostic surfaces the
  raw row state, and `held` reflects the freshness gate. No behavioral surprise,
  but worth noting in the CLI help if the distinction ever bites.
- The `DistillLockService`/`DreamLockService` duality is a naming alias, not a
  separate lock: they share the `distill_lock` rows and TTL, so a dream and a
  distill cannot both hold the project lock. This is the intended design (the
  brief says "REUSE step 12's DreamLockService"), but if step 12 and step 17 ever
  need different TTLs the wrapper would need to diverge.
