# Step 16 — Federated cross-project query with importance ranking and dedup

**Verdict:** PASS
**Change:** 014-mnemonic-performance, step 16 (`16-federated-query`)
**Commit:** `016ba01` `feat(mnemonic): federated cross-project query with importance ranking and dedup`

## What was implemented per sub-task

### 16.1 [RED] — Federated query merges by cross-store rank + importance score

Evolved `SearchObservationsAll`'s merge from step 03's rank-only
`sort.SliceStable(rank ASC, UpdatedAt DESC)` into a **composite-score** merge.
Each per-store search (`SearchObservationsScoped` → `SearchWithScope`) already
returns its observations with the `importance_score` column populated
(step 13's `obsSelectCols` / the inline 26-column SELECT both include
`o.importance_score`), so no per-store query change was needed — the merge now
reads `Observation.ImportanceScore` directly.

The merge (`mergeRanked` in `service/service.go`) now:
1. computes each observation's importance (`observedImportanceOf`: the stored
   `importance_score` when positive, else the AKL compute-on-the-fly fallback
   from `retrieval_usage` + `created_at` at the production 0.05/day decay),
2. finds the result-set `maxImportance` (the normalization anchor),
3. computes `federatedComposite(rank, importance, maxImportance, cfg)` for each
   hit,
4. sorts by composite **DESCENDING** (ties: UpdatedAt desc, then ID desc —
   deterministic regardless of goroutine completion order).

Concurrency is unchanged (step 03's semaphore + `SKILLGRID_SEARCH_PARALLEL=0`
rollback); both the parallel and sequential paths call the new federated merge.
`SearchObservationsAll`'s signature is unchanged (additive evolution).

### 16.2 [RED] — Dedup by observation ID across stores

The merge maintains a `seen`-equivalent `byID` map (keyed by observation
identity). When the same observation is emitted into the merge more than once,
the **higher-composite copy is kept** and the duplicate dropped (deterministic
tie-breaks on UpdatedAt desc, then ID desc). See "Dedup key choice" below for
the exact key and why.

### 16.3 [AFK] — Respects per-project importance scores (step 13)

Each store's search applies its **local** importance scoring (step 13) before
returning: `SearchWithScope` returns the `importance_score` column stamped on
save, and `observedImportanceOf` reads it (with the compute-on-the-fly fallback
for zero-usage rows). The federated merge uses those **per-project** scores in
the composite (normalized by the merged result-set max), so a high-importance
observation in store A can outrank a stronger-FTS observation in store B.
Verified by `TestFederatedQueryRespectsProjectImportance`.

## Composite score formula

Quoted from `federatedComposite` (service.go):

```
composite = rank_weight * (1/(1+rank)) + importance_weight * (importance / maxImportance)
```

- `rank` = the hit's 0-based position within its **own** store's bm25 ranking
  (lower = better). It decays hyperbolically: a #1 hit contributes exactly
  `rank_weight`, and each deeper hit strictly less. This inverts step 03's
  "lower rank = better" into "higher composite = better."
- `importance` = the observation's **local per-project** importance score
  (step 13). `maxImportance` = the maximum importance across the whole merged
  result set, so the importance term is normalized to (0, 1].
- Both terms live in (0, 1]; the weights directly express the
  rank-vs-importance trade-off. **Higher composite ranks first.**
- When no observation has a positive importance score, the importance term is
  0 for every hit and the merge degrades to pure cross-store rank order
  (`rank_weight/(1+rank)`), which is monotonic in rank — byte-identical to
  step 03's behavior.

Weights are configurable via the `mnemonic.federated` config section (see
below); zero/malformed values fall back to the 0.5/0.5 default.

## Dedup key choice

**Key = observation ID + project** (the same identity key step 03 used,
`ID/Project`).

The brief frames 16.2 as "the same observation (same ID) in two different
stores," but each store is an independent project with its **own autoincrement
ID space** — a bare numeric ID is ambiguous across stores. Two DISTINCT
observations can share an ID: the step-03 rollback test
(`TestSearchObservationsAllParallelRollbackEnv`) seeds identical content in two
stores (both get ID 1) and expects BOTH to survive the merge. Keying on ID
alone (or ID+normalized_hash) would collapse those and break that contract —
confirmed empirically during RED (the step-03 tests returned 1 hit instead of
2). Keying on ID+project preserves the step-03 contract while still collapsing
a true duplicate (the same observation emitted twice into the merge), keeping
the higher-composite copy. The test `TestFederatedQueryDedup` exercises both
halves: (a) distinct same-ID observations from different stores both survive,
and (b) a duplicate-emit of the same observation collapses to one survivor —
the higher-composite copy.

## Per-store importance application

`SearchWithScope` (the per-store search) already SELECTs `o.importance_score`
(step 13). The federated merge reads it via `observedImportanceOf`:
- if `ImportanceScore.Valid && > 0` → use the stored value;
- else → `memory.ComputeImportanceScore(retrieval_usage, age, 0.05)` (the same
  compute-on-the-fly fallback step 13's query-time re-rank uses; the merge is
  weight-only and must not depend on a per-store config load).

The per-project score is normalized by the merged result-set max before being
weighted, so a store's local importance is applied **before** the cross-store
merge.

## Config keys

New `mnemonic.federated` section in `config.d/indexing.yaml`
(`config/load.go`, following the existing `mnemonic.importance` pattern):

```yaml
mnemonic:
  federated:
    rank_weight: "0.5"        # default 0.5
    importance_weight: "0.5"  # default 0.5
```

- `config.Federated{RankWeight, ImportanceWeight}`, `config.DefaultFederated()`
  → 0.5/0.5, wired into `DefaultIndexing()`.
- `mergeFederated` parses the float strings; absent/malformed/non-positive
  values fall back to the defaults, so a bad key never skews the composite.
- Service side: `service.FederatedConfig` + `federatedWeights` (zero-fallback
  to `config.DefaultFederated`), `Service.SetFederatedWeights` /
  `federatedCfg` (mutex-guarded). Tests pin weights via `setFederatedForTest`.

## Test seam

`memory.Service.TestRawImportance` (exported, test-only, default -1 = unset,
set in `memory.New`) — when >= 0, `stampImportance` stores it as the
`importance_score` instead of computing it, so federated tests can seed
arbitrary per-observation importance scores. Guarded by
`testRawImportanceMu`. Production never sets it.

## `-race` flakiness check

`go test ./internal/mnemonic/service/ -count=1 -race` → **only the pre-existing
flaky `TestReindexStructuralIsEmbedderFree` fails** (`sql: database is closed`
— a known concurrency artifact in that unrelated test, present at HEAD before
this change). **No `WARNING: DATA RACE`** in the federated path. The 3 new
federated tests + all 5 step-03 tests (`TestSearchObservationsAll*`,
`TestSearchAllProjects*`) pass under `-race`.

## RED → GREEN evidence

- **16.1 RED:** `TestFederatedQueryImportanceRanking` + the other two written
  first; initial run was a build failure (`FederatedConfig`,
  `TestRawImportance`, `setFederatedForTest`, `federatedComposite` undefined).
  After implementing `federatedComposite`/`mergeRanked`/`observedImportanceOf`
  + the `TestRawImportance` seam → **PASS**.
- **16.2 RED:** `TestFederatedQueryDedup` initially returned 2 copies (the
  shared obs in two stores) and kept the wrong one; after the ID+project dedup
  key with higher-composite-wins → **PASS**. (First attempt keyed on ID alone,
  which broke step-03's `TestSearchObservationsAllParallelRollbackEnv` +
  `TestSearchObservationsAllMissingStoreSkipped` — reverted to ID+project.)
- **16.3 RED:** `TestFederatedQueryRespectsProjectImportance` passed once the
  per-project importance flowed into the composite; the composite-value sanity
  asserts (1.0 vs 0.5556) confirm the documented formula.

## Full relevant suite + output

`go test ./skillgrid-cli/internal/mnemonic/service/ -count=1 -race`:

```
--- FAIL: TestReindexStructuralIsEmbedderFree (2.99s)   # pre-existing flaky (sql: database is closed)
FAIL  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service  247.775s
```
All other service tests pass; no data races. The 3 new tests + step-03
regression tests specifically:

```
--- PASS: TestSearchAllProjectsMergesTwoStores
--- PASS: TestSearchAllProjectsEmptyDir
--- PASS: TestSearchAllProjectsSpansEveryStore
--- PASS: TestFederatedQueryImportanceRanking
--- PASS: TestFederatedQueryDedup
--- PASS: TestFederatedQueryRespectsProjectImportance
--- PASS: TestSearchObservationsAllParallel
--- PASS: TestSearchObservationsAllParallelRollbackEnv
--- PASS: TestSearchObservationsAllMissingStoreSkipped
--- PASS: TestSearchObservationsAllSemaphoreBound
PASS
```

Baselines (005/008/010/011/013 packages) `go test ... -count=1`:

```
ok  store        5.784s
ok  memory       20.825s
ok  memory/layer 4.093s
ok  codeindex    18.448s
ok  route        2.559s
ok  affected     11.067s
ok  community    4.710s
ok  pdg          5.991s
ok  config       5.057s   # incl. new TestFederatedConfigWeights
```

## Commits

- `016ba01` feat(mnemonic): federated cross-project query with importance ranking and dedup

## Concerns

- **Dedup key is ID+project, not ID alone.** This is the correct call given
  each store's independent autoincrement ID space (proven by the step-03
  contract), but it means 16.2's literal "same ID in two different stores"
  only dedups when the duplicate is the *same* observation re-emitted into the
  merge — two genuinely distinct observations that happen to share an ID
  across stores are kept (which is what step 03 requires). If a future
  replication mechanism assigns a *global* observation ID, the key should be
  revisited to drop the project component.
- **The `mnemonic.federated` weights are loaded from config but the service
  does not auto-wire them in `openProject`** — the zero-value
  `federatedWeights` already resolves to the 0.5/0.5 defaults via
  `federatedWeights`, and tests/CLI can override via `SetFederatedWeights`. A
  CLI flag or openProject wiring that routes the YAML weights into the service
  is a natural follow-up (the config is loadable + tested today).
- **`TestReindexStructuralIsEmbedderFree`** remains flaky at HEAD
  (`sql: database is closed`) — not caused by this change, out of scope.