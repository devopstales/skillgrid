## 17-distill-lock

### Goal

DistillLockService + DistillRollback on failure.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-17` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 09, 12

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: DistillLockService
- `internal/mnemonic/memory/dream_lock_test.go` — MODIFY: DistillLock + rollback tests

**Interfaces:**
- Consumes: DreamLockService from step 12; session promotion from step 09
- Produces: `DistillLockService` for distillation; `DistillRollback` on failure; lock status CLI

### Tasks

- [ ] 17.1 `[RED]` DistillLockService uses distill_lock row per project
  - [ ] 17.1.a Write failing test (`TestDistillLockRowPerProject`): call `DistillLockService.Acquire(projectA)`; verify a `distill_lock` row is created for projectA with a timestamp; call `Acquire(projectB)`; verify a separate row for projectB; call `Acquire(projectA)` again; verify it detects the existing lock; call `Release(projectA)`; verify the row is removed
  - [ ] 17.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject'` — Expected: FAIL
  - [ ] 17.1.c Minimal implementation — ensure `DistillLockService` (from step 12) uses a `distill_lock` table with `project_id TEXT PRIMARY KEY, locked_at TIMESTAMP, locked_by TEXT` columns; `Acquire` does an `INSERT OR FAIL` to prevent concurrent locks; `Release` does a `DELETE`
  - [ ] 17.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject'` — Expected: PASS
  - [ ] 17.1.e Commit — `feat(mnemonic): DistillLockService with per-project lock rows`
- [ ] 17.2 `[RED]` DistillRollback reverts observations to pre-distill state on failure
  - [ ] 17.2.a Write failing test (`TestDistillRollbackOnFailure`): capture pre-distill state (all observations); start a distillation; simulate a mid-distill failure; call `DistillRollback`; verify all observations match the pre-distill state exactly; verify the lock is released after rollback; verify no partial changes remain
  - [ ] 17.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillRollbackOnFailure'` — Expected: FAIL
  - [ ] 17.2.c Minimal implementation — implement `DistillRollback(ctx, projectID, snapshots []ObservationSnapshot)` that restores each observation to its pre-distill state from the snapshot list; wrap in a transaction for atomicity; release the distill lock after rollback completes
  - [ ] 17.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillRollbackOnFailure'` — Expected: PASS
  - [ ] 17.2.e Commit — `feat(mnemonic): DistillRollback reverts to pre-distill state on failure`
- [ ] 17.3 `[RED]` Lock timeout is 5 minutes with auto-release on error
  - [ ] 17.3.a Write failing test (`TestDistillLockTimeoutAutoRelease`): acquire a distill lock; set the lock timestamp to 6 minutes ago (simulate staleness); attempt to acquire the lock again; verify the stale lock is auto-released and the new acquisition succeeds; verify a lock that is only 1 minute old is not released
  - [ ] 17.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockTimeoutAutoRelease'` — Expected: FAIL
  - [ ] 17.3.c Minimal implementation — in `Acquire`, check if an existing lock has `locked_at < now - 5min`; if so, delete the stale lock and proceed with the new acquisition; log a warning for stale lock auto-release; ensure 5-minute timeout is configurable via `mnemonic.distill.lock_timeout`
  - [ ] 17.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockTimeoutAutoRelease'` — Expected: PASS
  - [ ] 17.3.e Commit — `feat(mnemonic): distill lock 5-minute timeout with auto-release`
- [ ] 17.4 `[AFK]` Lock status visible via CLI
  - [ ] 17.4.a Write failing test (`TestDistillStatusCLI`): acquire a distill lock for a project; invoke the distill status CLI; verify it shows the lock is held with timestamp and locked_by; release the lock; verify the CLI shows no active lock
  - [ ] 17.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestDistillStatusCLI'` — Expected: FAIL
  - [ ] 17.4.c Minimal implementation — add `mem distill status` subcommand to `mem.go`; query the `distill_lock` table; display active locks with project_id, locked_at, and locked_by; show "no active locks" when the table is empty
  - [ ] 17.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestDistillStatusCLI'` — Expected: PASS
  - [ ] 17.4.e Commit — `feat(mnemonic): add mem distill status CLI subcommand`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDistillLockRowPerProject\|TestDistillRollbackOnFailure\|TestDistillLockTimeoutAutoRelease'` | PASS | | |
| Acceptance `@step-17` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify lock timeout is 5 minutes with auto-release | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): DistillLockService with rollback and 5-minute timeout`

---

