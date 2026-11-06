## 20-snapshots

### Goal

Multi-version snapshots + transaction locking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-20` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01, 17

**Files:**
- `internal/mnemonic/memory/snapshots.go` — CREATE: multi-version snapshots + transaction locking
- `internal/mnemonic/memory/snapshots_test.go` — CREATE: snapshot tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `snapshot` subcommand

**Interfaces:**
- Consumes: cached store handles from step 01; DistillLock from step 17
- Produces: `snapshots` table; `Snapshot()` method; row-level locking; `mem snapshot` CLI

### Tasks

- [ ] 20.1 `[RED]` Snapshot() creates a point-in-time view of the store
  - [ ] 20.1.a Write failing test (`TestSnapshotCreatePointInTime`): create 5 observations; call `Snapshot(projectID)`; modify 2 observations and add 1; call `RestoreSnapshot(snapshotID)`; verify the store is back to the 5-observation state with original content; verify the snapshot captured the state hash
  - [ ] 20.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime'` — Expected: FAIL
  - [ ] 20.1.c Minimal implementation — create `snapshots.go` with a `snapshots` table: `id INTEGER PRIMARY KEY, project_id TEXT NOT NULL, state_hash TEXT NOT NULL, data BLOB NOT NULL, created_at TIMESTAMP`; `Snapshot(projectID)` serializes the current observations state to a BLOB, computes a hash, and stores it; `RestoreSnapshot(snapshotID)` deserializes and replaces the current state
  - [ ] 20.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime'` — Expected: PASS
  - [ ] 20.1.e Commit — `feat(mnemonic): point-in-time store snapshots`
- [ ] 20.2 `[RED]` Row-level locking prevents concurrent writes to same observation
  - [ ] 20.2.a Write failing test (`TestRowLevelLockingConcurrentWrites`): start two goroutines that both attempt to update the same observation simultaneously; verify the second write blocks until the first completes; verify no data corruption (final state is one of the two writes, not a mix); verify the lock is released after the transaction
  - [ ] 20.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRowLevelLockingConcurrentWrites'` — Expected: FAIL
  - [ ] 20.2.c Minimal implementation — add row-level locking using SQLite `BEGIN IMMEDIATE` transactions for writes; wrap each observation update in a transaction; concurrent writes to the same row are serialized by SQLite's write lock; add a 5-second timeout on lock acquisition
  - [ ] 20.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRowLevelLockingConcurrentWrites'` — Expected: PASS
  - [ ] 20.2.e Commit — `feat(mnemonic): row-level locking for concurrent writes`
- [ ] 20.3 `[AFK]` Auto-prune old snapshots to prevent storage bloat
  - [ ] 20.3.a Write failing test (`TestSnapshotAutoPrune`): create 20 snapshots; configure retention to keep only the last 5; trigger auto-prune; verify only the 5 most recent snapshots remain; verify the oldest 15 are deleted; verify the retention count is configurable
  - [ ] 20.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotAutoPrune'` — Expected: FAIL
  - [ ] 20.3.c Minimal implementation — implement `PruneSnapshots(projectID, keep int)` that deletes all but the `keep` most recent snapshots; trigger auto-prune after each new snapshot; make retention configurable via `mnemonic.snapshot.retention` (default 10)
  - [ ] 20.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotAutoPrune'` — Expected: PASS
  - [ ] 20.3.e Commit — `feat(mnemonic): auto-prune old snapshots with configurable retention`
- [ ] 20.4 `[AFK]` mem snapshot CLI creates and rolls back to snapshots
  - [ ] 20.4.a Write failing test (`TestMemSnapshotCLI`): invoke `mem snapshot create` and verify a snapshot is created with an ID; modify the store; invoke `mem snapshot restore <id>` and verify the store is restored; invoke `mem snapshot list` and verify all snapshots are listed with timestamps
  - [ ] 20.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSnapshotCLI'` — Expected: FAIL
  - [ ] 20.4.c Minimal implementation — add `mem snapshot create`, `mem snapshot restore <id>`, and `mem snapshot list` subcommands to `mem.go`; wire them to `Snapshot()`, `RestoreSnapshot()`, and a `ListSnapshots()` method
  - [ ] 20.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSnapshotCLI'` — Expected: PASS
  - [ ] 20.4.e Commit — `feat(mnemonic): add mem snapshot CLI subcommands`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSnapshotCreatePointInTime\|TestRowLevelLockingConcurrentWrites'` | PASS | | |
| Acceptance `@step-20` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify snapshots are opt-in and do not affect normal writes | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): multi-version snapshots with transaction locking`

---

