# Mnemonic Web UI Rewrite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use simple-execution to work this plan ticket-by-ticket from `tasks.md`. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** ACCEPTED (as-built — the code is commit `aa0aa791` on `release/2`; this plan documents and verifies it, and closes the gaps verification found)

**Tier:** T2

**Build shape:** Facade — the SPA shell (theme, nav IA, routes) stood up first against the mockup, then each panel was wired to its live Go endpoint. Recorded because the look needed sign-off (prototype 001) before the panel plumbing.

**Classification:** standard → verification floor L2 (full UI suite + bundle budget + Go build with and without `-tags ui`).

**Goal:** The embedded `skillgrid-ui` SPA matches prototype 001's information architecture, visual language, and panel set, keeps the production scaffolding, and uses the mockup's D3 force code graph.

**Architecture:** Vite + React 19 + TypeScript + Tailwind v4 + TanStack Router SPA under `skillgrid-ui/`, built into `skillgrid-cli/internal/mnemonic/http/ui/dist` and embedded by the Go server behind `-tags ui` (fallback shell otherwise). Panels fetch the Go read bridges through one `apiGet` helper that injects `project` and uses relative URLs in production (`apiOrigin()` returns `''` outside Vite dev). The code graph is a D3 v7 force simulation over `/mnemonic/graph/data?limit=500` with a node inspector.

**Tech Stack:** React 19, TypeScript 5.8, Vite 6, Tailwind 4, TanStack Router 1.x, Vitest 5 + Testing Library, d3 7 (ADR-0017), Go 1.22+ `embed`.

**Spec:** `./briefing.md` (intent, locked decisions, out of scope, success). Decisions: `.skillgrid/artifacts/04-adr-0017-d3-force-graph.md`. Manifest: `./adr.md`.

**Findings:** `.skillgrid/prototypes/001-mnemonic-webui-mockup/prototype.md` (PARTIAL verdict: 14 sections render; gaps are API-side) and `findings.md` (CORS prerequisite; `project` param required; graph `limit=500` ≈ 365 kB JSON, 500 nodes render fine in D3 v7).

## Terms

- **Prototype** (was *spike*) — a feasibility probe under `.skillgrid/prototypes/NNN-name/`; see `.skillgrid/artifacts/02-technical-terms.md`. Note: at `release/2` HEAD the SPA page and route still carry the old name (`SpikesPage`, `/project/spikes`); the repo-wide rename is in flight in the working tree outside this change.
- **Read bridge** — a Go HTTP GET handler that serves `.skillgrid/`, tracker, git, or Mnemonic store data as JSON to the SPA (`02-technical-terms.md`).

## Hypothesis

**Claim:** Porting the mockup's IA, tokens, and D3 graph into the existing Vite/React scaffolding yields a working admin console without changing the Go API contract.
**Right condition:** every nav leaf renders against the live Go server; `npm test`, `npm run build:check`, `go build ./...`, `go build -tags ui ./...` all exit 0; the graph loads 500 nodes with the inspector.
**Wrong condition:** a panel needs a Go endpoint that does not exist, or the bundle budget (400 kB index) is exceeded by d3.
**Thinnest MVP:** the shell (theme + nav + routes) plus the D3 Code Graph page against `/mnemonic/graph/data`.
**Door check:** Task 2 (D3 graph under the bundle budget). It held — index chunk 298 kB / 400 kB with d3 split into `vendor-d3`.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- The sidebar renders Overview · Project · Memory · Observe · Docs · System in the mockup's order with the mockup's leaves (`nav-matches-mockup`).
- `/git`, `/mnemonic/memories`, `/mnemonic/search`, `/decisions`, `/prototypes` still resolve to their pages (`nav-matches-mockup`).
- GraphPage fetches with `limit: 500`, runs `d3.forceSimulation`, and shows a Node Inspector; `sigma`/`graphology`/`@react-sigma/core` are gone from `package.json` (`d3-code-graph`).
- Compaction, Web Cache, Security fetch `/context`, `/context/compaction`, `/web/status`, `/web/search`, `/security/trivy`, and every one of those is registered in `server.go` (`panels-fetch-live-endpoints`).
- The Prototypes page fetches a path that `server.go` registers as a JSON handler (`panels-fetch-live-endpoints`) — **backstop**: at HEAD this is false (`/spikes` is unregistered); see Task 4.
- `apiOrigin()` is `''` outside `import.meta.env.DEV`; no `fetch()` call site hardcodes `127.0.0.1:7438` (`relative-api-urls`).
- A briefing containing `#012` renders an anchor to `/tracker?task=012` after the move to MarkdownView (`briefing-task-refs-linkified`) — **backstop**: fails at HEAD, fixed by Task 5.
- `npm test`, `npm run build:check`, `go build ./...`, `go build -tags ui ./...` exit 0 (`verification-floor`).

**Artifacts** (files that must exist with real implementation, not stubs):
- `skillgrid-ui/src/components/layout/AppLayout.tsx` — grouped NAV table + slide-over + project/density footer.
- `skillgrid-ui/src/styles/index.css` — surface/indigo tokens, `nav-item`, `kpi-card` primitives.
- `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` — D3 force layout + inspector (callers/callees highlight).
- `skillgrid-ui/src/features/observe/{CompactionPage,TelemetryPage,WebCachePage}.tsx`, `skillgrid-ui/src/features/security/SecurityPage.tsx` — live-endpoint panels.
- `skillgrid-ui/src/lib/api.ts`, `skillgrid-ui/src/lib/apiBase.ts` — `apiGet`/`apiFetch` + `apiOrigin`/`apiUrl`.
- `skillgrid-ui/src/features/plans/taskLinks.ts` — `linkifyTaskRefs` + `linkifyTaskRefsMarkdown`.

**Key links** (critical connections between artifacts that must work together):
- `app.tsx` route paths ⇄ `AppLayout.tsx` `NAV[].to` — every nav `to` has a route; `/project/*` and `/system/*` prefixes keep the SPA routes off the JSON paths `GET /prototypes` and `GET /security/trivy`.
- `GraphPage.tsx` → `features/mnemonic/graph/api.ts#fetchGraph` → `GET /mnemonic/graph/data?project=…&limit=500`.
- `PlanDetail.tsx` → `linkifyTaskRefsMarkdown(plan.briefing)` → `MarkdownView` (markdown links survive sanitisation; raw `<a>` did not).
- `vite.config.ts` `build.outDir` → `skillgrid-cli/internal/mnemonic/http/ui/dist` → `embed_fs_dist.go` (`-tags ui`).

**One-way-door decisions:** None. Removing Sigma is reversible (git revert of `aa0aa791`'s dependency hunk); no migration, no public API change.

## Global Constraints

- Go 1.22+ to build (`per ASSUMPTIONS.md § Locked constraints`).
- No new dependencies without an ADR — `d3` is covered by `per .skillgrid/artifacts/04-adr-0017-d3-force-graph.md`; no other dependency may be added by this change.
- Spec-zone changes commit before code-zone changes (`per ASSUMPTIONS.md § Locked constraints`).
- Serial development: one change at a time — this execution commits only its own paths; the concurrent uncommitted work in the main working tree (Teams view, agent observability, spikes→prototypes rename) is not swept into its commits.
- Bundle budget: `scripts/check-bundle-size.mjs` index chunk ≤ 400 kB.
- Routes: UI keeps `/project/prototypes` (at HEAD `/project/spikes`) and `/system/security` so JSON `GET /prototypes` and `GET /security/trivy` stay unambiguous (`per briefing.md § Locked decisions`).
- Out of scope (`per briefing.md § Out of scope`): auth, pagination, mobile polish beyond the slide-over, backend FTS5 search gap, doc-root configuration.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Routing (SPA route vs JSON route collision) | Applicable | SPA pages for colliding names live under `/project/*` and `/system/*`; `apiPrefixes` in `embed.go` returns JSON 404, never the shell, under API prefixes | `AppLayout.test.tsx` nav assertions + G2 route grep; Go `embed_test.go` already covers the JSON-404 rule |
| Markdown rendering of repo content (XSS via briefing/tasks) | Applicable | `MarkdownView` runs `rehype-sanitize`; task links are emitted as markdown links, not raw HTML (Task 5) | `taskLinks.test.ts` "emits a markdown link"; `PlansPage.linkedTasks.test.tsx` |
| Swagger iframe (framing the SPA inside itself) | Applicable | iframe src is `apiUrl('/swagger/')` — absolute origin only in Vite dev, relative in production | `SwaggerPage.test.tsx` |
| Shell commands / subprocesses / VCS automation | N/A: the SPA issues HTTP GETs only | — | — |
| Mnemonic tool contracts | N/A: no MCP tool surface changes | — | — |

## File Structure

- `skillgrid-ui/src/styles/index.css` — theme tokens + layout primitives (mockup surface/indigo).
- `skillgrid-ui/src/components/layout/AppLayout.tsx` — grouped sidebar nav, slide-over, footer (project, density).
- `skillgrid-ui/src/components/ui/Badges.tsx` — `PageHeader`, `EmptyState`, `ErrorState`, `LoadingState`, status badges shared by every panel.
- `skillgrid-ui/src/app.tsx` — TanStack route tree (lazy pages).
- `skillgrid-ui/src/lib/api.ts` / `apiBase.ts` — JSON fetch helpers; `project` injection; dev-only absolute origin.
- `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` + `graph/api.ts` + `graph/types.ts` — D3 force graph + inspector.
- `skillgrid-ui/src/features/observe/*.tsx` — Telemetry, Compaction, Web Cache.
- `skillgrid-ui/src/features/security/SecurityPage.tsx` — Trivy report panel.
- `skillgrid-ui/src/features/spikes/SpikesPage.tsx` (→ `features/prototypes/PrototypesPage.tsx` after the rename) — Project → Prototypes list.
- `skillgrid-ui/src/features/settings/SettingsPage.tsx` — config/state grid + HTTP token.
- `skillgrid-ui/src/features/plans/{PlanDetail.tsx,taskLinks.ts,taskLinks.test.ts}` — briefing task-ref links (regression fix).
- `skillgrid-ui/vite.config.ts` — dev proxy prefixes + `build.outDir` into the Go embed dir + manual chunks (`vendor-d3`).
- `skillgrid-cli/internal/mnemonic/http/embed.go` — serves dist (`-tags ui`) or fallback; API-prefix JSON 404s.

---

### Task 1: Theme tokens + nav IA + relative API base (as-built)

**Files:**
- Modify: `skillgrid-ui/src/styles/index.css`, `skillgrid-ui/src/components/layout/AppLayout.tsx`, `skillgrid-ui/src/app.tsx`, `skillgrid-ui/src/lib/api.ts`, `skillgrid-ui/src/lib/apiBase.ts`, `skillgrid-ui/vite.config.ts`
- Test: `skillgrid-ui/src/components/layout/AppLayout.test.tsx`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `apiGet<T>(path, params?, {project?})`, `apiFetch<T>(path)`, `apiOrigin(): string`, `apiUrl(path): string`; `NAV` entries `{id,to,label,icon,top?}` and `{group}`.
- Seam: `apiBase.ts#apiOrigin` — the only place the dev origin lives; production is `''`.
- Deletion test: without `api.ts`, every panel re-implements `project` injection + error unwrapping (CompactionPage, WebCachePage, TelemetryPage, SpikesPage, GraphPage, Overview).
- Adapters: 2 — Vite dev (absolute origin) and production (same-origin relative).

**SATISFIES:** `nav-matches-mockup`, `theme-tokens`, `relative-api-urls`

- [ ] **Step 1:** G1 `npx vitest run src/components/layout/AppLayout.test.tsx` — expect PASS.
- [ ] **Step 2:** G2 route grep — expect 5.
- [ ] **Step 3:** G3 token grep — expect indigo accent + surface tokens.
- [ ] **Step 4:** G8 origin grep — expect no `fetch()` site outside `apiBase.ts`.
- [ ] **Step 5:** Ledger line in `.skillgrid/sdd/2026-10-02-mnemonic-webui-rewrite/progress.md`; no commit (code already at `aa0aa791`).

### Task 2: D3 force code graph with inspector (ADR-0017) — door check

**Files:**
- Modify: `skillgrid-ui/src/features/mnemonic/GraphPage.tsx`, `skillgrid-ui/src/features/mnemonic/graph/api.ts`, `skillgrid-ui/package.json`
- Delete: `skillgrid-ui/src/features/mnemonic/graph/{VectorGraph,ExplorerPanel,layouts,camera,...}` (Sigma stack)
- Test: `skillgrid-ui/scripts/check-bundle-size.mjs` (budget), G4/G5 greps

**Interfaces:**
- Consumes: `apiGet` (Task 1).
- Produces: `fetchGraph({limit}) → GraphData {nodes, edges}`; `GraphPage` default export.
- Seam: `graph/api.ts#fetchGraph` (server `limit` param).
- Deletion test: pass-through for the page; the `limit` cap is the only logic worth keeping — findings say the full graph (4.8k nodes) is unusable client-side.
- Adapters: 1 (live server) + test fixture in GraphPage tests → 2.

**SATISFIES:** `d3-code-graph`

- [ ] **Step 1:** G4 grep — three lines (`limit: 500`, `forceSimulation`, `Node Inspector`).
- [ ] **Step 2:** G5 dependency grep — 0 Sigma matches, 1 `d3`.
- [ ] **Step 3:** `npm run build:check` — `vendor-d3` chunk split; index ≤ 400 kB.
- [ ] **Step 4:** Ledger line.

### Task 3: Observe + System panels on live endpoints (as-built)

**Files:**
- Create: `skillgrid-ui/src/features/observe/{CompactionPage,TelemetryPage,WebCachePage}.tsx`, `skillgrid-ui/src/features/security/SecurityPage.tsx`
- Modify: `skillgrid-ui/src/features/settings/SettingsPage.tsx`, `skillgrid-ui/src/features/swagger/SwaggerPage.tsx`
- Test: `SettingsPage.test.tsx`, `SwaggerPage.test.tsx`, G6 grep

**Interfaces:**
- Consumes: `apiGet`/`apiFetch`/`apiUrl` (Task 1); `PageHeader`/`EmptyState`/`ErrorState`/`LoadingState` (Badges).
- Produces: nothing downstream.
- Seam: none (in-process).
- Deletion test: pass-through pages; delete and the nav leaves 404.
- Adapters: n/a.

**SATISFIES:** `panels-fetch-live-endpoints` (Observe/System scenario)

- [ ] **Step 1:** G6 — 5 registered routes in `server.go`.
- [ ] **Step 2:** `npx vitest run src/features/settings src/features/swagger` — PASS.
- [ ] **Step 3:** Ledger line.

### Task 4: Prototypes panel on a registered endpoint

**Files:**
- Modify: `skillgrid-ui/src/features/spikes/SpikesPage.tsx` (→ `features/prototypes/PrototypesPage.tsx`), `skillgrid-ui/src/app.tsx`, `skillgrid-ui/src/components/layout/AppLayout.tsx`
- Go: a JSON `GET` handler listing `.skillgrid/prototypes/NNN-name/` (`prototype.md` metadata)

**Interfaces:**
- Consumes: `apiGet` (Task 1).
- Produces: `GET <path> → { prototypes: [{name, date, hypothesis, files}] }`.
- Seam: the Go read bridge.
- Deletion test: without a registered endpoint the panel renders ErrorState forever (the HEAD state).
- Adapters: live server + Go httptest → 2.

**SATISFIES:** `panels-fetch-live-endpoints` (Prototypes scenario)

**Blocked on:** the repo-wide spikes→prototypes rename, which is uncommitted in the main working tree (staged renames under `.skillgrid/prototypes/`, `docs_prototypes.go`, `PrototypesPage.tsx`, skills, docs). That work supplies both halves of this task and is owned by another session; this change cannot commit a subset of it without leaving the tree inconsistent.

- [ ] **Step 1:** G7 — at HEAD the fetch path is `/spikes`; `server.go` registers 0 handlers for it → RED (confirmed).
- [ ] **Step 2:** When the rename lands: re-run G7 → expect 1; `npx vitest run` → PASS.
- [ ] **Step 3:** Ledger line; re-open QA.

### Task 5: Briefing task-ref links survive MarkdownView (regression fix)

**Files:**
- Modify: `skillgrid-ui/src/features/plans/taskLinks.ts`, `skillgrid-ui/src/features/plans/PlanDetail.tsx`
- Test: `skillgrid-ui/src/features/plans/taskLinks.test.ts`, `skillgrid-ui/src/features/plans/PlansPage.linkedTasks.test.tsx`

**Interfaces:**
- Consumes: `TASK_REF_RE`, `MarkdownView`.
- Produces: `linkifyTaskRefsMarkdown(text: string): string` — emits `[match](/tracker?task=NNN)`.
- Seam: none (pure function).
- Deletion test: PlanDetail would have to know the renderer's sanitiser; keep it in `taskLinks.ts` next to `linkifyTaskRefs`.
- Adapters: n/a.

**SATISFIES:** `briefing-task-refs-linkified`

- [ ] **Step 1: RED** — at HEAD `npx vitest run src/features/plans` fails: `expected [...] to include '/tracker?task=012'`.
- [ ] **Step 2:** Add `linkifyTaskRefsMarkdown` to `taskLinks.ts`; `PlanDetail.tsx` renders `<MarkdownView body={linkifyTaskRefsMarkdown(plan.briefing)} />`; add the unit test.
- [ ] **Step 3: GREEN** — `npx vitest run src/features/plans` → 11 passed.
- [ ] **Step 4:** Commit only these three files: `fix(ui): keep briefing task refs linked under MarkdownView`.

### Task 6: Verification floor L2

**Files:** none (verification only).

**SATISFIES:** `verification-floor`

- [ ] **Step 1:** G10 `npm test && npm run build:check` → exit 0, budget OK.
- [ ] **Step 2:** G11 `go build ./... && go build -tags ui ./...` → exit 0.
- [ ] **Step 3:** Ledger line; mark `tasks.md` complete (or `blocked` if Task 4 is still open).

## Self-Review

1. Spec coverage — nav IA (T1), theme (T1), D3 graph (T2), new panels (T3+T4), route collisions (T1/T4), relative URLs (T1), success criteria (T6). Out-of-scope items not planned. ✓
2. Must-haves cover every truth; two are tagged backstop and map to T4/T5. ✓
3. One-way doors: none. ✓
4. Placeholders: none. ✓
5. Type consistency: `linkifyTaskRefsMarkdown` named identically in Must-Haves, Task 5, and tasks.md. ✓

**Owed-decision gate:** every displayed value has a named source — the `limit` (findings.md), nav order (briefing), tokens (prototype 001 `index.html`), endpoints (`server.go`). The Prototypes endpoint's shape is owned by the in-flight rename; recorded as Task 4's blocker, not invented here.

**Plan review verdict:** READY FOR EXECUTION (Tasks 1–3, 5, 6 now; Task 4 blocked, see above).
