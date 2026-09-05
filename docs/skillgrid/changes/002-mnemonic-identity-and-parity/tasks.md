# Tasks: 002-mnemonic-identity-and-parity

> **STATUS:** `in-progress` (2026-09-05) — 0/4 steps PASS · **gap-close revise** (interview D1–D6)
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax. **Do not rebuild shipped behaviour** — implement `[GAP]` deltas; `[VERIFY]` only needs PASS evidence.

**Goal:** Give Mnemonic a stable, clone-private project identity and Engram-parity recall (cross-store, lifecycle, optional embeddings) so memories stop scattering across invisible stores — closing remaining abort/write gaps and proving acceptance.

**Architecture:** Gap-close on existing `skillgrid-cli/internal/mnemonic` identity + recall stack. Harden binding-write abort and ambiguous-parent write abort; verify shipped cross-store / lifecycle / embedding paths. See `change.md` + `interview.md` D1–D6.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), HTTP mux with bearer-token auth on writes; optional embedder behind `MNEMONIC_EMBED`.

**Spec:** `docs/skillgrid/changes/002-mnemonic-identity-and-parity/change.md`

**Acceptance:** `docs/skillgrid/changes/002-mnemonic-identity-and-parity/acceptance.feature` (`@step-NN`)

**Interview:** `docs/skillgrid/changes/002-mnemonic-identity-and-parity/interview.md`

---

## Goal

Agents and any Mnemonic consumer get a stable, repo-bound project identity and Engram-parity recall so memories survive move/rename/remote-change; ambiguous parents and binding failures never invent unstable write targets; shipped parity features are acceptance-proven.

## Out of scope / Non-Goals

- Cloud sync (Engram `sync_*`) — keep Mnemonic local-first
- Rewriting the code index or web research cache
- Changing FTS5 to another search engine
- Expanding surface area beyond Step Blueprint capabilities 01–04
- Greenfield re-implementation of already-shipped tools/columns

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

- No cloud sync (`sync_*`); Mnemonic stays local-first
- Do not rewrite the code index or web research cache
- Do not replace FTS5 with another search engine
- Do not expand surface area beyond Step Blueprint capabilities 01–04
- All new SQL is additive migrations; no destructive rewrites of existing schema
- MCP tools follow existing `s.AddTool(toolDef, handlerFunc)` pattern
- HTTP routes follow existing `mux.HandleFunc` pattern with bearer-token auth on writes
- Project id is the single stable key; `store.Open` must remain idempotent when two cwds map to one id
- Optional embedding recall is off by default (`MNEMONIC_EMBED`); FTS5 remains the floor
- Ambiguous parent cwd (>1 child repo) → `abort` writes/store-open (`ErrAmbiguousProject` + `AvailableProjects`); never open/create under directory-hash fallback
- Identity binding write fails → `abort` with clear error; do not fall through to seed-without-binding
- Store open under remapped id → `warn+continue` if alias seed needed; idempotent open; silent SeedID→canonical merge on bind remains
- Cross-store search with empty / missing stores → `warn+continue`; empty merged result, not hard failure
- Invalid lifecycle state → `abort` with validation error
- Embedder unavailable while `MNEMONIC_EMBED=1` → `warn+continue`; degrade to FTS5-only
- HTTP write without bearer token → `abort` (401/403)
- Apply is **gap-close**: do not rewrite working shipped paths unless a `[GAP]` or failing `[VERIFY]` requires it

---

## State

```yaml
phase: apply         # spec | apply | verify | archive
current_step: 01-identity-binding
status: in_progress  # in_progress | blocked | done
updated: 2026-09-05T14:20:00+02:00
delivery: single-pr  # resolved ask-on-risk for step-01 gap batch
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
| Estimated changed lines (change) | ~250–400 (gaps) + verify |
| 400-line budget risk | Medium |
| Chained PRs recommended | Ask if step-01 gap PR grows |
| Delivery strategy | ask-on-risk |

---

## 01-identity-binding

### Goal

Close identity abort gaps and prove clone-private binding, ambiguity, bounded config, and alias/merge behaviour against `@step-01`.

### Out of scope / Non-Goals

- Cross-store merge UX beyond SeedID auto-merge (step 02)
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
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (OpenForCWD / resolve write paths as needed)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/` (only if write-path abort surfacing needs it)
- Modify: `docs/skillgrid/glossary/business.md` / `technical.md` (if terms drift)
- Test: `skillgrid-cli/internal/mnemonic/project/resolve_test.go`

**Interfaces:**
- Consumes: none
- Produces: stable project id via clone-private binding; `AvailableProjects` / `ErrAmbiguousProject`; binding-write abort; no store under ambiguous fallback; SeedID + silent merge; idempotent `store.Open` under remapped ids

### Tasks

- [x] 01.1 `[GAP]` `[RED]` Binding write failure aborts (no seed-without-binding) (threat: git repository selection) — Scenario: `Binding write failure does not fall through to path-hash`
  - [x] 01.1.a Write failing test (permission / unwritable common-dir)
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Binding|WriteFail|Permission' -count=1` — Expected: FAIL
  - [x] 01.1.c Minimal implementation — remove soft seed fallback on `writeBinding` error
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Binding|WriteFail|Permission' -count=1` — Expected: PASS
  - [x] 01.1.e Commit — `fix(mnemonic): abort when identity binding cannot be written`
- [x] 01.2 `[GAP]` `[RED]` Ambiguous parent never opens/creates a store under directory-hash fallback (threat: git repository selection) — Scenario: `Multi-repo parent returns AvailableProjects`
  - [x] 01.2.a Write failing test (OpenForCWD / write path from multi-repo parent)
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Ambiguous|AvailableProjects|OpenForCWD' -count=1` — Expected: FAIL
  - [x] 01.2.c Minimal implementation — hard-abort writes; require `MNEMONIC_PROJECT` / explicit `project=`
  - [x] 01.2.d Run to confirm pass — same Run — Expected: PASS
  - [x] 01.2.e Commit — `fix(mnemonic): refuse store open under ambiguous directory-hash fallback`
- [x] 01.3 `[VERIFY]` Worktree and main checkout share project id — Scenario: `Worktree and main checkout share project id` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Worktree|CommonDir|Identity' -count=1` — Expected: PASS
- [x] 01.4 `[VERIFY]` Remote-change and sibling path keep project id — Scenario: `Remote change and sibling path keep project id` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Remote|Sibling|Stable' -count=1` — Expected: PASS
- [x] 01.5 `[VERIFY]` Single child auto-promotes; multi-repo returns ambiguity — Scenario: `Single child auto-promotes` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'AutoPromote|Ambiguous' -count=1` — Expected: PASS
- [x] 01.6 `[VERIFY]` Config walk stops at repository root — Scenario: `Config walk stops at repository root` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Config|Bound' -count=1` — Expected: PASS
- [x] 01.7 `[VERIFY]` Prior keys alias / SeedID silent merge to canonical — Scenario: `Prior keys alias to canonical id` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/project/ -run 'Alias|Seed|Merge' -count=1` — Expected: PASS
- [x] 01.8 `[VERIFY]` `MNEMONIC_PROJECT` selects among candidates — Scenario: `MNEMONIC_PROJECT selects among candidates` — `Run: go test ./skillgrid-cli/internal/mnemonic/project/ -run 'Override|MNEMONIC_PROJECT' -count=1` — Expected: PASS
- [x] 01.9 `[VERIFY]` `store.Open` idempotent under remapped id — Scenario: `Store open is idempotent under remapped id` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Open|Idempotent' -count=1` — Expected: PASS
- [x] 01.10 `[AFK]` Glossary terms for identity / ambiguity abort semantics if drifted — `Run: true` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/project/ ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | PASS (apply) | Binding abort + ambiguous open refuse + worktree + SeedID alias |
| Acceptance `@step-01` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/project/ ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | PASS (apply) | Gaps 01.1–01.2 + VERIFY 01.3–01.9 |
| Global Constraints | — | held | held (apply) | Verdict owned by sdd-verify |

### Commit

When step DoD is met: `fix(mnemonic): harden identity abort paths for binding and ambiguity`

---

## 02-cross-store-recall

### Goal

Prove cross-store recall and alias unification; fix only acceptance/RED gaps.

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
- Touch only if `[VERIFY]` fails: `skillgrid-cli/internal/mnemonic/service/`, `mcp/`, `http/`
- Test: same packages

**Interfaces:**
- Consumes: stable project id + aliases from 01
- Produces: `mem_search(all_projects=true)` merged results; `mem_unify` idempotent; HTTP surfaces with write auth

### Tasks

- [ ] 02.1 `[VERIFY]` `[RED]` `all_projects` merges two seeded stores (threat: Mnemonic tool surface) — Scenario: `all_projects search merges two stores` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'AllProjects|CrossStore' -count=1` — Expected: PASS (if FAIL → promote to `[GAP]` micro-cycle)
- [ ] 02.2 `[VERIFY]` `[RED]` `mem_unify` idempotent (threat: Mnemonic tool surface) — Scenario: `mem_unify is idempotent on already-unified keys` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify' -count=1` — Expected: PASS
- [ ] 02.3 `[VERIFY]` Recall spans every store — Scenario: `Recall spans every store` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'CrossStore|Merge|SearchAll' -count=1` — Expected: PASS
- [ ] 02.4 `[VERIFY]` Fragmented stores one logical index — Scenario: `Fragmented stores are one logical index` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify|Alias' -count=1` — Expected: PASS
- [ ] 02.5 `[VERIFY]` Missing / empty → empty merged result — Scenario: `Missing data yields no result` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Empty|Missing' -count=1` — Expected: PASS
- [ ] 02.6 `[VERIFY]` HTTP cross-store / unify write auth — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | same | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met (only if gaps fixed): `fix(mnemonic): cross-store recall acceptance gaps` — else no commit; note verify-only in Verification.

---

## 03-lifecycle-parity

### Goal

Prove observation lifecycle parity; fix only acceptance/RED gaps.

### Out of scope / Non-Goals

- Embedding generation / RRF behaviour (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-cross-store-recall

**Files:**
- Touch only if `[VERIFY]` fails: `memory/`, `store/migrations/`, `service/`, `mcp/`, `http/`

**Interfaces:**
- Consumes: cross-store / alias contracts from 02
- Produces: lifecycle columns honoured; `mem_pin` / `mem_unpin`; expiry soft-exclude; `tool_name` on save

### Tasks

- [ ] 03.1 `[VERIFY]` `[RED]` pin/unpin reorder context; invalid pin structured error (threat: Mnemonic tool surface) — Scenario: `Pin and unpin reorder context` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Pin|Unpin' -count=1` — Expected: PASS
- [ ] 03.2 `[VERIFY]` `[RED]` expired soft-excluded; invalid lifecycle rejected (threat: Mnemonic tool surface) — Scenario: `Expired entries are soft-excluded` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ -run 'Expir|Lifecycle|Invalid' -count=1` — Expected: PASS
- [ ] 03.3 `[VERIFY]` Additive migration / columns present — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` — Expected: PASS
- [ ] 03.4 `[VERIFY]` Lifecycle columns honoured — Scenario: `Lifecycle columns are honoured` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Lifecycle|Recency|Duplicate|Pin' -count=1` — Expected: PASS
- [ ] 03.5 `[VERIFY]` Invalid lifecycle rejected — Scenario: `Invalid lifecycle state is rejected` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Invalid|Reject' -count=1` — Expected: PASS
- [ ] 03.6 `[VERIFY]` `tool_name` provenance on save — Scenario: `tool_name provenance is stored on save` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/memory/ -run 'ToolName|Provenance' -count=1` — Expected: PASS
- [ ] 03.7 `[VERIFY]` Lifecycle HTTP write auth if exposed — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/store/ -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met (only if gaps fixed): `fix(mnemonic): lifecycle parity acceptance gaps` — else verify-only note.

---

## 04-embedding-recall

### Goal

Prove optional embedding recall fused with FTS5; fix only acceptance/RED gaps.

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
- Touch only if `[VERIFY]` fails: `memory/embedding.go`, `memory/search_embed.go`, `mcp/`

**Interfaces:**
- Consumes: search path from 03
- Produces: embedder gate; RRF merge with FTS5; FTS5-only when flag off / vectors absent / embedder missing

### Tasks

- [ ] 04.1 `[VERIFY]` `[RED]` Flag on fuses; unset FTS5-only; missing embedder no 500 (threat: Mnemonic tool surface) — Scenario: `Flag on fuses vector and keyword results` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Embed|RRF|Fusion' -count=1` — Expected: PASS
- [ ] 04.2 `[VERIFY]` Vector recall behind flag — Scenario: `Vector recall is available behind the flag` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Fusion' -count=1` — Expected: PASS
- [ ] 04.3 `[VERIFY]` Keyword-only when vectors absent — Scenario: `Keyword-only fallback when vectors are absent` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Fallback|NoVector|RRF' -count=1` — Expected: PASS
- [ ] 04.4 `[VERIFY]` Missing embedder degrades — Scenario: `Missing embedder degrades to keyword-only` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Degrad' -count=1` — Expected: PASS
- [ ] 04.5 `[VERIFY]` Disabled flag no vector path — Scenario: `Disabled flag yields no vector recall` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'EmbedOff|FTSOnly|Embed' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | |
| Acceptance `@step-04` / `@p0` | same | PASS | | |
| Full suite | `go test ./...` (module root) | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met (only if gaps fixed): `fix(mnemonic): embedding recall acceptance gaps` — else verify-only note.

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
- [ ] Interview D1–D6 reflected in shipped behaviour
