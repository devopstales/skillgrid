# QA Report — Web Admin Dashboard (Vite + React SPA)

> Change: `.skillgrid/specs/2026-09-08-web-admin-dashboard/`
> Generated: 2026-09-19T13:30:00Z
> Gate: **PASS**

## Test Plan

> Derived from `tasks.md` (7 phases, per-phase Verdicts) and `acceptance.feature` (89 scenarios: 59 @p0, 24 @p1, 6 @edge).

### Risk Ranking

The highest-risk behaviors are the **read-only bridges to external state**: the git CLI bridge (`/git/*` — a non-repo must 503, not 500), the `.stitch/` prototype file serving (**path traversal** — the one user-input-to-filesystem seam), and the SDD/`docsCwd` resolution (repo-root vs data-dir). These are tested hardest (P0, integration layer, real temp repos / fixture `.stitch/`). The activity SSE stream is the second risk (goroutine leak on client disconnect). UI rendering is lower-risk (verified by browser smoke + tsc + vitest for the pure logic modules).

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | SPA embed + shell fallback (non-API route → index.html) | `GET /` | integration | `TestPhase1_Embed`, `TestPhase1_SPAFallback` | P0 | pr | covered |
| 2 | Multi-provider tracker (backlog list/get/status/deps) | `GET/POST /tracker/*` | integration | `TestPhase2_Backlog`, `_Get`, `_SetStatus`, `_Deps`, `_BoardMapping`, `CLI_Failure`, `BadOutput` | P0 | pr | covered |
| 3 | Docs render (markdown + mermaid + DOMPurify) | `GET /docs/*` + UI | integration+ui | `TestPhase3_*` (docs handlers) + browser smoke | P0 | pr | covered |
| 4 | Mnemonic vector graph (Sigma + graphology, progressive layout) | `GET /mnemonic/graph/data` + UI | integration+ui | `TestPhase4_*` + `layouts.test.ts` (7) + `camera.test.ts` (5) | P0 | pr | covered |
| 5 | Mnemonic suite (files/memories/sessions/audit/search) | `GET /mnemonic/*` | integration | `TestPhase5_*` (files/memories/sessions/search) | P0 | pr | covered |
| 6 | Activity events/stats/SSE (newest-first, leak-free) | `GET /activity/*` | integration | `TestPhase6_Activity`, `TestPhase6_ActivitySSE` | P0 | pr | covered |
| 7 | SDD plans/specs aggregation (status/progress/files/steps) | `GET /plans`, `/specs` | integration | `TestPhase6_Plans`, `TestPhase6_Specs` | P0 | pr | covered |
| 8 | Git bridge (commits/diff/history/blame) + non-repo 503 | `GET /git/*` | integration | `TestPhase6_Git` (real temp repo), `TestPhase6_GitNotRepo` | P0 | pr | covered |
| 9 | **Path traversal on `/prototypes`** (sandboxed to `.stitch/`) | `GET /prototypes/{id}` | integration | `TestPhase7_Prototypes` (../, %2e%2e, absolute, nested, unknown) | P0 | pr | covered |
| 10 | Route code-splitting + bundle budget | SPA build | ui | `npm run build:check` (index 294.3 kB < 400 kB budget) | P0 | pr | covered |
| 11 | Responsive + density + motion | SPA | ui | browser smoke (viewport + density toggle `<html data-density>`) | P1 | pr | covered |
| 12 | openapi documents all routes + swagger serves | `GET /openapi.yaml`, `/swagger-ui/` | integration | `TestPhase6_OpenAPI`, `TestPhase7_OpenAPI` + curl 200 | P0 | pr | covered |
| 13 | User manual documents views + tracker CLI | `docs/user-guide/09-serve-dashboard.md` | docs | `TestPhase7_UserManual` | P1 | pr | covered |
| 14 | Full DoD smoke (build+test+vet+tsc+browser) | all | all | per-phase Verdicts + this report's runs | P0 | pr | covered |

**Layer** — integration (real `http.Handler` + real temp repos / SQLite fixtures) for backend; ui (vitest for pure logic, browser smoke for rendering) for frontend. No e2e layer is configured (`testing.layers: [unit, integration]`).

**Cadence** — all `pr` (these block merge; they are the regression net for the dashboard).

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| Prototypes traversal | `../secret.html`, `..%2Fsecret.html`, `%2e%2e/secret.html` | 400/404, sibling `secret.html` NOT served | `TestPhase7_Prototypes` | covered |
| Prototypes absolute | `/etc/hostname`, drive-letter id | 400 (rejected) | `TestPhase7_Prototypes` | covered |
| Prototypes unknown id | `nope.html` | 404 | `TestPhase7_Prototypes` | covered |
| Prototypes nested | `nested/proto-b.html` | 200 (still sandboxed) | `TestPhase7_Prototypes` | covered |
| Git non-repo | served dir is not a git worktree | 503 | `TestPhase6_GitNotRepo` | covered |
| Activity SSE disconnect | client disconnects mid-stream | no goroutine leak | `TestPhase6_ActivitySSE` (leak check) | covered |
| Activity limit clamp | `limit=0` / `limit=999999` | clamped 1..500 | `TestPhase6_Activity` | covered |
| Plans no ledger | plan with no `.skillgrid/sdd` steps | `steps: []` (not null), UI `?.length ?? 0` | `TestPhase6_Plans` + `PlanDetail.tsx` | covered |
| SPA/API collision | `/plans` browser (text/html) vs API (json) | shell vs JSON | `TestPhase6_*` (Accept: application/json) + browser | covered |
| Bundle budget | index chunk > 400 kB | build:check exits 1 | `scripts/check-bundle-size.mjs` | covered |

### Out of Scope

- **Mermaid diagram rendering fidelity** — covered by react-markdown/mermaid's own behavior; we test the docs handlers + that diagrams render in-browser (smoke), not diagram-output byte-equality.
- **Sigma graph visual correctness** — we test the layout math (`layouts.test.ts`, `camera.test.ts`) + that the graph renders (smoke), not pixel-exact node positions.
- **Tracker third-party CLIs (github/linear)** — `gh`/`linear` behavior is their contract; we test the adapter (backlog provider, the built-in) + the degraded-state reporting.
- **4 pre-existing `service` test failures** — `TestOpenFor*` in `internal/mnemonic/service` (see Findings; environment issue, not this change).

## Goal-Backward Verification

**Stated goal** (from briefing.md): Operators can open `http://127.0.0.1:7438/` from a running `skillgrid serve` and administer all surfaces (Tracker, Mnemonic Graph/Files/Memories/Sessions/Search, Docs, Plans, Activity, Git, Prototypes) from one dashboard — embedded in the Go binary, no separate frontend process.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | `GET /` serves the SPA shell with every view | `TestPhase1_Embed`, `TestPhase1_SPAFallback`; browser: all 10 sidebar views render | VERIFIED |
| Truth | Multi-provider Kanban works on the active tracker | `TestPhase2_Backlog*` (list/get/status/deps); browser: Kanban renders | VERIFIED |
| Truth | Vector graph renders the memory graph (GitNexus parity) | `TestPhase4_*` + `layouts.test.ts` (progressive layout) + browser: Sigma renders 4379-node fixture | VERIFIED |
| Truth | Activity stream is live (SSE) + leak-free | `TestPhase6_ActivitySSE` (ready + live event + goroutine leak check) | VERIFIED |
| Truth | SDD Plans view reflects the `.skillgrid/` ledger | `TestPhase6_Plans` (status/progress/files/steps) + browser: 8 plan cards + spec viewer | VERIFIED |
| Truth | Git view (commits/diff/blame) is read-only | `TestPhase6_Git` (real temp repo) + `git.go` has no push/commit/merge/reset | VERIFIED |
| Truth | Prototypes gallery is sandboxed to `.stitch/` (traversal-guarded) | `TestPhase7_Prototypes` (traversal/absolute/nested/unknown) | VERIFIED |
| Truth | No separate frontend process (go:embed) | `TestPhase1_Embed`; `ui.go` serves embedded `ui/dist`; binary rebuilt post-ui-build | VERIFIED |
| Artifact | All 10 feature dirs + backend handlers exist | `features/{activity,plans,git,prototypes,docs,kanban,tracker,mnemonic,settings}`; `http/{activity,plans,git,prototypes}.go` | VERIFIED |
| Key Link | SPA route → lazy chunk → data fetch → render | browser smoke: each view loads (lazy) + fetches + renders, 0 console errors | VERIFIED |
| Data Flow | real SQLite → `/mnemonic/*` → React render | browser: fixture `aiskillgrid.sqlite` (4379 nodes) renders in Graph/Memories/Sessions | VERIFIED |

## Traceability Matrix

89 scenarios total (59 @p0, 24 @p1, 6 @edge). Grouped by phase; each phase's `@happy`/`@edge`/`@failure` scenarios are covered by the named phase tests (integration) + browser smoke (UI rendering scenarios). Every scenario's backend behavior has a passing integration test; UI-rendering scenarios are verified by the browser smoke pass (no dedicated e2e harness exists — `testing.layers` has no e2e).

| Scenario (grouped) | Test / Scenario (from test plan) | Ran | Result |
|---|---|---|---|
| Phase 1 (embed + shell, ~4 scenarios) | `TestPhase1_Embed`, `TestPhase1_SPAFallback` | yes | pass |
| Phase 2 (tracker kanban, ~14 scenarios) | `TestPhase2_Backlog*`, `_BoardMapping`, `CLI_Failure`, `BadOutput` | yes | pass |
| Phase 3 (docs render, ~6 scenarios) | `TestPhase3_*` docs handlers + browser mermaid smoke | yes | pass |
| Phase 4 (vector graph, ~12 scenarios) | `TestPhase4_*` + `layouts.test.ts` + `camera.test.ts` + browser | yes | pass |
| Phase 5 (mnemonic suite, ~20 scenarios) | `TestPhase5_*` (files/memories/sessions/audit/search) + browser | yes | pass |
| Phase 6 (activity/plans/git, ~14 scenarios) | `TestPhase6_Activity`, `_ActivitySSE`, `_Plans`, `_Specs`, `_Git`, `_GitNotRepo`, `_OpenAPI` | yes | pass |
| Phase 7 (prototypes/polish, ~19 scenarios) | `TestPhase7_Prototypes`, `_OpenAPI`, `_UserManual` + `build:check` + browser | yes | pass |

**Coverage:** 89/89 scenarios mapped to a covering test (backend, integration layer) that ran and passed. UI-rendering scenarios (the "…view renders…" @happy) are additionally verified by the browser smoke pass (all 10 views render, 0 console errors); there is no automated e2e harness (`testing.layers` excludes e2e), so these are browser-verified, not test-verified — recorded as an open item below, not a FAIL (the rendering logic that *can* be unit-tested is unit-tested: layouts/camera/explorer = 23 vitest tests).

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| UI "renders" scenarios have no automated e2e assertion | `acceptance.feature` @happy "…view renders…" rows | missing-oracle (no automated CHECK) | `testing.layers: [unit, integration]` — no e2e layer; rendering verified by manual browser smoke only | A UI refactor that breaks a view's render would not fail CI (caught only by manual smoke) |
| Mermaid diagram output not asserted byte-for-byte | `features/docs/MarkdownView.tsx` | broken-verification (low) | we assert the docs handlers + that diagrams render in-browser, not diagram SVG output | A mermaid version bump that changes output would not be caught |

Both gaps are **low-risk** and accepted (recorded as open items → a future e2e-harness change), not CRITICAL: the backend behavior of every scenario is test-covered, and the rendering is browser-verified.

## TDD Evidence Audit

Per-phase TDD was followed during execution (RED test written + confirmed failing, then minimal implementation + confirmed passing, committed). Spot-verified for the phases completed this session:

| Task | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|---|---|---|---|---|
| 7.1 `/prototypes` traversal | "Path traversal on /prototypes returns 400" | `prototypes_test.go` written first, `go test -run TestPhase7_Prototypes` → FAIL (SPA shell returned) | `prototypes.go` added → PASS (commit `67eac44`) | OK |
| 6.1 activity SSE | "GET /activity/stream emits new events live (SSE)" | `activity_test.go` TestPhase6_ActivitySSE RED | `activity.go` → PASS | OK |
| 6.2 plans/specs | "GET /plans returns SDD plans with status and progress" | `plans_test.go` RED | `plans.go` → PASS (commit `8d99fcf`) | OK |
| 6.3 git | "GET /git/commits returns the commit list" | `git_test.go` RED (503 / no handler) | `git.go` → PASS (commit `69900bc`) | OK |
| 7.3 code-splitting | "Heavy features are code-split and bundle stays within budget" | build showed single 814 kB index | `app.tsx` lazy routes + `build:check` → 294.3 kB PASS (commit `d0109bc`) | OK |

Earlier phases (1–5) followed the same RED→GREEN→commit pattern (per their per-phase Verdicts + commit history). No `MISSING_RED` found in the spot-audit.

## Test Quality Audit

| Test | Contract Violated | Evidence |
|---|---|---|
| (none) | — | Spot-audit of the Phase 6/7 tests: they assert on real handler output (status code + JSON body content), not on mocks. `TestPhase7_Prototypes` has explicit negative assertions (sibling `secret.html` must NOT be served). `TestPhase6_ActivitySSE` has a goroutine-leak counter-test (asserts no leak after client disconnect). No tautologies, snapshot-only, or source-text assertions. |

P0 triangulation: the traversal guard is tested with 4 distinct inputs (`../secret.html`, `..%2F`, `%2e%2e/`, absolute) — triangulated. The activity SSE is tested for both the initial `ready` and a live event — triangulated.

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration (no e2e).

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|---|---|---|---|---|
| Backend routes (all `/activity`, `/plans`, `/git`, `/prototypes`, `/tracker`, `/mnemonic`, `/docs`) | integration | no | no | real `http.Handler` + real temp repos/SQLite — the seam is the HTTP boundary |
| Layout/camera/explorer math | unit (vitest) | no | no | pure functions, no I/O — unit is the right layer |
| UI rendering | browser smoke (no e2e layer) | yes (degraded from e2e) | no | no e2e harness configured; recording the degradation |

**Layer distribution:** ~51 integration (Go phase tests) + 23 unit (vitest) + browser smoke. Reasonable: the dashboard's risk is at the HTTP seam (integration) and in pure layout math (unit); rendering is browser-verified.

## Security Audit

**Mode:** Manual fallback (no `security.trivy.command` configured).

### Manual Findings (Mode B)

#### Secrets Archaeology

| Location | Type | Evidence | Severity |
|---|---|---|---|
| (none) | — | `rg` over the new files (`activity.go`, `plans.go`, `git.go`, `prototypes.go`) for `password|secret|api_key|token=` — no hardcoded secrets. The only token reference is the existing `SKILLGRID_HTTP_TOKEN` bearer gate (write routes), unchanged. | — |

#### Dependency Audit

| Dependency | CVE / Issue | Severity | Fix Available |
|---|---|---|---|
| (none new) | — | This change adds no new Go or npm runtime dependencies (git via CLI, layout via existing `graphology-layout-forceatlas2`). `sigma`/`graphology` were added by the graph-upgrade change, not this one. | — |

#### OWASP Spot-Check

| File | OWASP Item | Pattern | Evidence |
|---|---|---|---|
| `prototypes.go` | A01 (Broken Access) / path traversal | user-input `id` → filesystem path | `stitchFile()` cleans + `filepath.Rel` prefix check; served with `X-Content-Type-Options: nosniff`; iframe uses `sandbox="allow-scripts"` (no `allow-same-origin`) |
| `git.go` | A03 (Injection) | user-input `sha`/`path` → `git` CLI | args passed as separate `exec.Command` args (not shell-interpolated); `sha`/`path` validated against the repo |
| `activity.go` (SSE) | A06 (DoS) | long-lived connection | poller is `ctx.Done()`-bounded; client disconnect stops the goroutine (leak-tested) |

**Security verdict:** PASS — 0 CRITICAL, 0 WARNING, 3 SUGGESTION (documented mitigations above). The one user-input-to-filesystem seam (`.stitch/` prototypes) is traversal-guarded and nosniff'd.

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|---|---|---|---|---|
| Coverage | (not configured) | `coverage_min: 0` | — | N/A |
| Mutation | (not configured) | `mutation_min: 0` | — | N/A |
| Lint | (not configured) | errors only | — | N/A |
| Typecheck | `npx tsc -b --noEmit` | errors only | 0 errors | PASS |
| Go vet | `go vet ./internal/mnemonic/http/...` | errors only | 0 | PASS |
| P0 pass rate | `go test -run TestPhase` (P0 subset) | 100 | 51/51 phase tests pass | PASS |
| P1 pass rate | P1 subset | 95 | pass | PASS |
| Trivy security | (not configured) | report only | — | N/A |
| Bundle budget | `npm run build:check` | index < 400 kB | 294.3 kB | PASS |

**Quality config status:** configured (thresholds explicit: coverage/mutation disabled at 0, P0=100, P1=95).

## Findings

### CRITICAL (must fix before merge)

- (none)

### WARNING (should fix before archive)

- **Pre-existing `service` test failures (not this change):** 4 tests in `internal/mnemonic/service` (`TestOpenFor*`) fail because the global conventional-commits git hook rejects `git commit -m init` in the test's temp repo. Confirmed pre-existing: they fail identically with the `http` (this change's) package stashed, and `store`/`route`/`search`/`http` all pass. **Path to resolution:** make the test fixtures use `--no-verify` on the temp-repo `git commit`, or a conventional subject (`chore: init`). Out of scope for this change (different package, environment issue).

### SUGGESTION (nice to have)

- **UI "renders" scenarios lack automated e2e assertions** — add an e2e harness (Playwright) for the @happy "…view renders…" scenarios so a UI-render regression fails CI. Currently browser-verified by hand.
- **Test-only dev hooks remain** — `__skillgridGraph` / `__skillgridGraphByOrder` on `globalThis` in `layouts.ts` (from the graph-upgrade change) are test-only; remove or gate behind a dev flag before a future release.
- **Mermaid diagram output not byte-asserted** — a mermaid version bump could change rendered SVG without a test catching it.

## Gate Decision

**Verdict:** **PASS**

**Reasoning:** All 8 falsifiable truths are VERIFIED by named integration tests (goal-backward, 4 levels). All 89 scenarios are mapped to a covering test that ran and passed (backend, integration); UI-rendering scenarios are additionally browser-verified (all 10 views render, 0 console errors) with the degradable rendering logic unit-tested (23 vitest tests). No CRITICAL findings from the verification-gap / TDD / test-quality / security audits. All code-quality gates PASS or N/A (tsc 0, go vet 0, P0 100%, bundle 294.3 kB < 400 kB). The only WARNING is the pre-existing `service` package failure, explicitly out of scope (environment issue, different package, confirmed unrelated by stashing this change's code).

**Open items (for a future change, non-blocking):**
- Pre-existing `service` test failures (`TestOpenFor*`) — fix the temp-repo `git commit` to use `--no-verify` or a conventional subject. Path: `internal/mnemonic/service/open_test.go` + `cross_store_test.go`.
- Add an e2e harness (Playwright) for the UI "renders" scenarios.
- Remove/gate the `__skillgridGraph` test hooks in `layouts.ts`.

**Waiver (if WAIVED):** N/A — no waiver; the gate is a machine PASS.

## Human Override

> A human decision always overrides this machine verdict.

**Human decision (fill in):** _pending — machine verdict is PASS; no override recorded._
