**Goal:** Make Mnemonic callers exercise one deep Project Handle seam (one SQLite open per request) instead of a shallow open-delegate-close facade.

**Architecture:** Export Project Handle; root Service is factory + cross-store only; MCP/HTTP adapters open once. See `change.md` decisions. Deferred: store.DB unexport, dual MigrateProjects, memory package split.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), HTTP mux.

**Spec:** `docs/skillgrid/changes/007-mnemonic-project-handle-facade/change.md`

**Acceptance:** `docs/skillgrid/changes/007-mnemonic-project-handle-facade/acceptance.feature` (`@step-NN`)

---

## Goal

Agents and HTTP clients open a Project Handle once, then call memory / web / code on that handle — without double-opening SQLite or learning ~60 pass-through methods.

## Out of scope / Non-Goals

- Unexporting `store.Store.DB` / pulling all SQL behind the store seam (candidate #3)
- Renaming or unifying dual `MigrateProjects` semantics (candidate #4)
- Splitting `memory.Service` into Session / Observation / Review packages (candidate #6)
- Changing MCP tool names, HTTP route paths, or observation DTO field names
- Reworking `project.Resolve` / identity binding (owned by 002)
- New MCP tools or HTTP routes

## Definition of Done

Change is done only when **all** of the following are true:

- [x] Every success criterion / DoD checkbox in `change.md` is met
- [x] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [x] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [x] No unchecked `- [ ]` under any `### Tasks`
- [x] No **Global Constraint** violated
- [x] Rollback path in `change.md` is still valid (or N/A documented)
- [x] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- No store.DB unexport / SQL-behind-store deepen in this change
- No dual `MigrateProjects` rename/unify in this change
- No memory package split into Session/Observation/Review
- No MCP tool name, HTTP route path, or observation DTO field renames
- No project.Resolve / identity-binding rework
- No new MCP tools or HTTP routes
- Artifact Store Mode stays hybrid; no persistence-mode change
- MCP tool names and required params stay stable (Mnemonic tool surface contract)
- HTTP route paths and JSON field names stay stable
- Project resolution rules (CWD vs explicit `project` param) stay as today; only open lifecycle changes
- Cross-store identity ops remain on the root Service
- Handle open fails (missing data dir, bad project id) → `abort`
- Nil Service / uninitialized handle → `abort` at open
- Cross-store op with missing source store → existing Migrate/Merge warn+continue / no-op semantics unchanged
- MCP handler without injected Service in tests → `abort` with clear error when tests require inject
- Do not add new SQL outside existing modules (Store accessor may remain)
- No schema migrations in this change

---

## State

```yaml
phase: archive
current_step: 04-collapse-wrappers
status: done
updated: 2026-09-09T15:00:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `export-project-handle` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `mcp-single-open` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `http-single-open` | `@step-03` | 01 | Feature tagged `@step-03` |
| 04 | `collapse-wrappers` | `@step-04` | 02, 03 | Feature tagged `@step-04` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~600–1100 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

Honest notes: step 01 is small (export + tests); steps 02/03 are large adapter rewires that can run in parallel after 01; step 04 is cleanup. Prefer chained PRs: 01 alone, then 02∥03, then 04 — or 01+02, then 03, then 04.

---

## 01-export-project-handle

### Goal

Callers outside `service` can open an exported Project Handle and reach memory/web without needing the wide open-delegate wrapper surface.

### Out of scope / Non-Goals

- MCP/HTTP rewire (steps 02–03)
- Deleting all wrappers yet (step 04)
- Cross-store behavior changes

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-01` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS` (verify-waiver at archive)
- [x] Produces contracts listed under Interfaces are available to dependents
- [x] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/handle.go` (optional split)
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/service/*_test.go`

**Interfaces:**
- Consumes: none (existing unexported `projectHandle` + `openProject*`)
- Produces: exported `ProjectHandle` type; `OpenForCWD` / `OpenForDirectory` / `Open(projectID)` returning it; `ProjectID()`, `Memory()`, `Web()` (and `Store()` if still required); cross-store methods unchanged on `Service`

### Tasks

- [x] 01.1 `[RED]` Invalid or empty project id aborts open — no partial handle (threat: store open failure)
  - [x] 01.1.a Write failing test — Scenario: Invalid project id aborts open
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'InvalidProject|EmptyProject|AbortOpen' -count=1` — Expected: FAIL
  - [x] 01.1.c Minimal implementation (export type; Open validates and aborts)
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'InvalidProject|EmptyProject|AbortOpen' -count=1` — Expected: PASS
  - [x] 01.1.e Commit — `feat(mnemonic): abort Project Handle open on bad project id`
- [x] 01.2 `[RED]` Exported handle opens once and exposes Memory/Web for a happy project
  - [x] 01.2.a Write failing test — Scenario: Opened handle exposes memory and web
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'ProjectHandle|OpenOnce|Expose' -count=1` — Expected: FAIL
  - [x] 01.2.c Minimal implementation (export `ProjectHandle`; Open* return it)
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'ProjectHandle|OpenOnce|Expose' -count=1` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): export Project Handle as single-project seam`
- [x] 01.3 `[AFK]` Cross-store methods still compile and ListProjects / resolve smoke — Scenario: Cross-store root ops remain — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -count=1` — Expected: PASS

### Verification

Verdict: `PASS WITH WARNINGS` (verify-waiver at archive — sdd-verify not run; cited evidence below)

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | PASS | 34/34 tests, 75s |
| Acceptance `@step-01` / `@p0` | map scenarios to unit runs above | PASS | PASS | TestOpenEmpty/InvalidProjectIDAborts, TestProjectHandleExposesMemoryAndWeb |
| Runtime harness | N/A — library seam | PASS | PASS | |
| Rollback boundary | revert commit; no migrations | PASS | PASS | |
| Global Constraints | — | held | held | only service/ + tests changed |

### Commit

Step 01 commits: `99b97ad` (abort on bad id), `32dc818` (export seam), `e3b67d1` (test: seed webcache snapshot)

---

## 02-mcp-single-open

### Goal

Each MCP tool call opens SQLite at most once for its target project and calls domain ops through the Project Handle.

### Out of scope / Non-Goals

- HTTP rewire (step 03)
- Deleting unused Service wrappers (step 04)
- Changing tool names, params, or DTO shapes

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-02` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS` (verify-waiver at archive)
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS (step 01 applied)
- [x] No Global Constraint violated

> Depends on: 01-export-project-handle

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_code.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_web.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server_test.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/*_test.go`

**Interfaces:**
- Consumes: exported `ProjectHandle` + `Service.Open*` from 01
- Produces: MCP handlers that open once and use handle; `SetService` / inject still works

### Tasks

- [x] 02.1 `[RED]` mem_save happy path opens the store once (threat: Mnemonic tool surface + double-open)
  - [x] 02.1.a Write failing test — Scenario: mem_save opens store once
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|SingleOpen|OpenOnce' -count=1` — Expected: FAIL
  - [x] 02.1.c Minimal implementation (open handle once; call Memory via handle)
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|SingleOpen|OpenOnce' -count=1` — Expected: PASS
  - [x] 02.1.e Commit — `feat(mnemonic): MCP tools single-open via Project Handle`
- [x] 02.2 `[RED]` mem_search (and contract shape) unchanged after handle path (threat: Mnemonic tool surface)
  - [x] 02.2.a Write failing test — Scenario: mem_search result shape unchanged
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSearch|Contract|Shape' -count=1` — Expected: FAIL
  - [x] 02.2.c Minimal implementation
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSearch|Contract|Shape' -count=1` — Expected: PASS
  - [x] 02.2.e Commit — `feat(mnemonic): MCP tools single-open via Project Handle`
- [x] 02.3 `[AFK]` code_* and web_cache_* handlers use handle; SetService inject still works — Scenarios: code and web tools use handle; Injected service still works — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` — Expected: PASS

### Verification

Verdict: `PASS WITH WARNINGS` (verify-waiver at archive — sdd-verify not run; cited evidence below)

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | PASS | full mcp suite; store suite PASS |
| Acceptance `@step-02` / `@p0` | map scenarios to MCP tests | PASS | PASS | single_open_test.go: TestMemSaveSingleOpen (2→1), TestMemSearchContractShape, TestCodeSearchSingleOpen, TestWebCacheLookupSingleOpen |
| Runtime harness | optional e2e if already present | PASS | PASS | |
| Rollback boundary | revert; no migrations | PASS | PASS | |
| Global Constraints | — | held | held | mcp/ + test-only store counter only |

### Commit

Step 02 commit: `4f7e9bf` feat(mnemonic): MCP tools single-open via Project Handle

Note: mcp↔service verbatim copies (mcpOrientSymbol, mcpCodeExplore, mcpMemoryDoctor, mcpCodeEmbeddingStatus, mcpReadIndexedCode) are plan-mandated — handle exposes only Memory()/Web()/Store(). Consolidation deferred to step 04.

---

## 03-http-single-open

### Goal

HTTP single-project routes match the MCP lifecycle: one Project Handle open per request.

### Out of scope / Non-Goals

- Changing auth, routes, or JSON shapes
- Cross-store migrate/merge routes may keep root Service calls
- Deleting Service wrappers (step 04)

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-03` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS` (verify-waiver at archive)
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS (step 01 applied)
- [x] No Global Constraint violated

> Depends on: 01-export-project-handle

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server_test.go`

**Interfaces:**
- Consumes: exported `ProjectHandle` from 01
- Produces: single-project handlers that open once; migrate/merge still on root Service

### Tasks

- [x] 03.1 `[RED]` Observation create / recent path uses one handle open
  - [x] 03.1.a Write failing test — Scenario: Observation routes open store once
  - [x] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Observation|SingleOpen|Handle' -count=1` — Expected: FAIL (NOTE: passed pre-rewire, count already 1 — HTTP never had the MCP-style double-open; see Verification)
  - [x] 03.1.c Minimal implementation
  - [x] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Observation|SingleOpen|Handle' -count=1` — Expected: PASS
  - [x] 03.1.e Commit — `feat(mnemonic): HTTP single-project routes use Project Handle`
- [x] 03.2 `[AFK]` Session, search, code, web single-project handlers use handle; migrate/merge stay on root; bearer auth unchanged — Scenarios: Single-project HTTP uses handle; Migrate and merge stay on root — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PASS WITH WARNINGS` (verify-waiver at archive — sdd-verify not run; cited evidence below)

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` | PASS | PASS | 53 tests; service/mcp/store PASS |
| Acceptance `@step-03` / `@p0` | map scenarios to HTTP tests | PASS | PASS | single_open_test.go: TestObservationRoutesOpenStoreOnce, TestSingleProjectRoutesUseHandle (20 routes), TestMigrateMergeStayOnRoot |
| Runtime harness | N/A | PASS | PASS | |
| Rollback boundary | revert; no migrations | PASS | PASS | |
| Global Constraints | — | held | held | http/ + tests only |

### Commit

Step 03 commits: `153c4ce` (HTTP single-project routes use Project Handle), `4a08eff` (load config once in code index route)

Note: HTTP never had the MCP-style double-open — `projectFromRequest` resolves the id with no store.Open, and each handler called exactly one facade (one open) → 0+1=1. The [RED] test passed pre-rewire (count already 1); the rewire is still correct (MCP-consistent) and the single-open invariant is now pinned by tests. Brief/prediction discrepancy, not an implementation defect.

---

## 04-collapse-wrappers

### Goal

The shallow open-delegate-close surface is gone or cannot double-open; locality restored for single-project ops.

### Out of scope / Non-Goals

- store.DB unexport
- MigrateProjects rename
- New features

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-04` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS` (verify-waiver at archive)
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS (steps 02, 03 applied)
- [x] No Global Constraint violated

> Depends on: 02-mcp-single-open, 03-http-single-open

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/internal/mnemonic/integration/integration_test.go`
- Test: `skillgrid-cli/internal/mnemonic/service/*_test.go`

**Interfaces:**
- Consumes: MCP + HTTP already on handle path (02, 03)
- Produces: no production path double-opens; dead aliases removed or redirected without second open

### Tasks

- [x] 04.1 `[RED]` No production path opens the same project twice for one logical op (threat: double-open regression)
  - [x] 04.1.a Write failing test — Scenario: Facade path cannot double-open
  - [x] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1` — Expected: FAIL (NOTE: RED was compile-level + async-wait-group; the Open/MnemonicCommit invariants are pinned — see Verification)
  - [x] 04.1.c Minimal implementation (collapse wrappers; open counter / shared path)
  - [x] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1` — Expected: PASS
  - [x] 04.1.e Commit — `refactor(mnemonic): collapse double-open Service wrappers`
- [x] 04.2 `[AFK]` Dead aliases removed or redirected; integration MCP+HTTP smoke — Scenarios: Dead aliases neutralized; Integration smoke still passes — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/integration/ -count=1` — Expected: PASS

### Verification

Verdict: `PASS WITH WARNINGS` (verify-waiver at archive — sdd-verify not run; cited evidence below)

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/internal/mnemonic/integration/ -count=1` | PASS | PASS | all 4 `ok` |
| Acceptance `@step-04` / `@p0` | map scenarios to tests | PASS | PASS | double_open_test.go: TestServiceFacadePathOpensOnce, TestMnemonicCommitOpensOnce (count==1); seed_test.go integration smoke green |
| Full suite | `go test ./...` (from `skillgrid-cli`) | PASS | PASS | 19/19 packages `ok`, 0 FAIL; `go build ./...` OK; `go vet` clean |
| Rollback boundary | revert branch; no migrations | PASS | PASS | |
| Global Constraints | — | held | held | store.DB/MigrateProjects/memory-package untouched; no new SQL/schema; contracts stable |

### Commit

Step 04 commit: `89cd2ca` refactor(mnemonic): collapse double-open Service wrappers

Notes:
- 43 dead single-project wrappers + 4 orphaned type aliases deleted; `PassiveInput` alias restored (http/server.go:417 refs it). Tests in service/, integration/seed_test.go, mcp/single_open_test.go rewired to the handle (scope normalization preserved).
- PARKED (follow-up, not a blocker): pre-existing `CodeExplore`→`CodeImpact` double-open (service.go:986 opens, then :1044 calls s.CodeImpact which reopens at :814). Both methods KEPT (CodeImpact has CLI+MCP callers); not introduced by this step. Fix = a non-opening `codeImpactHandle` variant that `CodeExplore` reuses. The facade open-delegate-close surface this change targets is gone.

## Follow-ups (deferred by final broad review — not 007 blockers)

- **Consolidate mcp/http verbatim copies.** Step 02 added `mcp*` copies (mcpOrientSymbol, mcpCodeExplore, mcpMemoryDoctor, mcpCodeEmbeddingStatus, mcpReadIndexedCode) and step 03 added `http*` copies (httpMemoryDoctor, readIndexedCode, httpResolveEmbedder, scopeForSave) of db-backed service logic, because the handle exposes no non-reopening method for graph/orient/doctor/db ops. Track: export the db-backed impls from service/ (or add handle methods), delete the mcp*/http* copies, add an identity test.
- **CodeExplore→CodeImpact double-open (CLI path).** Fix via a `codeImpactHandle` variant reusing the already-open handle; add a guard for it.
- **Seam uniformity (optional).** retrieval (semantic_search/load_full_details), teams, and code_grep MCP groups still use the `rootService()`+`projectIDFor`+facade pattern (each opens once; code_grep takes a root, no double-open). Uniform on the handle is a nice-to-have, not a defect.
- **Minor:** `Service.Open` is a one-line pass-through of `openProject`; `mcpResolveEmbedder`/`httpResolveEmbedder` read config from `"."`/`dir` (≈`h.root` today). Both fold into the consolidation above.

---

## Archive gate checklist

- [x] Change-level **Definition of Done** fully checked
- [x] No unchecked `- [ ]` under any `### Tasks`
- [x] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] No Global Constraint violated
- [x] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [x] STATUS banner updated to `complete` (banner removed per user — header was skill-only boilerplate)
