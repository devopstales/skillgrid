## 01-framework-routes

### Goal

Additive `013_*` schema + framework-route extractor (route + references + navigates edges) so URL→handler and screen→screen become one graph hop, while 005/008 tools stay unchanged. Applies the **drop-rather-than-guess** edge policy: ambiguous references are dropped at extraction (not stored as `AMBIGUOUS`) and excluded from blast-radius math.

### Out of scope / Non-Goals

- `code_affected` / `code_rename` (step 02); the `--base` merge-base diff + owner attribution (step 02)
- Watcher / staleness banner / fingerprint gate / warm embedder (step 03)
- Changing 005/008 tools; mobile bridging; browser UI

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none (005 done)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/013_framework_routes.sql`
- Create: `skillgrid-cli/internal/mnemonic/route/extract.go`
- Create: `skillgrid-cli/internal/mnemonic/route/frameworks.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_route.go`
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` (hook route extraction after 005 symbol/edge extraction, same tx)
- Test: `skillgrid-cli/internal/mnemonic/route/...`, `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/store/...`, `skillgrid-cli/internal/mnemonic/mcp/...`

**Interfaces:**
- Consumes: 005's `symbols`/`edges` tables + gotreesitter extraction; indexer single-transaction guard; existing migration runner
- Produces: `route` nodes + `references`/`navigates` edge types in the existing graph (additive `013_*`); per-framework node maps (routes→handler `references`, routers→screen `navigates`); same-tx route-extraction hook in `Indexer.Run`; route/navigates query MCP tools; the **drop-rather-than-guess** edge-resolution policy (ambiguous refs dropped + warned, never stored as `AMBIGUOUS`; `AMBIGUOUS` = resolved-heuristic only) that step 02's blast-radius math inherits

### Tasks

- [ ] 01.1 `[RED]` Framework routing files produce route nodes with references edges (Scenario: Web-framework routing files produce route nodes)
  - [ ] 01.1.a Write failing test — fixture routing files (Django `urls.py`, FastAPI `@app.get`, Express `app.get`, Gin `r.GET`, Rails `get '/x'`, Spring `@GetMapping`); after index, assert `route` nodes exist, each linked by a `references` edge to its handler, every edge EXTRACTED (explicit syntax), and a query for callers of a handler surfaces the URL pattern
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `013_framework_routes.sql` + `route/extract.go` dispatch + `route/frameworks.go` per-framework node maps + `Indexer.Run` hook (same tx as 005 extraction)
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): framework route nodes and references edges`
- [ ] 01.2 `[RED]` Routers produce navigates edges to named screens (Scenario: Router navigations produce navigates edges)
  - [ ] 01.2.a Write failing test — fixture routers (Next.js `router.push`/`<Link href>`, React Router `<Route path>`, SvelteKit `goto`, Vue Router `push({name})`); after index, assert `navigates` edge from the sending function to the named screen; literal destinations EXTRACTED; markup-written links INFERRED
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Navigates -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — router node maps in `route/frameworks.go` + navigates emission in `route/extract.go`
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Navigates -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): navigates edges from framework routers`
- [ ] 01.3 `[AFK]` Computed or unserved destinations stay unresolved (Scenario: Computed or unserved destination stays unresolved) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Unresolved -count=1` — Expected: PASS
- [ ] 01.4 `[RED]` Drop rather than guess — ambiguous references dropped, not AMBIGUOUS-kept (Scenario: Ambiguous references are dropped not guessed) — Business rule "Drop rather than guess"; edge policy inherited by step 02
  - [ ] 01.4.a Write failing test — a reference with no same-file match, no explicit specifier, and no unique global/owner-qualified match is **dropped at extraction**: no edge stored, and a warning (count + sample) is reported, not a silent discard; assert `AMBIGUOUS` is reserved for edges that *were* resolved via a heuristic (e.g. a markup-written `navigates` link) — a heuristic-resolved edge is stored `AMBIGUOUS`, an ambiguous-but-unresolvable one is absent
  - [ ] 01.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run DropNotGuess -count=1` — Expected: FAIL
  - [ ] 01.4.c Minimal implementation — drop-not-guess resolution policy in `route/extract.go` (drop + warn on ambiguity; keep `AMBIGUOUS` for heuristic-resolved only)
  - [ ] 01.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run DropNotGuess -count=1` — Expected: PASS
  - [ ] 01.4.e Commit — `feat(mnemonic): drop-not-guess edge resolution policy`
- [ ] 01.5 `[AFK]` Malformed routing file falls back and index continues (Scenario: Malformed routing file falls back and index continues) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [ ] 01.6 `[AFK]` Framework with no routes reports none (Scenario: Framework with no routes reports none) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -count=1` — Expected: PASS
- [ ] 01.7 `[RED]` Mnemonic tool surface — route tools register, `code_search` stable, bad args rejected (Scenario: Route tools register and code_search stays stable) — threat: Mnemonic tool surface
  - [ ] 01.7.a Write failing test — assert route/navigates query tools register with distinct `code_*` names; `code_search` name + required `query` param schema unchanged; 005/008 tools intact; bad route args rejected with a clear validation error
  - [ ] 01.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run RouteTools -count=1` — Expected: FAIL
  - [ ] 01.7.c Minimal implementation — `tools_code_route.go` + server registration without dropping existing `code_*`
  - [ ] 01.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run RouteTools -count=1` — Expected: PASS
  - [ ] 01.7.e Commit — `feat(mnemonic): register route and navigates query tools`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `skillgrid index` on a fixture web app; query route→handler→screen via SQL / MCP; confirm an ambiguous ref is dropped + warned | PASS | | |
| Rollback boundary | Drop `013_*` + `route/` + `tools_code_route.go`; revert indexer hook | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): framework route nodes, navigates edges, and drop-not-guess policy`

---

