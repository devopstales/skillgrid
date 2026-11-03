# Step 16 Review — Federated cross-project query (importance ranking + dedup)

**Change:** 014-mnemonic-performance, step 16 (`16-federated-query`)
**Reviewer:** fresh, no prior context
**Verdict:** `PASS WITH WARNINGS`
**Approved to merge:** yes

---

## Evidence checked (all 8 checks)

### 1. Composite formula (16.1) — VERIFIED, monotonic, correct degradation

`federatedComposite` at `service/service.go:990-998`:

```go
func federatedComposite(rank int, importance, maxImportance float64, cfg FederatedConfig) float64 {
	cfg = federatedWeights(cfg)
	rankTerm := cfg.RankWeight / (1 + float64(rank))
	impTerm := 0.0
	if maxImportance > 0 && importance > 0 {
		impTerm = cfg.ImportanceWeight * (importance / maxImportance)
	}
	return rankTerm + impTerm
}
```

- **Matches spec:** `rank_weight * (1/(1+rank)) + importance_weight * (importance / maxImportance)`. Exact.
- **Monotonic in rank:** `rankTerm = cfg.RankWeight/(1+rank)` — rank 0 → `RankWeight`, rank 1 → `RankWeight/2`, rank 2 → `RankWeight/3`. Strictly decreasing. Correct.
- **Monotonic in importance:** `importance/maxImportance` is strictly increasing in `importance` for `importance>0`. Correct.
- **maxImportance == 0** (no positive importance): the guard `if maxImportance > 0 && importance > 0` makes `impTerm = 0` for every hit → composite = `RankWeight/(1+rank)`, pure rank order. This is step-03 behavior. Correct.
- **importance == 0** (with maxImportance > 0): guard makes `impTerm = 0` for that hit → rank-only for that hit. Correct (a zero-importance hit gets no boost).
- **Weights configurable:** yes, via `cfg` (see check 6).

### 2. Dedup key (16.2) — VERIFIED, `ID+project` is the CORRECT key

`mergeRanked` at `service/service.go:1061-1084`:

```go
type dedupKey struct {
	id      int64
	project string
}
byID := map[dedupKey]int{} // obs identity -> index in `best`
...
key := dedupKey{id: r.obs.ID, project: r.obs.Project}
if idx, ok := byID[key]; ok {
	cur := best[idx]
	if composite > cur.composite ||
		(composite == cur.composite &&
			(r.obs.UpdatedAt > cur.obs.UpdatedAt ||
				(r.obs.UpdatedAt == cur.obs.UpdatedAt && r.obs.ID > cur.obs.ID))) {
		best[idx] = scored{obs: r.obs, composite: composite}
	}
	continue
}
byID[key] = len(best)
best = append(best, scored{obs: r.obs, composite: composite})
```

**Verdict: `ID+project` is correct, NOT `ID` alone.** Each store is an independent project with its own autoincrement ID space, so a bare numeric ID is ambiguous across stores. The step-03 rollback test `TestSearchObservationsAllParallelRollbackEnv` (parallel_search_test.go:121-141) seeds identical content into two stores (both get ID 1) and asserts `len(parHits) != 2` fails — i.e. **BOTH must survive**. Keying on ID alone would collapse those two distinct observations and break that contract. The implementer confirmed empirically (RED: ID-alone broke the step-03 tests, reverted to ID+project).

**Keeps the HIGHER-composite version** (not first-wins): yes — `if composite > cur.composite || (...)` replaces the stored copy with the higher-composite one, with deterministic tie-breaks (UpdatedAt desc, then ID desc). Step-03 used first-wins (`seen[key] = true; continue`); step 16 upgrades to higher-composite-wins. Correct and strictly better.

**Subtlety (see minor finding M1):** because the key is ID+project, the "higher-composite-wins" path is only reachable when the *same* observation (same ID in the *same* project) is emitted more than once. Two genuinely distinct observations that happen to share an ID across two stores are NOT collapsed — which is exactly what step-03 requires. The brief's literal phrasing "the same observation (same ID) in two different stores" is therefore only deduped in the re-emit case, not in the cross-store same-ID case. The implementer documents this trade-off honestly in the report's Concerns. This is the right call, but it means the test exercises the dedup via a synthetic double-emit (the same `storeRanked` slice fed twice), not via two real stores — see M2.

### 3. Per-store importance (16.3) — VERIFIED, column IS in the search SELECT

- `memory.Observation` has `ImportanceScore sql.NullFloat64` (`memory/service.go:281`, additive + omitempty).
- `SearchWithScope` (the per-store search behind `SearchObservationsScoped`) SELECTs `o.importance_score, o.recency_decay, o.maturity_tier, o.provenance` at `memory/service.go:573` and scans it via `scanObservations`. So each store's search returns observations with importance populated. Correct.
- `observedImportanceOf` (`service/service.go:1007-1016`) reads the stored `ImportanceScore` when positive, else falls back to `memory.ComputeImportanceScore(o.RetrievalUsage, age, 0.05)` (the compute-on-the-fly AKL fallback with the production 0.05/day decay). `ComputeImportanceScore` returns 0 for `retrievalCount <= 0` (`memory/importance.go:108-110`). So the per-project importance is applied **before** the cross-store merge. Correct.

### 4. Concurrency / -race — VERIFIED, no data race in the federated path

- Per-store searches run in goroutines (semaphore-bounded, step 03, unchanged); each appends to a **buffered channel** `results`, and a single goroutine drains it (`service/service.go:931-934`). `mergeRanked` is then called **once, in the parent goroutine** — the `seen`/`byID` map is mutated by a single goroutine, so it is thread-safe without a lock. Correct.
- `federatedWeights` is guarded by `federatedMu`; `federatedCfg()` locks, copies the struct, unlocks (`service/service.go:553-561`). `SetFederatedWeights` and `setFederatedForTest` also lock. No unsynchronized access. Correct.
- **`-race` credible:** I re-ran the 3 federated tests + all 5 step-03 `TestSearchObservationsAll*` tests under `-race` — all pass, no `WARNING: DATA RACE`. The only failing test in the full package (`TestReindexStructuralIsEmbedderFree`, `sql: database is closed`) is pre-existing and unrelated. Credible.

### 5. Backward compat (step-03 rollback) — VERIFIED, still works

- `SKILLGRID_SEARCH_PARALLEL=0` → `searchParallelEnabled()` false → `searchObservationsAllSequential` (`service/service.go:861-863`), which calls the same `mergeRanked(collected, limit, s.federatedCfg())` (line 882). The rollback path uses the federated merge. `TestSearchObservationsAllParallelRollbackEnv` passes (re-ran, 33s).
- When importance is disabled / all-zero: composite degrades to `RankWeight/(1+rank)` = pure cross-store rank order, i.e. step-03's behavior. (Not byte-identical to step-03's `(rank, UpdatedAt)` sort when composites tie — see minor M1 — but rank-order identical, which is what matters.)
- Step-03 regression tests all pass (re-ran).

### 6. Config (16.1) — VERIFIED, parses correctly, 0.5/0.5 defaults, malformed falls back

- `config/load.go`: `Federated{RankWeight, ImportanceWeight}`, `DefaultFederated()` → 0.5/0.5, wired into `DefaultIndexing()`.
- `mergeFederated` parses float strings; `strconv.ParseFloat` error OR `f <= 0` → left as default. Absent/malformed/non-positive → 0.5/0.5. Correct.
- **Zero-value `FederatedConfig` (production default) → 0.5/0.5:** yes. `Service.federatedWeights` starts at the zero value; `federatedWeights(cfg)` (`service.go:964-973`) maps `<= 0` fields to `config.DefaultFederated()`. Correct.
- `TestFederatedConfigWeights` covers default / absent-section / explicit / malformed. Correct.

### 7. Additive-ness / tool contract — VERIFIED, no `mem_*`/`code_*`/`web_*` signature change

- `SearchObservationsAll` signature is unchanged (`service.go:850`): `func (s *Service) SearchObservationsAll(ctx, query, matchMode, scope string, limit int) ([]memory.Observation, error)`. `mergeRanked` gained a `cfg` param, but it is a private helper — not a tool contract.
- The MCP `mem_search` handler (`mcp/tools_memory.go:296-316`) for `all_projects=true` calls `svc.SearchAllProjects` → `SearchObservationsAll` (`service.go:1128-1129`). The tool's parameter set and result shape are untouched. Correct.
- Purely additive: the change evolves the *merge* (composite + dedup), keeping the same entry point, the same per-store search, and the same concurrency. The `strconv` import was removed from `service.go` (no longer needed after the dedup key became a struct) — confirmed no remaining `strconv.` references and `go build` clean.

### 8. TDD discipline — VERIFIED, 3 tests cover the three sub-tasks, -race clean

- `TestFederatedQueryImportanceRanking` (16.1): 3 stores, varying importance; asserts the highest-composite ranks first and the discriminating case (a rank-1 high-importance hit outranks a rank-0 low-importance hit — something a pure rank-merge would not do). Passes.
- `TestFederatedQueryDedup` (16.2): asserts distinct same-ID observations from different stores both survive (step-03 contract), then a synthetic double-emit collapses to one survivor keeping the higher-composite copy. Passes.
- `TestFederatedQueryRespectsProjectImportance` (16.3): store A importance 9.0 vs store B importance 1.0 with stronger FTS; asserts A ranks first despite B's stronger FTS, and pins the exact composite values (1.0 vs 0.5556). Passes.
- `-race`: the tests do not set `-race` themselves (that's a `go test` flag), but I re-ran them under `-race` — clean. The report's `-race` claim is credible.

---

## Findings

### M1 (minor) — "Byte-identical to step 03" is rank-order-identical, not byte-identical, when composites tie

`mergeRanked` (service.go:1086-1094) breaks composite ties on `UpdatedAt desc`, then `ID desc`. Step 03's sort was `(rank asc, UpdatedAt desc)` with **no ID tiebreak**. When importance is all-zero the composites are `RankWeight/(1+rank)`, so two hits with the **same rank** (e.g. two different stores' #1 hits, or two hits at the same rank in one store) get an `ID desc` tiebreak that step 03 did not have. The ordering is therefore *rank-order-identical* to step 03 (a rank-0 hit is never behind a rank-1 hit) but not byte-identical in the same-rank edge case.

The comment at service.go:988-989 says "byte-identical to step 03" — that is an overclaim. The step-03 regression test only asserts hit *count* (not order), so it does not catch the tiebreak difference.

**Fix:** reword the comment to "rank-order-identical to step 03" (or "degrades to pure cross-store rank order"). No code change needed — the behavior is correct, the comment is just imprecise.

### M2 (minor) — The dedup "higher-composite-wins" path is only exercised via a synthetic double-emit, not via two real stores

Because the dedup key is ID+project, two *distinct* observations sharing an ID across two real stores are never collapsed (by design — step-03 contract). So `TestFederatedQueryDedup` cannot trigger the higher-composite-wins branch through two real stores; it synthesizes the duplicate by feeding the same `storeRanked` slice twice into `mergeRanked` directly (federated_query_test.go:391-396). That is a valid unit-level exercise of the dedup logic, but it does not prove the duplicate would arise organically from the live search path. This is acceptable for a unit test, but worth noting the "dedup across stores" story is really "dedup of re-emitted same-project rows."

**Fix (optional):** add a comment in the test (or the report) explicitly stating the organic trigger is a same-project re-emit, and that cross-store same-ID rows are intentionally kept. The report already does this in Concerns; a one-line test comment would make it self-documenting.

### m1 (nit) — `TestRawImportance` seam is a package-wide global in effect per-service

`memory.Service.TestRawImportance` is a field on each `*memory.Service` instance (memory/service.go:89-99), but `testRawImportanceMu` is a **package-level** mutex (importance.go:242) guarding a per-instance field. `seedFederatedObs` sets `h.memory.TestRawImportance = rawScore` and restores it after the save (federated_query_test.go:259-272). This is fine because the tests run sequentially per service and restore the value, but the mutex is package-wide while the state it guards is per-instance — a slight mismatch. It works (no race observed under `-race`), but a per-instance field with a package mutex is a minor design smell.

**Fix (optional):** nothing required. If the seam is ever used concurrently across multiple `*memory.Service` instances, the package mutex is over-broad (safe but slow) — acceptable for a test-only field.

### m2 (nit) — `observedImportanceOf` hardcodes the 0.05 decay in the fallback

`observedImportanceOf` (service.go:1015) uses `memory.ComputeImportanceScore(o.RetrievalUsage, age, 0.05)` with a hardcoded 0.05 decay, while the per-store `SetImportance` config could have a different `DecayRate`. The comment explains the rationale (the merge is weight-only and must not depend on a per-store config load). This is defensible, but if a store is configured with a non-default decay, the merge's fallback importance won't match the store's stamped score for zero-usage rows. In practice the stamped `ImportanceScore` is positive for any row with usage, so the fallback only fires for 0-usage rows (which score 0 anyway). Net effect is negligible.

**Fix (optional):** none required; the rationale is sound. If the config decay ever diverges from 0.05, revisit.

---

## Summary of the 8 checks

| # | Check | Result |
|---|-------|--------|
| 1 | Composite formula monotonic + degradation | PASS — exact formula, monotonic, correct zero/zero degradation |
| 2 | Dedup key ID+project + higher-composite-wins | PASS — correct key, keeps higher composite (see M1/M2) |
| 3 | Per-store importance in SELECT | PASS — column in SELECT, populated, applied pre-merge |
| 4 | Concurrency / -race | PASS — channel + single-goroutine merge, mutex-guarded weights, -race clean |
| 5 | Backward compat / step-03 rollback | PASS — sequential path uses federated merge, degrades to rank-only, regression passes |
| 6 | Config parse + defaults | PASS — 0.5/0.5 default, malformed falls back, zero-value maps to default |
| 7 | Additive / tool contract | PASS — no `mem_*` signature change, same entry point, strconv removed cleanly |
| 8 | TDD discipline + -race | PASS — 3 tests cover 16.1/16.2/16.3, -race clean |

**Findings by severity:** critical 0, major 0, minor 2 (M1, M2), nit 2 (m1, m2).

**Verdict: PASS WITH WARNINGS.** All 8 checks pass; no critical or major findings. The two minor findings are a comment imprecision (M1) and a test-coverage nuance (M2) — neither blocks merge. The composite formula, dedup key, concurrency, and backward-compat are all correct and verified empirically.

**Approved to merge: yes.**
