## 03-parallel-search

### Goal

Concurrent cross-project search with bounded goroutines.

### Out of scope / Non-Goals

- Store pooling (depends on 01), FTS trigram, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/service/service.go` — MODIFY: cached Open, parallel search
- `internal/mnemonic/service/retrieval.go` — MODIFY: parallel cross-project search

**Interfaces:**
- Consumes: cached store handles from step 01; existing `SearchObservationsScoped` per-store search
- Produces: parallel `SearchObservationsAll` with bounded goroutine pool; cross-store rank merge; `seen` dedup map preserved

### Tasks

- [ ] 03.1 `[RED]` Concurrent search completes in parallel with bounded concurrency
  - [ ] 03.1.a Write failing test (`TestSearchObservationsAllParallel`): create 10 project stores each with matching observations; call `SearchObservationsAll` and measure elapsed time; verify it completes faster than sequential execution (use a sleep-based latency marker per store); verify results are merged and deduped
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel'` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — rewrite `SearchObservationsAll` to launch one goroutine per store bounded by a semaphore of size `min(len(stores), runtime.NumCPU())`; each goroutine calls `SearchObservationsScoped`; collect results via a channel; merge using cross-store rank (highest rank across all stores wins)
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel'` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): parallel cross-project search with bounded goroutines`
- [ ] 03.2 `[RED]` Missing stores are skipped with warning, not fatal
  - [ ] 03.2.a Write failing test (`TestSearchObservationsAllMissingStoreSkipped`): create 3 project stores, delete the file for one, call `SearchObservationsAll`; verify it returns results from the 2 valid stores without error; verify a warning is logged for the missing store
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllMissingStoreSkipped'` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — in each goroutine, check store existence before searching; if missing, log a warning and return nil results (non-fatal); the merge step skips nil result sets
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllMissingStoreSkipped'` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): skip missing stores in parallel search with warning`
- [ ] 03.3 `[AFK]` Semaphore limits concurrent stores to prevent resource exhaustion
  - [ ] 03.3.a Write failing test (`TestSearchObservationsAllSemaphoreBound`): create 50 project stores; call `SearchObservationsAll` with a mock that tracks concurrent goroutine count; verify max concurrency never exceeds `runtime.NumCPU()`; verify all 50 stores are searched
  - [ ] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllSemaphoreBound'` — Expected: FAIL
  - [ ] 03.3.c Minimal implementation — use a buffered channel as semaphore (size `min(len(stores), runtime.NumCPU())`); each goroutine acquires before searching and releases after; add 5s timeout on semaphore acquire (connection pool exhausted → queue with timeout)
  - [ ] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllSemaphoreBound'` — Expected: PASS
  - [ ] 03.3.e Commit — `feat(mnemonic): add semaphore to bound parallel search concurrency`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel\|TestSearchObservationsAllMissingStoreSkipped'` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/service/` | PASS | | |
| Rollback boundary | verify sequential search still works when parallel is disabled | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): concurrent cross-project search with bounded goroutines`

---

