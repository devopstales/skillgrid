## 13-importance-scoring

### Goal

AKL importance scoring + recency decay in improve() and query ranking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-13` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 08

**Files:**
- `internal/mnemonic/memory/importance.go` — CREATE: AKL importance scoring + recency decay
- `internal/mnemonic/memory/importance_test.go` — CREATE: importance scoring tests
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: importance scoring integration

**Interfaces:**
- Consumes: `retrieval_usage` column; improve() from step 08; observation age data
- Produces: `importance_score`, `maturity_tier`, `recency_decay` columns; query-time ranking boost

### Tasks

- [ ] 13.1 `[RED]` importance_score is computed from retrieval_count × recency_factor
  - [ ] 13.1.a Write failing test (`TestImportanceScoreComputation`): create observations with varying `retrieval_usage` (0, 5, 50, 100) and varying ages (1 day, 7 days, 30 days); call the importance scorer; verify importance_score increases with retrieval_count; verify older observations have lower scores (exponential decay); verify the formula is `retrieval_count * exp(-decay_rate * age_days)`
  - [ ] 13.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation'` — Expected: FAIL
  - [ ] 13.1.c Minimal implementation — create `importance.go` with `ComputeImportanceScore(retrievalCount int, age time.Duration, decayRate float64) float64` using exponential decay; add `importance_score` (FLOAT), `maturity_tier` (TEXT enum), `recency_decay` (FLOAT) columns to observations via migration; populate on save and on periodic recompute
  - [ ] 13.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation'` — Expected: PASS
  - [ ] 13.1.e Commit — `feat(mnemonic): AKL importance score with exponential recency decay`
- [ ] 13.2 `[RED]` maturity_tier transitions from fresh → mature → archival based on age + retrieval
  - [ ] 13.2.a Write failing test (`TestMaturityTierTransitions`): create a new observation (age 0) → verify tier is `fresh`; age it to 7 days with 10 retrievals → verify tier is `mature`; age it to 30 days with 0 retrievals → verify tier is `archival`; verify transitions are monotonic (no downgrade)
  - [ ] 13.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMaturityTierTransitions'` — Expected: FAIL
  - [ ] 13.2.c Minimal implementation — implement `ComputeMaturityTier(age time.Duration, retrievalCount int) string` with thresholds: `fresh` (age < 7d), `mature` (age >= 7d AND retrievalCount > 0), `archival` (age >= 30d OR retrievalCount == 0 AND age > 14d); store the tier on the observation
  - [ ] 13.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMaturityTierTransitions'` — Expected: PASS
  - [ ] 13.2.e Commit — `feat(mnemonic): maturity tier computation (fresh/mature/archival)`
- [ ] 13.3 `[RED]` mem_search applies importance score as query-time ranking boost
  - [ ] 13.3.a Write failing test (`TestImportanceScoreQueryRanking`): create 3 observations matching a query — one with high importance (score 9.5), one medium (5.0), one low (0.5); run `mem_search`; verify the high-importance observation ranks first; verify the importance score is applied as a multiplicative boost on top of the base relevance score
  - [ ] 13.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreQueryRanking'` — Expected: FAIL
  - [ ] 13.3.c Minimal implementation — in `SearchOwnerScoped`, after computing base relevance scores, multiply by a normalized importance factor (`importance_score / max_importance`); make the decay rate configurable via `mnemonic.importance.decay` config key; `improve()` uses importance score instead of binary retrieval_usage
  - [ ] 13.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreQueryRanking'` — Expected: PASS
  - [ ] 13.3.e Commit — `feat(mnemonic): importance score as query-time ranking boost`
- [ ] 13.4 `[AFK]` Configurable decay rate and thresholds
  - [ ] 13.4.a Write failing test (`TestImportanceConfigurableDecay`): set `mnemonic.importance.decay` to a high value; verify older observations lose importance faster; set it to a low value; verify older observations retain importance longer; verify default decay rate is reasonable
  - [ ] 13.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceConfigurableDecay'` — Expected: FAIL
  - [ ] 13.4.c Minimal implementation — read `mnemonic.importance.decay` from config (default 0.05 per day); pass it to `ComputeImportanceScore`; make tier thresholds configurable via `mnemonic.importance.tier_thresholds`
  - [ ] 13.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceConfigurableDecay'` — Expected: PASS
  - [ ] 13.4.e Commit — `feat(mnemonic): configurable importance decay rate and tier thresholds`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestImportanceScoreComputation\|TestMaturityTierTransitions\|TestImportanceScoreQueryRanking'` | PASS | | |
| Acceptance `@step-13` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify importance scoring does not break default search ordering | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): AKL importance scoring with recency decay and query-time ranking`

---

