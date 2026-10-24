# Task 13 — AKL importance scoring + recency decay (review)

Fresh reviewer. Scope: `review-step13-scoped.diff` (1167 lines, 6 files) + the
existing `importance.go` / `improve.go` / `service.go` / `retrieve.go` /
`config/load.go` / `migrations/025_importance.sql`. All four focused tests
run locally and PASS; the full `memory`, `config`, `store` packages PASS.

## Verdict: `PASS WITH WARNINGS`

The math is correct, the tier monotonic invariant is sound, the query-time
boost is a genuine multiplicative, improve()-gated, in-memory re-rank, and the
migration is purely additive. Findings are all minor/nit: a brief deviation on
"improve() uses importance_score" (realized by replacement, not re-pointing), a
double-set of importance config per mem_* read, and an unused index. None
break a global constraint or the tool contract.

---

## Findings

### F1 — minor — `improve.go` / `importance.go`: `improve()` (step 08) is NOT updated to use importance_score
The brief 13.3 says "`improve()` uses importance score instead of binary
retrieval_usage." The implementer instead replaced `improve()` **at the call
site**: `SearchOwnerScoped` now calls `importanceRerank` (importance-based),
and the step-08 `improve()` is left in `improve.go` still using binary
`retrieval_usage` (its `improveBoost`/`improveDecay` are unchanged).

```go
// service.go:632 — the only production call site now uses importance
return s.importanceRerank(ctx, out), nil
```

The old `improve()` remains referenced only by the step-08 test
(`improve_test.go:174` calls `fx.svc.improve(ctx, raw)` directly). This is a
behaviorally-correct realization of the brief's intent (the search path is now
importance-driven), and the report documents it explicitly (report Concern #2).
But the literal deliverable — `improve()` itself using importance_score — was
not done. Fix (optional, can be deferred to a cleanup step): either re-point
`improve()` to delegate to `importanceRerank`, or fold the two and delete the
now-dead binary path once the step-08 tests are re-pointed.

### F2 — minor — `service/service.go:344` + `service/retrieval.go:256`: importance config set twice per mem_* read
Both the project-open path (`OpenProject`, service.go:344) and the per-read
path (`budgetedRetrievalHits`, retrieval.go:256) call `mem.SetImportance(...)`.
The per-read path re-opens a fresh `store` + `memory.New` (retrieval.go:223-228)
and re-reads `config.Load(root)` (retrieval.go:232), so it is self-contained and
the open-path set is effectively dead on that route. It mirrors the pre-existing
step-08 pattern (improve is set in the same two places), so it is consistent
with the codebase and harmless — just redundant. Fix (optional): set it in only
one place, or document why both are needed.

### F3 — minor — `store/migrations/025_importance.sql:15`: index is created but never used
```sql
CREATE INDEX IF NOT EXISTS idx_obs_importance
    ON observations (project, importance_score DESC)
    WHERE deleted_at IS NULL AND importance_score IS NOT NULL;
```
The query-time boost reads stored importance **in-memory**
(`observedImportance` → `o.ImportanceScore.Float64`), and no SQL query in this
step filters/orders by `importance_score`. The report itself says it is "for
prune/dream phases to pre-filter" (future steps). It is purely additive and
correct, but currently dead. Fix (optional): drop it until a prune/dream step
consumes it, or add a comment naming the consuming step.

### F4 — nit — `memory/importance.go`: `errors_IsNoRows` is a needless wrapper
```go
func errors_IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
```
Used exactly once (`stampImportance`), with an odd name. Inline the
`errors.Is(err, sql.ErrNoRows)` call or drop the helper.

### F5 — nit — `memory/service.go`: `SearchWithScope` has stray leading-whitespace changes
The diff touches `SearchWithScope`'s `ORDER BY` / `AND expires_at` lines with a
leading-tab change (e.g. `  AND (...)` → ` 	 AND (...)`, same for the
`ORDER BY` line). Whitespace-only in a Go raw string, no semantic effect, but it
is an unintended cosmetic churn in a function that is not otherwise part of step
13's behavioral surface (it only gained the 3 SELECT columns, which is needed).

---

## Evidence checked (all 9)

1. **Importance formula (13.1)** — VERIFIED CORRECT.
   `importance.go:106` `ComputeImportanceScore`:
   ```go
   if retrievalCount <= 0 { return 0 }
   if decayRate < 0 { decayRate = 0 }
   return float64(retrievalCount) * recencyFactor(age, decayRate)
   ```
   `recencyFactor` = `math.Exp(-decayRate * ageDays(age))` → exactly
   `retrieval_count * exp(-decay_rate * age_days)`. Monotonic in
   retrieval_count (linear × positive factor), strictly decreasing in age
   (exponential). `ageDays` clamps negative age to 0 (`importance.go:91`) so
   clock skew cannot amplify the score past the undiminished value. 0
   retrieval → exactly 0. Test asserts the 22.3 anchor
   (`100*exp(-0.05*30)`), strict increase in count, strict decrease in age,
   the exponential (non-linear) ratio check, and the negative-age no-amplify
   bound. PASS.

2. **Maturity tier monotonic (13.2)** — VERIFIED CORRECT.
   Thresholds match the brief (`importance.go:138`): archival `age>=30d OR
   (usage==0 AND age>14d)`; mature `age>=7d AND usage>0`; fresh else. Order of
   checks (archival first, then mature) is correct. Monotonic enforcement in
   `NextMaturityTier` (`importance.go:157`): `stored = max(previous, computed)`
   in the `fresh(0) < mature(1) < archival(2)` rank map; empty/unknown previous
   behaves as `fresh`. The direct answer to the scenario — "mature at 7d/10
   retrievals, re-evaluated at 30d with 0 NEW retrievals": computed at 30d is
   `archival` (age>=30d dominates), `max(mature, archival) = archival` → it
   DOES become archival. A hypothetical re-evaluation at 2d/100 (which computes
   `fresh`) is guarded by `max(mature, fresh) = mature` (test
   `importance_test.go:703`). PASS.

3. **Query-time boost (13.3)** — VERIFIED CORRECT.
   `importanceRerank` (`importance.go:367`): base `= 1 + position` (every doc
   positive), factor `= sc / maxScore` (normalized to (0,1], `maxScore` = max
   stored score in the result set), `score = base * factor` — a genuine
   **multiplicative** boost on the base relevance, not additive. In-memory
   re-rank via `sort.SliceStable` (SQL `ORDER BY` untouched). Gated by
   `s.improveCfg.Enabled` (the `mnemonic.improve` opt-in) + the shared
   `improveLast` cooldown anchor; when disabled it returns the input unchanged
   (byte-identical default). `maxScore <= 0` → no-op. PASS.

4. **improve() integration (13.3)** — SEE F1. The search path is now
   importance-driven (correct), but the step-08 `improve()` function itself was
   not re-pointed to importance_score (left on binary retrieval_usage, kept for
   the step-08 tests). Opt-in gate still works.

5. **Config (13.4)** — VERIFIED CORRECT.
   `mnemonic.importance.decay` parsed as a float string
   (`config/load.go:78` `strconv.ParseFloat(section.DecayRate, 64)`);
   absent/malformed/negative → left `0` → `SetImportance`/`effectiveImportance`
   apply the `0.05` default (`importance.go:71`, `183`). Tier thresholds
   (`mature_age_days`/`archival_age_days`/`unused_archival_days`) configurable,
   zero → defaults 7/30/14. Test verifies a higher decay erodes old scores
   faster and a lower one retains them longer, plus the 0.2/day ranking flip.
   PASS.

6. **Migration (13.1)** — VERIFIED CORRECT. `025_importance.sql` is three
   `ALTER TABLE ... ADD COLUMN` (all nullable, no existing schema touched) + one
   `CREATE INDEX IF NOT EXISTS`. Migration is numbered 025, correctly after
   024. No backfill: pre-025 rows stay NULL and are computed on next
   save/recompute; `observedImportance` has a compute-on-the-fly fallback for
   rows with NULL importance (so ranking never breaks on legacy rows). PASS.

7. **Population on save** — VERIFIED CORRECT. `Save()` INSERTs with constant
   defaults (`importance_score=0, recency_decay=1, maturity_tier='fresh'`,
   service.go:428) then calls `stampImportance(ctx, id, 0, now)` (service.go:441)
   which recomputes and UPDATEs. Best-effort: a tier-lookup error returns early
   and the UPDATE error is swallowed — a stamp failure never fails the save.
   `RecomputeImportance` is idempotent + monotonic. PASS.

8. **Additive-ness / tool contract** — VERIFIED. No `mem_*`/`code_*`/`web_*`
   signature changed: the `mem_search` handler (`tools_memory.go:267`) is
   unchanged; the `Observation` struct gained 3 `omitempty` fields
   (pre-025 rows leave them unset → JSON contract unchanged). The re-rank lives
   in-memory in `SearchOwnerScoped`/`SearchOwnerScopedRetrieveFTS`; the SQL
   `ORDER BY (pinned DESC, bm25)` is untouched. `TestImproveDisabledNoRegression`
   (exercises the search path with improve off) and
   `TestImportanceScoreQueryRanking` (disabled leg) confirm the default is
   byte-identical. `modernc.org/sqlite` untouched (no CGO). PASS.

9. **TDD discipline** — VERIFIED. The four tests cover (a) formula monotonic +
   exponential (TestImportanceScoreComputation), (b) tier transitions fresh/
   mature/archival + monotonic (TestMaturityTierTransitions), (c) query ranking
   high-first + multiplicative + gate (TestImportanceScoreQueryRanking), (d)
   configurable decay high-vs-low (TestImportanceConfigurableDecay). RED→GREEN
   evidence is plausible (build failures on the new symbols). All four PASS
   locally; full `memory`/`config`/`store` packages PASS. The one reported
   failure (`TestReindexStructuralIsEmbedderFree`, `sql: database is closed`)
   is in the `service` package and is documented as pre-existing/flaky, not in
   this step's changed code.

## Approved to merge: yes

The deliverable is correct and additive; the findings are refinements, not
defects. Recommend addressing F1 (re-point or remove the dead `improve()` path)
as a follow-up so step 08 and step 13 do not permanently maintain two parallel
re-rank implementations.
