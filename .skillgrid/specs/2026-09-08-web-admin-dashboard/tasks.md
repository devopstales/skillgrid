# Tasks: 009-web-admin-dashboard — SkillGrid Web Dashboard (Vite + React SPA)

> **STATUS:** `planning` (2026-09-16) — re-planned to the full Vite + React 19 SPA architecture (7 phases); 0/7 phases implemented
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement phase-by-phase. Phases use checkbox (`- [ ]`) syntax.
>
> **Supersession note (2026-09-16):** the 2026-09-11 vanilla-embedded plan (P1–P6, 4 phases PASS) is **superseded**. Its Go tracker bridge (`internal/mnemonic/http/tracker`) and service layer are **preserved and refactored** behind the new `TicketProvider` registry; the vanilla `ui/` assets (kanban layout, tracker, docs viewer, memory governance) are a **design reference** to be re-expressed in React. This tasks.md starts fresh at 0/7.

**Goal:** Replace the minimal embedded data viewer with a full web dashboard — multi-provider Kanban, GitNexus-style vector graph, OpenViking-style file tree, activity stream, SDD plans, git history, rendered docs with Mermaid, and a prototype viewer — as a Vite + React SPA embedded in the Go binary via `go:embed`, served by `skillgrid serve` at `/` with no separate frontend process.

**Architecture:** `skillgrid-ui/` (Vite + React 19 + TS, sibling to `skillgrid-cli/`) builds into `internal/mnemonic/http/ui/dist/`, embedded with `//go:embed all:ui/dist`. `ui.go` serves the SPA with an index.html fallback for non-API/non-asset routes (preserving `/swagger/` + `/openapi.yaml`). The Go server gains route groups `/tracker/*`, `/mnemonic/*`, `/activity/*`, `/docs/*`, `/plans/*`, `/git/*`, `/prototypes/*`. Real-time via SSE. Graph via Sigma.js + graphology; docs via react-markdown + mermaid + DOMPurify. See briefing.md Architecture decisions.

**Tech Stack:** Go 1.22+ (`skillgrid-cli`); existing SQLite/MCP service layer (reused); React 19 + TypeScript + Vite 6 + Tailwind CSS v4 + shadcn/ui; TanStack Router + TanStack Query + Zustand; Sigma.js + graphology (+ forceatlas2 / louvain / metrics); react-markdown + remark/rehype + mermaid + DOMPurify; @dnd-kit; TanStack Table v8; Recharts; diff2html.

**Spec:** `briefing.md`

**Acceptance:** `acceptance.feature` (`@phase-NN`)

---

## Goal

Operators can open `http://127.0.0.1:7438/` from a running `skillgrid serve` and administer all surfaces from one dashboard: Tracker (multi-provider Kanban), Mnemonic (Graph/Files/Memories/Sessions/Search), Docs (rendered + Mermaid), Plans (SDD progress), Activity (live SSE), Git (commits/diffs/blame), Prototypes (`.stitch/`) — without leaving the browser and without a separate frontend process. After any phase, shipped views work and future views show as disabled stubs.

## Out of scope / Non-Goals

- MCP surface is frozen — no tool, param, return shape, or error code changes (no `tools_*.go` touched)
- Changes to Mnemonic storage schema, indexing, or retrieval logic
- Mutations other than: tracker status change, memory pin/unpin/soft-delete/update, and 013 governance mutations (edit/share/status)
- Authentication beyond the existing `SKILLGRID_HTTP_TOKEN` bearer gate on write routes; no login UI
- CORS, remote hosting, or any non-127.0.0.1 serving story by default
- Multi-tenant teams / role layers / LLM proxy
- AI chat panel in the dashboard — operator-facing, not an agent client
- In-browser LLM/embedding inference — embeddings computed by the Go index
- Session relay / cleave handoff surfaces (change 006)
- D3.js for the main vector graph (Sigma.js only; D3 secondary at most)
- Editing docs from the browser (Docs is read-only)

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `briefing.md` is met
- [ ] Every `@phase-NN` Feature in `acceptance.feature` has passing `@happy`, `@edge`, and `@failure` scenarios
- [ ] Every phase below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `briefing.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `briefing.md` (Error handling + Non-Goals + stack rules). Every phase inherits these — do not restate per phase.

- Active tracker CLI (CLI-backed providers) not on PATH → `warn+continue`: `GET /tracker/*` → 503 `{error, provider}`; view disabled with provider name. Backlog.md (file-based) is unaffected
- Tracker CLI non-zero exit / auth failure / timeout (>10s) → `warn+continue`: → 502 with stderr excerpt (200 chars) + provider
- Tracker output not valid / unknown version → `warn+continue`: → 502 with reason; no partial render
- Tracker provider unresolvable → `warn+continue`: → 501 with reason; view disabled
- Tracker status transition unsupported → `warn+continue`: → 501 listing what the provider supports
- Path traversal on `GET /docs/content?path=...` → `abort`: 400; only declared roots resolve
- Unknown doc path / plan id / spec path / memory id / session id → `abort`: 404
- Graph depth filter exceeds data → `warn+continue`: 200 + `truncated: true`; UI shows "depth capped"
- SSE client disconnect → `warn+continue`: server drops the stream, no leak, no error to other clients
- Write route without token when `SKILLGRID_HTTP_TOKEN` set → `abort`: 401 (existing `requireWriteAuth`)
- MCP tool names, signatures, return shapes, error codes frozen — no `tools_*.go` touched
- SPA build is deterministic + offline-capable — no external CDN at runtime; all assets from the binary (Sigma.js-from-CDN allowed only inside the exported graph HTML file)
- Server stays bound to `127.0.0.1` by default; no new serving env vars (except `SKILLGRID_TRACKER`)
- Tracker mutations go through the provider registry, never direct file writes (Backlog.md status write is the one file mutation, via `task edit`) and never invented statuses/labels
- Docs reads sandboxed to `.skillgrid/sdd/`, `openspec/`, `.backlog/tasks/`, `docs/`, root `*.md`; read-only; rendered as markdown, never executed
- 013 is a soft dependency: governance/layer views render 013 fields when present, forward-compat placeholder when absent; a failed/absent 013 field kills that widget only
- 005/008 are soft dependencies for the graph edge set: graph degrades to node-only + file-list fallback when edge/community data is absent
- `GET /memory/project` keeps its current CWD-based behavior (quirk, not a regression)
- Each phase adds its view; not-yet-built views render as disabled stubs, never dead links
- Each backend phase documents its routes in `openapi.yaml` as it lands

---

## State

```yaml
phase: spec
current_phase: 1-embed-pipeline-shell
status: planned
updated: 2026-09-16T00:00:00Z
```

## Phase map

| NN | Phase | Tag | Blocked by | Acceptance |
|----|-------|-----|------------|------------|
| 1 | `embed-pipeline-shell` | `@phase-1` | — | Feature tagged `@phase-1` |
| 2 | `kanban` | `@phase-2` | 1 | Feature tagged `@phase-2` |
| 3 | `docs` | `@phase-3` | 1 | Feature tagged `@phase-3` |
| 4 | `mnemonic-graph` | `@phase-4` | 1, (soft 005/008) | Feature tagged `@phase-4` |
| 5 | `mnemonic-files-memories` | `@phase-5` | 1, (soft 013) | Feature tagged `@phase-5` |
| 6 | `activity-plans-git` | `@phase-6` | 1, 3 | Feature tagged `@phase-6` |
| 7 | `prototypes-polish` | `@phase-7` | 1, 6 | Feature tagged `@phase-7` |

Execution runs 1 → 7 in order. Each phase is independently shippable.

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~9000 (SPA ~5500 across 9 feature modules, Go ~1800 incl. provider registry + docs/graph/git/activity/prototypes handlers + tests ~1200, openapi ~300) |
| 400-line budget risk | High (each phase commits separately; SPA feature modules are the bulk) |
| Chained PRs recommended | Yes (per-phase; Phase 1 + Phase 2 are the largest single diffs) |
| Delivery strategy | per-phase commits, stacked |

---

## 1-embed-pipeline-shell

### Goal

The embed pipeline works end-to-end and `GET /` serves the SPA shell with every view as a stub — the foundation every later phase builds on.

### Out of scope / Non-Goals

- Any view content beyond disabled stubs (belongs to Phases 2–7)
- Any new backend data route (only the SPA + preserved swagger/openapi)
- OpenAPI changes beyond the SPA + preserved swagger

### Definition of Done

This phase is done only when:

- [ ] `npm run build` produces `ui/dist/index.html` + `ui/dist/assets/`
- [ ] `go build` embeds `ui/dist/` and `skillgrid serve` serves the SPA at `/`
- [ ] SPA fallback serves `index.html` for non-API/non-asset routes; assets + API routes are not shadowed
- [ ] `/swagger/` + `/openapi.yaml` still load
- [ ] Sidebar renders with all routes as labeled disabled stubs, dark theme applied
- [ ] `tsc --noEmit` clean; no external CDN assets
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-1` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-ui/` (Vite + React + TS scaffold: `package.json`, `vite.config.ts`, `tailwind.config.ts`, `index.html`, `src/app/`)
- Create: `skillgrid-ui/src/features/*/` (placeholder route components for all 9 views)
- Create: `skillgrid-cli/internal/mnemonic/http/embed.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui.go`
- Modify: `Taskfile.yml`
- Modify: `.gitignore` (add `ui/dist/`)

**Interfaces:**
- Consumes: existing server (`skillgrid serve` :7438), existing `/projects` for the project selector
- Produces: the embedded SPA shell (sidebar + router + stubs + dark theme) that Phases 2–7 extend one view each; `//go:embed all:ui/dist`

### Tasks

- [ ] 1.1 `[AFK]` Scaffold `skillgrid-ui/` (Vite + React 19 + TS + Tailwind v4 + shadcn/ui); `vite.config.ts` `build.outDir` → `../skillgrid-cli/internal/mnemonic/http/ui/dist`, `emptyOutDir: true`, content-hash `entryFileNames`/`chunkFileNames`/`assetFileNames`; dev `server.proxy` `/api` → `:8080`
- [ ] 1.2 `[AFK]` App shell: TanStack Router with placeholder routes for all 9 views; sidebar nav (Tracker, Mnemonic▸Graph/Files/Memories/Sessions/Search, Docs, Plans, Activity, Git, Prototypes, Settings); dark theme (`#0a0a0b`/`#141416`/`#27272a`/`#6366f1`); Inter + JetBrains Mono; project selector persists to localStorage
- [ ] 1.3 `[RED]` Create `embed.go`: `//go:embed all:ui/dist`, `//go:embed ui/openapi.yaml`, `//go:embed all:ui/swagger`
  - [ ] 1.3.a Write failing test: `GET /` returns the SPA `index.html`; `GET /assets/index-[hash].js` returns the hashed asset (not `index.html`); `GET /mnemonic/nope` (an API prefix) returns 404 JSON, not `index.html`.
  - [ ] 1.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_Embed` — Expected: FAIL
  - [ ] 1.3.c Minimal implementation: `embed.go` + `ui.go` SPA fallback (`fs.Sub` file server; non-`/api/`, non-asset → `index.html`).
  - [ ] 1.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_Embed` — Expected: PASS
  - [ ] 1.3.e Commit — `feat(ui): go:embed Vite dist + SPA fallback`
- [ ] 1.4 `[RED]` Threat: SPA fallback does not shadow API/asset routes
  - [ ] 1.4.a Write failing test: `GET /openapi.yaml` → 200 yaml; `GET /swagger/` → 200; a non-asset, non-API path (e.g. `/tracker`) → 200 `index.html` (client router); a known API route that does not exist yet → 404 JSON (not `index.html`).
  - [ ] 1.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_SPAFallback` — Expected: FAIL
  - [ ] 1.4.c Minimal implementation: prefix-based fallback (API prefixes → 404 JSON; assets → file server; everything else → `index.html`).
  - [ ] 1.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_SPAFallback` — Expected: PASS
  - [ ] 1.4.e Commit — `feat(ui): SPA fallback preserves API + asset routes`
- [ ] 1.5 `[AFK]` Future views render as labeled disabled stubs, never dead links; no external CDN assets (offline check) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 1.6 `[AFK]` Taskfile `ui:build` (`dir: skillgrid-ui`, `npm ci && npm run build`) + `build:all` (`deps: [ui:build]`, `go build -o dist/skillgrid ./skillgrid-cli/cmd/skillgrid`); `.gitignore` adds `ui/dist/` — `Run: go test ./skillgrid-cli/internal/mnemonic/http/...` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase1_'` | PASS | — | embed + SPA fallback |
| SPA build | `cd skillgrid-ui && npm ci && npx tsc --noEmit && npm run build` | PASS | — | produces `ui/dist/` |
| Acceptance `@phase-1` / `@p0` | manual smoke: `skillgrid serve` + browser — sidebar + stubs + dark theme + swagger link | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` | PASS | — | full package green |
| Rollback boundary | `git revert` + `skillgrid serve` — old viewer still works | PENDING | N/A (additive) | new `skillgrid-ui/` + `embed.go` + `ui.go` rewrite; revert restores old `ui/` serving |
| Global Constraints | — | held | — | offline, 127.0.1 unchanged, no `tools_*.go` touched |

### Commit

When phase DoD is met: `feat(ui): Vite/React SPA shell + go:embed pipeline (Phase 1)`

---

## 2-kanban

### Goal

The Tracker Kanban is fully functional on the active tracker (and, where configured, other providers): Go provider registry + `/tracker/*` CRUD + SSE + `@dnd-kit` board.

### Out of scope / Non-Goals

- Invented statuses/labels or guessed Jira project keys
- Any other view (stubs stay)

### Definition of Done

This phase is done only when:

- [ ] Provider detection + all four adapters work with tests
- [ ] Backlog.md adapter parses `.backlog/tasks/*.md` frontmatter (no CLI dependency)
- [ ] `/tracker/*` CRUD + `/tracker/stream` SSE covered by tests (incl. all error rows)
- [ ] Kanban board/detail/drawer/dependency-graph UI works in-browser
- [ ] `openapi.yaml` documents the tracker routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-2` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/tracker/tracker.go` (`TicketProvider` interface + `UnifiedTask` DTO + registry + handlers)
- Modify: `skillgrid-cli/internal/mnemonic/http/tracker/backlog.go` (CLI + direct frontmatter parse)
- Modify: `skillgrid-cli/internal/mnemonic/http/tracker/{github,gitlab,jira}.go` (refactored shell-out)
- Create/Modify: `skillgrid-cli/internal/mnemonic/http/tracker/*_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/tracker/*` + `/tracker/stream`)
- Create: `skillgrid-ui/src/features/kanban/` (BoardView, ListView, TaskCard, TaskDetail, FilterBar, DependencyGraph, providers/, hooks/, types.ts)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: active tracker CLIs on PATH (CLI-backed); `.backlog/tasks/*.md`; tracker doc + `config.yaml:issue_tracker` + `SKILLGRID_TRACKER`; existing `requireWriteAuth`
- Produces: `GET /tracker/providers`, `GET /tracker/tasks?provider=`, `POST /tracker/tasks`, `PATCH /tracker/tasks/{id}`, `GET /tracker/tasks/{id}`, `GET /tracker/tasks/{id}/deps`, `GET /tracker/stream` (SSE) + the Kanban view

### Tasks

- [ ] 2.1 `[RED]` Threat: Taxonomy — provider detection never guesses
  - [ ] 2.1.a Write failing test: unknown `SKILLGRID_TRACKER` → 501; Jira without project key in tracker doc → 501 with reason; valid `backlogmd` resolves to the backlog adapter; `?provider=github` override validated.
  - [ ] 2.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_Detection` — Expected: FAIL
  - [ ] 2.1.c Minimal implementation: `TicketProvider` interface + registry + detection (issue-tracker doc + `config.yaml:issue_tracker` + `SKILLGRID_TRACKER`, default `backlogmd`).
  - [ ] 2.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_Detection` — Expected: PASS
  - [ ] 2.1.e Commit — `feat(tracker): TicketProvider registry + detection`
- [ ] 2.2 `[RED]` Threat: Subprocess — CLI-backed provider missing → 503 with provider name
  - [ ] 2.2.a Write failing test: empty PATH, `GET /tracker/tasks?provider=github` → 503 `{error: "gh CLI not found", provider: "github"}`; Backlog.md (file-based) still → 200.
  - [ ] 2.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_CLI_Missing` — Expected: FAIL
  - [ ] 2.2.c Minimal implementation: `exec.LookPath` check per CLI-backed adapter; 503 with JSON reason + provider.
  - [ ] 2.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_CLI_Missing` — Expected: PASS
  - [ ] 2.2.e Commit — `feat(tracker): 503 when CLI-backed provider missing`
- [ ] 2.3 `[RED]` Threat: Subprocess — CLI non-zero exit / auth failure → 502; timeout >10s → 502; bad output → 502
  - [ ] 2.3.a Write failing test: fixture CLI exiting 1 (stderr "boom") → 502 with excerpt (200 chars) + provider; auth-failure fixture → 502; fixture sleeping 11s → 502 timeout; fixture printing garbage / unknown `schemaVersion` → 502.
  - [ ] 2.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_CLI_Failure` — Expected: FAIL
  - [ ] 2.3.c Minimal implementation: `exec.CommandContext` 10s; capture + truncate stderr; per-adapter parse + `schemaVersion` gate.
  - [ ] 2.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_CLI_Failure` — Expected: PASS
  - [ ] 2.3.e Commit — `feat(tracker): 502 on CLI failure/timeout/bad-output`
- [ ] 2.4 `[RED]` Backlog.md adapter: parse `.backlog/tasks/*.md` frontmatter into `UnifiedTask` (no CLI dependency) + status write via `task edit`
  - [ ] 2.4.a Write failing test: temp `.backlog/tasks/` with 3 real `.md` files → `GET /tracker/tasks?provider=backlogmd` returns normalized `UnifiedTask` (id, title, status, priority, assignee, labels, dependencies, milestone, board column); unknown id → 404; `PATCH /tracker/tasks/{id}` status write runs `task edit` (verified by re-reading the file); without token → 401; invalid status → 400.
  - [ ] 2.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_Backlog` — Expected: FAIL
  - [ ] 2.4.c Minimal implementation: frontmatter parser (`id`/`status`/`priority`/`assignee`/`labels`/`dependencies`/`milestone`/`dueDate`/`parent`) + `board` column mapping + `task edit` status write.
  - [ ] 2.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_Backlog` — Expected: PASS
  - [ ] 2.4.e Commit — `feat(tracker): backlog adapter (frontmatter parse + status write)`
- [ ] 2.5 `[RED]` GitHub + GitLab + Jira adapters: JSON list/get/patch behind the interface
  - [ ] 2.5.a Write failing test: fixture `gh`/`glab`/`jira` JSON → normalized `UnifiedTask`; `UpdateTask` maps to close/reopen (gh/glab) / `issue move` (jira); custom status → 501; unknown transition → 400/501; missing project key → 501 (never guessed); without token → 401.
  - [ ] 2.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_RemoteAdapters` — Expected: FAIL
  - [ ] 2.5.c Minimal implementation: refactored shell-out adapters behind `TicketProvider` (`gh issue list --json`/view/close-reopen; `glab issue list -F json`/view/close-reopen; `jira issue list -q JQL`/view/`issue move` after transition discovery).
  - [ ] 2.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestPhase2_RemoteAdapters` — Expected: PASS
  - [ ] 2.5.e Commit — `feat(tracker): github + gitlab + jira adapters`
- [ ] 2.6 `[RED]` Threat: SSE — `/tracker/stream` (fsnotify on `.backlog/tasks/`) live-updates + no goroutine leak
  - [ ] 2.6.a Write failing test: an SSE client subscribes; a `.backlog/tasks/*.md` change is emitted as an event; a client that disconnects is cleaned up (goroutine count before/after equal); a slow consumer does not block other clients.
  - [ ] 2.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase2_TrackerStream` — Expected: FAIL
  - [ ] 2.6.c Minimal implementation: `GET /tracker/stream` SSE + fsnotify watcher + per-client buffered channel + context-cancel cleanup.
  - [ ] 2.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase2_TrackerStream` — Expected: PASS
  - [ ] 2.6.e Commit — `feat(tracker): /tracker/stream SSE (fsnotify, leak-free)`
- [ ] 2.7 `[AFK]` Mount `/tracker/*` CRUD + `/tracker/stream` on the mux (reads open, writes token-gated) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase2_Routes` — Expected: PASS
- [ ] 2.8 `[AFK]` Kanban UI: provider tabs, board/list toggle, `@dnd-kit` drag across columns, filter bar (provider/assignee/label/milestone/priority/date), task detail drawer (markdown + dependency mini-graph), dependency graph — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 2.9 `[AFK]` Kanban degraded states (503/501 → disabled view with provider + reason; per-widget error isolation) + SSE live board updates — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 2.10 `[AFK]` openapi.yaml documents the tracker routes (+ CRUD + SSE) with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase2_OpenAPI` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (provider) | `go test ./skillgrid-cli/internal/mnemonic/http/tracker/...` | PASS | — | detection, missing/exit-1/timeout/bad-output, 4 adapters, frontmatter parse |
| Focused test (SSE/routes) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase2_'` | PASS | — | stream leak-free, routes, openapi |
| Acceptance `@phase-2` / `@p0` | manual smoke: Backlog.md board round-trip + one more provider + browser | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + SPA `tsc --noEmit` | PASS | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | N/A (additive) | new tracker registry + routes + kanban feature; revert restores Phase 1 shell |
| Global Constraints | — | held | — | shell-out only, no invented statuses, Jira key never guessed, no CDN |

### Commit

When phase DoD is met: `feat(tracker): provider registry + Kanban (Phase 2)`

---

## 3-docs

### Goal

The Docs view renders repo markdown with Mermaid diagrams across `.skillgrid/sdd/`, `openspec/`, `.backlog/`, `docs/`, root `*.md`.

### Out of scope / Non-Goals

- Editing docs from the browser (read-only)
- Docs outside the declared roots
- Any other view (stubs stay)

### Definition of Done

This phase is done only when:

- [ ] `/docs/*` routes covered by traversal + happy-path tests
- [ ] Tree, rendered view, Mermaid SVG, TOC, frontmatter chips, search work in-browser
- [ ] Mermaid/markdown XSS guarded (strict + sanitize + DOMPurify)
- [ ] `openapi.yaml` documents the docs routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-3` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/docs/*`)
- Create: `skillgrid-cli/internal/mnemonic/http/docs/docs.go` (sandboxed tree/content/search/render readers + handlers)
- Create: `skillgrid-cli/internal/mnemonic/http/docs/docs_test.go`
- Create: `skillgrid-ui/src/features/docs/` (DocsTree, MarkdownView, MermaidBlock, FrontmatterChips, OnThisPage, DocSearch, hooks/useDocs.ts, lib/mermaid.ts)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: repo markdown on disk (`.skillgrid/sdd/*`, `openspec/*`, `.backlog/tasks/*.md`, `docs/**/*.md`, root `*.md`)
- Produces: `GET /docs/tree?root=...`, `GET /docs/content?path=...` (markdown + frontmatter + relatedPlans), `GET /docs/search?q=...`, `GET /docs/render?path=...` + the Docs view

### Tasks

- [ ] 3.1 `[RED]` Threat: path traversal — `..`, absolute paths, symlink escape, unknown paths blocked
  - [ ] 3.1.a Write failing test: `GET /docs/content?path=../secret`, `path=/etc/passwd`, `path=docs/../x`, unknown `path` → 400/404; happy `path` under a declared root returns the file + frontmatter + relatedPlans.
  - [ ] 3.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_Traversal` — Expected: FAIL
  - [ ] 3.1.c Minimal implementation: clean + prefix-check every path against the declared roots; read-only; render as markdown, never execute.
  - [ ] 3.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_Traversal` — Expected: PASS
  - [ ] 3.1.e Commit — `feat(docs): sandboxed doc readers`
- [ ] 3.2 `[RED]` Threat: Mermaid/markdown XSS — untrusted markdown + `language-mermaid` sanitized
  - [ ] 3.2.a Write failing test: a doc with a `<script>` in markdown body does not yield raw `<script>` in the rendered output (rehype-sanitize); a `language-mermaid` block renders to SVG that is DOMPurify-sanitized; `mermaid` `securityLevel` is `strict` (asserted from the SPA config fixture / a Go-side default).
  - [ ] 3.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_MermaidSanitize` — Expected: FAIL
  - [ ] 3.2.c Minimal implementation: `rehype-sanitize` + DOMPurify on `mermaid.render()` SVG; `securityLevel: 'strict'`; SVG cached by content hash.
  - [ ] 3.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_MermaidSanitize` — Expected: PASS
  - [ ] 3.2.e Commit — `feat(docs): mermaid + markdown XSS guards`
- [ ] 3.3 `[AFK]` `GET /docs/tree?root=sdd|openspec|backlog|docs|all` returns the grouped tree (path, title, updatedAt, frontmatter status) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_Tree` — Expected: PASS
- [ ] 3.4 `[AFK]` `GET /docs/search?q=...` returns matches (path + snippet) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_Search` — Expected: PASS
- [ ] 3.5 `[AFK]` `GET /docs/render?path=...` (optional SSR fallback) returns rendered HTML — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestPhase3_Render` — Expected: PASS
- [ ] 3.6 `[AFK]` Docs UI: tree sidebar grouped by root (file icons, updated-at badges, name filter, expand/collapse), MarkdownView (GFM, anchors, task lists, syntax highlighting), **MermaidBlock**, frontmatter chips (status/author/updated), on-this-page sticky TOC with scroll-spy, cross-links to SPA routes, copy/download/print — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 3.7 `[AFK]` "View plan progress →" cross-link to the Plans view (Phase 6 stub target) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 3.8 `[AFK]` openapi.yaml documents the docs routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase3_OpenAPI` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (traversal + XSS) | `go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run 'TestPhase3_'` | PASS | — | traversal RED→GREEN, mermaid sanitize |
| Focused test (routes) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase3_'` | PASS | — | tree/search/render/openapi |
| Acceptance `@phase-3` / `@p0` | manual smoke: `skillgrid serve` + browser — tree, rendered view, Mermaid SVG, TOC, search | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + SPA `tsc --noEmit` | PASS | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | N/A (additive) | new `docs/` pkg + routes + docs feature; revert restores Phase 2 shell |
| Global Constraints | — | held | — | read-only sandbox, no CDN, markdown rendered not executed |

### Commit

When phase DoD is met: `feat(docs): rendered markdown + Mermaid viewer (Phase 3)`

---

## 4-mnemonic-graph

### Goal

The Mnemonic vector graph renders the memory graph with GitNexus parity: Sigma.js + graphology, ForceAtlas2/tree/circles, Louvain coloring, depth filter, search highlight.

### Out of scope / Non-Goals

- files/memories/sessions views (Phase 5)
- In-browser embedding inference
- Any other view (stubs stay)

### Definition of Done

This phase is done only when:

- [ ] `/mnemonic/graph*` routes covered by depth/truncate + happy-path tests
- [ ] `mnemonicGraphToGraphology` converter + layouts + communities + filters work
- [ ] VectorGraph (force/tree/circles, Louvain, depth slider, search highlight, legend/zoom/hover, HTML export) works in-browser
- [ ] Degrades to node-only + file-list fallback when 005/008 edge data absent
- [ ] `openapi.yaml` documents the graph routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-4` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell, (soft: 005/008)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/mnemonic/graph*`)
- Create: `skillgrid-cli/internal/mnemonic/http/mnemonic_graph.go` (graph/nodes handlers: depth filter, node cap, communities)
- Create: `skillgrid-cli/internal/mnemonic/http/mnemonic_graph_test.go`
- Create: `skillgrid-ui/src/features/mnemonic/graph/` (VectorGraph.tsx, converters.ts, layouts.ts, communities.ts, filters.ts, controls/{LayoutSwitcher,SearchPanel,Legend,DepthSlider}, hooks/useMnemonicGraph.ts)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: `internal/mnemonic/graph` + `codeindex` (nodes + edges: imports/calls/semantic_similarity), `memory` (node metadata)
- Produces: `GET /mnemonic/graph?node_id=&depth=2` (nodes + edges + `truncated`), `GET /mnemonic/graph/nodes?limit=500` + the VectorGraph view

### Tasks

- [ ] 4.1 `[RED]` Threat: Graph scale — depth filter + node cap + `truncated` flag
  - [ ] 4.1.a Write failing test: a graph deeper than 2 hops with `GET /mnemonic/graph?depth=2` returns only the 2-hop neighborhood + `truncated: true`; `/graph/nodes?limit=500` caps at 500 nodes; the converter drops edges whose endpoints are absent from the node set.
  - [ ] 4.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_GraphScale` — Expected: FAIL
  - [ ] 4.1.c Minimal implementation: BFS depth filter on the graph + node cap + edge pruning + `truncated` flag.
  - [ ] 4.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_GraphScale` — Expected: PASS
  - [ ] 4.1.e Commit — `feat(http): /mnemonic/graph with depth filter + node cap`
- [ ] 4.2 `[AFK]` `/mnemonic/graph` happy path returns nodes (id, label, type, path, degree, community) + edges (source, target, type, weight) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_GraphHappy` — Expected: PASS
- [ ] 4.3 `[RED]` Threat: 005/008 soft dep — graph degrades to node-only + file-list fallback when edge/community data absent
  - [ ] 4.3.a Write failing test: against a store with nodes but no edge/community data, `/mnemonic/graph` returns 200 with nodes + empty edges + a `degraded: true` flag; the file-list fallback data is present.
  - [ ] 4.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_GraphDegraded` — Expected: FAIL
  - [ ] 4.3.c Minimal implementation: detect absent edge/community data; return `degraded: true` + file-list fallback.
  - [ ] 4.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_GraphDegraded` — Expected: PASS
  - [ ] 4.3.e Commit — `feat(http): graph degrades to node-only + file-list when 005/008 absent`
- [ ] 4.4 `[AFK]` `mnemonicGraphToGraphology` converter (node label/type/path/size-by-degree/x/y/color; edge type/weight/size/color; multi + directed) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 4.5 `[AFK]` VectorGraph: `SigmaContainer` + `useLoadGraph`; `force` (ForceAtlas2 bounded iterations) / `tree` / `circles` layouts via `layouts.ts`; Louvain community coloring via `communities.ts`; `filterGraphByDepth` + visibleLabels via `filters.ts` — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 4.6 `[AFK]` Graph controls: LayoutSwitcher, SearchPanel, Legend, DepthSlider; zoom/pan; hover tooltips (node type/path/degree/community); semantic search → node highlight — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 4.7 `[AFK]` Lightweight HTML export (Sigma.js from CDN in the exported file only, not the app) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 4.8 `[AFK]` openapi.yaml documents the graph routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase4_OpenAPI` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (scale + degrade) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase4_'` | PASS | — | depth/truncate, happy, degraded, openapi |
| Acceptance `@phase-4` / `@p0` | manual smoke: `skillgrid serve` + browser — force/tree/circles, Louvain, depth, search highlight, export | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + SPA `tsc --noEmit` | PASS | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | N/A (additive) | new `mnemonic_graph.go` + graph feature; revert restores Phase 3 shell |
| Global Constraints | — | held | — | Sigma.js (not D3), no CDN at runtime, 005/008 soft-dep honored |

### Commit

When phase DoD is met: `feat(mnemonic): Sigma.js vector graph (Phase 4)`

---

## 5-mnemonic-files-memories

### Goal

The rest of the Mnemonic suite: OpenViking file tree (L0/L1/L2), memory browser + governance, session browser, audit trail, semantic search.

### Out of scope / Non-Goals

- graph (Phase 4)
- relay/cleave surfaces (006)
- Any change to 013's data model (013 owns it; this renders it)

### Definition of Done

This phase is done only when:

- [ ] `/mnemonic/files|memories|sessions|audit|search` routes covered by integration tests
- [ ] OpenViking tree + L0/L1/L2 content, memory grid/timeline/detail, session browser, audit trail, semantic search work in-browser
- [ ] Governance renders 013 data (forward-compat placeholder when absent), write-gated
- [ ] `openapi.yaml` documents the mnemonic routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-5` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell, (soft: 013)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/mnemonic/files|memories|sessions|audit|search`)
- Create: `skillgrid-cli/internal/mnemonic/http/mnemonic_files.go` (files tree/content, memories, sessions, audit, search handlers)
- Create: `skillgrid-cli/internal/mnemonic/http/mnemonic_files_test.go`
- Create: `skillgrid-ui/src/features/mnemonic/files/` (FileTree, ContentPanel)
- Create: `skillgrid-ui/src/features/mnemonic/memories/` (MemoryGrid, MemoryDetail, MemoryTimeline)
- Create: `skillgrid-ui/src/features/mnemonic/sessions/` (SessionBrowser)
- Create: `skillgrid-ui/src/features/mnemonic/search/` (SearchPanel)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: `internal/mnemonic/memfs` (files tree/content), `memory` (memories, pin/unpin, governance via 013 `mem_*`), sessions rows, audit log, hybrid search
- Produces: `GET /mnemonic/files/tree?path=/`, `GET /mnemonic/files/content?uri=mnemonic://...`, `GET /mnemonic/memories?limit=&offset=`, `GET /mnemonic/memories/{id}`, `GET /mnemonic/sessions`, `GET /mnemonic/audit`, `GET /mnemonic/search?q=&mode=hybrid` + the files/memories/sessions/search views

### Tasks

- [ ] 5.1 `[RED]` `GET /mnemonic/files/tree?path=/` returns the memfs tree (node icon/name/memory-count/last-indexed); `GET /mnemonic/files/content?uri=mnemonic://...` returns L0/L1/L2 tiers
  - [ ] 5.1.a Write failing test: a memfs with nested nodes → `/files/tree?path=/` returns the tree with counts; `/files/content?uri=mnemonic://...` returns L0 (abstract) + L1 (overview) + L2 (details); unknown uri → 404.
  - [ ] 5.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Files` — Expected: FAIL
  - [ ] 5.1.c Minimal implementation: memfs tree walker + content tier reader wired to handlers.
  - [ ] 5.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Files` — Expected: PASS
  - [ ] 5.1.e Commit — `feat(http): /mnemonic/files tree + content (L0/L1/L2)`
- [ ] 5.2 `[RED]` `GET /mnemonic/memories?limit=&offset=` + `GET /mnemonic/memories/{id}` (full content + 013 governance fields)
  - [ ] 5.2.a Write failing test: store with 60 memories → `/memories?limit=50&offset=0` returns 50 + total; `/memories/{id}` returns full content + owner/version/status/usage/visibility (when 013 present); unknown id → 404.
  - [ ] 5.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Memories` — Expected: FAIL
  - [ ] 5.2.c Minimal implementation: paginated memory list + detail (via `memory.Get`) + 013 governance fields.
  - [ ] 5.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Memories` — Expected: PASS
  - [ ] 5.2.e Commit — `feat(http): /mnemonic/memories list + detail + governance fields`
- [ ] 5.3 `[RED]` `GET /mnemonic/sessions` + `GET /mnemonic/audit` + `GET /mnemonic/search?q=&mode=hybrid`
  - [ ] 5.3.a Write failing test: `/sessions` returns session list (id, title, started_at, count, entities); `/audit` returns the hash-chained log; `/search?q=auth&mode=hybrid` returns ranked results with relevance; empty query → 200 empty list.
  - [ ] 5.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_SessionsAuditSearch` — Expected: FAIL
  - [ ] 5.3.c Minimal implementation: session list + audit log + hybrid search wired to handlers.
  - [ ] 5.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_SessionsAuditSearch` — Expected: PASS
  - [ ] 5.3.e Commit — `feat(http): /mnemonic/sessions + /audit + /search`
- [ ] 5.4 `[AFK]` OpenViking file tree UI: mirror memfs, node icon + name + memory-count badge + last-indexed; content panel with L0/L1/L2 tiers; `mnemonic://` breadcrumbs; scoped search; inline memory annotations — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 5.5 `[AFK]` Memory content view: card grid (title, source, timestamp, tags, relevance, preview), timeline (grouped by day/session), detail modal (full content, source context, vector-similar memories, edit/delete), session browser (group by session id, summary, count, timeline, entities), audit trail (hash-chained log) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 5.6 `[RED]` Threat: Governance mutation — write-gated edit/share/status (renders 013 data; forward-compat placeholder when absent)
  - [ ] 5.6.a Write failing test: with `SKILLGRID_HTTP_TOKEN` set, in-place edit / explicit share / status change without token → 401; with token → 200; edit appends a 013 version (re-readable); share idempotent + 400 on unknown target; **with 013 absent**, governance widgets render labeled placeholders + flat view and the view stays interactive (per-widget isolation).
  - [ ] 5.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Governance` — Expected: FAIL
  - [ ] 5.6.c Minimal implementation: governance mutation handlers (write-gated, version-appending edit, idempotent share, status) + 013-absent placeholder detection.
  - [ ] 5.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_Governance` — Expected: PASS
  - [ ] 5.6.e Commit — `feat(http): memory governance (write-gated, 013 forward-compat)`
- [ ] 5.7 `[AFK]` Semantic search UI: `mode=hybrid` results list → jump to graph highlight / memory detail — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 5.8 `[AFK]` openapi.yaml documents the mnemonic routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase5_OpenAPI` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (files/memories/sessions/audit/search) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase5_'` | PASS | — | files, memories, sessions+audit+search, governance |
| Acceptance `@phase-5` / `@p0` | manual smoke: `skillgrid serve` + browser — tree + L0/L1/L2, memory grid + detail + actions, session browser, audit, search | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + SPA `tsc --noEmit` | PASS | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | N/A (additive) | new `mnemonic_files.go` + features; revert restores Phase 4 shell |
| Global Constraints | — | held | — | 013 soft-dep honored, write-gated, per-widget isolation, no CDN |

### Commit

When phase DoD is met: `feat(mnemonic): files + memories + sessions + search (Phase 5)`

---

## 6-activity-plans-git

### Goal

The Activity stream (SSE), SDD Plans view, and Git view are fully functional.

### Out of scope / Non-Goals

- Feature work beyond these three views (bugs fixed, nothing new)
- Any git write/commit/push/PR (Git view is read-only)

### Definition of Done

This phase is done only when:

- [ ] `/activity/*` (SSE + events + stats), `/plans/*`, `/git/*` routes covered by tests (incl. SSE)
- [ ] Activity live feed + filters + stats + agent health + alerts work in-browser
- [ ] Plan cards + detail + spec viewer (reuses Docs) + dependency DAG work in-browser
- [ ] Commit graph + list + detail (diff) + file history + blame work in-browser
- [ ] `openapi.yaml` documents the activity/plans/git routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-6` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell, 3-docs (spec viewer reuses Docs rendering)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/activity/*`, `/plans/*`, `/git/*`)
- Create: `skillgrid-cli/internal/mnemonic/http/activity.go` (event log + SSE + stats)
- Create: `skillgrid-cli/internal/mnemonic/http/plans.go` (plans/specs readers)
- Create: `skillgrid-cli/internal/mnemonic/http/git.go` (commits/diff/file-history/blame)
- Create: `skillgrid-cli/internal/mnemonic/http/{activity,plans,git}_test.go`
- Create: `skillgrid-ui/src/features/activity/` (ActivityStream, EventCard, StatsBar, AgentHealth, AlertBanner)
- Create: `skillgrid-ui/src/features/plans/` (PlanList, PlanDetail, SpecViewer, PlanDependencyGraph)
- Create: `skillgrid-ui/src/features/git/` (CommitGraph, CommitList, CommitDetail, DiffViewer, BlameView)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: event log (sessions, observations, index status), `.skillgrid/sdd/*` (plans/specs frontmatter + step files), `git log/diff/blame`
- Produces: `GET /activity/stream` (SSE), `GET /activity/events?limit=100`, `GET /activity/stats`, `GET /plans`, `GET /plans/{id}`, `GET /plans/{id}/steps`, `GET /specs`, `GET /specs/{path}`, `GET /git/commits?limit=50`, `GET /git/commits/{sha}`, `GET /git/diff/{sha}`, `GET /git/file-history?path=...`, `GET /git/blame?path=&line=` + the activity/plans/git views

### Tasks

- [ ] 6.1 `[RED]` `/activity/events?limit=100` + `/activity/stats` + `/activity/stream` (SSE, leak-free)
  - [ ] 6.1.a Write failing test: seeded events → `/activity/events?limit=100` returns them (newest first, type/source/actor/severity/relatedIds); `/activity/stats` returns the counters; `/activity/stream` emits new events live and cleans up on disconnect (goroutine count before/after equal).
  - [ ] 6.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Activity` — Expected: FAIL
  - [ ] 6.1.c Minimal implementation: event log + SSE + stats handlers.
  - [ ] 6.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Activity` — Expected: PASS
  - [ ] 6.1.e Commit — `feat(http): /activity events + stats + SSE`
- [ ] 6.2 `[RED]` `/plans` + `/plans/{id}` + `/plans/{id}/steps` + `/specs` + `/specs/{path}` (aggregated from `.skillgrid/sdd/*`)
  - [ ] 6.2.a Write failing test: a `.skillgrid/sdd/` change with frontmatter + step files → `/plans` returns the plan (id, title, status, progress); `/plans/{id}` returns detail (step checklist, acceptance, linked files/tasks/commits); `/specs/{path}` returns the spec markdown; unknown id/path → 404.
  - [ ] 6.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Plans` — Expected: FAIL
  - [ ] 6.2.c Minimal implementation: `.skillgrid/sdd/*` frontmatter + step-file parser + specs reader.
  - [ ] 6.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Plans` — Expected: PASS
  - [ ] 6.2.e Commit — `feat(http): /plans + /specs (SDD aggregation)`
- [ ] 6.3 `[RED]` `/git/commits?limit=50` + `/git/commits/{sha}` + `/git/diff/{sha}` + `/git/file-history?path=...` + `/git/blame?path=&line=`
  - [ ] 6.3.a Write failing test: a fixture git repo → `/git/commits?limit=50` returns commits (sha, message, author, date, +/- stats, branch lane); `/git/commits/{sha}` + `/git/diff/{sha}` return the diff; `/git/file-history?path=...` + `/git/blame?path=&line=` return history + blame; unknown sha/path → 404; not-a-repo → 503.
  - [ ] 6.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Git` — Expected: FAIL
  - [ ] 6.3.c Minimal implementation: `git log/diff/show/blame` invocations (read-only) + diff parsing + handlers.
  - [ ] 6.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_Git` — Expected: PASS
  - [ ] 6.3.e Commit — `feat(http): /git commits + diff + file-history + blame`
- [ ] 6.4 `[AFK]` Activity UI: infinite-scroll live feed (newest first), compact cards (icon/timestamp/summary/severity border), filters (type/source/severity/actor/time), stats bar, agent health panel, alert banners — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 6.5 `[AFK]` Plans UI: plan cards (status, progress bar), plan detail (step checklist, acceptance, linked files/tasks/commits), dependency DAG, spec viewer (reuses Docs MarkdownView + MermaidBlock), "Read in Docs →" — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 6.6 `[AFK]` Git UI: commit graph (branch lanes, merge nodes, author avatars), commit list (SHA copyable, message, author, +/-), commit detail (unified/split diff, changed-file tree), file history, blame view, commit↔task linking via message conventions — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 6.7 `[AFK]` openapi.yaml documents the activity/plans/git routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase6_OpenAPI` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (activity/plans/git) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase6_'` | PASS | — | activity SSE, plans, git, openapi |
| Acceptance `@phase-6` / `@p0` | manual smoke: `skillgrid serve` + browser — live feed, plan cards + spec viewer, commit graph + diff + blame | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + SPA `tsc --noEmit` | PASS | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | N/A (additive) | new activity/plans/git pkgs + features; revert restores Phase 5 shell |
| Global Constraints | — | held | — | git read-only, SSE leak-free, no CDN |

### Commit

When phase DoD is met: `feat(dashboard): activity + plans + git (Phase 6)`

---

## 7-prototypes-polish

### Goal

The Prototypes gallery ships; the whole surface is polished, performant, responsive, and the full DoD smoke passes.

### Out of scope / Non-Goals

- New feature views (bugs fixed, nothing new)
- Any schema/config migration

### Definition of Done

This phase is done only when:

- [ ] `/prototypes/*` routes covered by tests
- [ ] `.stitch/` gallery + sandboxed iframe + device viewport + code viewer + export work in-browser
- [ ] Performance: code-splitting per feature, lazy-load graph + git, bundle-size budget met
- [ ] Responsive (desktop/tablet/mobile) + density modes + motion
- [ ] `openapi.yaml` final review; user manual updated
- [ ] `tsc --noEmit` + `npm run build` + `go test ./...` + `go vet ./...` green
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-7` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on phase already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 1-embed-pipeline-shell, 6-activity-plans-git

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (mount `/prototypes/*`)
- Create: `skillgrid-cli/internal/mnemonic/http/prototypes.go` (`.stitch/` gallery readers)
- Create: `skillgrid-cli/internal/mnemonic/http/prototypes_test.go`
- Create: `skillgrid-ui/src/features/prototype/` (PrototypeGallery, SandboxPreview, CodeViewer)
- Modify: `docs/user-manual/` (serve page)
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: `.stitch/` design prototypes on disk
- Produces: `GET /prototypes`, `GET /prototypes/{id}`, `GET /prototypes/{id}/html` + the Prototypes view + the polished, smoke-tested dashboard

### Tasks

- [ ] 7.1 `[RED]` Threat: repo file serving (path traversal) — `/prototypes*` sandboxed to `.stitch/`
  - [ ] 7.1.a Write failing test: `GET /prototypes/{id}` with `..`, absolute, or unknown id → 400/404; happy id returns the prototype HTML; reads sandboxed to `.stitch/`; no file execution.
  - [ ] 7.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase7_Prototypes` — Expected: FAIL
  - [ ] 7.1.c Minimal implementation: `.stitch/` reader + clean + prefix-check + handlers.
  - [ ] 7.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase7_Prototypes` — Expected: PASS
  - [ ] 7.1.e Commit — `feat(http): /prototypes (sandboxed .stitch/ readers)`
- [ ] 7.2 `[AFK]` Prototypes UI: gallery grid, sandboxed iframe preview (`sandbox` attr), device viewport toggle (desktop/tablet/mobile), code viewer (HTML/CSS/JS side-by-side), version history + diffs, standalone-HTML export / copy — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 7.3 `[AFK]` Performance: TanStack Router code-splitting per feature route; lazy-load graph + git; content-hash assets; bundle-size budget checked (report + threshold) — `Run: cd skillgrid-ui && npm run build` — Expected: PASS
- [ ] 7.4 `[AFK]` Responsive (desktop/tablet/mobile) + density modes (comfortable/compact) + motion (150ms ease; activity stream animates from top) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
- [ ] 7.5 `[AFK]` openapi.yaml final review: every route documented with valid examples; `/swagger/` exercises each; old routes still documented — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase7_OpenAPI` — Expected: PASS
- [ ] 7.6 `[AFK]` User-manual serve section documents the views + the per-provider tracker-CLI dependency (install/auth + degraded states) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase7_UserManual` — Expected: PASS
- [ ] 7.7 `[AFK]` Full DoD smoke: `npm run build` + `go test ./...` + `go vet ./...` + `tsc --noEmit` + manual browser pass over all views — `Run: go test ./...` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (prototypes + final) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase7_'` | PASS | — | prototypes traversal, openapi, user manual |
| SPA build + types | `cd skillgrid-ui && npm ci && npx tsc --noEmit && npm run build` | PASS | — | bundle-size report |
| Full suite | `go test ./...` + `go vet ./...` | PASS | — | |
| Acceptance `@phase-7` / `@p0` | manual smoke: `skillgrid serve` + browser — gallery + iframe + viewport + export; all views render; DoD checklist | PENDING | — | |
| Rollback boundary | `git revert` + `go test ./...` | PENDING | PASS | reverting Phase 7 restores Phase 6 state |
| Global Constraints | — | held | — | prototypes sandboxed, no CDN, bundle-size within budget |

### Commit

When phase DoD is met: `feat(dashboard): prototypes + polish + DoD smoke (Phase 7)`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every phase Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
