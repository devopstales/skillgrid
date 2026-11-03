# Step 01 — Framework Route Nodes, Navigates Edges, Drop-Not-Guess

Change: `010-mnemonic-framework-routes-affected`, step `01-framework-routes`

## Status: COMPLETE (all sub-tasks green, all suites pass, 005/008 baseline intact)

## What was implemented per sub-task

### 01.1 — Framework routing files produce route nodes with references edges (RED)
- `013_framework_routes.sql` (additive): `route_meta` (framework-specific payload: framework/method/path_pattern/screen, FK→symbols+files) + `route_drops` (per-file drop count). No columns added to 005 `symbols`/`edges` — route nodes are `kind='route'` rows in `symbols`, `references`/`navigates` are rows in the 005 `edges` table.
- `route/extract.go`: `ExtractFile` dispatch (per-file-type handler) + per-framework extractors for Django/FastAPI (.py), Express/Vue (.js/.mjs/.cjs), Gin (.go), Rails (.rb), Spring (.java), SvelteKit (.svelte), Next.js/React Router (.tsx/.jsx). Each produces `Route` (web) and/or `Navigation` (router) records with `Explicit`/`Markup` flags.
- `route/frameworks.go`: `Build` (node map) producing `RouteNode`s (route nodes) + `NavigationNode`s (navigates) + `Built.Dropped/DropSample`. `RouteNode` carries the resolved `HandlerSymbol` (0 when dropped).
- `codeindex/indexer.go`: `extractRoutes` hook in `Indexer.Run`, AFTER the 005 `writeFileGraph`, in the SAME tx. `txRouteStore` implements `route.Store` (upserts route nodes as `kind='route'` symbols byte-identical to the 005 upsert, upserts references/navigates edges, records drops). Route-node UIDs added to the orphan-prune target set.
- Web frameworks: Django `urls.py` (as_view/view refs), FastAPI `@app.get` (decorator → next `def`), Express `app.get` (handler id), Gin `r.GET` (handler id), Rails `get '/x', to:` (controller#action), Spring `@GetMapping` (next Java method). All explicit → EXTRACTED.

### 01.2 — Routers produce navigates edges to named screens (RED)
- `route/frameworks.go` `buildNavigates` + `route/extract.go` router extractors: Next.js `router.push`/`<Link href>`, React Router `<Route path>`/`navigate`, SvelteKit `goto`, Vue Router `push({name})`/`push('/x')`.
- `NavigationNode.FromSymbol` = file's first 005 symbol (top-level nav); literal programmatic dest → EXTRACTED, markup-written link → INFERRED.
- Sibling-router union in dispatch: `.tsx` (Next.js first) also unions React Router; `.js` (Express first) also unions Vue Router — so the first file-type handler does not drop the others.

### 01.3 — Computed or unserved destinations stay unresolved (AFK)
- `buildNavigates` skips `Dest == ""` (computed, not a literal) — no fabricated navigates edge.
- `TestUnresolvedComputedDestinations` / `TestUnresolvedWebRouteNoRouteServes`: computed `router.push(variable)` → no nav; unserved handler → route node exists, references edge dropped (HandlerSymbol=0), drop recorded.

### 01.4 — Drop rather than guess (RED)
- `txRouteStore.ResolveHandler`: same-file match first → then unique global match; multiple or zero matches → `(0,"",false)` (dropped). `buildRoutes` records a drop (`Built.Dropped`/`DropSample`) instead of storing an AMBIGUOUS references edge.
- `AMBIGUOUS` is NOT used for dropped refs — it stays reserved for heuristic-resolved edges (markup `navigates` is INFERRED, not AMBIGUOUS). A drop is a warning (count + sample in `route_drops`), not a silent discard.

### 01.5 — Malformed routing file falls back and index continues (AFK)
- `ExtractFile`/`Build` never error: a malformed file yields an empty (non-nil) result. `extractRoutes` is non-fatal (warns, continues). `TestMalformedFileFallsBackAndContinues` / `TestIndexMalformedRouteFileFallsBack` confirm the index re-runs without error and prior route nodes survive.

### 01.6 — Framework with no routes reports none (AFK)
- `TestNoRoutesFrameworkReportsNone`: empty `urlpatterns = []` and a Spring controller with no `@*Mapping` → 0 route nodes, 0 references (reports "none", not a fabricated route).

### 01.7 — Mnemonic tool surface (RED)
- `mcp/tools_code_route.go`: `code_route` (handler→routes / path→route) + `code_navigates` (from/to screen query), distinct `code_*` names, query-only (no `code_affected`). Registered in `server.go` (`Start` + `NewServer`) without dropping any 005/008 tool.
- `mcp/server_test.go` `TestAllToolsRegistered` baseline updated 63→65 (additive). `code_search` name + required `query` param unchanged (baseline lock `TestCodeSearchSchemaStable` / `TestToolSurfaceBaseline` / `TestCodeToolRequiredParamsUnchanged` still pass).

## RED→GREEN evidence

### 01.1 (route nodes + references)
- **FAIL** (before `013`/extract/frameworks existed): `go test ./internal/mnemonic/route/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/store/... -count=1` → route package did not exist (`no Go files`); codeindex had no route nodes (`expected >=2 route symbols, got 0`).
- **PASS** (after 013 + extract.go + frameworks.go + indexer hook): `go test ./internal/mnemonic/route/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/store/... -count=1` → `ok` (all 4 packages).

### 01.2 (navigates)
- **FAIL** (`-run Navigates` before router extractors): `go test ./internal/mnemonic/route/ -run Navigates -count=1` → `expected navigates edges, got none` for react-router/vue-router (sibling dispatch dropped them) and nextjs.
- **PASS** (after router node maps + union): `go test ./internal/mnemonic/route/ -run Navigates -count=1` → `ok`.

### 01.4 (drop-not-guess)
- **FAIL** (`-run DropNotGuess` before the policy): `go test ./internal/mnemonic/route/ -run DropNotGuess -count=1` → ambiguous handler reference was stored as a references edge / not reported (`expected 1 dropped reference, got 0`).
- **PASS** (after `ResolveHandler` same-file→unique-global + drop recording): `go test ./internal/mnemonic/route/ -run DropNotGuess -count=1` → `ok`.

### 01.7 (route tools)
- **FAIL** (`-run RouteTools` before tools existed): `go test ./internal/mnemonic/mcp/ -run RouteTools -count=1` → `no tests to run` / tools not registered.
- **PASS** (after `tools_code_route.go` + server registration): `go test ./internal/mnemonic/mcp/ -run RouteTools -count=1` → `ok` (TestRouteToolsRegistered, TestRouteToolsBadArgs).

## Full relevant suites (final)

```
$ go test ./internal/mnemonic/route/... ./internal/mnemonic/codeindex/... \
          ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route      0.558s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  3.768s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store      3.605s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp       27.249s
```

Route package tests (all PASS):
`TestRouteNodesForWebFrameworks` (django/fastapi/express/gin/rails/spring), `TestRouteNodesForDjangoExplicit`, `TestNavigatesForRouters` (nextjs/react-router/sveltekit/vue-router), `TestNavigatesMarkupInferred`, `TestNavigatesNoRouterProducesNone`, `TestUnresolvedComputedDestinations`, `TestUnresolvedWebRouteNoRouteServes`, `TestDropNotGuess`, `TestDropNotGuessWarningIsReported`, `TestMalformedFileFallsBackAndContinues`, `TestNoRoutesFrameworkReportsNone`.

codeindex route integration tests (all PASS):
`TestIndexProducesRouteNodesAndReferences`, `TestIndexRouteNavigatesEdges`, `TestIndexUnresolvedDestinationStaysUnresolved`, `TestIndexMalformedRouteFileFallsBack`.

MCP route tool tests (all PASS):
`TestRouteToolsRegistered`, `TestCodeRouteQuery`, `TestCodeNavigatesQuery`, `TestRouteToolsBadArgs`.

## 005/008 baseline confirmation

```
$ go test ./internal/mnemonic/community/... ./internal/mnemonic/hybrid/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community  0.775s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid     1.057s
```
005 tool-surface baseline lock also green: `TestToolSurfaceBaseline`, `TestCodeSearchSchemaStable`, `TestCodeToolRequiredParamsUnchanged`, `TestAllToolsRegistered` (updated 63→65 additively). `code_search`/`code_read`/`code_index`/`code_status` + all 008 `code_communities`/`code_god_nodes`/`code_explain_community` unchanged.

## Commits
| SHA | Subject |
|-----|---------|
| d3ca0fc | feat(mnemonic): framework route nodes and references edges |
| a3af415 | feat(mnemonic): navigates edges from framework routers |
| cddafb5 | feat(mnemonic): drop-not-guess edge resolution policy |
| bb82f7c | feat(mnemonic): register route and navigates query tools |
| b14a34d | feat(mnemonic): same-transaction route extraction hook in Indexer.Run |
| 972c647 | test(mnemonic): add code_route and code_navigates to the tool surface baseline |

## Global constraints held
- **CGo-free**: route package uses stdlib only (`regexp`, `strings`, `hash/fnv`, `path/filepath`); no CGo dependency introduced.
- **Additive on 005/008**: `013_*` adds two new tables; 005 `symbols`/`edges` untouched (route nodes are `kind='route'` rows, references/navigates are `edges` rows). No 005/008 `code_*` tool name or required param changed.
- **Confidence labels**: route/navigates edges carry EXTRACTED (explicit syntax / literal) or INFERRED (convention / markup-written). No AMBIGUOUS stored for dropped refs.
- **Unresolved, not fabricated**: computed/unserved destinations stay unresolved (no edge); markup links INFERRED.
- **Drop-not-guess**: ambiguous refs (no same-file, no explicit specifier, no unique global match) dropped at extraction + warned (count + sample in `route_drops`); excluded from blast-radius math (no edge row).

## Concerns
- **`route_meta.screen` column is currently unused** (navigates store the screen in `edges.to_name`). Kept for step 02 / future screen-node promotion. Could be removed, but additive + harmless.
- **Handler resolution is name-only** (same-file, then unique global). Owner-qualified (`pkg.Handler`) matching is not yet parsed from the source; a dotted handler ref is trimmed to the bare name before resolution. This is conservative (favors dropping over guessing), consistent with the policy.
- **`config.d/indexing.yaml` default include** does not scan `.js`/`.rb`/`.java`/`.py`/`.svelte` by default; route extraction only fires for files the indexer scans. The codeindex + mcp fixtures add an `indexing.yaml` (or use an explicit include set) so all framework file types are indexed. Production projects must include the relevant extensions in their indexing config for route extraction to apply — this is expected (the 005 extractor has the same include-gating).

---

## Review findings (step-01 re-work)

### 1. Blast-radius edge-kind reconciliation (IMPORTANT for step 02)

**Problem**: `graph/impact.go` traversed `e.kind IN ('calls','imports','reference','route','extends','implements')` — the step-01 `references` (route→handler) edges were absent. Also the walk filtered `to_id IS NULL` only, but a *resolved* route→handler `references` edge carries a non-null `to_id`. Both were fixed so step-02's `code_affected` can traverse a resolved route→handler reference.

**`impact.go` before:**
```sql
SELECT e.from_id, e.kind, e.confidence, e.line
FROM edges e
WHERE e.to_id IS NULL
  AND e.to_name IN (SELECT name FROM symbols WHERE id = ?)
  AND e.kind IN ('calls','imports','reference','route','extends','implements')
ORDER BY e.id
```

**`impact.go` after:**
```sql
SELECT e.from_id, e.kind, e.confidence, e.line
FROM edges e
WHERE (
    e.to_id = ?
    OR (e.to_id IS NULL AND e.to_name IN (SELECT name FROM symbols WHERE id = ?))
  )
  AND e.kind IN ('calls','imports','reference','route','extends','implements','references','navigates')
ORDER BY e.id
```
The 005 singular `'reference'` kind is left intact; step-01's plural `references` + `navigates` are added explicitly (commented as step-01 kinds included for blast radius). The by-id match is safe because 005 calls/imports edges always set `to_id` (from `ToUID`), so no over-matching on names.

**Test**: `TestImpactTraversesRouteReferencesEdges` (`graph/impact_test.go`) asserts a resolved route→handler `references` edge IS traversed (the route appears in the handler's blast radius), and that a dropped (unresolvable, no stored edge) reference does NOT appear — drop-not-guess still holds.

### 2. AMBIGUOUS three-way split (01.4.a)

**Problem**: the route package emitted only `EXTRACTED`/`INFERRED`; `AMBIGUOUS` was produced solely by name-only resolution in `graph.go:~193`. The 01.4.a contract ("a heuristic-resolved edge is stored `AMBIGUOUS`, an ambiguous-but-unresolvable one is absent") was never emitted or tested by the route package.

**Change** (route `extract.go`/`frameworks.go` + `codeindex/indexer.go`):
- `SymbolIndex.ResolveHandler` / `Store.ResolveHandler` now return a `Resolution` (id, uid, confidence) instead of `(id, uid, ok)`.
- Resolution policy in `txRouteStore.ResolveHandler`:
  - **same-file** match → `EXTRACTED` (an explicit handler reference).
  - **name-only** match (no same-file hit, but a unique global symbol by that name) → `AMBIGUOUS` (a best-effort name guess — resolved, stored, low-confidence).
  - **multiple or zero** global matches → unresolvable → dropped + warned (drop-not-guess).
- `StoreReferencesEdges` now writes `node.HandlerConfidence` (EXTRACTED or AMBIGUOUS) rather than re-deriving from `HandlerExplicit`. (INFERRED is not a references-edge confidence — it labels convention-derived navigates.)
- Three-way distinction documented at the top of `extract.go`: `EXTRACTED` = explicit syntax; `INFERRED` = convention/heuristic (markup navigates); `AMBIGUOUS` = name-only best-effort guess (stored, low-confidence); Dropped = unresolvable, not stored.

**Test**: `TestHeuristicResolvedIsAmbiguous` (`route/edge_test.go`) asserts BOTH halves of 01.4.a — (a) a name-only-resolved handler is stored `AMBIGUOUS` (not dropped), and (b) an unresolvable handler is absent (dropped + warned). Existing `TestDropNotGuess`, `TestUnresolvedWebRouteNoRouteServes`, and `TestIndexUnresolvedDestinationStaysUnresolved` were adjusted (they used the old `(id, uid, ok)` shape / `dummyIndex{known:...}` literal) and all pass unchanged in intent.

### Full relevant suite (after review fixes)
```
ok  internal/mnemonic/route      0.651s
ok  internal/mnemonic/codeindex  3.604s
ok  internal/mnemonic/store      3.501s
ok  internal/mnemonic/mcp        27.602s
ok  internal/mnemonic/graph      2.835s
```
005/008 baseline intact:
```
ok  internal/mnemonic/community  0.656s
ok  internal/mnemonic/hybrid     1.049s
```
Tool-surface baseline (65 tools) unchanged; CGo-free stance unchanged; same-transaction indexer hook unchanged.

### Commits
| SHA | Subject |
|-----|---------|
| ac92533 | feat(mnemonic): three-way EXTRACTED/INFERRED/AMBIGUOUS route edge policy |
| 8c099a5 | feat(graph): include route references edges in blast radius |
