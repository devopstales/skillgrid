# Task 08 — Self-improvement feedback loop: review

Fresh reviewer, no prior context. Scope: the 5-file scoped diff
(`config/load.go` + test, `memory/improve.go` [NEW], `memory/improve_test.go`
[NEW], `memory/service.go`). Verdict drives against the 9 mandated checks.

## Verdict

**PASS WITH WARNINGS**

The boost/decay math, the opt-in default, the in-memory re-rank, the cooldown,
determinism, TDD discipline, and the tool contract are all correct and verified
by execution. The findings are **consistency** problems, not regressions:
(a) the retrieval-path `SetImprove` wiring is a no-op that contradicts its own
comment, and (b) the scoped diff omits the two wiring files the report
describes as in-scope. Neither changes the default (disabled) byte-identity —
that property holds regardless.

## Findings

### 1. MAJOR — `SetImprove` in the retrieval path is a no-op; comment contradicts behavior
- **Where:** `skillgrid-cli/internal/mnemonic/service/retrieval.go:239-252`
  (comment + `mem.SetImprove(...)`), with the FTS leg at
  `skillgrid-cli/internal/mnemonic/memory/retrieve.go:225-257`.
- **What:** The comment at `retrieval.go:239-242` claims *"the mem_* read paths
  (mem_context / mem_timeline) also honor the mnemonic.improve opt-in so the
  re-rank is consistent across every search surface."* But the ranked output
  for the FTS fact-mode leg is produced by
  `rrfFallbackOwnerScopedFTS` → `s.BlendedSearch(ctx, query, matchMode, "", Vector{},
  limit*3)` (`retrieve.go:233`). `improve()` is only ever called from
  `SearchOwnerScoped` (`service.go:586`). `BlendedSearch` and the RRF/vector
  legs never pass their ranked slice through `improve()`, so the
  `SetImprove` wired here has **zero effect** on the ordering that
  `mem_context`/`mem_timeline` (fact mode) return. The re-rank is NOT
  "consistent across every search surface" as the comment promises.
  (Bootstrap mode, `retrieve.go:118-119`, ranks `personas` — no
  `retrieval_usage` at all — so it is unaffected too.)
- **Fix:** Either (a) call `improve()` on the `visible` slice inside
  `rrfFallbackOwnerScopedFTS` before `layerHitsFromObservations` (note: the
  leg uses `limit*3`, so the re-rank must apply to the full pre-truncate
  slice), or (b) delete the `SetImprove` call in `retrieval.go` and rewrite
  the comment to say only the `mem_search` (owner-scoped FTS) path re-ranks.
  Do not ship a dead config hook with a claim it satisfies.

### 2. MINOR — scoped diff omits the wiring files the report lists as in-scope
- **Where:** `review-step08-scoped.diff` (5 `diff --git` headers) vs
  `task-08-improve-loop-report.md:216-217` ("Files changed" table).
- **What:** The report's table lists
  `skillgrid-cli/internal/mnemonic/service/service.go` (`openProject`) and
  `service/retrieval.go` (`budgetedRetrievalHits`) as changed. Neither file is
  in the scoped diff. Both ARE committed (git shows both at `69ced40`, clean
  tree) and both compile (`go build ./...` = exit 0), so the feature is whole
  and coherent on the working tree. But the reviewer is told to judge "the
  scoped diff (5 files)," and the wiring that makes the feature reachable from
  `openProject` is invisible in it. A reviewer diffing only the scoped file
  would see `improve()` defined and called in `SearchOwnerScoped`, but would
  never see `SetImprove` ever invoked from config → could wrongly conclude the
  loop is permanently disabled.
- **Fix:** Regenerate the scoped diff to include `service/service.go` and
  `service/retrieval.go` (the two `SetImprove` call sites), or annotate the
  brief/report that these two wiring hunks are deliberately out of the
  "5-file" review scope.

### 3. MINOR — uncommitted working-tree drift on a reviewed file
- **Where:** `git diff skillgrid-cli/internal/mnemonic/memory/improve.go`.
- **What:** `improve.go` has an uncommitted change (comment-only: the
  `Cooldown` doc line expanded to mention `time.Nanosecond` disabling the
  gating). The reviewed diff captures the committed version; the on-disk
  version differs. Harmless (comment only, tests still pass), but the
  review-target and the working tree are not the same bytes.
- **Fix:** Commit the comment tweak (or fold it into the step-08 commit) so
  the reviewed state matches `HEAD`.

### 4. NIT — decay magnitude is unbounded in age/TTL
- **Where:** `improve.go:180-194` (`improveDecay`); formula
  `float64(excess)/float64(ttl) * DecayRate`.
- **What:** Decay grows linearly with `age - TTL` with no cap, so a very old
  never-accessed doc (e.g. 100×TTL) decays by `99 * DecayRate` = `24.75` at
  the default `DecayRate=0.25`. Boost is capped (max `MaxUsage * BoostRate`
  = 5.0), but decay is not — an old cold doc can be pushed arbitrarily far
  down. The implementer already flagged this as intentional in the report
  (Concern 4); it is correct behavior, just worth an operator note in the
  config docs (that `DecayRate` is an *unbounded* scale).
- **Fix:** None required. Optionally add a `maxDecay` cap for symmetry with
  the boost cap, or document the unbounded scale.

### 5. NIT — `baseRankScore` is absolute position, so re-rank strength scales with result size
- **Where:** `improve.go:145` (`scores[i] = float64(i)`).
- **What:** The base rank is the absolute 0-based position, not a normalized
  score. A boost of 5.0 can move a doc at most ~5 positions, but because
  higher-indexed positions start with a larger penalty, the *effective*
  re-ranking power of the same boost differs between a 3-result page and a
  100-result page. This is a deliberate perturbation-of-SQL-order design (and
  the tests confirm it), but it means "boost high usage" is position-relative,
  not an absolute relevance injection. Acceptable and documented; noting for
  completeness.
- **Fix:** None required.

## Evidence checked (the 9 mandated checks)

1. **Opt-in default — BYTE-IDENTICAL when disabled** ✅ **VERIFIED.**
   Gate at `improve.go:134`:
   ```go
   if s == nil || len(results) == 0 || !s.improveCfg.Enabled {
       return results
   }
   ```
   When `Enabled == false` (the default — zero-value `improveCfg` for a
   directly-constructed `Service`, and `Enabled:false` from
   `config.Improvement` when the `mnemonic.improve` section is absent,
   `config/load.go:88-101` `mergeImprovement` + `improvementSection`
   `yaml:"improve"`), `improve()` returns the input slice **unchanged** —
   same order, same pointers, no copy, no score compute, no cooldown stamp.
   `SearchOwnerScoped` (`service.go:580-586`) does the usage bump first, then
   `return s.improve(ctx, out), nil` — the bump is unchanged and the
   re-rank is a no-op. The SQL `ORDER BY COALESCE(o.pinned, 0) DESC,
   bm25(observations_fts)` (`service.go:568`) is untouched. Proven by
   `TestImproveDisabledNoRegression` (asserts mixed-usage query returns the
   raw SQL order `[0, 100, 10]` verbatim when disabled) and
   `TestImprovementConfigDefaultOff` (config loader defaults the section
   off, absent or no-file). I ran both: PASS. **No regression on the default
   path.**

2. **In-memory re-rank, NOT SQL** ✅ **VERIFIED.** No change to the SQL
   `ORDER BY` in `SearchOwnerScoped` (line 568 is identical to pre-improve).
   `improve()` is applied *after* the query, *after* the usage bump:
   ```go
   for _, o := range out { s.BumpRetrievalUsage(ctx, o.ID) }  // service.go:580-581
   return s.improve(ctx, out), nil                            // service.go:586
   ```
   It is a second in-memory `sort.SliceStable` over the SQL-ranked slice
   (`improve.go:154`). Correct per the brief ("IN-MEMORY re-rank, NOT SQL
   ORDER BY change").

3. **Boost math** ✅ **VERIFIED monotonic + bounded + threshold-gated.**
   `improve.go:164-173`:
   ```go
   if usage <= Threshold { return 0 }
   c := usage; if c > MaxUsage { c = MaxUsage }
   return float64(c) * BoostRate
   ```
   Monotonic in `retrieval_usage` (strictly increasing until the `MaxUsage`
   cap, then flat). Bounded: max = `MaxUsage * BoostRate` = 5.0 at defaults —
   cannot overflow or dominate unbounded. Threshold (`default 5`) is
   configurable and gated: `usage <= Threshold` → 0 boost, so a single search
   hit (usage=1) earns nothing. The threshold/rate/cap are all in
   `ImproveConfig` (see check 6).

4. **Decay math** ✅ **VERIFIED: only usage==0 AND age>TTL; proportional to age; fresh obs don't decay.**
   `improve.go:180-194`:
   ```go
   if usage != 0 || ttl <= 0 { return 0 }        // never-accessed only
   created, err := parseObsTimestamp(createdAt)
   if err != nil || created.IsZero() { return 0 }
   age := now.Sub(created)
   excess := age - ttl
   if excess <= 0 { return 0 }                    // age < TTL → NO decay
   return float64(excess) / float64(ttl) * DecayRate
   ```
   Applied **only** when `retrieval_usage == 0` AND `age > TTL`. A
   never-accessed-but-recent obs (`age < TTL`) returns 0 (no decay) — the
   `excess <= 0` guard. An ever-retrieved obs (`usage > 0`) returns 0. Decay
   is proportional to `age - TTL` (linear, older → larger). Proven by
   `TestImproveDecaysNeverAccessed` (the 30×TTL doc decays more than the 20×TTL
   doc, flipping the raw SQL order). I ran it: PASS.

5. **Cooldown** ✅ **VERIFIED: per-service; first search is NOT blocked; consecutive-within-window are.**
   `improve.go:137-141`:
   ```go
   now := s.improveCfg.Now()
   if !s.improveLast.IsZero() && now.Sub(s.improveLast) < s.improveCfg.Cooldown {
       return results
   }
   s.improveLast = now
   ```
   `improveLast` is a per-`Service` `time.Time`, reset to zero by
   `SetImprove` (`improve.go:111`). **The first search is NOT blocked** — the
   `!s.improveLast.IsZero()` guard means a zero-valued anchor passes through
   and the re-rank runs. This directly answers the brief's concern ("Could it
   cause the FIRST search to not apply improve()?") — no, the first search
   re-ranks. Within the window, subsequent searches return the raw SQL order
   unchanged (no re-rank, no re-stamp). It is **per-service, not per-query**
   (two different queries on the same service share the window — the
   implementer flagged this as intended in Concern 2). Proven by
   `TestImproveDisabledNoRegression`'s cooldown section: `first[0].RetrievalUsage
   == 100` (boosted) and `second[0].RetrievalUsage == 1` (raw SQL order, cold
   doc first), with `ids(first) != ids(second)`. I ran it: PASS.

6. **Config struct** ✅ **VERIFIED: full struct, all rates configurable, sane defaults, malformed → default (no crash).**
   `config/load.go:22-29` `Improvement{Enabled bool, Threshold int, MaxUsage
   int, BoostRate float64, DecayRate float64, Cooldown time.Duration}` — a
   struct, not a bare bool. All five tunables are configurable.
   `improvementSection` (`config/load.go:61-68`) maps `enabled`/`threshold`/
   `max_usage`/`boost_rate`/`decay_rate`/`cooldown`. Rate fields are YAML
   **strings** parsed via `strconv.ParseFloat`; `cooldown` via
   `time.ParseDuration`. A missing/malformed value leaves the field zero, and
   `SetImprove` (`improve.go:87-111`) falls back to production defaults
   (`Threshold=5, MaxUsage=50, BoostRate=0.1, DecayRate=0.25,
   Cooldown=1min`) — so malformed config degrades to defaults, it does not
   crash. `Enabled` defaults to `false` (opt-in). Proven by
   `TestImprovementConfigDefaultOff` (parses `0.2`/`0.5`/`30s`/`10`/`100`).
   I ran it: PASS.

7. **Determinism / ties** ✅ **VERIFIED stable.** `improve.go:154`:
   ```go
   sort.SliceStable(idx, func(a, b int) bool { return scores[idx[a]] > scores[idx[b]] })
   ```
   `sort.SliceStable` preserves the relative order of equal scores. Since
   `idx` is initialized to `[0,1,2,…]` (the SQL rank order), ties keep the
   SQL rank. The score function is deterministic (no map iteration, no
   random; `Now()` is a fixed clock seam in tests). **Deterministic and
   stable tie-break.**

8. **TDD discipline** ✅ **VERIFIED — all 3 tests exercise the real re-rank and the disabled test compares against explicit SQL order.**
   - `TestImproveBoostsHighUsageObservations`: seeds 3 obs (100/10/0) where
     raw SQL bm25 order is the *opposite* of usage order, asserts the re-rank
     yields `[101, 11, 1]`. It even guards against a broken fixture
     (`if slices.Equal(rawUsage, []int{100,10,0}) { t.Fatalf("test setup
     broken…") }`). Real re-rank, not a stub.
   - `TestImproveDecaysNeverAccessed`: seeds two never-accessed (20×TTL,
     30×TTL) + one accessed (usage=50, fresh); asserts accessed first and the
     age-proportional ordering (20×TTL before 30×TTL). Uses a direct-SQL
     `rawSQLSearch` helper (no usage bump) to isolate decay. Real decay.
   - `TestImproveDisabledNoRegression`: **explicitly** compares the disabled
     output against the no-improve SQL order (`wantSQL := []int{0, 100, 10}`,
     `slices.Equal(usageOrder, wantSQL)`), then re-enables and asserts the
     order *changes*, then runs the cooldown (first boosted / second raw).
     This is the strongest possible proof of the disabled = pre-improve
     property. I ran all three: PASS (0.14s/0.15s/0.35s); full memory suite
     also PASS (11.3s).

9. **Tool contract** ✅ **VERIFIED — no `mem_*`/`code_*`/`web_*` signature change; no tool registration touched.**
   The scoped diff has **no** references to `mcp/` or `tools_memory` (confirmed
   by grep: "no mcp references"). The only call-site in `tools_memory.go:341`
   (`h.Memory().SearchOwnerScoped(...)`) is **not** in the scoped diff — the
   `mem_search` tool body, its args, and its return shape are untouched.
   `improve()` returns `[]Observation` in the same shape, only re-ordered.
   The `Observation` struct gains no exported field (the `improveCfg`/
   `improveLast` are unexported `Service` fields; `RetrievalUsage`/`CreatedAt`
   pre-existed). Tool contracts hold.

## Approved to merge

**Yes — with the two findings above tracked.** The core deliverables (boost,
decay, opt-in default byte-identity, cooldown, in-memory re-rank, config,
determinism, TDD, tool contract) are all correct and execution-verified. The
blocker-grade concern (the no-op retrieval-path wiring + its contradicting
comment, Finding 1) and the scoped-diff completeness gap (Finding 2) should be
resolved or explicitly waived before archive, but neither breaks the
default-disabled byte-identity property that the Global Constraints hinge on.
Recommended: fix Finding 1 (delete the dead `SetImprove` or wire `improve()`
into `rrfFallbackOwnerScopedFTS`) and Finding 2 (add the two wiring files to
the scoped diff) in a follow-up, then re-diff.
