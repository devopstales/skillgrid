## 16-federated-query

### Goal

Evolve parallel search into federated cross-project query with importance ranking.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-16` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 03, 13

**Files:**
- `internal/mnemonic/service/service.go` — MODIFY: federated query
- `internal/mnemonic/service/retrieval.go` — MODIFY: parallel → federated cross-project search

**Interfaces:**
- Consumes: parallel search from step 03; importance scoring from step 13
- Produces: federated query pipeline with importance ranking and dedup; `mem search all_projects=true` uses federated pipeline

### Tasks

- [ ] 16.1 `[RED]` Federated query merges results by cross-store rank + importance score
  - [ ] 16.1.a Write failing test (`TestFederatedQueryImportanceRanking`): create 3 project stores, each with observations of varying importance scores (high, medium, low); run `SearchObservationsAll` (federated mode); verify results are ranked by a composite score combining cross-store rank and importance; verify the highest composite score ranks first
  - [ ] 16.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking'` — Expected: FAIL
  - [ ] 16.1.c Minimal implementation — evolve `SearchObservationsAll` into a federated pipeline: each store returns `{observations, importance_scores, store_id}`; the merge pipeline computes a composite score = `cross_store_rank_weight * rank + importance_weight * importance_score`; sort by composite score descending; make weights configurable
  - [ ] 16.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking'` — Expected: PASS
  - [ ] 16.1.e Commit — `feat(mnemonic): federated query with importance-based merge ranking`
- [ ] 16.2 `[RED]` Dedup by observation ID across stores
  - [ ] 16.2.a Write failing test (`TestFederatedQueryDedup`): create the same observation (same ID) in two different stores; run `SearchObservationsAll`; verify the observation appears only once in the results; verify the dedup preserves the higher-ranked version; verify the `seen` map is used for dedup
  - [ ] 16.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryDedup'` — Expected: FAIL
  - [ ] 16.2.c Minimal implementation — in the federated merge, maintain a `seen` map keyed by observation ID; when a duplicate is found, keep the version with the higher composite score; skip the duplicate in the final result set
  - [ ] 16.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryDedup'` — Expected: PASS
  - [ ] 16.2.e Commit — `feat(mnemonic): federated query dedup by observation ID`
- [ ] 16.3 `[AFK]` Federated query respects per-project importance scores from step 13
  - [ ] 16.3.a Write failing test (`TestFederatedQueryRespectsProjectImportance`): create 2 stores — store A has an observation with importance 9.0, store B has an observation with importance 1.0 but higher FTS relevance; run federated query; verify the composite ranking balances both factors; verify per-project importance is applied before cross-store merge
  - [ ] 16.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryRespectsProjectImportance'` — Expected: FAIL
  - [ ] 16.3.c Minimal implementation — ensure each store's search applies its local importance scoring (from step 13) before returning results; the federated merge uses the per-project importance scores in the composite ranking; document the weight configuration
  - [ ] 16.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryRespectsProjectImportance'` — Expected: PASS
  - [ ] 16.3.e Commit — `feat(mnemonic): federated query respects per-project importance scores`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -run 'TestFederatedQueryImportanceRanking\|TestFederatedQueryDedup'` | PASS | | |
| Acceptance `@step-16` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/service/` | PASS | | |
| Rollback boundary | verify parallel search fallback works when federated is disabled | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): federated cross-project query with importance ranking and dedup`

---

