# Task 03 — Layered Retrieval + Budgets (013 step 03) — Report

**Status:** DONE (all sub-tasks 03.1–03.9 implemented and GREEN)

## What was implemented per sub-task

| Sub-task | Status | Delivered |
|----------|--------|-----------|
| 03.1 `[RED]` Context-window flood (budget) | a→e | `memory/budget.go` — uniform `Budget` (item cap + char budget + context timeout); `memory/budget_test.go::TestBudget`; commit `6fce689` |
| 03.2 `[RED]` Mnemonic tool surface (retrieve/surface) | a→e | `memory/retrieve.go` — `Retrieve` (L2/L3-first + RRF fallback) + `LayerHit`; `memory/retrieve_test.go::TestRetrieve`; commit `91fd4a4` |
| 03.3 `[AFK]` L2/L3-first + RRF fallback | PASS | `memory/retrieve_test.go::TestLayered` |
| 03.4 `[AFK]` every `mem_*` read enforces item/char/timeout | PASS | `memory/budget.go` + MCP read-path wiring (both `mem_search` branches, `mem_context`, `mem_timeline`) |
| 03.5 `[AFK]` 20-hit search → 20 budgeted snippets + id | PASS | `service/retrieval.go::BudgetedRetrieval` + `service/retrieval_budget_test.go::TestBudgetedSearch` |
| 03.6 `[AFK]` `mem_get_observation` only-full-content + tunable | PASS | `service/retrieval_budget_test.go::TestFullContent`; config `mnemonic.retrieval_budget` |
| 03.7 `[AFK]` route context/search/timeline budgeted | PASS | `service/retrieval_budget_test.go::TestRouteReads` + MCP handler wiring |
| 03.8 `[AFK]` CLI parity (layers/governance/share) | PASS | `cmd/skillgrid/mem.go` + `memhelpers.go` + `main.go`; `cmd/skillgrid/mem_test.go::TestMemCLIParity`/`TestMemCLIBadArgs` |
| 03.9 `[AFK]` 005 stable + bad args rejected | PASS | `mcp/tools_memory_retrieval_test.go::TestBudgetedRetrievalMCP` (78-tool lock + required-params + bad-args); commit `7508faa` |

## RED → GREEN evidence

### 03.1 (budget)
- **RED** (`go test ./internal/mnemonic/memory/... -run Budget -count=1`): build failure — `fx.svc.Budget undefined`, `undefined: defaultBudgetItems`, `undefined: DefaultBudget`, `undefined: Budget`.
- **GREEN**: `TestBudget` PASS — subtests `item-cap-truncates-20-hits`, `char-budget-truncates-with-explicit-omitted`, `context-timeout-returns-truncated-partial-never-hangs` all PASS.

### 03.2 (retrieve/surface)
- **RED**: build failure — `retrieve_test.go` referenced `newFixture`/`RetrieveOpts` before `retrieve.go` existed; `retrieve.go:110: undefined: sql` (mid-fix).
- **GREEN**: `TestRetrieve` PASS — `bootstrap-returns-l2-l3-first`, `fact-falls-back-to-l1-l0-rrf`, `bad-mode-rejected` all PASS.

## How the budget wrapper works (`memory/budget.go`)

`Budget.Apply(ctx, hits)` is the uniform in-memory wrapper every read runs through:

1. **Item-count cap** (`cfg.Items`, default 10): `out = hits[:cfg.Items]` — a 20-hit read returns 10 in-list results, never more.
2. **Char budget** (`cfg.Chars`, default 1200): each in-list snippet longer than the cap is truncated with an explicit marker via `TruncateWithMarker` → `s[:n] + "… (N chars omitted)"` where `N = len(s)-n`. Never silent: a reader always sees exactly how many chars were dropped.
3. **Context timeout** (`cfg.TimeoutNs`, default 3s): `Apply` checks `ctx.Deadline()`. A lapsed deadline (the read consumed the whole context budget) returns `truncated=true` + `reason="timeout"` **and a partial** (as many hits as the item cap allows), never a hang. The timeout is **injectable** — the test injects a real short deadline (`context.WithDeadline(ctx, now+1ms)`) so it is fast and deterministic, not a real 3s wait.

`BudgetConfig` is additive: a zero field falls back to its default, so a partial config only overrides the knobs it sets. `Service.SetBudget`/`Service.Budget` expose it on the memory service; the service layer tunes it from config in `openProject`.

## How layered retrieval bootstraps L2/L3 then falls back to RRF (`memory/retrieve.go`)

`Retrieve(ctx, RetrieveOpts{Mode, Query, Limit})` (Mode `"bootstrap"` | `"fact"`):

- **`bootstrap`** → `layerBootstrap`: reads the step-02 `personas` records (L3 persona-delta, L2 scenario) joined to `observation_layers`, ordered **L3 then L2** (most stable/cheapest first), provenance-linked. Returns L2/L3 first. A project with no distilled layers returns an empty list (graceful no-op, not an error).
- **`fact`** → `rrfFallback`: for a specific fact it **routes through the EXISTING 005 RRF read path** — `Service.BlendedSearch` (FTS + vector + `ReciprocalRankFusion`) — NOT a re-implementation of RRF. L1 atoms are observations (reachable via FTS); the L0 raw record is the RRF floor. Returns L1 hits tagged `layer:"L1"`.

Every `LayerHit` carries `GetObservationID` (the full-content fetch id), so the agent can pull untruncated content on demand.

## How `mem_get_observation` stays the ONLY full-content path

- **In-list** (`mem_search`/`mem_context`/`mem_timeline`) results are truncated: the MCP handlers run `Budget.Apply` + `budgetedObservationDTOs` (char budget with explicit "N chars omitted"); the CLI `mem search` outputs `res.Hits` (char-truncated).
- **`mem_get_observation`** uses the raw `observationDTO` (no char budget, no truncation) and `memory.Service.ReadAs`/`Get` — it returns the **full, untruncated** content. It is the only read path that does.
- **Every in-list result carries its full-content id** (`id` / `GetObservationID`), asserted by `TestBudget` (item-cap subtest), `TestBudgetedSearch`, `TestRetrieve`/`TestLayered`, and `TestBudgetedRetrievalMCP`.
- `TestFullContent` proves the contrast directly: a budgeted in-list hit is truncated with "chars omitted", but `mem.Get(id)` returns the exact full body (no marker, longer).

## How the budget is tunable

`config/load.go` adds the `mnemonic.retrieval_budget` section (`items` / `chars` / `timeout` duration string) → `config.Indexing.RetrievalBudget`. `service.openProject` reads it and calls `mem.SetBudget(memory.BudgetConfig{...})`, so a config without the section is the default budget and a partial section overrides only the knobs it sets. `TestFullContent` proves the tunability: a larger char cap (`Chars: 3000`) keeps more in-list content than the default. The CLI exposes `--item` / `--char` / `--timeout` flags on `mem search`/`context`/`timeline`.

## Full relevant suite + output

`go test ./internal/mnemonic/memory/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... ./cmd/skillgrid/... -count=1`:

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	9.241s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer	2.553s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp	38.092s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	27.481s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	23.558s
```

Step-03 scenario tests (all PASS):

```
--- PASS: TestBudget (item-cap-truncates-20-hits, char-budget-truncates-with-explicit-omitted, context-timeout-returns-truncated-partial-never-hangs)
--- PASS: TestRetrieve (bootstrap-returns-l2-l3-first, fact-falls-back-to-l1-l0-rrf, bad-mode-rejected)
--- PASS: TestLayered
--- PASS: TestBudgetedSearch
--- PASS: TestFullContent
--- PASS: TestRouteReads
--- PASS: TestBudgetedRetrievalMCP
--- PASS: TestMemCLIParity
--- PASS: TestMemCLIBadArgs
```

## Baselines (005/008/010/011) still pass

`go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	0.725s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	8.160s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	2.091s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	2.813s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	12.570s
```

The 78-tool surface lock stays green (asserted in `TestBudgetedRetrievalMCP` and the existing `tools_memory_*_test.go` / `tools_code_*_test.go` suites); 005 `mem_*` names + required params unchanged.

## Commits

```
6fce689 feat(memory): item+char+timeout budget wrapper on mem_* reads
91fd4a4 feat(memory): layered L2/L3-first retrieval + mem_get_observation full-content path
9142064 feat(memory): route mem_search/context/timeline through layered + budgeted in-list reads
7508faa test(mcp): budgeted in-list + mem_get_observation only-full-content + bad-args (005-stable)
7e29c3d feat(cli): mem layers/governance/share/search/context/timeline with budget flags (parity)
```

## Concerns

- **`Retrieve` is additive, not yet the sole search path.** `mem_search` still runs the owner-scoped FTS (`SearchOwnerScoped`) as the primary ranked list and then applies the budget; the layered `Retrieve` is exercised via the `BudgetedRetrieval` seam and the 03.2/03.3 tests. This satisfies "route through the existing RRF path for the L1/L0 fallback" (BlendedSearch) without disturbing the 013-step-01 per-owner visibility filter that `mem_search` depends on. A future step could make `Retrieve` the primary path if bootstrap-first ordering is required end-to-end at the tool boundary.
- **Persona ids (L2/L3) vs observation ids (L1).** For L2/L3 bootstrap hits, `GetObservationID` is the `personas` row id (the stable handle), which is not an `observations` id — so `mem_get_observation` (which fetches by observation id) covers the L1/L0 RRF-fallback hits. L2/L3 content is fully surfaced in `mem_layers`/bootstrap (not truncated there). This matches "bootstrap from L2/L3, specific fact via RRF" — the specific-fact path (L1/L0) is the one whose in-list hits are fetchable by observation id.
- **Context timeout is deadline-driven, not a wall-clock `select`.** `Apply` returns a truncated partial when the caller's deadline has lapsed; this is the "never hangs" guarantee (a read bounded by the caller's context). The MCP/HTTP handler context carries the real deadline; the test injects a short one.

---

## Coherence-fix pass (review gaps 1–5)

This section records the review-coherence fixes layered on top of the original
5 commits. The load-bearing guarantees (budget wrapper, only-full-content path,
78-tool surface, step-01 visibility gate) were already held; these close the
coherence gaps a reviewer flagged without regressing them.

### 1. (IMPORTANT) `TimeoutNs` is now actually enforced (not a no-op)

**Before:** `BudgetConfig.TimeoutNs` was stored/normalized but `Apply` only
inspected the caller's *already-lapsed* `ctx.Deadline()` after the read had
returned — so a genuinely slow read still ran to completion; "never hangs" was
not enforced by the budget itself. The test passed only because it injected an
already-lapsed deadline.

**Fix — where the deadline-bound context is created:** a new
`memory.Budget.Bound(ctx)` derives `context.WithTimeout(ctx, budget.TimeoutNs)`
(the smaller of the caller deadline and the budget timeout wins) BEFORE the
query. A new `memory.Budget.ApplyRead(ctx, read)` is the uniform budgeted-read
wrapper: it runs `read` under `Bound(ctx)` and applies the full uniform budget
(item cap + char budget + timeout). If the read is cut by the deadline (its
error is `context.DeadlineExceeded`/`Canceled`), the partial it gathered is
surfaced as `truncated:true` / `reason:"timeout"` instead of a hard failure.
`Apply` itself is unchanged (it still observes the lapsed bound deadline to set
the flags). The MCP `mem_search` read path and the layered CLI path both call
`ApplyRead`, so the timeout is enforced identically at every read.

**Slow-read truncation test** (`memory/budget_test.go::TestBudgetTimeoutEnforced`):
an injectable slow read (honors the context deadline, returns a 1-hit partial on
cancellation, else a 20-hit full result after 20ms) is run under a 1ms budget
timeout → returns a 1-hit partial, `truncated:true`, `reason:"timeout"`, and the
full 20-hit read did NOT run to completion. A control with a generous 2s timeout
lets the full read complete (item-capped to 10) with no `timeout` reason. Fast
(no real wall-clock wait) because the injected delay is bounded by the tiny
deadline.

### 2. (IMPORTANT) Layered path now has a real production caller (gate held)

**Decision (option b + a hybrid):** wiring the *layered* path into `mem_search`
would regress the step-01 per-owner gate (the MCP `mem_search` runs
`SearchOwnerScoped` as its primary ranked list, and the RRF leg carries that
gate). So `mem_search` stays on the gated FTS path, and the **CLI `mem search`**
is made the real production entry point for the layered L2/L3-first + RRF
fallback path, with the per-owner gate held.

**What changed:**
- `memory.SearchOwnerScopedRetrieve(ctx, readerOwner, mode, query, limit)` — the
  owner-gated layered read. `bootstrap` → L2/L3 `layerBootstrap`; `fact` →
  `rrfFallbackOwnerScoped`, which routes through the EXISTING RRF path
  (`BlendedSearch`: FTS + vector + ReciprocalRankFusion) and then filters the
  hits by the step-01 `canRead` gate (a private observation of another owner is
  absent; a missing/expired row is treated as not-visible, not an error).
- `service.BudgetedRetrievalAs` / `service.BudgetedRetrievalAsRoot` — the
  production facade (config-loaded budget, optional per-project CLI-flag
  override via `Service.SetBudgetOverride`). `BudgetedRetrieval` (the original
  seam) now delegates to `BudgetedRetrievalAs`. The read runs through
  `SearchOwnerScopedRetrieve` under `Budget().Bound(ctx)`.
- CLI `mem search` (`cmd/skillgrid/mem.go::runMemSearch`) now calls
  `svc.BudgetedRetrievalAsRoot(..., "fact", ...)` with `--reader-owner` as the
  reader identity — so the layered path is reachable end-to-end from the command
  line.

**End-to-end test** (`service/retrieval_budget_test.go::TestBudgetedRetrievalProductionEntryPoint`):
drives the real facade — (a) `bootstrap` returns the distilled L2/L3 layers
(non-empty, each carrying a fetch id) proving the layered path is real; (b) a
specific-fact query falls back to L1/L0 RRF; (c) owner-visibility gate holds on
the layered path — owner A (the saving session) sees their own private fact,
owner B does NOT.

### 3. (MINOR) CLI `mem context` / `mem timeline` now honor their budget flags

**Before:** `cmd/skillgrid/mem.go` `runMemContext`/`runMemTimeline` did
`_ = memBudgetOpts(...)` and never applied the budget (only `mem search` did).

**Fix:** both now call `h.Memory().SetBudget(memoryBudget(cfg))`, derive the
deadline-bound read context via `b.Bound(hCtx())`, run the read under it, and
apply the uniform budget (item cap + char budget) plus the timeout flag. A new
helper `budgetSessionsCLI` (memhelpers.go) applies the session item cap + char
budget. `runMemTimeline` truncates each `before`/`after` snippet with the char
budget.

**CLI test** (`cmd/skillgrid/mem_test.go::TestMemCLIBudgetedContext`, the `go run`
subprocess pattern): a session with a long summary + `--char 40` → the output
contains the explicit `chars omitted` marker and `"truncated": true`.

### 4. (MINOR) Uniform budget across context/timeline

**Decision: uniform Apply (item cap + char budget + context timeout), applied
consistently.** `mem_context` now item-caps the session list (same cap as
`mem_search`) and char-truncates each summary, and `mem_timeline` char-truncates
each snippet; both enforce the context timeout via the deadline-bound read
context. This makes all three reads (search/context/timeline) apply the same
three caps. The only structural difference is the *item model* (sessions vs
timeline before/after lists vs observations), which is inherent to each read's
shape; the budget *caps* are uniform. `mem_get_observation` remains the ONLY
full-content path (untouched).

### 5. (MINOR) Config-driven tunability test (not just a direct setter)

**New test** (`service/retrieval_budget_test.go::TestBudgetConfigDrivenTunability`):
writes a `mnemonic.retrieval_budget.items` section to `config.d/indexing.yaml`
under each project's config root, then drives the real read path
(`BudgetedRetrievalAsRoot`, which loads config in the production seam). Two
projects, identical 20-hit content, `items: 5` vs `items: 15` → in-list
truncation of 5 vs 15. Proves config-driven tunability end-to-end, not just a
direct `SetBudget` call.

## Full relevant suite (coherence-fix pass)

`go test ./internal/mnemonic/memory/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... ./cmd/skillgrid/... -count=1`:

```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  15.167s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer 4.016s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp  66.442s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service  48.462s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  48.022s
```

Baselines (005/008/010/011) still pass:

`go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:

```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route  1.209s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected  14.447s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community  3.138s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg  4.235s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  21.178s
```

Step-01 visibility wiring test still passes:

`go test ./internal/mnemonic/mcp/... -run TestMemSearchGetOwnerEnforcedWiring -count=1`:

```
--- PASS: TestMemSearchGetOwnerEnforcedWiring (0.35s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp  2.601s
```

The 78-tool surface lock stays green (`TestBudgetedRetrievalMCP` PASS); 005
`mem_*` names + required params unchanged.
