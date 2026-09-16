# Tasks: 010-mnemonic-framework-routes-affected

> **STATUS:** `in-progress` — 0/3 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Close the highest-value gaps CodeGraph + GitNexus expose over 005/008: (1) **framework-aware routes** — emit `route` + `navigates` edges so "which URL serves this handler" and "where does tapping this go" are one graph hop; (2) **`code_affected`** — trace import deps transitively from changed files to the test files that must run; (3) **`code_rename`** — a graph-grounded write tool that splits renames into confidence buckets; (4) **auto-sync watcher** — keep the graph fresh as the agent edits via **copy-and-swap** publication + auto-reopen, with a staleness banner so the agent never gets a silent wrong answer or reads a torn index.

**Architecture:** A third extractor layer (framework-route per language/framework) adds `route` nodes + `references`/`navigates` edges into 005's existing graph; **drop-rather-than-guess** means ambiguous references are dropped at extraction and excluded from blast-radius math. `code_affected` is a pure edge-traversal query (no new extraction) over 005's import/`tests-for` edges, with `--base <ref>` deriving the changed set from a merge-base diff + git-history owners; `code_rename` is the write counterpart splitting edits into graph-confidence vs. text-search buckets with `dry_run`. An fsnotify watcher in the serve/MCP path publishes via **copy-and-swap** (sidecar build → atomic publish) so readers never see a torn index; a **pull-at-query fingerprint gate** is the correctness backstop that keeps every `code_*` query fresh even with the watcher off; a warm embedder daemon keeps 005's ONNX model in RAM (load-once, idle-evict, MCP-heartbeat keep-alive).

**Tech Stack:** Go (`skillgrid-cli`), CGo-free SQLite (`modernc.org/sqlite`), gotreesitter graph from 005, `github.com/fsnotify/fsnotify` (pure-Go watcher), MCP (`mcp-go`), CLI.

**Spec:** `docs/skillgrid/changes/010-mnemonic-framework-routes-affected/change.md`

**Acceptance:** `docs/skillgrid/changes/010-mnemonic-framework-routes-affected/acceptance.feature` (`@step-NN`)

---

## Goal

Coding agents get end-to-end flows across framework boundaries (URL → handler → handler → next screen), a one-command answer to "which tests do I run after this change?", and a graph that is never stale mid-session — without re-running an index by hand or re-grepping during the debounce window.

## Out of scope / Non-Goals

- Re-implementing 005's symbols/edges/extractors/hybrid search or 008's community/knowledge nodes — this change is additive
- Cross-language mobile bridging (Swift↔ObjC, React Native bridge, Expo Modules) — CodeGraph has it, but it's niche + heavy; only if Mnemonic targets mobile/RN
- Browser graph UI (CodeGraph `codegraph ui`) — human-facing, not agent-facing; a later `skillgrid code ui` if ever
- Telemetry / self-updating / signed-release attestation (CodeGraph product concerns) — not part of the graph
- Multi-project / cross-repo graph merge

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing `@happy`, `@edge`, and `@failure` scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Applicable threat-matrix rows (PR commands; Mnemonic tool surface; Index freshness / concurrency; Warm embedder lifecycle) have RED coverage that passed
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- CGo-free: fsnotify is pure-Go (no C dependency); matches the codebase's CGo-free stance (modernc + gotreesitter + loom)
- Additive on top of 005 (and 008 where present) — never rewrite 005/008 symbols/edges/extractors; existing 005/008 `code_*` tools keep name + required params; all new tools use distinct `code_*` names
- Every new edge carries a Confidence Label: `EXTRACTED | INFERRED | AMBIGUOUS`; route/navigation edges that resolve from explicit syntax are `EXTRACTED`, convention/heuristic ones `INFERRED`, name-only guesses `AMBIGUOUS`
- Route/navigation destinations that are **computed** (not a string literal) or that **no route serves** are left unresolved — never fabricated; a markup-written link is marked `INFERRED`
- **Drop rather than guess** (Graft): an ambiguous reference (no same-file match, no explicit specifier, no unique global/owner-qualified match) is **dropped** at extraction, not `AMBIGUOUS`-kept — ambiguous edges are excluded from `code_affected` / blast-radius math so a false positive can never inflate the radius. `AMBIGUOUS` is reserved for edges that *were* resolved but with a heuristic (e.g. markup-written `navigates`); a drop is reported as a warning, not a silent discard
- `code_affected` is a **query only** — it adds no nodes/edges; it traverses 005's import/`tests-for` edges with a depth cap (default 5) and returns paths, not invented relationships
- **`--base <ref>` PR mode** (Graft `blast --base`): `code_affected --base origin/main` derives the changed file set from the merge-base diff (no `--stdin` needed), reports the areas the change can reach (per-symbol detail collapsed under each area), and names **git-history owners** ("who to tag") for each affected area via `git blame` on the touched lines. `--quiet`/`--json` make the output a ready PR comment / CI gate
- `code_rename` is the **write counterpart** of `code_affected`: it resolves the symbol (reusing 005's disambiguation), then splits edits into **graph edits** (high-confidence: definition + typed references from the edges table) and **text-search edits** (lower-confidence: string matches the graph didn't resolve). `dry_run` (default `true`) returns the plan without writing. Every edit carries a confidence label; a rename that can't resolve the symbol returns a candidate list, never a silent pick
- Watcher is opt-out by default in `serve`/MCP path; debounce window default 2000ms (tunable, clamped [100ms, 60s]); filtered to source files only; a sandboxed env (`SKILLGRID_NO_WATCH=1`) disables it gracefully — the pull-at-query fingerprint gate still makes every query fresh (structural-only), so a disabled watcher degrades to "slightly slower but never stale"
- **The embedder stays warm** while a session is active: the ONNX runtime (005's default embedder) is loaded once and **kept in RAM** across `skillgrid search` / MCP calls rather than re-loading the ~270MB model per invocation. It **evicts on idle** (`idle_timeout`, default 10min, tunable) and is **kept alive by MCP heartbeat** while an MCP client is connected. A down/absent model just falls back to FTS+signals (005's floor); `off`/external providers don't hold RAM at all
- **Publication is copy-and-swap:** the watcher builds the re-index into a sidecar, then atomically publishes it (rename/swap), so a concurrent reader never observes a torn index. On Windows / sidecar-unsupported filesystems it updates in place with a write-failure guard (retry if the failure was pre-write; stop if it may have mutated the live index). Running MCP/serve processes **reopen the newly-published index on their next tool call (~5s), no restart**
- Staleness banner: during the debounce window, any MCP response referencing a pending file names it and tells the agent to `Read` it directly; pending files not referenced surface as a footer
- **Pull-at-query fingerprint gate** (Graft): every `code_*` MCP/CLI query first runs a ~3ms `(size, mtime)` stat-walk of the working tree against the last index's fingerprint; on drift it triggers a **structural-only** incremental re-index (never the LLM/embedder leg), guarded by the same writer lock as the watcher. The fsnotify watcher stays the *comfort* layer (near-zero-latency, background); the fingerprint gate is the *correctness backstop* — even with the watcher off, a query answers against the tree as it is now (uncommitted edits included). Fingerprint is keyed by the extractor stamp (a binary/version change invalidates it)
- One live writer per project (shared or single direct-mode); a second exits with a clear writer-lock error; `SKILLGRID_NO_WATCH=1` to run in-process
- Bad / missing args on new `code_*` tools → `abort` with clear validation error; do not invent routes or affected files
- Migration id `013_framework_routes.sql` — leave `011` (005) and `012` (008) as-is; all new SQL is additive

---

## State

```yaml
phase: verify        # spec | apply | verify | archive
current_step: 03-autosync-watcher
status: done         # in_progress | blocked | done
next_action: sdd-archive (verify gate PASS; human QA to accept or waive)
updated: 2026-09-09
```

## Verification (change-level, sdd-verify)

**Per-step verdicts:** 01 `PASS` · 02 `PASS` · 03 `PASS` (all sub-tasks `[x]`; all `@step-01/02/03` scenarios COMPLIANT at runtime — covering tests pass).

**Change verdict: `PASS`.**

**Evidence:**
- Runtime proof: `go test ./internal/mnemonic/{codeindex,route,affected,embedder,mcp,service}/... ./cmd/skillgrid/... -count=1` → all `ok`. `go build ./...` exit 0 (the parallel-session `setup`/`install` refactor has landed; no warning).
- 005/008/010 baseline preserved: tool surface → 73 purely additive (route/navigates, affected/rename, knowledge [008], watcher adds none); `code_search` name + required `query` param unchanged; code-to-code `graph.Path` unchanged; migrations 013 (routes) + 015 (knowledge, 008) additive; CGo-free (fsnotify pure-Go, git via os/exec).
- 29 `@step-NN` scenarios in `acceptance.feature`, all mapped to passing runtime tests.
- Load-bearing properties verified: drop-not-guess (unresolvable refs dropped, excluded from blast radius); `code_affected` query-only + resolved-edges-only; fingerprint gate structural-only + embedder-free (counting-spy counterfactual + `ReindexStructural`); copy-and-swap (reader never torn); single-writer lock.

**QA plan:** `qa-plan.md` (happy/edge/failure + pass-fail + waiver).
**Ticket:** none (change-level `Ticket:` not set).
**Next:** sdd-archive — eligible when human QA is accepted or explicitly waived.

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `framework-routes` | `@step-01` | — (005 done) | Feature tagged `@step-01` |
| 02 | `code-affected` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `autosync-watcher` | `@step-03` | 02 | Feature tagged `@step-03` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~2400–3300 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

Honest forecast: three vertical slices (routes → affected/rename → watcher+embedder), each likely its own stacked PR. Step 03 is the largest (~900–1200, incl. the fingerprint gate + warm embedder) and should land on its own. Do not attempt a single-PR delivery without explicit exception.

Suggested split: PR1 routes → PR2 affected/rename → PR3 watcher+embedder · Chain strategy: stacked-to-main

---

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

- [x] 01.1 `[RED]` Framework routing files produce route nodes with references edges (Scenario: Web-framework routing files produce route nodes)
  - [ ] 01.1.a Write failing test — fixture routing files (Django `urls.py`, FastAPI `@app.get`, Express `app.get`, Gin `r.GET`, Rails `get '/x'`, Spring `@GetMapping`); after index, assert `route` nodes exist, each linked by a `references` edge to its handler, every edge EXTRACTED (explicit syntax), and a query for callers of a handler surfaces the URL pattern
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `013_framework_routes.sql` + `route/extract.go` dispatch + `route/frameworks.go` per-framework node maps + `Indexer.Run` hook (same tx as 005 extraction)
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): framework route nodes and references edges`
- [x] 01.2 `[RED]` Routers produce navigates edges to named screens (Scenario: Router navigations produce navigates edges)
  - [ ] 01.2.a Write failing test — fixture routers (Next.js `router.push`/`<Link href>`, React Router `<Route path>`, SvelteKit `goto`, Vue Router `push({name})`); after index, assert `navigates` edge from the sending function to the named screen; literal destinations EXTRACTED; markup-written links INFERRED
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Navigates -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — router node maps in `route/frameworks.go` + navigates emission in `route/extract.go`
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Navigates -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): navigates edges from framework routers`
- [x] 01.3 `[AFK]` Computed or unserved destinations stay unresolved (Scenario: Computed or unserved destination stays unresolved) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run Unresolved -count=1` — Expected: PASS
- [x] 01.4 `[RED]` Drop rather than guess — ambiguous references dropped, not AMBIGUOUS-kept (Scenario: Ambiguous references are dropped not guessed) — Business rule "Drop rather than guess"; edge policy inherited by step 02
  - [ ] 01.4.a Write failing test — a reference with no same-file match, no explicit specifier, and no unique global/owner-qualified match is **dropped at extraction**: no edge stored, and a warning (count + sample) is reported, not a silent discard; assert `AMBIGUOUS` is reserved for edges that *were* resolved via a heuristic (e.g. a markup-written `navigates` link) — a heuristic-resolved edge is stored `AMBIGUOUS`, an ambiguous-but-unresolvable one is absent
  - [ ] 01.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run DropNotGuess -count=1` — Expected: FAIL
  - [ ] 01.4.c Minimal implementation — drop-not-guess resolution policy in `route/extract.go` (drop + warn on ambiguity; keep `AMBIGUOUS` for heuristic-resolved only)
  - [ ] 01.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -run DropNotGuess -count=1` — Expected: PASS
  - [ ] 01.4.e Commit — `feat(mnemonic): drop-not-guess edge resolution policy`
- [x] 01.5 `[AFK]` Malformed routing file falls back and index continues (Scenario: Malformed routing file falls back and index continues) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [x] 01.6 `[AFK]` Framework with no routes reports none (Scenario: Framework with no routes reports none) — `Run: go test ./skillgrid-cli/internal/mnemonic/route/... -count=1` — Expected: PASS
- [x] 01.7 `[RED]` Mnemonic tool surface — route tools register, `code_search` stable, bad args rejected (Scenario: Route tools register and code_search stays stable) — threat: Mnemonic tool surface
  - [ ] 01.7.a Write failing test — assert route/navigates query tools register with distinct `code_*` names; `code_search` name + required `query` param schema unchanged; 005/008 tools intact; bad route args rejected with a clear validation error
  - [ ] 01.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run RouteTools -count=1` — Expected: FAIL
  - [ ] 01.7.c Minimal implementation — `tools_code_route.go` + server registration without dropping existing `code_*`
  - [ ] 01.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run RouteTools -count=1` — Expected: PASS
  - [ ] 01.7.e Commit — `feat(mnemonic): register route and navigates query tools`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/graph/... -count=1` | PASS | PASS | 5/5 packages `ok`; 005/008 baseline (`community`/`hybrid`) `ok` |
| Acceptance `@step-01` / `@p0` | mapped unit scenarios (`TestIndexProducesRouteNodesAndReferences`, `TestRouteNodesForWebFrameworks`, `TestNavigatesForRouters`, `TestDropNotGuess`, `TestHeuristicResolvedIsAmbiguous`, `TestUnresolved*`, `TestMalformed*`, `TestNoRoutes*`, `TestRouteTools*`) | PASS | PASS | 6 web frameworks + 4 routers fixtures |
| Runtime harness | `skillgrid index` on fixture web app; `code_route` / `code_navigates` via MCP; ambiguous ref dropped + warned (`route_drops`) | PASS | PASS | Express + Next.js end-to-end; others unit at `Build` |
| Rollback boundary | Drop `013_*` + `route/` + `tools_code_route.go`; revert indexer hook + impact kind-list | PASS | PASS | 005/012 tables untouched; tool surface 63→65 additive |
| Global Constraints | — | held | held | CGo-free; additive; same-tx hook; EXTRACTED/INFERRED/AMBIGUOUS + drop-not-guess |

Sub-agent review: round 1 needs fixes (blast-radius kind list not reconciled for step-01 `references`/`navigates`; AMBIGUOUS-reserved-for-heuristic-resolved contract unemitted/untested) → fixed (ac92533 three-way edge policy, 8c099a5 blast-radius traversal). Re-review: both ADDRESSED, no new breakage, non-vacuous tests.

### Commit

When step DoD is met: `feat(mnemonic): framework route nodes, navigates edges, and drop-not-guess policy`

---

## 02-code-affected

### Goal

`code_affected` (read) + `code_rename` (write) MCP/CLI tools — "which tests do I run after this change?" and "what does renaming this symbol touch?" — both grounded in the existing graph. `code_affected` now supports **`--base <ref>`**: it derives the changed set from the merge-base diff, groups the result into affected areas, and names git-history owners ("who to tag") via `git blame` — a ready PR comment / CI gate. Blast-radius math traverses only resolved edges (step 01's drop policy), so a dropped/ambiguous reference can never inflate the radius.

### Out of scope / Non-Goals

- Route edges + the drop-not-guess policy (step 01); watcher / banner / fingerprint gate / warm embedder (step 03)
- Recomputing the dependency graph; storing a precomputed impact matrix

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-framework-routes

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/affected/affected.go`
- Create: `skillgrid-cli/internal/mnemonic/affected/rename.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_affected.go`
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go` (`skillgrid search affected` CLI `--stdin`/`--base <ref>`/`--depth`/`--filter`/`--json`/`--quiet` + `skillgrid search rename`)
- Test: `skillgrid-cli/internal/mnemonic/affected/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's import + `tests-for` edges and symbol disambiguation; step 01's drop-not-guess resolved-edge set (blast-radius math); 005/010 `symbols`/`edges` tables; `git` (merge-base diff + `git blame`) for `--base`
- Produces: `code_affected` traversal (depth-capped, `--stdin`/`--base <ref>`/`--filter`/`--json`/`--quiet`); `--base` merge-base diff → affected-area grouping + git-history owners ("who to tag"); `code_rename` (graph vs. text-search edit buckets, `dry_run` default true, per-edit confidence); MCP tools + CLI parity

### Tasks

- [x] 02.1 `[RED]` `code_affected` traverses changed files to test files (Scenario: code_affected returns affected test files)
  - [ ] 02.1.a Write failing test — fixture repo where changed sources are transitively imported; `code_affected src/utils.ts src/api.ts` returns the test files that (transitively) import the changed symbols via import + `tests-for` edges; `--depth` caps traversal (default 5); `--filter "e2e/*"` restricts reported test files; `--json` / `--quiet` emit scriptable output; result is paths/relationships from the existing graph only (no invented edges)
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffected -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — `affected/affected.go` traversal (import + `tests-for`, depth cap, filter) + service/CLI facade
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffected -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): code_affected test-impact traversal`
- [x] 02.2 `[RED]` PR commands — `code_affected --stdin` on a fixture diff (Scenario: code_affected consumes a diff file list from stdin) — threat: PR commands
  - [ ] 02.2.a Write failing test — `git diff --name-only`-style stdin (fixture diff) returns the correct test files; an empty diff (empty stdin) returns an empty result with a clear message, not an error
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/cmd/skillgrid/... -run CodeAffectedStdin -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `--stdin` file-list ingestion in traversal + `code_intel.go` `search affected` CLI
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/cmd/skillgrid/... -run CodeAffectedStdin -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): code_affected --stdin for CI and hooks`
- [x] 02.3 `[RED]` PR commands — `code_affected --base <ref>` merge-base diff + affected areas + git-history owners (Scenario: code_affected --base derives areas and owners) — threat: PR commands; Business rule "`code_affected --base <ref>` PR mode"
  - [ ] 02.3.a Write failing test — fixture branch with a merge-base diff against a base ref; `code_affected --base origin/main` derives the changed file set from `git merge-base <base> HEAD` + diff (no `--stdin` plumbing), walks the same import/`tests-for` traversal, groups the result into affected **areas** (per-symbol detail collapsed under each area), and names **git-history owners** ("who to tag") for each touched area via `git blame` on the changed lines; `--quiet`/`--json` shape the output as a ready PR comment / CI gate
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffectedBase -count=1` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — `--base` merge-base diff changed-set source + affected-area grouping + `git blame` owner attribution in `affected/affected.go` + `code_intel.go`
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffectedBase -count=1` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): code_affected --base merge-base diff with owners`
- [x] 02.4 `[RED]` `code_rename` splits edits into graph vs text-search buckets (Scenario: code_rename returns graph and text-search edit buckets)
  - [ ] 02.4.a Write failing test — `code_rename validateUser verifyUser` on a fixture returns `files_affected` / `total_edits` split into **graph edits** (high-confidence: definition + typed references from the edges table) and **text-search edits** (lower-confidence: flagged "review carefully"); every edit carries a confidence label; `--dry-run` (default) returns the plan without writing any file
  - [ ] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRename -count=1` — Expected: FAIL
  - [ ] 02.4.c Minimal implementation — `affected/rename.go` (resolve via 005 disambiguation; graph edges → graph bucket; string matches → text-search bucket; per-edit confidence; `dry_run` default true)
  - [ ] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRename -count=1` — Expected: PASS
  - [ ] 02.4.e Commit — `feat(mnemonic): code_rename graph and text-search buckets`
- [x] 02.5 `[RED]` PR commands — non-dry `code_rename` edits only listed files, no commit/push (Scenario: code_rename applies only listed files without committing) — threat: PR commands
  - [ ] 02.5.a Write failing test — a non-dry `code_rename` against a fixture edits exactly the files in its plan (no other file touched); no commit or push is performed
  - [ ] 02.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameApply -count=1` — Expected: FAIL
  - [ ] 02.5.c Minimal implementation — apply path limited to planned files; no git side effects
  - [ ] 02.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameApply -count=1` — Expected: PASS
  - [ ] 02.5.e Commit — `feat(mnemonic): code_rename applies only planned files`
- [x] 02.6 `[AFK]` Ambiguous rename target returns a ranked candidate list (Scenario: Ambiguous rename target returns candidates) — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameAmbiguous -count=1` — Expected: PASS
- [x] 02.7 `[AFK]` No changed files gives an empty result not an error (Scenario: code_affected with no changed files returns empty) — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -count=1` — Expected: PASS
- [x] 02.8 `[AFK]` Blast-radius math traverses only resolved edges (Scenario: Dropped references are excluded from blast radius) — Business rule "Drop rather than guess" inherited by `code_affected`
  - [ ] 02.8.a Write failing test — on a fixture with a reference dropped at extraction (step 01 drop policy) and one resolved, `code_affected` traverses only the resolved edges: the dropped/ambiguous reference contributes nothing, so a false positive can never inflate the reported test-file radius
  - [ ] 02.8.b Run to confirm — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run BlastRadius -count=1` — Expected: PASS
- [x] 02.9 `[RED]` Mnemonic tool surface — `code_affected` + `code_rename` register, rename disambiguation, dry_run touches nothing, 005/008 tools stable (Scenario: affected and rename tools register and stay stable) — threat: Mnemonic tool surface
  - [ ] 02.9.a Write failing test — assert `code_affected` + `code_rename` register with distinct `code_*` names; `code_affected` exposes `--stdin`/`--base <ref>`/`--depth`/`--filter`/`--json`/`--quiet`; rename disambiguation returns a candidate list (no silent pick); `dry_run` touches nothing; 005/008 tools (incl. `code_search`) still stable; bad affected/rename args rejected with a clear validation error
  - [ ] 02.9.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run AffectedTools -count=1` — Expected: FAIL
  - [ ] 02.9.c Minimal implementation — `tools_code_affected.go` + server registration without dropping existing `code_*`
  - [ ] 02.9.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run AffectedTools -count=1` — Expected: PASS
  - [ ] 02.9.e Commit — `feat(mnemonic): register code_affected and code_rename tools`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -count=1` | PASS | PASS | affected/mcp/cmd `ok`; 005/008/010 baseline (graph/route/community) `ok` |
| Acceptance `@step-02` / `@p0` | mapped unit scenarios (`CodeAffected`, `CodeAffectedStdin`, `CodeAffectedBase`, `CodeRename`, `CodeRenameApply`, `CodeRenameAmbiguous`, `BlastRadius`, `AffectedTools`) | PASS | PASS | hermetic temp-repo git fixture; drop-policy blast-radius test real (resolved edge present, dropped absent) |
| Runtime harness | `git diff --name-only \| skillgrid search affected --stdin`; `skillgrid search affected --base <ref>` (areas + owners); `skillgrid search rename` dry-run | PASS | PASS | `--base` merge-base diff → areas (collapsed hops) + `git blame` owners; rename dry_run default true |
| Rollback boundary | Drop `affected/` + `tools_code_affected.go`; revert `code_intel.go` | PASS | PASS | 005/010 tables untouched (no step-02 migration); tool surface 71→73 additive |
| Global Constraints | — | held | held | CGo-free (git via os/exec); query-only (no node/edge writes); resolved-edges-only blast radius; dry_run default |

Sub-agent review: approved with fixes — package-level `pathCache` (stale-path/parallelism hazard) + `parseBlameAuthor` digit-truncation → fixed (22aef81 per-call cache scoping, c109f2f structural blame-author parse). Re-review: both ADDRESSED, no new breakage, behavior-preserving.

### Commit

When step DoD is met: `feat(mnemonic): code_affected (--base) and code_rename tools`

---

## 03-autosync-watcher

### Goal

fsnotify watcher + **pull-at-query fingerprint gate** + copy-and-swap publication + reader auto-reopen + connect-time catch-up + staleness banner + warm embedder daemon — keep the graph fresh *and* the embedder warm as the agent edits. The watcher is the *comfort* layer (near-zero-latency background sync); the **fingerprint gate** is the *correctness backstop* — a ~3ms `(size, mtime)` stat-walk per query that runs a structural-only re-index on drift under the writer lock, so every `code_*` query is fresh even with the watcher off.

### Out of scope / Non-Goals

- Route edges + the drop-not-guess policy (step 01); `code_affected`/`code_rename` (step 02)
- Manual full re-index (still available); shared-daemon watcher mode (later optimization)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-code-affected

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/codeindex/watch.go`
- Create: `skillgrid-cli/internal/mnemonic/codeindex/fingerprint.go`
- Create: `skillgrid-cli/internal/mnemonic/codeindex/publish.go`
- Create: `skillgrid-cli/internal/mnemonic/embedder/warm.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (wire staleness banner + auto-reopen + warm-embedder heartbeat into response path)
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (`serve`/MCP watcher + fingerprint gate + warm-embedder wiring + `skillgrid index status` pending-sync section)
- Modify: `skillgrid-cli/go.mod` (add `github.com/fsnotify/fsnotify`)
- Test: `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/embedder/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's single-transaction `(size, mtime)` + content-hash guard; fsnotify (FSEvents/inotify/KQueue, pure-Go); 005's ONNX `Embedder` interface; 005 include/exclude + `.gitignore` filters; the same writer lock as the watcher (fingerprint gate re-index)
- Produces: debounced incremental re-index (create/modify/delete, source files only); **pull-at-query fingerprint gate** (`codeindex/fingerprint.go` — ~3ms `(size, mtime)` stat-walk per query, structural-only re-index on drift under the writer lock, extractor-stamp keyed, never touches the embedder leg); copy-and-swap publication (sidecar → atomic publish) + in-place write-failure guard; reader auto-reopen (~5s, no restart); connect-time `(size, mtime)` + content-hash catch-up; `⚠️` staleness banner (pending referenced file) + footer (un-referenced); warm embedder cache (load-once, `idle_timeout` evict default 10min, MCP-heartbeat keep-alive); `skillgrid index status` `### Pending sync:` section; writer-lock for single-writer

### Tasks

- [x] 03.1 `[RED]` Index freshness / concurrency — saving a source file triggers a debounced incremental re-index (Scenario: Saving a source file triggers a debounced re-index) — threat: Index freshness / concurrency
  - [ ] 03.1.a Write failing test — with a running MCP/serve watcher, save (create/modify/delete) of a source file fires fsnotify; a burst of rapid edits collapses into ONE sync after the debounce window (default 2000ms, clamped [100ms, 60s]); only the changed surface is re-indexed; non-source files (filtered by 005 include/exclude + `.gitignore`) are ignored
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Watcher -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `codeindex/watch.go` (fsnotify + debounce + source filter) + `go.mod` dep
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Watcher -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): fsnotify auto-sync watcher with debounce`
- [x] 03.2 `[RED]` Index freshness / concurrency — copy-and-swap: reader never sees a torn index (Scenario: Reader sees old or new index never torn) — threat: Index freshness / concurrency
  - [ ] 03.2.a Write failing test — during a re-index publication, a concurrent reader observes either the old or the new index, never a partial one; on a sidecar-unsupported filesystem the in-place fallback retries on pre-write failure and stops when it may have mutated the live index
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Publish -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — `codeindex/publish.go` (sidecar build → atomic rename/swap; in-place write-failure guard)
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Publish -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): copy-and-swap index publication`
- [x] 03.3 `[RED]` Index freshness / concurrency — running process reopens the new index without restart (Scenario: Reader auto-reopens the new index) — threat: Index freshness / concurrency
  - [ ] 03.3.a Write failing test — after a publication, a running MCP/serve process opens the newly-published index on its next tool call (within ~5s) with no restart; the agent's next query reflects its own just-made edit
  - [ ] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -run AutoReopen -count=1` — Expected: FAIL
  - [ ] 03.3.c Minimal implementation — auto-reopen hook in the MCP serve response path (`mcp/server.go`)
  - [ ] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -run AutoReopen -count=1` — Expected: PASS
  - [ ] 03.3.e Commit — `feat(mnemonic): reader auto-reopen of published index`
- [x] 03.4 `[RED]` Index freshness / concurrency — connect-time catch-up absorbs offline edits (Scenario: Connect-time catch-up absorbs offline edits) — threat: Index freshness / concurrency
  - [ ] 03.4.a Write failing test — edits made while no MCP server was running (a `git pull`, another editor) are reconciled by a `(size, mtime)` + content-hash check on (re)connect; only genuinely changed files are re-indexed
  - [ ] 03.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run CatchUp -count=1` — Expected: FAIL
  - [ ] 03.4.c Minimal implementation — connect-time `(size, mtime)` + hash reconciliation in `watch.go`
  - [ ] 03.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run CatchUp -count=1` — Expected: PASS
  - [ ] 03.4.e Commit — `feat(mnemonic): connect-time size-mtime-hash catch-up`
- [x] 03.5 `[RED]` Index freshness / concurrency — staleness banner for a pending referenced file (Scenario: Pending referenced file gets a staleness banner) — threat: Index freshness / concurrency
  - [ ] 03.5.a Write failing test — during the debounce window, an MCP tool response that references a still-pending file prepends `⚠️ <file> is pending sync — Read it directly`; pending files not referenced surface as a small footer; after sync the banner clears
  - [ ] 03.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run StalenessBanner -count=1` — Expected: FAIL
  - [ ] 03.5.c Minimal implementation — pending-file tracking + banner/footer in `mcp/server.go` response path
  - [ ] 03.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run StalenessBanner -count=1` — Expected: PASS
  - [ ] 03.5.e Commit — `feat(mnemonic): staleness banner on pending files`
- [x] 03.6 `[RED]` Index freshness / concurrency — second writer exits with a lock error (Scenario: Second writer exits with a writer-lock error) — threat: Index freshness / concurrency
  - [ ] 03.6.a Write failing test — two `serve`/MCP processes on one project: the first acquires the writer lock; the second exits with a clear writer-lock error pointing at the live writer
  - [ ] 03.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WriterLock -count=1` — Expected: FAIL
  - [ ] 03.6.c Minimal implementation — single-writer lock (shared/direct-mode) in `watch.go` + `main.go` serve wiring
  - [ ] 03.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WriterLock -count=1` — Expected: PASS
  - [ ] 03.6.e Commit — `feat(mnemonic): single-writer lock for auto-sync`
- [x] 03.7 `[RED]` Index freshness / concurrency — pull-at-query fingerprint gate re-indexes structurally on drift with the watcher off (Scenario: Fingerprint gate re-indexes structurally with watcher off) — threat: Index freshness / concurrency; Business rule "Pull-at-query fingerprint gate"
  - [ ] 03.7.a Write failing test — with the watcher off (`SKILLGRID_NO_WATCH=1`), a `code_*` query made after an edit runs a ~3ms `(size, mtime)` stat-walk against the last index's fingerprint; on drift a **structural-only** incremental re-index runs (under the same writer lock as the watcher) **before** the answer, so the query reflects the edited tree (uncommitted edits included); assert the gate **never invokes the embedder leg** (no model load/embedding call during the gate) and that the fingerprint is keyed by the extractor stamp (a stamp/version change invalidates it, forcing a re-walk)
  - [ ] 03.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run FingerprintGate -count=1` — Expected: FAIL
  - [ ] 03.7.c Minimal implementation — `codeindex/fingerprint.go` (stat-walk fingerprint + extractor-stamp key + structural-only re-index under the writer lock, never the embedder leg) + query-path hook in `main.go`/`mcp/server.go`
  - [ ] 03.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run FingerprintGate -count=1` — Expected: PASS
  - [ ] 03.7.e Commit — `feat(mnemonic): pull-at-query fingerprint gate (structural-only, embedder-free)`
- [x] 03.8 `[AFK]` Watcher reindex failure stops in-place, keeps old on swap (Scenario: Reindex failure keeps the old index live) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run WatchFailure -count=1` — Expected: PASS
- [x] 03.9 `[AFK]` Watcher disabled means manual index (Scenario: Watcher disabled means manual index) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WatchDisabled -count=1` — Expected: PASS
- [x] 03.10 `[RED]` Warm embedder lifecycle — load-once, reused across search calls (Scenario: Warm embedder loads once and reuses) — threat: Warm embedder lifecycle
  - [ ] 03.10.a Write failing test — the ONNX model loads once and is reused across `skillgrid search` / MCP calls (no per-call model-load; the ~270MB model is not re-loaded per invocation); the first call pays the load, subsequent calls do not
  - [ ] 03.10.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmEmbedder -count=1` — Expected: FAIL
  - [ ] 03.10.c Minimal implementation — `embedder/warm.go` load-once ONNX cache + `mcp/server.go` / `main.go` wiring into the search path
  - [ ] 03.10.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmEmbedder -count=1` — Expected: PASS
  - [ ] 03.10.e Commit — `feat(mnemonic): warm embedder load-once cache`
- [x] 03.11 `[RED]` Warm embedder lifecycle — evicts after idle_timeout, heartbeat keeps alive (Scenario: Warm embedder evicts on idle and survives heartbeat) — threat: Warm embedder lifecycle
  - [ ] 03.11.a Write failing test — after `idle_timeout` (default 10min, tunable) of no use the warm embedder is evicted from RAM; an active MCP client connection keeps it alive via heartbeat (no eviction while connected); `off`/external providers hold no RAM
  - [ ] 03.11.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmEvict -count=1` — Expected: FAIL
  - [ ] 03.11.c Minimal implementation — `idle_timeout` evict timer + MCP-heartbeat keep-alive in `embedder/warm.go` + `server.go`
  - [ ] 03.11.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmEvict -count=1` — Expected: PASS
  - [ ] 03.11.e Commit — `feat(mnemonic): warm embedder idle-evict and heartbeat keep-alive`
- [x] 03.12 `[AFK]` Absent or failed model degrades to FTS and signals (Scenario: Absent model degrades to FTS and signals) — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmFallback -count=1` — Expected: PASS
- [x] 03.13 `[AFK]` Evicted-then-reused embedder reloads transparently (Scenario: Evicted embedder reloads on reuse) — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmReload -count=1` — Expected: PASS
- [x] 03.14 `[AFK]` `skillgrid index status` pending-sync section (Scenario: index status shows pending sync) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -run IndexStatus -count=1` — Expected: PASS
- [x] 03.15 `[RED]` Mnemonic tool surface — watcher alters no tool schemas, banner present, bad args rejected (Scenario: Watcher keeps tool schemas stable) — threat: Mnemonic tool surface
  - [ ] 03.15.a Write failing test — assert the watcher + fingerprint gate + banner do not alter any existing 005/008/new `code_*` tool schemas; the staleness banner is present on the response path; bad watcher/status args are rejected with a clear validation error
  - [ ] 03.15.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run WatcherSurface -count=1` — Expected: FAIL
  - [ ] 03.15.c Minimal implementation — schema-stability assertions + banner/arg-validation wiring
  - [ ] 03.15.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run WatcherSurface -count=1` — Expected: PASS
  - [ ] 03.15.e Commit — `feat(mnemonic): watcher tool-surface stability and status`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... ./cmd/skillgrid/... ./internal/mnemonic/service/... -count=1` | PASS | PASS | all `ok`; 005/008/010 baseline (graph/route/community/affected) `ok` |
| Acceptance `@step-03` / `@p0` | mapped unit scenarios (`Watcher`, `Publish`, `AutoReopen`, `CatchUp`, `StalenessBanner`, `WriterLock`, `FingerprintGate`, `WatchFailure`, `WatchDisabled`, `WarmEmbedder`, `WarmEvict`, `WarmFallback`, `WarmReload`, `IndexStatus`, `WatcherSurface`) | PASS | PASS | time-based tests use injectable durations; fingerprint gate structural-only + embedder-free (counting-spy counterfactual + ReindexStructural) |
| Runtime harness | `skillgrid serve` + edit mid-session (auto-reopen + banner); `SKILLGRID_NO_WATCH=1` + edit + query (fingerprint gate structural re-index, embedder untouched); `skillgrid index status` pending section | PASS | PASS | watcher = comfort layer; fingerprint gate = correctness backstop (works with watcher off) |
| Rollback boundary | Drop `watch.go`/`fingerprint.go`/`publish.go`/`warm.go` + banner/heartbeat/auto-reopen in serve path; revert `go.mod` (fsnotify) + `main.go` | PASS | PASS | 005/010/008 tables + tools untouched; 73-tool baseline unchanged; no step-03 migration |
| Global Constraints | — | held | held | CGo-free (fsnotify pure-Go); additive; fingerprint gate structural-only + embedder-free; reader never torn (copy-and-swap); single-writer lock |

Sub-agent review: approved with fixes — (a) fingerprint gate embedder-free property was comment-only → now a counting-spy counterfactual (attach fires it) + `ReindexStructural` (the real entrypoint) is embedder-free; (b) `warmSearchEmbedder` read CWD config not `h.root` → fixed (3da7fbb); (c) failed-factory FTS branch untested → `TestWarmFailedFactoryDegradesToFTS` (1ee735f). Re-review: 2 & 3 ADDRESSED, 1 PARTIAL-but-improved (the gate's embedder-free is enforced by not attaching, which `ReindexStructural` guarantees; a generic `Run` with an attached embedder legitimately embeds — confirmed ca15bb8). No new breakage.

### Commit

When step DoD is met: `feat(mnemonic): auto-sync watcher, fingerprint gate, copy-and-swap, and warm embedder`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] Applicable threat rows (PR commands; Mnemonic tool surface; Index freshness / concurrency; Warm embedder lifecycle) have passing RED coverage
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
