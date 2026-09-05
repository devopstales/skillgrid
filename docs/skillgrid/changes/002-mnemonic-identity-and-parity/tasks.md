# Tasks: 002-mnemonic-identity-and-parity

> **STATUS:** `in-progress` (2026-09-04) — 0/4 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Give Mnemonic a stable, clone-private project identity and Engram-parity recall (cross-store, lifecycle, optional embeddings) so memories stop scattering across invisible stores.

**Architecture:** Project resolution in `skillgrid-cli/internal/mnemonic/project` is the sole identity seam; store open stays idempotent under the canonical id; service + memory own alias unification, lifecycle, and optional embedding fusion; MCP/HTTP adapters expose new contracts without changing `mem_save` shape. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), HTTP mux with bearer-token auth on writes; optional embedder behind `MNEMONIC_EMBED`.

**Spec:** `docs/skillgrid/changes/002-mnemonic-identity-and-parity/change.md`

**Acceptance:** `docs/skillgrid/changes/002-mnemonic-identity-and-parity/acceptance.feature` (`@step-NN`)

---

## Goal

Agents and any Mnemonic consumer get a stable, repo-bound project identity and Engram-parity recall (cross-store merge, pin/expiry/recency, optional vector fusion) so memories survive move/rename/remote-change and no longer strand across invisible SQLite files.

## Out of scope / Non-Goals

- Cloud sync (Engram `sync_*`) — keep Mnemonic local-first; revisit once identity is stable
- Rewriting the code index or web research cache (both already exceed Engram)
- Changing FTS5 to another search engine
- Expanding surface area beyond the four Step Blueprint capabilities

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

- No cloud sync (`sync_*`); Mnemonic stays local-first
- Do not rewrite the code index or web research cache
- Do not replace FTS5 with another search engine
- Do not expand surface area beyond Step Blueprint capabilities 01–04
- All new SQL is additive migrations; no destructive rewrites of existing schema
- MCP tools follow existing `s.AddTool(toolDef, handlerFunc)` pattern
- HTTP routes follow existing `mux.HandleFunc` pattern with bearer-token auth on writes
- Project id is the single stable key; `store.Open` must remain idempotent when two cwds map to one id
- Optional embedding recall is off by default (`MNEMONIC_EMBED`); FTS5 remains the floor
- Ambiguous parent cwd (>1 child repo) → `abort` (`ErrAmbiguousProject` + `AvailableProjects`); never invent a silent directory-hash write target
- Identity binding write fails (permissions on common-dir) → `abort` with clear error; do not fall through to unstable path-hash
- Store open under remapped id → `warn+continue` if alias seed needed; idempotent open; seed alias when prior store exists
- Cross-store search with empty / missing stores → `warn+continue`; return empty merged result, not hard failure
- Invalid lifecycle state (bad pin id, malformed `expires_at`) → `abort` with validation error
- Embedder unavailable while `MNEMONIC_EMBED=1` → `warn+continue`; degrade to FTS5-only
- HTTP write without bearer token → `abort` (401/403)

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-identity-binding
status: in_progress  # in_progress | blocked | done
updated: 2026-09-04T21:58:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `identity-binding` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `cross-store-recall` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `lifecycle-parity` | `@step-03` | 02 | Feature tagged `@step-03` |
| 04 | `embedding-recall` | `@step-04` | 03 | Feature tagged `@step-04` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

---

## 01-identity-binding

### Goal

Establish clone-private identity binding, child auto-promote, ambiguity, bounded config walk, and seed aliases so the same clone yields one stable project id across remote-change, sibling copy, and worktree.

### Out of scope / Non-Goals

- Cross-store merge UX (step 02)
- Lifecycle columns (step 03)
- Embeddings (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/project/resolve.go`
- Modify: `skillgrid-cli/internal/mnemonic/project/resolve_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/store/store.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `docs/skillgrid/agents/glossary/business.md`
- Modify: `docs/skillgrid/agents/glossary/technical.md`
- Test: `skillgrid-cli/internal/mnemonic/project/resolve_test.go`

**Interfaces:**
- Consumes: none
- Produces: stable project id via clone-private binding; `AvailableProjects` / `ErrAmbiguousProject`; seed aliases into `project_aliases`; idempotent `store.Open` under remapped ids

### Tasks

- [ ] 01.1 `[RED]` Linked worktree and main checkout share the same project id (threat: git repository selection) — Scenario: `Worktree and main checkout share project id`
  - [ ] 01.1.a Write failing test
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Worktree|CommonDir|Identity' -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Worktree|CommonDir|Identity' -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): bind project id via git common-dir`
- [ ] 01.2 `[RED]` Remote-change and absolute sibling path keep the same project id (threat: git repository selection) — Scenario: `Remote change and sibling path keep project id`
  - [ ] 01.2.a Write failing test
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Remote|Sibling|Stable' -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Remote|Sibling|Stable' -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): keep project id across remote and path change`
- [ ] 01.3 `[RED]` Multi-repo parent cwd returns ambiguity with AvailableProjects and does not write a directory-hash bucket (threat: git repository selection) — Scenario: `Multi-repo parent returns AvailableProjects`
  - [ ] 01.3.a Write failing test
  - [ ] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Ambiguous|AvailableProjects' -count=1` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation
  - [ ] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Ambiguous|AvailableProjects' -count=1` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(mnemonic): surface AvailableProjects on ambiguous parent cwd`
- [ ] 01.4 `[AFK]` Rewrite project resolution to bind once to the clone; never re-derive id from mutable git state after binding — Scenario: `Project binds to its clone` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -count=1` — Expected: PASS
- [ ] 01.5 `[AFK]` Exactly one child repo auto-promotes with soft warning; more than one returns ambiguity — Scenario: `Single child auto-promotes` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'AutoPromote|Ambiguous' -count=1` — Expected: PASS
- [ ] 01.6 `[AFK]` Bound config walk to enclosing repo root — Scenario: `Config walk stops at repository root` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Config|Bound' -count=1` — Expected: PASS
- [ ] 01.7 `[AFK]` Seed aliases so prior directory-hash / remote keys route to canonical id — Scenario: `Prior keys alias to canonical id` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/project/ -run 'Alias|Seed' -count=1` — Expected: PASS
- [ ] 01.8 `[AFK]` `MNEMONIC_PROJECT` selects among ambiguous candidates — Scenario: `MNEMONIC_PROJECT selects among candidates` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Override|MNEMONIC_PROJECT' -count=1` — Expected: PASS
- [ ] 01.9 `[AFK]` `store.Open` remains idempotent when two cwds map to one id — Scenario: `Store open is idempotent under remapped id` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Open|Idempotent' -count=1` — Expected: PASS
- [ ] 01.10 `[AFK]` Update glossary terms for identity / recall capabilities (business + technical) — `Run: true` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/project/ ./skillgrid-cli/internal/mnemonic/store/ -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/project/ -count=1` | PASS | | Scenarios: Project binds to its clone; Worktree and main checkout share project id; Multi-repo parent returns AvailableProjects |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/integration/ -count=1` | PASS | | if present |
| Rollback boundary | additive only — no destructive schema | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): clone-private identity binding with ambiguity and aliases`

---

## 02-cross-store-recall

### Goal

Deliver cross-store recall and alias unification so `mem_search(all_projects=true)` merges stores and `mem_unify` folds aliases idempotently.

### Out of scope / Non-Goals

- New lifecycle columns (step 03)
- Embedding fusion (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-identity-binding

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Test: `skillgrid-cli/internal/mnemonic/service/` ; `skillgrid-cli/internal/mnemonic/mcp/` ; `skillgrid-cli/internal/mnemonic/http/`

**Interfaces:**
- Consumes: stable project id + aliases from 01
- Produces: `mem_search(all_projects=true)` merged/re-ranked results; `mem_unify` admin path; matching HTTP surfaces with write auth

### Tasks

- [ ] 02.1 `[RED]` `mem_search` with `all_projects=true` merges two seeded stores (threat: Mnemonic tool surface) — Scenario: `all_projects search merges two stores`
  - [ ] 02.1.a Write failing test
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'AllProjects|CrossStore' -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'AllProjects|CrossStore' -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): merge mem_search across all projects`
- [ ] 02.2 `[RED]` `mem_unify` is idempotent and records alias without 500 on already-unified keys (threat: Mnemonic tool surface) — Scenario: `mem_unify is idempotent on already-unified keys`
  - [ ] 02.2.a Write failing test
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify' -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify' -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): idempotent mem_unify for aliases`
- [ ] 02.3 `[AFK]` Support recalling across every store under the store dir, merged and re-ranked — Scenario: `Recall spans every store` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'CrossStore|Merge' -count=1` — Expected: PASS
- [ ] 02.4 `[AFK]` Unify aliases so fragmented stores become one logical index — Scenario: `Fragmented stores are one logical index` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify|Alias' -count=1` — Expected: PASS
- [ ] 02.5 `[AFK]` Missing / empty stores yield empty merged result (not error storm) — Scenario: `Missing data yields no result` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Empty|Missing' -count=1` — Expected: PASS
- [ ] 02.6 `[AFK]` Expose HTTP surfaces for cross-store recall / unify with write auth — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | Scenarios: Recall spans every store; all_projects search merges two stores |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/integration/ -count=1` | PASS | | if present |
| Rollback boundary | remove new tools/routes; additive only | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): cross-store recall and alias unification`

---

## 03-lifecycle-parity

### Goal

Deliver observation lifecycle parity so pinning, expiry, duplicate/recency, and tool provenance are honoured in context/search/review.

### Out of scope / Non-Goals

- Embedding generation / RRF behaviour (step 04; columns may land in shared migration)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-cross-store-recall

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/008_obs_lifecycle.sql`
- Modify: `skillgrid-cli/internal/mnemonic/memory/lifecycle.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/` ; `skillgrid-cli/internal/mnemonic/service/` ; `skillgrid-cli/internal/mnemonic/mcp/`

**Interfaces:**
- Consumes: cross-store / alias contracts from 02
- Produces: additive lifecycle columns; `mem_pin` / `mem_unpin`; expiry soft-exclude; `tool_name` on save; review honouring `expires_at`

### Tasks

- [ ] 03.1 `[RED]` `mem_pin` / `mem_unpin` reorder context; invalid pin id returns structured error not 500 (threat: Mnemonic tool surface) — Scenario: `Pin and unpin reorder context`
  - [ ] 03.1.a Write failing test
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Pin|Unpin' -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Pin|Unpin' -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): mem_pin and mem_unpin lifecycle tools`
- [ ] 03.2 `[RED]` Expired `expires_at` excluded from live hits; invalid lifecycle state rejected (threat: Mnemonic tool surface) — Scenario: `Expired entries are soft-excluded`
  - [ ] 03.2.a Write failing test
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ -run 'Expir|Lifecycle' -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ -run 'Expir|Lifecycle' -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): honour expires_at and reject invalid lifecycle`
- [ ] 03.3 `[AFK]` Additive migration for lifecycle columns (`pinned`, `expires_at`, `duplicate_count`, `last_seen_at`, `tool_name`) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` — Expected: PASS
- [ ] 03.4 `[AFK]` Honour pinning, expiry, duplicate count, and recency in ordering/exclusion — Scenario: `Lifecycle columns are honoured` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Lifecycle|Recency|Duplicate' -count=1` — Expected: PASS
- [ ] 03.5 `[AFK]` Invalid lifecycle state is rejected with validation error — Scenario: `Invalid lifecycle state is rejected` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Invalid|Reject' -count=1` — Expected: PASS
- [ ] 03.6 `[AFK]` Store `tool_name` provenance on save when provided — Scenario: `tool_name provenance is stored on save` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/memory/ -run 'ToolName|Provenance' -count=1` — Expected: PASS
- [ ] 03.7 `[AFK]` Lifecycle-related HTTP routes (if exposed) keep write auth — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | Scenarios: Lifecycle columns are honoured; Pin and unpin reorder context |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/integration/ -count=1` | PASS | | if present |
| Rollback boundary | additive columns harmless if unused | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): observation lifecycle parity`

---

## 04-embedding-recall

### Goal

Deliver optional embedding recall fused with FTS5 via reciprocal-rank fusion behind `MNEMONIC_EMBED`, with FTS5-only behaviour when the flag is off or vectors are absent.

### Out of scope / Non-Goals

- Changing FTS5 engine
- Requiring a cloud embedder

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 03-lifecycle-parity

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/embedding.go`
- Modify: `skillgrid-cli/internal/mnemonic/memory/search_embed.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/` ; `skillgrid-cli/internal/mnemonic/mcp/`

**Interfaces:**
- Consumes: lifecycle / search path from 03
- Produces: embedder gate; cosine + RRF merge with FTS5; `mem_search` fusion when embeddings present

### Tasks

- [ ] 04.1 `[RED]` `MNEMONIC_EMBED=1` with embeddings returns fused ranking; flag unset returns FTS5-only shape; missing embedder does not 500 (threat: Mnemonic tool surface) — Scenario: `Flag on fuses vector and keyword results`
  - [ ] 04.1.a Write failing test
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Embed|RRF|Fusion' -count=1` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Embed|RRF|Fusion' -count=1` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(mnemonic): optional embedding recall with RRF fusion`
- [ ] 04.2 `[AFK]` Vector recall available behind flag, fused with keyword via reciprocal-rank fusion — Scenario: `Vector recall is available behind the flag` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Fusion' -count=1` — Expected: PASS
- [ ] 04.3 `[AFK]` Keyword-only fallback when vectors absent even if flag is on — Scenario: `Keyword-only fallback when vectors are absent` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Fallback|NoVector' -count=1` — Expected: PASS
- [ ] 04.4 `[AFK]` Disabled flag yields FTS5-only path (no vector recall) — Scenario: `Disabled flag yields no vector recall` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'EmbedOff|FTSOnly' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | |
| Acceptance `@step-04` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` | PASS | | Scenarios: Vector recall is available behind the flag; Flag on fuses vector and keyword results |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/... -count=1` | PASS | | |
| Full suite | `go test ./...` | PASS | | from module root |
| Rollback boundary | leave `MNEMONIC_EMBED` unset | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): optional embedding recall with RRF`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
