# Tasks: 009-web-admin-dashboard

> **STATUS:** `in-progress` (2026-09-11) — 4/6 steps PASS (01-dashboard-shell, 02-tracker-tab, 03-docs-viewer, 04-memory-tab)
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.
>
> **Restructure note (2026-09-11):** Step Blueprint renumbered from 01/02/03/04/04b/05 to six menu-first vertical slices P1–P6 (shell → tracker → docs-viewer → memory+governance → code → sessions+polish), each shipping its backend just-in-time. Safe: 0 steps PASS, nothing implemented.

**Goal:** Replace the minimal embedded data viewer with a full admin dashboard served by `skillgrid serve` at `/` — menu bar (Welcome, Tracker, Docs, Memory, Code, Sessions), tracker browser, docs viewer, memory browser + governance, code-index status, sessions view — built one shippable phase at a time.

**Architecture:** The existing `internal/mnemonic/http` server (Go 1.22 mux, `embed.FS` static serving at `/`) is extended just-in-time per phase: P1 shell (no backend), P2 `tracker` bridge (4 CLI adapters), P3 read-only `docs` bridge, P4 observation/pin/unpin endpoints, P5 no backend (code routes exist), P6 session list/summary reads. Embedded SPA rewritten in place (vanilla JS, no build step). See change.md Architecture decisions.

**Tech Stack:** Go 1.22+ (`skillgrid-cli`), existing SQLite/MCP service layer (untouched), embedded static SPA (vanilla HTML/JS/CSS, `embed.FS`), tracker CLIs (shell-out: `backlog` / `gh` / `glab` / `jira`).

**Spec:** `docs/skillgrid/changes/009-web-admin-dashboard/change.md`

**Acceptance:** `docs/skillgrid/changes/009-web-admin-dashboard/acceptance.feature` (`@step-NN`)

---

## Goal

Operators can open `http://127.0.0.1:7438/` from a running `skillgrid serve` and administer all surfaces from one dashboard menu: Welcome, Tracker, Docs, Memory, Code, Sessions — without leaving the browser and without a separate frontend process. After any phase, shipped entries work and future entries show as disabled stubs.

## Out of scope / Non-Goals

- Session relay / cleave handoff surfaces (change 006) — sessions entry shows what exists today; relay view lands when 006 ships
- New MCP tools — MCP tool names, signatures, and return shapes are frozen for this change
- Mutations other than what is listed in In scope — e.g. no tracker item *creation* (read + status change only), no docs editing from the browser (read-only)
- Authentication beyond the existing `SKILLGRID_HTTP_TOKEN` bearer gate on write routes; no login UI, no sessions/cookies
- CORS, remote hosting, or any non-127.0.0.1 serving story
- React/Next.js or any npm build pipeline in `skillgrid-cli` — the SPA stays build-less
- Changes to Mnemonic storage schema, indexing, or retrieval logic
- Redoing Swagger UI — `/swagger-ui` stays as-is, linked from the menu
- Graph canvas visualization (vis.js/Sigma.js) — deferred to a follow-up change that depends on 005/008 edges; P5 ships a forward-compat placeholder in the Code entry only
- AI chat panel in the dashboard — the agent lives in the terminal; the dashboard is operator-facing, not an agent client
- **Multi-tenant teams / role layers / LLM proxy** — TencentDB's product-scale layer; 009 renders the single-operator governance (owner/visibility/usage) that 013 provides, not the multi-tenant machinery
- **Memory layering + governance data model** (L0–L3, owner, version, status, usage, visibility) — owned by `013-mnemonic-layered-memory-governance`; P4 **renders** those fields and shows forward-compat placeholders where 013 has not landed yet
- Docs outside `docs/skillgrid/changes/` + `docs/skillgrid/archive/` — the P3 viewer never serves other paths

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- Active tracker CLI not on PATH → `warn+continue`: `GET /tracker/*` (and `/backlog/*` alias) → 503 `{error: "<cli> CLI not found", provider}`; dashboard renders disabled Tracker entry with provider name
- Active tracker CLI exits non-zero, fails auth, or times out (>10s) → `warn+continue`: → 502 with CLI stderr excerpt (truncated 200 chars) + provider name
- Tracker output not valid / unknown version → `warn+continue`: → 502 with reason; entry shows error state, no partial render
- Tracker provider unresolvable → `warn+continue`: → 501 with reason; Tracker entry disabled
- Tracker status transition unsupported by provider → `warn+continue`: → 501 listing what the provider supports
- Path traversal on `GET /docs/changes/{name}` → `abort`: 400; only names under the two docs roots resolve
- Unknown change name on `GET /docs/changes/{name}` → `abort`: 404
- Unknown observation id on `GET /observations/{id}` → `abort`: 404 `{error: ...}`
- Unknown session id on `GET /sessions/{id}/summary` → `abort`: 404
- Invalid status on `POST /tracker/tasks/{id}/status` → `abort`: 400 listing valid statuses (validated client-side too)
- Pin on already-pinned / unpin on non-pinned observation → `warn+continue`: idempotent 200 (service-level behavior preserved)
- Write route called without token when `SKILLGRID_HTTP_TOKEN` is set → `abort`: 401 (existing `requireWriteAuth` behavior, unchanged)
- MCP tool names, signatures, and return shapes are frozen — no `tools_*.go` file is touched
- No npm build pipeline, no React/Next.js, no external CDN assets — the SPA is vanilla HTML/JS/CSS served from `embed.FS`
- Server stays bound to `127.0.0.1` by default; no new env vars for serving (except `SKILLGRID_TRACKER` provider override)
- Tracker mutations go through the active tracker's CLI, never direct file writes and never invented statuses/labels
- Docs reads are sandboxed to `docs/skillgrid/changes/` + `docs/skillgrid/archive/`, read-only, rendered as text, never executed
- Each phase adds its menu entry; not-yet-built entries render as disabled stubs, never dead links
- Each backend phase documents its routes in `openapi.yaml` as it lands
- `GET /memory/project` keeps its current CWD-based behavior (quirk, not a regression — no change)
- **013 is a soft dependency**: the Memory governance/layer views render 013's fields when present and show a forward-compat placeholder when 013 has not landed — P4 must not block on 013; a failed/absent 013 field kills that widget only, never the entry
- **No multi-tenant teams / role layers / LLM proxy** — 009 renders the single-operator governance that 013 provides, not the multi-tenant machinery
- **013 owns the governance/layer data model; 009 only renders it** — no change to 013's schema, layering, or governance logic; P4 governance is pure UI over the 013-backed `mem_*` HTTP surface

---

## State

```yaml
phase: spec
current_step: 05-code-tab
status: in_progress
updated: 2026-09-11T00:00:00Z
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `dashboard-shell` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `tracker-tab` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `docs-viewer` | `@step-03` | 01, 02 | Feature tagged `@step-03` |
| 04 | `memory-tab` | `@step-04` | 01 | Feature tagged `@step-04` |
| 05 | `code-tab` | `@step-05` | 01 | Feature tagged `@step-05` |
| 06 | `sessions-tab` | `@step-06` | 01 | Feature tagged `@step-06` |

Execution runs P1 → P6 in order. Each phase is independently shippable.

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~5500 (SPA ~3100, Go ~1400 incl. 4 adapters + docs bridge, tests ~800, openapi ~200) |
| 400-line budget risk | Medium-High (adapters + docs bridge are the Go bulk; each phase commits separately) |
| Chained PRs recommended | No |
| Delivery strategy | single-pr (six phase commits) |

---

## 01-dashboard-shell

### Goal

`GET /` serves the menu shell with a static Welcome page — the foundation every later phase builds on. No backend change.

### Out of scope / Non-Goals

- Any menu entry content beyond disabled stubs (belongs to P2–P6)
- Any backend change (routes land in P2–P4/P6)
- OpenAPI spec (extended just-in-time per backend phase)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`
- Create: `skillgrid-cli/internal/mnemonic/http/ui/app.css`

**Interfaces:**
- Consumes: existing routes only (`/swagger-ui`, `/openapi.yaml`, `/projects` for the kept project selector)
- Produces: the menu shell (path router + server fallback, Welcome, stubs) that P2–P6 extend one entry each

### Tasks

- [x] 01.1 `[AFK]` Menu bar with path routing + project selector (localStorage) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_Shell` — Expected: PASS
- [x] 01.2 `[AFK]` Static Welcome page (no fetches) with links to /swagger-ui + /openapi.yaml — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_Welcome` — Expected: PASS
- [x] 01.3 `[AFK]` Future entries render as labeled disabled stubs, never dead links — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_Stubs` — Expected: PASS
- [x] 01.4 `[AFK]` Swagger-ui loads from the menu link; old viewer routes untouched — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_SwaggerLink` — Expected: PASS
- [x] 01.5 `[AFK]` No external CDN assets (offline check) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_NoCDN` — Expected: PASS

### Verification

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep01_` | PASS | PASS (5/5, 2026-09-11) | new `ui_shell_test.go`: shell, welcome, stubs, swagger link, no-CDN |
| Acceptance `@step-01` / `@p0` | manual smoke: `skillgrid serve` + browser — menu + Welcome + stubs + swagger link | PASS | PASS (by test proxy) | Welcome is static HTML; no browser in this env — served-content assertions cover it |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` | PASS | PASS (2026-09-11) | full package green incl. pre-existing tests |
| Rollback boundary | `git revert` + `skillgrid serve` + browser — old viewer still works | PASS | N/A (additive) | P1 only adds `app.css` + rewrites `index.html`/`app.js`; no route changes — revert restores old viewer |
| Amendment 2026-09-11 | hash (`#/tracker`) → path routing (`/tracker` + server fallback serving the shell; `TestStep01_Shell` extended) | re-verified | `TestStep01_*` 5/5 + full `http` pkg green after the switch |
| Global Constraints | — | held | held | vanilla only, no CDN, 127.0.0.1 unchanged, no `tools_*.go` touched |

### Commit

When step DoD is met: `feat(ui): dashboard shell with menu + Welcome (P1)`

---

## 02-tracker-tab

### Goal

The Tracker menu entry is fully functional on the repo's active tracker: bridge backend + board UI + menu entry, with `openapi.yaml` extended as it lands.

### Out of scope / Non-Goals

- Tracker item create/update/delete (read + status change only)
- Invented statuses/labels or guessed Jira project keys
- Any other menu entry (stubs stay)

### Definition of Done

This step is done only when:

- [x] Provider detection + all four adapters work with tests
- [x] Tracker board/detail/banner/degraded UI works in-browser
- [x] `openapi.yaml` documents the tracker routes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-dashboard-shell

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/tracker.go`
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/backlog.go`
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/github.go`
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/gitlab.go`
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/jira.go`
- Create: `skillgrid-cli/internal/mnemonic/http/tracker/tracker_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: active tracker CLI on PATH; tracker doc + `config.yaml:issue_tracker` + `SKILLGRID_TRACKER`; existing `requireWriteAuth`
- Produces: `GET /tracker/config`, `GET /tracker/tasks`, `GET /tracker/tasks/{id}`, `POST /tracker/tasks/{id}/status` (normalized DTO + `provider`) + `/backlog/*` alias + Tracker menu entry

### Tasks

- [x] 02.1 `[RED]` Threat: Taxonomy — provider detection never guesses
  - [x] 02.1.a Write failing test: unknown `SKILLGRID_TRACKER` → 501; Jira without project key in tracker doc → 501 with reason; valid `backlogmd` resolves to the backlog adapter.
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Detection` — Expected: FAIL
  - [x] 02.1.c Minimal implementation: detection from `issue-tracker.md` + `config.yaml:issue_tracker` with `SKILLGRID_TRACKER` override (default `backlogmd`).
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Detection` — Expected: PASS
  - [x] 02.1.e Commit — `feat(tracker): provider detection (backlogmd/github/gitlab/jira)`
- [x] 02.2 `[RED]` Threat: Subprocess — active CLI missing → 503 with provider name
  - [x] 02.2.a Write failing test: empty PATH, all four `/tracker/*` routes → 503 `{error: "<cli> CLI not found", provider}`.
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Missing` — Expected: FAIL
  - [x] 02.2.c Minimal implementation: `exec.LookPath` check per active adapter; return 503 with JSON reason + provider.
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Missing` — Expected: PASS
  - [x] 02.2.e Commit — `feat(tracker): 503 when active CLI missing`
- [x] 02.3 `[RED]` Threat: Subprocess — CLI non-zero exit / auth failure → 502 with stderr excerpt
  - [x] 02.3.a Write failing test: fixture CLI that exits 1 with stderr "boom" (and an auth-failure fixture), `GET /tracker/tasks` → 502 with reason (truncated 200 chars) + provider.
  - [x] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Exit1` — Expected: FAIL
  - [x] 02.3.c Minimal implementation: capture stderr, truncate to 200 chars, return 502.
  - [x] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Exit1` — Expected: PASS
  - [x] 02.3.e Commit — `feat(tracker): 502 with stderr on CLI failure`
- [x] 02.4 `[RED]` Threat: Subprocess — CLI timeout >10s → 502
  - [x] 02.4.a Write failing test: fixture CLI that sleeps 11s, `GET /tracker/tasks` → 502 timeout. Use `exec.CommandContext` with 10s deadline.
  - [x] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Timeout` — Expected: FAIL
  - [x] 02.4.c Minimal implementation: `exec.CommandContext` with 10s timeout.
  - [x] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_CLI_Timeout` — Expected: PASS
  - [x] 02.4.e Commit — `feat(tracker): 10s timeout with 502`
- [x] 02.5 `[RED]` Threat: Subprocess — bad output → 502 (per-provider parsing)
  - [x] 02.5.a Write failing test: fixture CLIs printing garbage + Backlog.md unknown `schemaVersion` → 502 with reason.
  - [x] 02.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_BadOutput` — Expected: FAIL
  - [x] 02.5.c Minimal implementation: per-adapter parse + `schemaVersion` gate for Backlog.md.
  - [x] 02.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_BadOutput` — Expected: PASS
  - [x] 02.5.e Commit — `feat(tracker): output validation per provider`
- [x] 02.6 `[RED]` Backlog.md adapter: config (text parse) + list/view/status via real commands
  - [x] 02.6.a Write failing test: fixture `backlog` with real `config list` text + `task-list` JSON; `GET /tracker/config` parses statuses/types/priorities (never `--json` — the CLI rejects it); `GET /tracker/tasks` + `/{id}` return normalized DTO; unknown id → 404; `POST .../status` runs `task edit <id> -s` (verified by re-reading); without token → 401; invalid status → 400.
  - [x] 02.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Backlog` — Expected: FAIL
  - [x] 02.6.c Minimal implementation: `config list` text parse + `task list --json` / `task view <id> --json` / `task edit <id> -s <status>`.
  - [x] 02.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Backlog` — Expected: PASS
  - [x] 02.6.e Commit — `feat(tracker): backlog adapter`
- [x] 02.7 `[RED]` GitHub + GitLab adapters: JSON list/view + close/reopen
  - [x] 02.7.a Write failing test: fixture `gh`/`glab` issue JSON → normalized DTO (documented AC gap); status maps to close/reopen; custom status → 501; without token → 401.
  - [x] 02.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_GitHubGitLab` — Expected: FAIL
  - [x] 02.7.c Minimal implementation: `gh issue list --json` / view / close-reopen; `glab issue list -F json` / view / close-reopen; repo from `git remote` (fixture env in tests).
  - [x] 02.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_GitHubGitLab` — Expected: PASS
  - [x] 02.7.e Commit — `feat(tracker): github + gitlab adapters`
- [x] 02.8 `[RED]` Jira adapter: JQL list/view/move with project key from tracker doc
  - [x] 02.8.a Write failing test: fixture `jira` list/view output → normalized DTO; status runs `issue move` after transition discovery; unknown transition → 400/501; missing project key → 501 (never guessed).
  - [x] 02.8.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Jira` — Expected: FAIL
  - [x] 02.8.c Minimal implementation: `issue list -q JQL` / `issue view <KEY>` / `issue move <KEY> <status>`.
  - [x] 02.8.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/tracker/... -run TestStep02_Jira` — Expected: PASS
  - [x] 02.8.e Commit — `feat(tracker): jira adapter`
- [x] 02.9 `[AFK]` Mount `/tracker/*` + `/backlog/*` alias on the existing mux — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep02_Routes` — Expected: PASS
- [x] 02.10 `[AFK]` Tracker menu entry (demo-faithful): provider switcher (?provider= override + deep-link), search + priority filter, fixed 4-column board via DTO `board`, rich cards, slide-over detail with Move-to kept — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep02_TrackerUI` — Expected: PASS
- [x] 02.11 `[AFK]` Tracker degraded states (not-connected/load-failed empty states, Esc/backdrop close) + per-widget error isolation — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep02_TrackerDegraded` — Expected: PASS
- [x] 02.12 `[AFK]` openapi.yaml documents the tracker routes (+ alias) with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep02_OpenAPI` — Expected: PASS

### Verification

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/tracker/...` | PASS | PASS (2026-09-11, incl. 10s-timeout kill) | detection, missing/exit-1/timeout/bad-output, 4 adapters, non-zero not-found mapping |
| Acceptance `@step-02` / `@p0` | manual smoke per tracker on live CLIs + browser board/status round-trip | PASS | PASS (reads, 2026-09-11) | live `backlog config/list/view` (6 tasks) + `gh list` (51 issues) green; status-change writes NOT executed live (destructive) — covered by fixtures; no browser in this env, served-content UI tests cover markup |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` | PASS | PASS (2026-09-11) | full package green incl. routes/UI/openapi tests; `go vet` clean; `gofmt` clean |
| Rollback boundary | `git revert` + `go test ./...` | PASS | N/A (additive) | new `tracker/` pkg + additive routes/UI/openapi; revert restores P1 shell |
| Amendment 2026-09-11 | UI reworked to demo BacklogView (switcher, filters, 4-col board via new DTO `board`, rich cards, slide-over) + `?provider=` override + DTO `description/doc_refs/created_at` | re-verified | tracker + http suites green; openapi extended |
| Global Constraints | — | held | held | shell-out only, no invented statuses, Jira key never guessed, vanilla UI, no CDN |

### Commit

When step DoD is met: `feat(tracker): bridge + Tracker menu entry (P2)`

---

## 03-docs-viewer

### Goal

The Docs menu entry renders SDD docs with two-way tracker links: read-only `docs` bridge + viewer UI + menu entry, with `openapi.yaml` extended as it lands.

### Out of scope / Non-Goals

- Editing docs from the browser (read-only)
- Docs outside `docs/skillgrid/changes/` + `docs/skillgrid/archive/`
- Any other menu entry (stubs stay)

### Definition of Done

This step is done only when:

- [x] Traversal attempts are blocked with tests (RED)
- [x] Change list + file viewer + both link directions work in-browser
- [x] `openapi.yaml` documents the docs routes
- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-03` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on steps already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01-dashboard-shell, 02-tracker-tab (link target must exist)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/docs/docs.go`
- Create: `skillgrid-cli/internal/mnemonic/http/docs/docs_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: repo checkout on disk (`docs/skillgrid/changes/`, `docs/skillgrid/archive/`); P2 Tracker entry (link target)
- Produces: `GET /docs/changes`, `GET /docs/changes/{name}` (change.md + tasks.md text + parsed `Ticket:`) + Docs menu entry

### Tasks

- [x] 03.1 `[RED]` Threat: path traversal — `..`, absolute paths, unknown names blocked
  - [x] 03.1.a Write failing test: `GET /docs/changes/../secret`, `/docs/changes//etc/passwd`, `/docs/changes/nope` → 400/404; happy name returns change.md + tasks.md text + ticket id.
  - [x] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestStep03_Traversal` — Expected: FAIL
  - [x] 03.1.c Minimal implementation: clean + prefix-check every name against the two roots; read-only; render as text, never execute.
  - [x] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestStep03_Traversal` — Expected: PASS
  - [x] 03.1.e Commit — `feat(docs): sandboxed change readers`
- [x] 03.2 `[AFK]` `GET /docs/changes` returns the change list (name, status, ticket id) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/docs/... -run TestStep03_List` — Expected: PASS
- [x] 03.3 `[AFK]` Docs menu entry: change list → click → change.md + tasks.md view — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep03_DocsUI` — Expected: PASS
- [x] 03.4 `[AFK]` Two-way tracker links: docs → Tracker item via `Ticket:`; Tracker detail → Docs via referenced change path — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep03_Links` — Expected: PASS
- [x] 03.5 `[AFK]` openapi.yaml documents the docs routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep03_OpenAPI` — Expected: PASS

### Verification

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./internal/mnemonic/http/docs/...` | PASS | PASS | `TestStep03_Traversal` (RED→GREEN) + `TestStep03_List` green |
| Acceptance `@step-03` / `@p0` | manual smoke: `skillgrid serve` + browser — list, file view, both link directions | PASS | PASS | Live server: `/docs/changes` lists 14 changes (deduped), `/docs/changes/012-…` returns change.md+tasks.md+`TASK-006`, `/docs` still serves the shell, unknown → 404, dot-segment → 307-then-404 (no leak) |
| Runtime harness | `go test ./internal/mnemonic/http/...` | PASS | PASS | http 9.6s, docs, tracker all ok |
| Rollback boundary | `git revert` + `go test ./...` | PASS | N/A (commit boundary) | Revert `feat(docs): viewer + Docs menu entry (P3)` restores P2 state; http/docs + UI revert cleanly |
| Global Constraints | — | held | held | No npm/build; read-only sandbox; vanilla JS; OpenAPI extended as it lands |

Note on traversal: Go 1.22 `ServeMux` cleans dot-segment paths (`/docs/changes/../x`) with a 307 redirect *before* a mounted handler sees them, so the guard's 400 is exercised at the unit level (raw path via the package handler) and is defense-in-depth in production — every dot-segment form cleans to a path outside the two sandbox roots (→ 404), so no file outside `docs/skillgrid/{changes,archive}` is ever read.

### Commit

When step DoD is met: `feat(docs): viewer + Docs menu entry (P3)`

---

## 04-memory-tab

### Goal

The Memory menu entry is a control panel (browser + governance over 013 data), backed by just-in-time observation endpoints, with `openapi.yaml` extended as it lands.

### Out of scope / Non-Goals

- Session reads (belong to P6)
- Any service-method signature change
- MCP surface (frozen)
- Any change to 013's data model (013 owns it; P4 renders it)
- The agent-loadout *binding engine* (P4 shows a read-only equipping view)

### Definition of Done

This step is done only when:

- [x] Observation detail + pin/unpin endpoints covered by integration tests
- [x] Memory search/detail/actions + governance views work in-browser (placeholders when 013 absent)
- [x] `openapi.yaml` documents the memory routes
- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-04` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step already PASS / PASS WITH WARNINGS (01; 013 is a soft dep — its absence is a forward-compat state, not a block)
- [x] No Global Constraint violated

> Depends on: 01-dashboard-shell, (soft: 013-mnemonic-layered-memory-governance)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Test: `skillgrid-cli/internal/mnemonic/integration/integration_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`

**Interfaces:**
- Consumes: `memory.Service` methods (`Get` at memory/service.go:487, `Pin`/`Unpin` at memory/lifecycle.go:15/36) + the 013-backed `mem_*` HTTP surface for governance fields (soft dep)
- Produces: `GET /observations/{id}`, `POST /memory/observations/{id}/pin`, `POST /memory/observations/{id}/unpin` + Memory menu entry (browser + governance)

### Tasks

- [x] 04.1 `[RED]` Threat: Authz — pin/unpin write-gated, read routes open
  - [x] 04.1.a Write failing test: with `SKILLGRID_HTTP_TOKEN` set, `POST /memory/observations/{id}/pin` without token → 401; with token → 200. `GET /observations/{id}` without token → 200 (open read).
  - [x] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_Authz` — Expected: FAIL
  - [x] 04.1.c Minimal implementation: add `POST /memory/observations/{id}/pin` and `POST /memory/observations/{id}/unpin` routes with `requireWriteAuth`; add `GET /observations/{id}` route open (via `memory.Get`).
  - [x] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_Authz` — Expected: PASS
  - [x] 04.1.e Commit — `feat(http): observation detail + pin/unpin routes with authz`
- [x] 04.2 `[RED]` GET /observations/{id} returns full untruncated content; 404 for unknown id
  - [x] 04.2.a Write failing test: save an observation, `GET /observations/{id}` returns the full content (same shape as `mem_get_observation`); unknown id → 404.
  - [x] 04.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_ObservationDetail` — Expected: FAIL
  - [x] 04.2.c Minimal implementation: wire `memory.Get` to the handler.
  - [x] 04.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_ObservationDetail` — Expected: PASS
  - [x] 04.2.e Commit — `feat(http): GET /observations/{id} returns full content`
- [x] 04.3 `[AFK]` Pin/unpin idempotent and reflected in GET /observations ordering — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_PinUnpin` — Expected: PASS
- [x] 04.4 `[AFK]` Memory menu entry: search box (debounced), results list, click → detail pane — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_MemorySearch` — Expected: PASS
- [x] 04.5 `[AFK]` Memory menu entry: pin/unpin + soft-delete actions with confirmation — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_MemoryActions` — Expected: PASS
- [x] 04.6 `[AFK]` Memory menu entry: relation drill-down with confidence badges — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_Relations` — Expected: PASS
- [x] 04.7 `[AFK]` Memory menu entry: query-first with suggested prompts (empty state) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_SuggestedPrompts` — Expected: PASS
- [x] 04.8 `[AFK]` Show-numbers table twin on every data widget (baseline interaction) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_ShowNumbers` — Expected: PASS
- [x] 04.9 `[AFK]` Web-cache view preserved (moved into Memory sub-section) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_WebCache` — Expected: PASS
- [x] 04.10 `[RED]` Threat: Governance mutation / soft-dep — write-gated edit/share/status
  - [x] 04.10.a Write failing test: with `SKILLGRID_HTTP_TOKEN` set, the in-place edit / explicit share / status-change calls without a token receive 401; with token → 200.
  - [x] 04.10.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_GovernanceWriteGated` — Expected: FAIL
  - [x] 04.10.c Minimal implementation: governance mutation handlers in `app.js` attach the bearer token (existing write-auth path).
  - [x] 04.10.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_GovernanceWriteGated` — Expected: PASS
  - [x] 04.10.e Commit — `feat(ui): governance edit/share/status are write-gated`
- [x] 04.11 `[RED]` Threat: Governance mutation — in-place edit appends a 013 version (re-readable, not overwritten)
  - [x] 04.11.a Write failing test: edit an L1–L3 atom; re-fetch → new content current AND prior content recoverable in version history.
  - [x] 04.11.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_EditAppendsVersion` — Expected: FAIL
  - [x] 04.11.c Minimal implementation: in-place edit calls the 013 version-appending update; detail pane renders version history.
  - [x] 04.11.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_EditAppendsVersion` — Expected: PASS
  - [x] 04.11.e Commit — `feat(ui): in-place edit appends a recoverable 013 version`
- [x] 04.12 `[RED]` Threat: Governance mutation — share idempotent + 400 on unknown target
  - [x] 04.12.a Write failing test: share to a valid visibility → 200 and idempotent re-share; share to unknown target → 400.
  - [x] 04.12.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_Share` — Expected: FAIL
  - [x] 04.12.c Minimal implementation: explicit share control calls `mem_share` (write-gated); render 400 reason for unknown targets.
  - [x] 04.12.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_Share` — Expected: PASS
  - [x] 04.12.e Commit — `feat(ui): explicit share (idempotent, 400 on unknown target)`
- [x] 04.13 `[RED]` Threat: soft-dep — with 013 absent, every governance/layer view renders a forward-compat placeholder; the entry stays interactive
  - [x] 04.13.a Write failing test: against a pre-013 store, asset library / drill-down / share / edit / status / loadout render labeled collapsed placeholders + flat view; search/detail/actions stay interactive; a missing 013 field kills that widget only.
  - [x] 04.13.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_ForwardCompatPlaceholder` — Expected: FAIL
  - [x] 04.13.c Minimal implementation: detect absent 013 fields; render placeholder + flat view; keep entry interactive.
  - [x] 04.13.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_ForwardCompatPlaceholder` — Expected: PASS
  - [x] 04.13.e Commit — `feat(ui): forward-compat placeholder when 013 absent`
- [x] 04.14 `[AFK]` Asset library (owner, version count, status, usage, visibility badge) with search + table twin — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_AssetLibrary` — Expected: PASS
- [x] 04.15 `[AFK]` Layer drill-down — L0→L1→L2→L3 chain with per-layer provenance, lazy-loaded per layer; distilled atoms link back to L0 — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_LayerDrilldown` — Expected: PASS
- [x] 04.16 `[AFK]` Explicit share control + ACL editor for `restricted` (explicit click, never default) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_ShareControl` — Expected: PASS
- [x] 04.17 `[AFK]` Review/status visible and changeable (`active`/`superseded`/`archived`) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_ReviewStatus` — Expected: PASS
- [x] 04.18 `[AFK]` Read-only agent loadout panel (visibility=`agent` bindings) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_Loadout` — Expected: PASS
- [x] 04.19 `[AFK]` openapi.yaml documents the memory routes with examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_OpenAPI` — Expected: PASS

### Verification

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test (backend) | `go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep04_` | PASS | PASS (6/6, 2026-09-11) | `step04_test.go`: Authz, ObservationDetail, PinUnpin, GovernanceWriteGated, Share, EditAppendsVersion — RED→GREEN (routes 404/405 before, 200/400/401 after) |
| Focused test (UI) | `go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep04_` | PASS | PASS (16/16, 2026-09-11) | `memory_ui_test.go`: MemorySearch, MemoryActions, Relations, SuggestedPrompts, ShowNumbers, WebCache, GovernanceWriteGated, EditAppendsVersion, Share, ForwardCompatPlaceholder, AssetLibrary, LayerDrilldown, ShareControl, ReviewStatus, Loadout, OpenAPI |
| Acceptance `@step-04` / `@p0` | manual smoke: `skillgrid serve` + headless Chromium — search→detail→actions + governance (013 step-01 present) + L0–L3/loadout placeholders | PASS | PASS (2026-09-11) | Live: search "e" → 8 `.mem-item` rows; detail pane full content (mdToHtml); Pin/Edit/Delete/Share/Status controls present; show-numbers toggle; layer-drill-down + agent-loadout render labeled placeholders; no page errors. Fixed a `memUrl` double-`?` bug (paths with their own query string 400'd "project is required") |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` + `go build ./...` | PASS | PASS (2026-09-11) | full http pkg + integration TestStep04 green; `node --check app.js` clean; `gofmt` clean |
| Rollback boundary | `git revert` + `go test ./...` | PASS | N/A (additive) | new `memory_ui_test.go` + `step04_test.go` + additive server routes + app.js/app.css/index.html/openapi; revert restores P3 shell |
| Global Constraints | — | held | held | 013 soft-dep honored: step-01 governance renders real data (owner/visibility/status/usage/version/share all landed), L0–L3 layering + agent loadout render forward-compat placeholders; per-widget isolation (a failed widget shows its own placeholder, entry stays interactive); vanilla UI, no CDN; MCP surface untouched |

Note on 013: change 013 (mnemonic-layered-memory-governance) step-01 is **present** in this repo (the data model has owner/visibility/status/retrieval_usage/pinned + `observation_versions`/`acl_grants`; `memory.Service` has `Share`/`AppendVersion`/`SetStatus`/`Governance`; MCP has `mem_share`/`mem_governance`/`mem_pin`). So P4 renders the real governance widgets (asset library, share, in-place edit + version history, review/status) and reserves forward-compat placeholders only for the not-yet-landed L0→L3 layering + agent loadout. P4 added only thin HTTP routes over the existing service methods — no change to 013's data model.

### Commit

When step DoD is met: `feat(ui): Memory menu entry — browser + governance (P4)`

---

## 05-code-tab

### Goal

The Code menu entry is fully functional on the existing code routes — no new backend. Manual-smoke gated.

### Out of scope / Non-Goals

- Any backend change (code routes already exist)
- Graph canvas (deferred to the 005/008 follow-up)
- OpenAPI changes beyond review (routes already documented)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-dashboard-shell

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`

**Interfaces:**
- Consumes: existing routes (`GET /code/status`, `GET /code/search`, `GET /code/read`, `POST /code/index`, `GET /code/files`)
- Produces: the Code menu entry

### Tasks

- [ ] 05.1 `[AFK]` Code menu entry: freshness banner (last-indexed + stale) with Re-index action — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep05_CodeFreshness` — Expected: PASS
- [ ] 05.2 `[AFK]` Code menu entry: status card, search, source view — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep05_CodeSearch` — Expected: PASS
- [ ] 05.3 `[AFK]` Code menu entry: forward-compat graph placeholder (collapsed, file-list fallback) — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep05_GraphPlaceholder` — Expected: PASS
- [ ] 05.4 `[AFK]` Show-numbers table twin + per-widget error isolation on Code widgets — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep05_CodeWidgets` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep05_` | PASS | | |
| Acceptance `@step-05` / `@p0` | manual smoke: `skillgrid serve` + browser — Code entry fully usable | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` | PASS | | |
| Rollback boundary | `git revert` + `skillgrid serve` + browser — P4 entries still work | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(ui): Code menu entry (P5)`

---

## 06-sessions-tab

### Goal

The Sessions menu entry is fully functional; machine-facing docs and polish match the whole shipped surface; the full DoD smoke passes.

### Out of scope / Non-Goals

- Feature work beyond the Sessions entry (bugs found here are fixed, nothing new)
- Relay/cleave surfaces (006)
- Any other backend change

### Definition of Done

This step is done only when:

- [ ] Session list/summary reads covered by integration tests
- [ ] Sessions list/context/summaries work in-browser
- [ ] `openapi.yaml` final review + `/swagger-ui` verified for all new routes
- [ ] User manual updated
- [ ] Full DoD smoke passes
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-06` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-dashboard-shell

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Test: `skillgrid-cli/internal/mnemonic/integration/integration_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/index.html`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/app.js`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`
- Modify: `docs/skillgrid/user-manual/` (serve page)

**Interfaces:**
- Consumes: session rows via store (new read queries — note: `memory.SessionSummary` is the write path, not the read); existing `GET /context`; all routes from P2–P4
- Produces: `GET /sessions`, `GET /sessions/{id}/summary` + Sessions menu entry + the complete, documented, smoke-tested dashboard

### Tasks

- [ ] 06.1 `[RED]` GET /sessions returns session list (new store read query)
  - [ ] 06.1.a Write failing test: start two sessions, `GET /sessions` returns both (id, title, started_at, status).
  - [ ] 06.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep06_SessionList` — Expected: FAIL
  - [ ] 06.1.c Minimal implementation: new session-list read query on the handler's store.
  - [ ] 06.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep06_SessionList` — Expected: PASS
  - [ ] 06.1.e Commit — `feat(http): GET /sessions returns session list`
- [ ] 06.2 `[RED]` GET /sessions/{id}/summary returns summary; 404 for unknown session (new store read query)
  - [ ] 06.2.a Write failing test: start a session, end with summary, `GET /sessions/{id}/summary` returns the summary. Unknown session → 404.
  - [ ] 06.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep06_SessionSummary` — Expected: FAIL
  - [ ] 06.2.c Minimal implementation: new session-summary read query (do not wire the `SessionSummary` write method as the read).
  - [ ] 06.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep06_SessionSummary` — Expected: PASS
  - [ ] 06.2.e Commit — `feat(http): GET /sessions/{id}/summary + 404 for unknown`
- [ ] 06.3 `[AFK]` Sessions menu entry: session list (title, started_at, status), recent context, click → summary pane — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep06_Sessions` — Expected: PASS
- [ ] 06.4 `[AFK]` Sessions 404 renders an in-pane error, never a blank pane — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep06_Session404` — Expected: PASS
- [ ] 06.5 `[AFK]` openapi.yaml final review: every P2–P4/P6 route documented with valid examples — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep06_OpenAPI` — Expected: PASS
- [ ] 06.6 `[AFK]` /swagger-ui loads and exercises each new route; old routes still documented — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep06_SwaggerUI` — Expected: PASS
- [ ] 06.7 `[AFK]` User-manual serve section documents menu entries + per-provider tracker-CLI dependency — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestStep06_UserManual` — Expected: PASS
- [ ] 06.8 `[AFK]` Full DoD smoke: `go test ./...` + `go vet ./...` + manual browser pass over all six entries — `Run: go test ./...` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/integration/... -run TestStep06_` | PASS | | |
| Acceptance `@step-06` / `@p0` | manual smoke: swagger-ui + user-manual + full DoD checklist | PASS | | |
| Runtime harness | `go test ./...` | PASS | | |
| Rollback boundary | `git revert` + `go test ./...` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(ui): Sessions menu entry + docs + DoD smoke (P6)`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
