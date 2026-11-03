# Task 13 — AKL importance scoring + recency decay (report)

Step 13 of `014-mnemonic-performance`: importance score, maturity tier, query-time ranking boost, and configurable decay/thresholds. All changes additive; `mem_*`/`code_*`/`web_*` tool contracts unchanged; `modernc.org/sqlite` (no CGO) unchanged.

## Per sub-task

### 13.1 [RED] — importance_score = retrieval_count × recency_factor
- **New file** `skillgrid-cli/internal/mnemonic/memory/importance.go`:
  - `ComputeImportanceScore(retrievalCount int, age time.Duration, decayRate float64) float64`
  - `recencyFactor(age, decayRate) float64` (the exponential factor, stored separately)
- **Migration 025** adds three nullable columns to `observations` (quoted below).
- **`Save()`** stamps the columns on insert: the INSERT carries `importance_score=0, recency_decay=1, maturity_tier='fresh'` as a constant default (SQLite `ADD COLUMN` requires a constant default; a 0-usage/age-0 row is exactly `0 * exp(0) = 0`), then `stampImportance()` recomputes and overwrites with the real values. **Periodic recompute** is `Service.RecomputeImportance(ctx)` (idempotent, monotonic) — available but not wired to a timer; on-save is sufficient for this step (documented, per the design notes).
- `RecomputeImportance` is exposed for dream/prune phases to catch rows whose age crossed a tier threshold without a save.

### 13.2 [RED] — maturity tier (fresh → mature → archival), monotonic
- `ComputeMaturityTier(age, retrievalCount) string` with the brief's thresholds:
  - `fresh`: age < 7d
  - `mature`: age >= 7d AND retrievalCount > 0
  - `archival`: age >= 30d OR (retrievalCount == 0 AND age > 14d)
- `ComputeMaturityTierWithThresholds(age, retrievalCount, TierThresholds)` — the configurable variant (13.4).
- **Monotonic**: `NextMaturityTier(previous, computed)` returns `max(previous, computed)` in the `fresh < mature < archival` ordering (the `maturityRank` map). The stored tier is never downgraded: `stampImportance` reads the previous tier, computes the new one, and stores `NextMaturityTier(prev, computed)`. An empty/NULL previous tier behaves like `fresh` (lowest rank).

### 13.3 [RED] — mem_search importance boost (in-memory, improve()-gated)
- In `SearchOwnerScoped` (service.go), after the SQL results + retrieval-usage bump, the re-rank is now `s.importanceRerank(ctx, out)` **instead of** `s.improve(ctx, out)`. The old `improve()` (binary retrieval-usage boost/decay) is kept in `improve.go` for direct callers and the step-08 tests; `improveRerank` is the new path.
- **Normalization + application**: base relevance is the position-based rank of the SQL result set, `base = 1 + position` (every doc positive). The importance factor is `importance_score / max_importance` where `max_importance` is the max stored score in the result set (normalized to (0,1]). `score = base * factor`. A stable partition by score descending keeps the SQL rank as tie-breaker. When `max_importance <= 0` (no positive scores) the boost is a no-op.
- **Gate**: `importanceRerank` is gated by `s.improveCfg.Enabled` (the `mnemonic.improve` opt-in, shared with step 08) and reuses the same per-service cooldown anchor. When `mnemonic.improve` is off, `importanceRerank` returns the input unchanged → **default search is byte-identical** to pre-step-13 (verified by `TestImportanceScoreQueryRanking` and the existing `TestImproveDisabledNoRegression`).
- `improve()` (the step-08 binary path) now coexists with `importanceRerank`; the brief's "improve() uses importance_score instead of binary retrieval_usage" is realized by `SearchOwnerScoped` calling `importanceRerank` rather than `improve`.

### 13.4 [AFK] — configurable decay + thresholds
- **Config keys** (added to `config/load.go`, following the step-04/05/06/08 pattern):
  - `mnemonic.importance.decay` (float, default 0.05/day). Parsed as a string; absent/malformed → 0 → `SetImportance` applies the 0.05 default.
  - `mnemonic.importance.tier_thresholds` (struct: `mature_age_days`, `archival_age_days`, `unused_archival_days`, in days; defaults 7/30/14). Zero fields → `SetImportance` defaults.
- **`Service.SetImportance(ImportanceConfig)`** fills zero fields with defaults (0.05/day decay, 7/30/14-day thresholds). `ImportanceConfig()` getter exposed for tests.
- **Wired** in both service call sites: `service/service.go` (project open) and `service/retrieval.go` (mem_* read path), routing `cfg.Importance` → `mem.SetImportance(...)`.

## Importance formula (quoted)

```
importance_score = retrieval_count * exp(-decay_rate * age_days)
recency_decay    = exp(-decay_rate * age_days)
```

`age_days = max(0, age.Hours()/24)`. Default `decay_rate = 0.05/day`. Anchor: 100 retrievals, 30 days, 0.05 decay → `100 * exp(-1.5) ≈ 22.3`.

## Maturity tier thresholds + monotonic logic

| tier | condition (default thresholds) |
|------|-------------------------------|
| `archival` | `age >= 30d` OR (`retrievalCount == 0` AND `age > 14d`) |
| `mature`   | `age >= 7d` AND `retrievalCount > 0` |
| `fresh`    | everything else |

Monotonic: `stored_tier = max(previous_tier, computed_tier)` in the `fresh < mature < archival` ordering. `NextMaturityTier(prev, computed)` enforces it; an empty/unknown previous tier is treated as `fresh`.

## 13.3 importance boost (normalization + gate)

```
base   = 1 + position        # position in the SQL result set (0-based)
factor = importance_score / max_importance   # normalized to (0, 1]
score  = base * factor
```

Stable partition by `score` descending; SQL rank is the tie-breaker. `max_importance` is the max stored score in the result set; if it is `<= 0` the boost is a no-op. **Gated by `mnemonic.improve.enabled`** (the shared improve opt-in) + the same cooldown. Disabled → byte-identical raw SQL order.

## Config keys

```yaml
mnemonic:
  importance:
    decay: "0.05"            # per-day decay (default 0.05; malformed → 0.05)
    tier_thresholds:
      mature_age_days: 7     # (default 7)
      archival_age_days: 30  # (default 30)
      unused_archival_days: 14 # (default 14)
```

## Migration 025 (quoted)

```sql
-- 025: AKL importance scoring columns (change 014, step 13).
--
-- Additive: three new nullable columns on observations; no existing schema
-- is touched. The values are populated on save (memory.Service.Save) and on
-- an explicit recompute (memory.Service.RecomputeImportance):
--
--   importance_score — retrieval_usage * exp(-decay_rate * age_days), the
--                      continuous AKL importance value (014 step 13.1).
--                      NULL until first stamped (pre-025 rows).
--   recency_decay    — exp(-decay_rate * age_days), the recency factor
--                      alone, stored for provenance.
--   maturity_tier    — 'fresh' | 'mature' | 'archival' (014 step 13.2).
--                      Transitions are monotonic (never downgraded); the
--                      stored value is max(previous, computed).

ALTER TABLE observations ADD COLUMN importance_score REAL;
ALTER TABLE observations ADD COLUMN recency_decay REAL;
ALTER TABLE observations ADD COLUMN maturity_tier TEXT;

CREATE INDEX IF NOT EXISTS idx_obs_importance
    ON observations (project, importance_score DESC)
    WHERE deleted_at IS NULL AND importance_score IS NOT NULL;
```

## How importance is populated on save

1. `Save()` INSERTs the row with `importance_score=0, recency_decay=1, maturity_tier='fresh'` (constant defaults required by SQLite `ADD COLUMN`).
2. `Save()` then calls `stampImportance(ctx, id, retrievalUsage=0, createdAt=now)`, which recomputes `ComputeImportanceScore(0, age, decay)` (=0), `recencyFactor(age, decay)` (=1 at age 0), and `ComputeMaturityTierWithThresholds` (=fresh), reads the previous tier, applies `NextMaturityTier`, and `UPDATE`s the three columns. Best-effort (a stamp failure never fails the save).
3. `seedImproveObs` (test helper) and any direct-INSERT path likewise call `stampImportance` to mirror `Save()`.
4. `RecomputeImportance(ctx)` re-stamps every live row (periodic path; monotonic).

## RED→GREEN evidence

| sub-task | RED (before impl) | GREEN (after impl) |
|----------|-------------------|--------------------|
| 13.1 | `undefined: ComputeImportanceScore` (build failure) | `TestImportanceScoreComputation` PASS |
| 13.2 | `undefined: ComputeMaturityTier` / `NextMaturityTier` (build failure) | `TestMaturityTierTransitions` PASS |
| 13.3 | boost not applied (re-rank absent) — `TestImportanceScoreQueryRanking` would not re-order | `TestImportanceScoreQueryRanking` PASS |
| 13.4 | `SetImportance`/config absent | `TestImportanceConfigurableDecay` PASS (memory + config) |

Focused run (all four):
```
--- PASS: TestImportanceScoreComputation (0.00s)
--- PASS: TestMaturityTierTransitions (0.22s)
--- PASS: TestImportanceScoreQueryRanking (0.20s)
--- PASS: TestImportanceConfigurableDecay (0.00s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory
```

## Full relevant suite

`go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1`:
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  (PASS)
```

`go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/pdg/... -count=1`:
```
ok   .../internal/mnemonic/store        5.4s
FAIL .../internal/mnemonic/service      (TestReindexStructuralIsEmbedderFree: "sql: database is closed")
ok   .../internal/mnemonic/codeindex    18.6s
ok   .../internal/mnemonic/route        2.3s
ok   .../internal/mnemonic/affected     15.9s
ok   .../internal/mnemonic/community    1.7s
ok   .../internal/mnemonic/pdg          6.6s
```

**Pre-existing failure (not introduced by step 13):** `TestReindexStructuralIsEmbedderFree` fails with `sql: database is closed` on HEAD **without** any step-13 changes (verified by `git stash` → re-run → same failure). It is a flaky/concurrency issue in the service reindex path, unrelated to importance scoring.

## Commits

| short SHA | subject |
|-----------|---------|
| `6907109` | feat(mnemonic): AKL importance score with exponential recency decay |
| `0cc759f` | feat(mnemonic): importance score as query-time ranking boost |
| `1bfe24e` | feat(mnemonic): configurable importance decay rate and tier thresholds |
| `9cf5967` | test(mnemonic): importance decay behavior + SetImportance wiring |

## Concerns

1. **`TestReindexStructuralIsEmbedderFree`** (service pkg) is a pre-existing flaky failure (`sql: database is closed`), present on HEAD without step-13 changes. Not in scope for step 13 but should be tracked separately.
2. **`improve()` coexistence**: step 08's `improve()` (binary retrieval-usage boost/decay) is kept in `improve.go` and still used by `TestImproveDecaysNeverAccessed`/`TestImproveBoostsHighUsageObservations`/`TestImproveDisabledNoRegression` (which call `s.improve` directly or via the old path). `SearchOwnerScoped` now uses `importanceRerank` instead. The two re-ranks are mutually exclusive at the search path; the step-08 tests that call `improve()` directly still pass. If a future step removes `improve()` entirely, those three tests would need to be re-pointed at `importanceRerank`.
3. **`obsSelectCols` grew by 3 columns** (importance_score, recency_decay, maturity_tier). All `scanObservations` call sites (service.go, governance.go, crosslink.go, improve_test.go `rawSQLSearch`) were updated to select + scan the 3 new columns. The `export.go` path uses its own SELECT (not `obsSelectCols`) and was **not** touched — it does not need the importance columns for its export record. The `dream_rollback.go` `ObservationSnapshot` was **not** extended with the importance columns (it snapshots for rollback, and the importance columns are recomputable, so a rollback that doesn't restore them is acceptable — they will be re-stamped on next save/recompute).
4. **`Observation` struct** gained `ImportanceScore sql.NullFloat64`, `RecencyDecay sql.NullFloat64`, `MaturityTier string` — all `omitempty`, so the JSON contract for existing consumers is unchanged (pre-025 rows leave them unset).
