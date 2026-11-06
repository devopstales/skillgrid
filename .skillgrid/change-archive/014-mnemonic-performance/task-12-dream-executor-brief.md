## 12-dream-executor

### Goal

consolidate/synthesize/prune distillation decomposition.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-12` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 04, 08

**Files:**
- `internal/mnemonic/memory/dream.go` — CREATE: DreamExecutor (consolidate/synthesize/prune); DreamLockService; DreamRollback
- `internal/mnemonic/memory/dream_test.go` — CREATE: dream executor tests
- `internal/mnemonic/memory/dream_lock_test.go` — CREATE: DreamLock + rollback tests

**Interfaces:**
- Consumes: TTL defaults from step 04; improve() scoring from step 08; AKL importance scoring
- Produces: `DreamExecutor` with `consolidate()`, `synthesize()`, `prune()`; `DreamLockService`; `DreamRollback`

### Tasks

- [ ] 12.1 `[RED]` consolidate() merges facts from multiple observations into coherent knowledge
  - [ ] 12.1.a Write failing test (`TestDreamConsolidate`): create 5 observations with overlapping facts about the same topic; call `DreamExecutor.consolidate()`; verify the overlapping facts are merged into a single coherent observation; verify no fact is lost in the merge; verify the source observations are marked as consolidated
  - [ ] 12.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate'` — Expected: FAIL
  - [ ] 12.1.c Minimal implementation — create `dream.go` with `DreamExecutor` struct; implement `consolidate(ctx, observations []Observation) (ConsolidatedResult, error)` that groups observations by topic and merges overlapping facts using LLM summarization; mark source observations as consolidated (set a status flag)
  - [ ] 12.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate'` — Expected: PASS
  - [ ] 12.1.e Commit — `feat(mnemonic): DreamExecutor consolidate phase`
- [ ] 12.2 `[RED]` synthesize() creates higher-level summaries from lower-tier memories
  - [ ] 12.2.a Write failing test (`TestDreamSynthesize`): create L0/L1 session summaries; call `DreamExecutor.synthesize()`; verify a higher-level L2 summary is created that captures the key points from the lower-tier memories; verify the summary references the source sessions
  - [ ] 12.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamSynthesize'` — Expected: FAIL
  - [ ] 12.2.c Minimal implementation — implement `synthesize(ctx, memories []Memory) (Summary, error)` that takes lower-tier memories and creates a higher-level summary via LLM; store the summary as a new observation with a higher tier level
  - [ ] 12.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamSynthesize'` — Expected: PASS
  - [ ] 12.2.e Commit — `feat(mnemonic): DreamExecutor synthesize phase`
- [ ] 12.3 `[RED]` prune() removes low-importance observations based on AKL scoring
  - [ ] 12.3.a Write failing test (`TestDreamPrune`): create 10 observations with varying importance scores (high, medium, low); call `DreamExecutor.prune()`; verify only low-importance observations are soft-deleted; verify high and medium importance observations remain; verify pruning respects the AKL maturity tier
  - [ ] 12.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamPrune'` — Expected: FAIL
  - [ ] 12.3.c Minimal implementation — implement `prune(ctx, threshold float64) (PrunedResult, error)` that queries observations with importance_score below threshold and maturity_tier `archival`; soft-deletes them (set `deleted_at`); returns a count of pruned observations
  - [ ] 12.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamPrune'` — Expected: PASS
  - [ ] 12.3.e Commit — `feat(mnemonic): DreamExecutor prune phase`
- [ ] 12.4 `[RED]` DreamLockService prevents concurrent distillation per project
  - [ ] 12.4.a Write failing test (`TestDreamLockPreventsConcurrent`): acquire a dream lock for project A; attempt to acquire another dream lock for project A; verify the second acquisition fails (or blocks); acquire a lock for project B; verify it succeeds; release the project A lock; verify project A can be locked again
  - [ ] 12.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamLockPreventsConcurrent'` — Expected: FAIL
  - [ ] 12.4.c Minimal implementation — create `DreamLockService` using a `distill_lock` row per project in the store; `Acquire(projectID)` inserts or updates the lock row with a timestamp; if the lock is held and older than 5 minutes, auto-release; `Release(projectID)` removes the lock row
  - [ ] 12.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamLockPreventsConcurrent'` — Expected: PASS
  - [ ] 12.4.e Commit — `feat(mnemonic): DreamLockService with 5-minute timeout`
- [ ] 12.5 `[RED]` DreamRollback reverts observations to pre-distill state on failure
  - [ ] 12.5.a Write failing test (`TestDreamRollbackOnFailure`): start a dream (consolidate); capture the pre-distill state; simulate a failure during synthesize; call `DreamRollback`; verify all observations are restored to their pre-distill state; verify the lock is released after rollback
  - [ ] 12.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamRollbackOnFailure'` — Expected: FAIL
  - [ ] 12.5.c Minimal implementation — implement `DreamRollback(ctx, projectID, preDistillState []ObservationSnapshot)` that restores observations to their pre-distill state from the captured snapshot; wrap consolidate/synthesize/prune in a transaction; on error, call rollback and release the lock
  - [ ] 12.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamRollbackOnFailure'` — Expected: PASS
  - [ ] 12.5.e Commit — `feat(mnemonic): DreamRollback reverts on failure`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDreamConsolidate\|TestDreamRollbackOnFailure'` | PASS | | |
| Acceptance `@step-12` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify DreamLock timeout is 5 minutes with auto-release | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): DreamExecutor with consolidate/synthesize/prune, DreamLock, and rollback`

---

