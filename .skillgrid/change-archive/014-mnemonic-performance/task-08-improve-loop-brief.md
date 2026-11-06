## 08-improve-loop

### Goal

Self-improvement feedback loop via retrieval usage.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, extraction, embedder, triple-store linkage.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-08` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: improve() feedback loop

**Interfaces:**
- Consumes: `retrieval_usage` column on observations (already tracked); triple-store linkage from step 07
- Produces: `improve()` method that re-weights observations by retrieval usage; boost/decay in `mem_search` ordering

### Tasks

- [ ] 08.1 `[RED]` High-usage observations are boosted in mem_search ordering
  - [ ] 08.1.a Write failing test (`TestImproveBoostsHighUsageObservations`): create 3 observations — one with `retrieval_usage=100`, one with `retrieval_usage=10`, one with `retrieval_usage=0`; call `improve()`; run `mem_search` for a query matching all three; verify the high-usage observation ranks first
  - [ ] 08.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations'` — Expected: FAIL
  - [ ] 08.1.c Minimal implementation — add `improve()` method on Service that reads `retrieval_usage` from all observations; compute a boost factor for observations with `retrieval_usage > threshold`; apply the boost in `mem_search` ordering (in-memory re-rank before returning results); make boost/decay rates configurable via `mnemonic.improve` config key; disable by default (opt-in)
  - [ ] 08.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations'` — Expected: PASS
  - [ ] 08.1.e Commit — `feat(mnemonic): improve() boosts high-usage observations in search`
- [ ] 08.2 `[RED]` Never-accessed observations decay in rank over time
  - [ ] 08.2.a Write failing test (`TestImproveDecaysNeverAccessed`): create 2 observations — one with `retrieval_usage=0` and age > TTL, one with `retrieval_usage=5`; call `improve()`; verify the never-accessed observation ranks below the accessed one; verify decay is proportional to age
  - [ ] 08.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDecaysNeverAccessed'` — Expected: FAIL
  - [ ] 08.2.c Minimal implementation — in `improve()`, compute a decay factor for observations with `retrieval_usage == 0` and age > TTL; reduce their rank score proportionally to age; ensure `improve()` is called before `SearchOwnerScoped` (transparent to callers)
  - [ ] 08.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDecaysNeverAccessed'` — Expected: PASS
  - [ ] 08.2.e Commit — `feat(mnemonic): improve() decays never-accessed observations`
- [ ] 08.3 `[AFK]` improve() is opt-in and does not regress when disabled
  - [ ] 08.3.a Write failing test (`TestImproveDisabledNoRegression`): disable `mnemonic.improve` via config; run `mem_search` with mixed retrieval_usage observations; verify results are ordered by the default ranking (no boost/decay applied); verify the search returns the same results as pre-improve() behavior
  - [ ] 08.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDisabledNoRegression'` — Expected: FAIL
  - [ ] 08.3.c Minimal implementation — gate `improve()` behind the `mnemonic.improve` config flag (default `false`); when disabled, skip the boost/decay computation entirely; add a cooldown period to prevent excessive re-weighting on consecutive searches
  - [ ] 08.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveDisabledNoRegression'` — Expected: PASS
  - [ ] 08.3.e Commit — `feat(mnemonic): make improve() opt-in with cooldown`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations\|TestImproveDecaysNeverAccessed'` | PASS | | |
| Acceptance `@step-08` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify improve() disabled by default | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): self-improvement feedback loop via retrieval usage`

---

