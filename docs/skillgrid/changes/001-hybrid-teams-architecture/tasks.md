# Tasks: 001-hybrid-teams-architecture

> **STATUS:** `in-progress` (2026-09-04) — 0/5 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Give Mnemonic a hybrid control/data plane so agents can spawn, claim, complete, and peer-review team tasks end-to-end.

**Architecture:** SQLite metadata + filesystem markdown under `.skillgrid/files/`; domain on `service.Service`; MCP and HTTP adapters. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), HTTP mux with bearer-token auth on writes.

**Spec:** `docs/skillgrid/changes/001-hybrid-teams-architecture/change.md`

**Acceptance:** `docs/skillgrid/changes/001-hybrid-teams-architecture/acceptance.feature` (`@step-NN`)

---

## Goal

Orchestrators and sub-agents can spawn, pull, read, submit, review, and mark done team tasks with SQL metadata and filesystem markdown content, so teamwork skills have a real backend.

## Out of scope / Non-Goals

- Embeddings / vector search (separate change)
- Python microservice / Hermes memory
- Session relay / Cleave-style handoffs
- Issue tracker / `project_name` integration
- MCP inbox tools (`agent_send_message` / `agent_read_inbox`)
- Tiered L0/L1/L2 storage (change **003**); only no-op content-write seam
- Migration `010_*` (owned by change **003**)

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

- No embeddings / vector search; no Python microservice; no session relay; no tracker integration
- No MCP inbox tools in this change; `messages` table may exist for HTTP readiness only
- No L0/L1/L2 tiered storage; only a no-op content-write seam for later change 003
- Do not create migration `010_*` (owned by change 003); use additive `009_*` only
- SQLite stores metadata only; content is `.md` under `{project}/.skillgrid/files/`
- Write filesystem content first, then SQL; on SQL failure delete the orphaned file
- MCP tools follow existing `s.AddTool(toolDef, handlerFunc)` pattern
- HTTP routes follow existing `mux.HandleFunc` with `requireWriteAuth` on writes
- Teams HTTP under `/teams/…` must not collide with `/memory/reviews`
- SQL insert fails after FS write → `abort` + delete orphan file
- Pull with no pending tasks → `abort`; clear error; nothing claimed
- Unknown task id → `abort`; clear error; no panic
- Bad / missing MCP spawn args → `abort`; structured tool error; no panic; no orphan
- HTTP write without bearer when `SKILLGRID_HTTP_TOKEN` set → `abort` (401)

---

## State

```yaml
phase: apply         # spec | apply | verify | archive
current_step: 01-schema
status: in_progress  # in_progress | blocked | done
updated: 2026-09-05T12:20:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `schema` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `service-facade` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `mcp-tools` | `@step-03` | 02 | Feature tagged `@step-03` |
| 04 | `http-routes` | `@step-04` | 02 | Feature tagged `@step-04` |
| 05 | `tests` | `@step-05` | 03, 04 | Feature tagged `@step-05` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1300 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | single-pr |

---

## 01-schema

### Goal

Add additive SQLite migration for teams/tasks/messages/reviews and ContentPlane filesystem helpers so store open creates tables safely and content writes keep SQL metadata-only.

### Out of scope / Non-Goals

- Service facade methods, MCP tools, HTTP routes
- Tiered storage / migration `010_*`

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/009_teams_schema.sql`
- Create: `skillgrid-cli/internal/mnemonic/files/content.go`
- Test: `skillgrid-cli/internal/mnemonic/store/` (migration apply / idempotence), `skillgrid-cli/internal/mnemonic/files/` (if package tests added)

**Interfaces:**
- Consumes: existing store migration embed + `index_meta`
- Produces: teams schema tables; `ContentPlane` Write/Read with FS-first rollback + no-op post-write seam

### Tasks

- [x] 01.1 `[RED]` Store open adds teams tables without rewriting observations — Scenario: Store open adds teams tables safely
  - [x] 01.1.a Write failing test
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Teams|009|Migration'` — Expected: FAIL
  - [x] 01.1.c Minimal implementation — create `009_teams_schema.sql` with `teams`, `team_members`, `tasks`, `messages`, `task_results`, `reviews` (paths/status only; `_path` columns); do not use `010_*`
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Teams|009|Migration'` — Expected: PASS
  - [x] 01.1.e Commit — `feat(mnemonic): add 009 teams schema migration`
- [x] 01.2 `[RED]` SQL fail after FS write rolls back content file — Scenario: SQL fail after FS write rolls back
  - [x] 01.2.a Write failing test
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/files/ -run 'Rollback|ContentPlane'` — Expected: FAIL
  - [x] 01.2.c Minimal implementation — create `files/content.go` ContentPlane Write/Read under `{project}/.skillgrid/files/{tasks,messages,reviews}/…`; FS-first; delete file on SQL failure; no-op post-write seam (no L0/L1/L2)
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/files/ -run 'Rollback|ContentPlane'` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): add ContentPlane FS-first writer`
- [x] 01.3 `[AFK]` Markdown on disk with SQL paths only (no tiered layers) — Scenario: Markdown on disk SQL paths only — `Run: go test ./skillgrid-cli/internal/mnemonic/files/ ./skillgrid-cli/internal/mnemonic/store/` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/files/` | PASS | PASS (apply) | exit 0 |
| Acceptance `@step-01` / `@p0` | Map scenarios in `acceptance.feature` `@step-01` | PASS | covered by store+files tests | verify owns verdict |
| Runtime harness | store open with existing observations | PASS | PASS (apply) | TestTeamsSchema* |
| Rollback boundary | SQL fail after FS write leaves no orphan | PASS | PASS (apply) | TestContentPlaneWriteRollbackOnSQLFail |
| Global Constraints | — | held | held (apply) | no 010_*; no tiers |

### Commit

When step DoD is met: `feat(mnemonic): teams schema and ContentPlane`

---

## 02-service-facade

### Goal

Add Service methods (spawn, pull, read, submit output, submit review, mark done) wired through `openProject` so orchestrators can run the task lifecycle.

### Out of scope / Non-Goals

- MCP registration and HTTP routes (steps 03–04)
- Inbox messaging tools

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-schema

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/teams.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: deferred focused suite in step 05; smoke via service package as needed

**Interfaces:**
- Consumes: ContentPlane; teams schema tables
- Produces: `SpawnTask`, `PullNextTask`, `ReadTask`, `SubmitOutput`, `SubmitReview`, `MarkDone`

### Tasks

- [ ] 02.1 `[RED]` Spawn returns pending task id — Scenario: Spawn returns pending task id
  - [ ] 02.1.a Write failing test
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'SpawnTask'` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — `SpawnTask` (brief via ContentPlane → SQL `pending`, return id)
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'SpawnTask'` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): spawn team task on service facade`
- [ ] 02.2 `[AFK]` Pull claims top priority; read returns brief — Scenario: Pull claims top priority with brief — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Pull|ReadTask'` — Expected: PASS
- [ ] 02.3 `[AFK]` Output → review_spec; review sets passed; mark done → done + task_results — Scenario: Output review done advance status — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Submit|MarkDone|Status'` — Expected: PASS
- [ ] 02.4 `[AFK]` Empty pull fails clearly — Scenario: Empty pull fails clearly — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Pull.*Empty|EmptyPull'` — Expected: PASS
- [ ] 02.5 `[AFK]` Wire teams via `openProject` / project handle in `service.go` as needed — `Run: go test ./skillgrid-cli/internal/mnemonic/service/` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Spawn|Pull|Submit|MarkDone'` | PASS | | |
| Acceptance `@step-02` / `@p0` | Map scenarios in `acceptance.feature` `@step-02` | PASS | | |
| Runtime harness | spawn → pull → read → submit → review → done | PASS | | |
| Rollback boundary | ContentPlane rollback still held | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): teams service facade lifecycle`

---

## 03-mcp-tools

### Goal

Register six team/agent MCP tools that call the service facade, with RED coverage for the Mnemonic tool-surface threat.

### Out of scope / Non-Goals

- Inbox MCP tools
- Changing mem/code/web tool contracts
- HTTP routes (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-service-facade

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_teams.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server_test.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (and early `tools_teams_test.go` stubs if needed)

**Interfaces:**
- Consumes: service teams methods
- Produces: `team_spawn_task`, `agent_pull_next_task`, `agent_read_task`, `agent_submit_output`, `agent_submit_review`, `agent_mark_done` via `registerTeamsTools`

### Tasks

- [ ] 03.1 `[RED]` (threat: Mnemonic tool surface) Six team tools registered; inbox absent — Scenario: Six team tools are registered
  - [ ] 03.1.a Write failing test — update `TestAllToolsRegistered` want list with the six names
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TestAllToolsRegistered'` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — create `tools_teams.go` with six `s.AddTool` handlers; call `registerTeamsTools(s)` from `server.go`
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TestAllToolsRegistered'` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): register six teams MCP tools`
- [ ] 03.2 `[RED]` (threat: Mnemonic tool surface) Bad spawn → structured tool error not panic — Scenario: Bad spawn errors without orphan state
  - [ ] 03.2.a Write failing test
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'BadSpawn|Spawn.*Invalid'` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — validate args; return tool error result
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'BadSpawn|Spawn.*Invalid'` — Expected: PASS
  - [ ] 03.2.e Commit — `fix(mnemonic): structured error on bad team spawn`
- [ ] 03.3 `[AFK]` Spawn pull read submit stay consistent — Scenario: Spawn pull read submit stay consistent — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Teams|Spawn|Pull'` — Expected: PASS
- [ ] 03.4 `[AFK]` Unknown id or empty queue errors without panic — Scenario: Unknown id or empty queue errors — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unknown|Empty'` — Expected: PASS
- [ ] 03.5 `[AFK]` mem/code/web tool names unchanged — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TestAllToolsRegistered'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TestAllToolsRegistered|BadSpawn|Teams'` | PASS | | |
| Acceptance `@step-03` / `@p0` | Map scenarios in `acceptance.feature` `@step-03` | PASS | | |
| Runtime harness | list tools; spawn→pull→read→submit | PASS | | |
| Rollback boundary | bad spawn leaves no orphan | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): teams MCP tool surface`

---

## 04-http-routes

### Goal

Add `/teams/…` CRUD with `requireWriteAuth` on writes and no collision with `/memory/reviews`.

### Out of scope / Non-Goals

- MCP tools; changing memory review routes
- New auth mechanisms beyond existing bearer pattern

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-service-facade

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Test: deferred focused HTTP tests in step 05

**Interfaces:**
- Consumes: service teams methods; `requireWriteAuth`
- Produces: `/teams/…` CRUD endpoints (writes auth-gated; GETs open)

### Tasks

- [ ] 04.1 `[RED]` Authenticated write under teams path succeeds — Scenario: Authenticated write under teams path succeeds
  - [ ] 04.1.a Write failing test
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams.*Auth|TeamsWrite'` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation — add `/teams/…` CRUD; wrap writes with `requireWriteAuth`; leave GETs open
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams.*Auth|TeamsWrite'` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(mnemonic): add teams HTTP routes`
- [ ] 04.2 `[AFK]` GETs stay open; teams paths distinct from memory reviews — Scenario: Gets stay open and teams paths stay distinct — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams|Reviews'` — Expected: PASS
- [ ] 04.3 `[AFK]` Unauthenticated write returns 401 — Scenario: Unauthenticated write returns 401 — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams.*401|Unauth'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams'` | PASS | | |
| Acceptance `@step-04` / `@p0` | Map scenarios in `acceptance.feature` `@step-04` | PASS | | |
| Runtime harness | POST with/without bearer; GET without bearer | PASS | | |
| Rollback boundary | no `/memory/reviews` collision | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): teams HTTP CRUD routes`

---

## 05-tests

### Goal

Close automated coverage for facade atomicity, MCP registration/dispatch, and HTTP write auth, including remaining Mnemonic tool-surface RED parity.

### Out of scope / Non-Goals

- New production features beyond test coverage
- Changing mem/code/web contracts

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 03-mcp-tools, 04-http-routes

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/teams_test.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_teams_test.go`
- Create: `skillgrid-cli/internal/mnemonic/http/teams_test.go`

**Interfaces:**
- Consumes: teams facade, MCP tools, HTTP routes
- Produces: regression suite proving DoD criteria

### Tasks

- [ ] 05.1 `[RED]` (threat: Mnemonic tool surface) Registry/dispatch includes six names; bad spawn tool error — Scenario: Registry has six team tools and keeps memory and code tools
  - [ ] 05.1.a Write failing test in `tools_teams_test.go`
  - [ ] 05.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TeamsTools|Dispatch'` — Expected: FAIL
  - [ ] 05.1.c Minimal implementation — dispatch + FS/SQL parity coverage so assertion holds
  - [ ] 05.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'TeamsTools|Dispatch'` — Expected: PASS
  - [ ] 05.1.e Commit — `test(mnemonic): cover teams MCP dispatch`
- [ ] 05.2 `[RED]` (threat: Mnemonic tool surface) Atomicity + bad spawn structured error fixtures — Scenario: Tests assert rollback and bad spawn structured error
  - [ ] 05.2.a Write failing test in `teams_test.go` / `tools_teams_test.go`
  - [ ] 05.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Rollback|BadSpawn|Atomicity'` — Expected: FAIL
  - [ ] 05.2.c Minimal implementation — complete fixture coverage
  - [ ] 05.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Rollback|BadSpawn|Atomicity'` — Expected: PASS
  - [ ] 05.2.e Commit — `test(mnemonic): cover teams atomicity and bad spawn`
- [ ] 05.3 `[AFK]` Service tests: pull priority + status transitions — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Pull|Status|Teams'` — Expected: PASS
- [ ] 05.4 `[AFK]` HTTP teams_test: 401 without bearer; open GET — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Teams'` — Expected: PASS
- [ ] 05.5 `[AFK]` Full suite covers facade, MCP, HTTP — Scenario: Suite covers facade MCP and HTTP teams behavior — `Run: go test ./skillgrid-cli/...` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/` | PASS | | |
| Acceptance `@step-05` / `@p0` | Map scenarios in `acceptance.feature` `@step-05` | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/...` | PASS | | |
| Rollback boundary | FS+SQL rollback fixtures | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `test(mnemonic): hybrid teams coverage suite`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
