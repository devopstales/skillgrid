# Task 08 — Self-improvement feedback loop via retrieval usage

## Status: COMPLETE

## Commits

| SHA | Subject |
|-----|---------|
| `536c08f` | feat(mnemonic): improve() boosts high-usage observations in search |
| `bc0a9fc` | feat(mnemonic): improve() decays never-accessed observations |
| `69ced40` | feat(mnemonic): make improve() opt-in with cooldown |

## Per sub-task

### 08.1 — High-usage observations are boosted in mem_search ordering [RED]

**Test**: `TestImproveBoostsHighUsageObservations` (`improve_test.go:61`)
Seeds 3 observations (retrieval_usage 100/10/0) where the SQL bm25 order is
the *opposite* of the usage order (the 0-usage doc matches the most query
tokens). Asserts `SearchOwnerScoped` returns them in usage-desc order
(100 > 10 > 0), proving `improve()` re-ranked them.

**RED evidence**: test failed with `improve() undefined` (compile) and, once
the API existed, failed with `got [1 101 11], want [101 11 1]` when
`improve()` was stubbed to identity.

**GREEN evidence**: PASS after `improve()` was implemented.

### 08.2 — Never-accessed observations decay in rank over time [RED]

**Test**: `TestImproveDecaysNeverAccessed` (`improve_test.go:119`)
Seeds 3 observations: two never-accessed (retrieval_usage=0, age 20×TTL and
30×TTL) and one accessed (retrieval_usage=50, fresh). The 30×TTL doc is
seeded to lead the raw SQL ranking (more matching tokens). Asserts
`improve()` flips it behind the 20×TTL doc (decay proportional to age) and
keeps the accessed doc first (no decay for used obs).

**RED evidence**: test failed with `decay must be proportional to age: raw
[3 2 1], got [3 2 1]` (the 30×TTL doc stayed first without decay). Verified
by stubbing `improveDecay` to 0 and making the accessed doc usage=0 — the
test then failed with `accessed observation must rank first: raw [3 2 1],
got [1 2 3]`, proving decay is the differentiator.

**GREEN evidence**: PASS after decay was implemented.

### 08.3 — improve() is opt-in and does not regress when disabled [AFK]

**Test**: `TestImproveDisabledNoRegression` (`improve_test.go:196`)
Three parts:
1. **Disabled default**: a `Service` with no `SetImprove` call has
   `ImproveConfig().Enabled == false`. A search with mixed usage (0/100/10)
   returns the raw SQL order verbatim (no boost/decay) — the cold doc (0
   usage, most tokens) leads.
2. **Opt-in changes order**: re-enabling `improve()` re-ranks the same query
   (hot doc first), proving the disabled path skipped the re-rank.
3. **Cooldown**: a fresh service with `Cooldown: 1h` — the first search is
   boosted (hot doc first, usage=100 pre-bump), the second search (within
   the window) returns the raw SQL order (cold doc first, usage=1 post-bump),
   proving the cooldown blocked the re-rank.

**Config loader test**: `TestImprovementConfigDefaultOff`
(`config/load_test.go:44`) — `mnemonic.improve` defaults to `enabled: false`
(absent or no-config), opts in with `enabled: true`, and parses
`threshold`/`max_usage`/`boost_rate`/`decay_rate`/`cooldown` from YAML.

**RED evidence**: the disabled test failed with `first search must be
boosted: got [100 10 0]` before the cooldown used a fresh service (the
earlier searches had bumped the hot doc's usage, so the "first" search saw
102 instead of 100).

**GREEN evidence**: PASS after the fresh-service fix.

## Boost/decay formulas

Quoted from `improve.go:36-40`:

```
boost = min(retrieval_usage, MaxUsage) * BoostRate   (when retrieval_usage > Threshold)
decay = max(0, age - TTL) / TTL * DecayRate          (when retrieval_usage == 0)
score = baseRankScore + boost - decay
```

- `baseRankScore` = the position in the SQL result set (0 for top, +1 per
  position). This is the pre-improve rank, so the re-rank is a perturbation
  of the SQL order, not a wholesale replacement.
- `age` = `now - created_at` (parsed via `parseObsTimestamp`, RFC3339 with a
  SQLite-shape fallback).
- `TTL` = the effective default soft expiry (`s.effectiveTTL()`, the
  `mnemonic.ttl` override or `defaultMemoryTTL` = 7d).
- Boost is capped linear: `min(usage, MaxUsage) * BoostRate`. Only applies
  when `usage > Threshold` (default 5), so a single search hit (usage=1) does
  not earn a boost.
- Decay is proportional to how far the age exceeds the TTL:
  `max(0, age - TTL) / TTL * DecayRate`. Only applies when
  `retrieval_usage == 0` (never accessed). A fresh observation (age < TTL)
  or an ever-retrieved one never decays.

## Config struct (`mnemonic.improve`)

`config/load.go` — `Improvement` struct + `improvementSection` YAML mapping:

```go
type Improvement struct {
    Enabled   bool
    Threshold int
    MaxUsage  int
    BoostRate float64
    DecayRate float64
    Cooldown  time.Duration
}

type improvementSection struct {
    Enabled   bool   `yaml:"enabled"`
    Threshold int    `yaml:"threshold"`
    MaxUsage  int    `yaml:"max_usage"`
    BoostRate string `yaml:"boost_rate"`   // parsed as float
    DecayRate string `yaml:"decay_rate"`   // parsed as float
    Cooldown  string `yaml:"cooldown"`     // Go duration, e.g. "60s"
}
```

- `Enabled` defaults to `false` (opt-in). Absent section → disabled.
- Rate fields are zero by default; `SetImprove` falls back to the memory
  package defaults (`Threshold=5`, `MaxUsage=50`, `BoostRate=0.1`,
  `DecayRate=0.25`, `Cooldown=1min`).
- `boost_rate`/`decay_rate` are YAML strings (e.g. `"0.2"`) parsed via
  `strconv.ParseFloat`; a missing or malformed value leaves the field zero so
  `SetImprove` applies its default.
- `cooldown` is a Go duration string (e.g. `"60s"`); a missing or malformed
  value leaves the field zero so `SetImprove` applies its default.

The `mnemonic.improve` section is wired into both service open paths:
- `service/service.go:315` (`openProject`) — the MCP `mem_search` path.
- `service/retrieval.go:242` (`budgetedRetrievalHits`) — the `mem_context` /
  `mem_timeline` read paths.

## Cooldown mechanism

`improve.go:136-139`:

```go
now := s.improveCfg.Now()
if !s.improveLast.IsZero() && now.Sub(s.improveLast) < s.improveCfg.Cooldown {
    return results
}
s.improveLast = now
```

- `improveLast` is a `time.Time` on the `Service` struct, reset to zero by
  `SetImprove` (so a config change re-arms the loop).
- On each `improve()` call, if the last re-rank was less than `Cooldown`
  ago, the input slice is returned unchanged (no re-rank). Otherwise the
  re-rank runs and `improveLast` is stamped.
- Default cooldown is `1 * time.Minute` (`improveDefaultCooldown`). A
  sub-millisecond value (e.g. `time.Nanosecond`) effectively disables the
  gating — used by tests that issue back-to-back searches.
- This prevents excessive re-weighting on consecutive searches: within the
  window, the SQL order is returned as-is.

## Where `improve()` is called in the search path

`service.go:586` — inside `SearchOwnerScoped`, after the retrieval-usage
bump and before returning:

```go
for _, o := range out {
    s.BumpRetrievalUsage(ctx, o.ID)
}
// Self-improvement feedback loop (014 step 08): opt-in in-memory
// re-rank by retrieval usage before the results go out. A no-op
// (byte-identical order) when the mnemonic.improve config is off.
return s.improve(ctx, out), nil
```

This is the per-owner read path behind `mem_search` (the MCP tool calls
`h.Memory().SearchOwnerScoped(...)` at `mcp/tools_memory.go:341`). The
`improve()` call is transparent to callers — they get a `[]Observation` in
the same shape, just re-ordered when the loop is enabled.

The `SearchWithScope` path (the un-owned `mem_search_all` variant) does NOT
call `improve()` — it keeps the pre-improve SQL ordering. This is consistent
with the brief's scope ("apply the boost in `mem_search` ordering") and keeps
the all-projects read path unchanged.

## How the disabled case is byte-identical

`improve.go:132-133`:

```go
if s == nil || len(results) == 0 || !s.improveCfg.Enabled {
    return results
}
```

When `Enabled` is `false` (the default), `improve()` returns the input slice
**unchanged** — same order, same pointers, no copy, no score computation. The
SQL `ORDER BY COALESCE(o.pinned, 0) DESC, bm25(observations_fts)` is
untouched (the re-rank is in-memory, after the query). So the default search
behavior is byte-identical to pre-improve: the same rows, the same order, the
same usage bump, the same error handling. The only difference is the
`improve()` call itself, which is a single early-return when disabled.

`TestImproveDisabledNoRegression` proves this: a service with no `SetImprove`
call (zero-value cfg, `Enabled=false`) returns the raw SQL order verbatim for
a mixed-usage query.

## Files changed

| File | Change |
|------|--------|
| `skillgrid-cli/internal/mnemonic/memory/improve.go` | NEW — `ImproveConfig`, `SetImprove`, `ImproveConfig`, `improve()`, `improveBoost`, `improveDecay`, `parseObsTimestamp` |
| `skillgrid-cli/internal/mnemonic/memory/improve_test.go` | NEW — `TestImproveBoostsHighUsageObservations`, `TestImproveDecaysNeverAccessed`, `TestImproveDisabledNoRegression`, `seedImproveObs`, `rawSQLSearch`, `ids`, `usageOrders` helpers |
| `skillgrid-cli/internal/mnemonic/memory/service.go` | `Service` struct: `improveCfg`/`improveLast` fields; `SearchOwnerScoped`: `return s.improve(ctx, out), nil`; `EffectiveTTL()` accessor |
| `skillgrid-cli/internal/mnemonic/config/load.go` | `Improvement` struct, `improvementSection`, `mnemonic.improve` YAML key, `mergeImprovement`, `Indexing.Improvement` field |
| `skillgrid-cli/internal/mnemonic/config/load_test.go` | `TestImprovementConfigDefaultOff` |
| `skillgrid-cli/internal/mnemonic/service/service.go` | `openProject`: `mem.SetImprove(...)` from `cfg.Improvement` |
| `skillgrid-cli/internal/mnemonic/service/retrieval.go` | `budgetedRetrievalHits`: `mem.SetImprove(...)` from `cfg.Improvement` |

## Verification

| Check | Run | Result |
|-------|-----|--------|
| Focused test | `go test ./internal/mnemonic/memory/ -run 'TestImproveBoostsHighUsageObservations\|TestImproveDecaysNeverAccessed\|TestImproveDisabledNoRegression' -count=1` | PASS (3/3) |
| Config loader | `go test ./internal/mnemonic/config/ -run 'TestImprovementConfigDefaultOff' -count=1` | PASS |
| Full memory suite | `go test ./internal/mnemonic/memory/ -count=1` | PASS (10.5s) |
| Store | `go test ./internal/mnemonic/store/... -count=1` | PASS (5.0s) |
| Codeindex | `go test ./internal/mnemonic/codeindex/... -count=1` | PASS (13.2s) |
| Route | `go test ./internal/mnemonic/route/... -count=1` | PASS (2.8s) |
| Affected | `go test ./internal/mnemonic/affected/... -count=1` | PASS (12.0s) |
| Community | `go test ./internal/mnemonic/community/... -count=1` | PASS (0.9s) |
| PDG | `go test ./internal/mnemonic/pdg/... -count=1` | PASS (4.4s) |
| Service | `go test ./internal/mnemonic/service/... -count=1` | FAIL — `TestReindexStructuralIsEmbedderFree` (pre-existing, fails on clean tree too: `sql: database is closed`) |
| Rollback boundary | improve() disabled by default | PASS — `TestImproveDisabledNoRegression` + `TestImprovementConfigDefaultOff` |
| Global Constraints | additive only; no SQL ORDER BY change; no tool contract change; no CGO | held |

## Concerns

1. **`SearchWithScope` (all-projects) does not call `improve()`** — only
   `SearchOwnerScoped` does. If the all-projects `mem_search_all` path should
   also be re-ranked, `improve()` needs to be added to `SearchWithScope`
   too. The brief scoped it to `mem_search` (owner-scoped), so this is
   intentional but worth confirming.
2. **Cooldown is per-service, not per-query** — two different queries on the
   same service share the cooldown window. A burst of distinct queries within
   the window will only re-rank the first. This is the intended behavior
   (prevent excessive re-weighting) but means the re-rank is "sticky" for the
   window's duration.
3. **`TestReindexStructuralIsEmbedderFree` fails on the clean tree** —
   pre-existing, unrelated to step 08 (`sql: database is closed` in the
   reindex path). Not a regression from this change.
4. **Boost/decay magnitudes** — with the default rates (`BoostRate=0.1`,
   `DecayRate=0.25`, `MaxUsage=50`), the max boost is `50 * 0.1 = 5.0` and
   the max decay is unbounded (proportional to age/TTL). A very old
   never-accessed doc (e.g. 100×TTL) decays by `99 * 0.25 = 24.75`, which
   can push it far down the ranking. This is intentional (old unused memory
   should sink) but operators tuning `DecayRate` should be aware of the
   unbounded scale.
