# Step 17 — DistillLockService + DistillRollback (transaction fix) — Review

**Change:** 014-mnemonic-performance · **Step:** 17-distill-lock
**Reviewer:** fresh (no prior context)
**Scope judged:** `review-step17-scoped.diff` (702 lines, 6 files) + supporting existing code.

## Verdict

**PASS**

---

## Findings

### Minor

**M1 — `distill_lock` column names are an artifact of the dream/distill duality.**
`distill_lock.go:97` (and `dream_lock.go:55`,57,84,102) all read/write a table named
`distill_lock` from `DreamLockService`. That is pre-existing step-12 naming (migration
`024_dream_lock.sql:24` is named `distill_lock` while its header comment calls it the
"distillation (dream) lock"). Step 17 inherits it. Not a defect — both names refer to
the same lock by design — but a future reader landing on `DreamLockService.Acquire`
and seeing `INSERT INTO distill_lock` will be momentarily confused. Cosmetic; no action
required for this step.

**M2 — `DistillLockService` holds a `db` it only uses for the read-only status query.**
`distill_lock.go:19` stores `db *sql.DB` alongside `inner *DreamLockService`. The `db`
is used only by `DistillLockStatus` (`:97`). `inner` already holds its own `*sql.DB`.
This is a one-field denormalization, not a bug, and it keeps `DistillLockStatus` from
reaching through `inner` (which has no `DB()` accessor). Acceptable; noting for
completeness.

### Nit

**N1 — `errDistillLockUninit` is a package-level var, not an inline sentinel.**
`distill_lock.go:113-124` builds the sentinel via a constructor + named type. It works
and mirrors the dream lock's "not initialized" wording, but an unexported `var
errDistillLockUninit = errors.New(...)` would be simpler. Cosmetic.

**N2 — `TestDistillStatusCLI` is ~10x slower than its sibling.**
`mem_distill_test.go` `TestDistillStatusCLI` (3.97s) vs `TestDistillStatusCLITimestamp`
(1.40s). The delta comes from `runMemCLI` spawning the real binary (store open/migrate
each invocation). No defect; the package as a whole passes in 7s.

**N3 — `runMemDistill` uses `hCtx()` rather than the request `ctx`.**
`mem.go:62` calls `locks.DistillLockStatus(hCtx(), projID)`. Consistent with sibling
mem subcommands (they all use the handle's context). No issue.

---

## Evidence checked

### 1. Reuse correctness (17.1) — ✅ THIN WRAPPER, verified

`DistillLockService` is a genuine thin alias. It holds one `*DreamLockService` and
delegates every mutation to it:

```go
// distill_lock.go
type DistillLockService struct {
    db    *sql.DB
    inner *DreamLockService
}
func NewDistillLockService(db *sql.DB) *DistillLockService {
    return &DistillLockService{db: db, inner: NewDreamLockService(db)}
}
func (l *DistillLockService) Acquire(ctx, projectID, holder) error {
    if l == nil || l.inner == nil { return errDistillLockUninit }
    return l.inner.Acquire(ctx, projectID, holder)   // delegate — no lock logic
}
func (l *DistillLockService) Release(ctx, projectID) error {
    if l == nil || l.inner == nil { return errDistillLockUninit }
    return l.inner.Release(ctx, projectID)           // delegate
}
func (l *DistillLockService) IsLocked(ctx, projectID) (bool, error) {
    if l == nil || l.inner == nil { return false, errDistillLockUninit }
    return l.inner.IsLocked(ctx, projectID)          // delegate
}
```

Zero lock logic is duplicated. The `AsDreamLock()` bridge (`:65`) exposes `inner` so
step-12's `DreamRollback` (which releases through `*DreamLockService`) lands its
release on the same row `Acquire` created. It uses migration 024's existing
`distill_lock` table — **no new table, no new migration** (confirmed: the only
`distill_lock` DDL in the repo is `024_dream_lock.sql:24`).

### 2. Transaction-wrapping fix (17.2) — ✅ CRITICAL, VERIFIED CORRECT

`dream_rollback.go` now wraps restore + orphan-sweep in ONE transaction. The exact code:

```go
// dream_rollback.go:140
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin rollback tx: %w", err)
}
committed := false
defer func() {
    if !committed {
        _ = tx.Rollback()          // ← defer tx.Rollback() present
    }
}()

// 1. Restore the captured rows inside the tx.
for i := range pre {
    if err := execSnapshot(ctx, tx, &pre[i], exec); err != nil {
        return fmt.Errorf("restore observation %d: %w", pre[i].ID, err)
    }
}
// 2. Orphan sweep, inside the SAME tx.
if err := deleteOrphansTx(ctx, tx, projectID, idsOf(pre), exec); err != nil {
    return err
}
if err := tx.Commit(); err != nil {          // :164 — single atomic commit
    return fmt.Errorf("commit rollback: %w", err)
}
committed = true

// 3. Release the lock — OUTSIDE the tx, after the data commit succeeded.
if locks != nil {
    if err := locks.Release(ctx, projectID); err != nil {
        return fmt.Errorf("release lock after rollback: %w", err)
    }
}
return nil
```

- `BeginTx` at :140, `Commit` at :164 — ✅
- `defer tx.Rollback()` gated by `committed` — ✅
- Lock release moved OUTSIDE the tx (after `committed = true`) — ✅
- `TestDistillRollbackAtomic` proves a mid-rollback crash leaves the project
  UNCHANGED: it injects a `failingExec` that errors on the 2nd statement, then asserts
  all rows are still `consolidated` (first restore rolled back) AND the orphan still
  exists (sweep never ran) — exactly the F1 "never half-restored" guarantee. PASS.

### 3. Per-project lock (17.1) — ✅ verified

`TestDistillLockRowPerProject` asserts exactly the brief's sequence:
Acquire(A) → 1 A row; Acquire(B) → separate B row (A untouched, total 2);
Acquire(A) again → errors (existing fresh lock detected, still 1 A row);
Release(A) → A removed, B intact (total 1). Lock is per-project (PRIMARY KEY
`project_id`), not global. PASS.

### 4. Lock release on rollback (17.2) — ✅ both paths

- **Successful rollback** → lock released: `TestDistillRollbackOnFailure` asserts
  `locks.IsLocked(ctx, pid)` is `false` after `DreamRollback`. PASS.
- **Failed rollback** → lock KEPT (held): `TestDistillRollbackAtomic` asserts
  `locks.IsLocked(ctx, pid)` is `true` after the mid-restore crash — fail-loud. PASS.

### 5. CLI (17.3/17.4) — ✅ verified

`mem distill status` (`mem.go:48 runMemDistill`) is read-only: it calls
`DistillLockStatus` (never acquires/releases). No lock → prints `{"project":…,
"status":"no active locks"}`; held → prints `project_id`, `locked_at`, `locked_by`,
`held`. `TestDistillStatusCLI` (no lock → lock → release → no lock) and
`TestDistillStatusCLITimestamp` (locked_at parses as RFC3339) both PASS.

### 6. Step-12 regression — ✅ no breakage

Ran the full memory-package focused set at HEAD:

```
--- PASS: TestDistillLockRowPerProject (0.19s)
--- PASS: TestDistillLockTimeoutAutoRelease (0.15s)
--- PASS: TestDistillRollbackOnFailure (0.35s)
--- PASS: TestDistillRollbackAtomic (0.34s)
--- PASS: TestDreamLockPreventsConcurrent (0.13s)   ← step 12
--- PASS: TestDreamLockTTLBoundary (0.14s)          ← step 12
--- PASS: TestDreamRollbackOnFailure (0.38s)        ← step 12
```

The transaction fix changed `dream_rollback.go` (restore/sweep are now tx-bound via the
`distillRollbackExec` seam), but `DreamRollback`'s public signature is unchanged — it
delegates to `DreamRollbackWith(ctx, pid, pre, locks, de.txExec)`, so step 12's
`TestDreamRollbackOnFailure` passes **unmodified**. The old `restoreSnapshot` /
`deleteOrphans` methods were replaced by `execSnapshot` / `deleteOrphansTx` (same SQL,
now bound to the tx). `dream_lock.go` (DreamLockService) is **byte-for-byte unchanged**
(`git diff 3edcafe..9c577eb -- dream_lock.go` is empty).

### 7. Additive-ness / tool contract — ✅ held

- `git diff 3edcafe..9c577eb --name-only` lists exactly 7 files: `mem.go`,
  `mem_distill_test.go` (new), `distill_lock.go` (new), `distill_lock_test.go` (new),
  `distill_lock_timeout_test.go` (new), `distill_rollback_test.go` (new),
  `dream_rollback.go` (the F1 tx fix). No `mcp/tools_*.go` touched → **no
  `mem_*`/`code_*`/`web_*` tool signature changed**.
- `DreamLockService` behavior is unchanged (file untouched). `DistillLockService` is a
  purely additive new wrapper; `DistillLockStatus` and `mem distill status` are new
  additive surfaces.
- **No CGO:** uses `database/sql` `BeginTx`/`ExecContext`/`QueryRowContext` (driver-
  agnostic) over the existing `modernc.org/sqlite` driver. No new import.
- **No new migration:** `distill_lock` already exists from step 12 (migration 024).

---

## Approved to merge

**yes**

All 7 checks verified against the diff and the live tree. The two things the brief
flagged as the focus — reuse correctness and the F1 transaction-wrapping fix — are both
correct and test-locked. The only residual is the pre-existing `distill_lock` /
`DreamLockService` naming duality (M1), which is a cosmetic artifact of step 12, not a
step-17 regression.
