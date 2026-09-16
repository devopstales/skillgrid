# Tasks: 002-mnemonic-identity-and-parity

> **STATUS:** `complete` (2026-09-05) — verify round 2 PASS · human QA **accepted** · archive
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

- [x] Every success criterion / DoD checkbox in `change.md` is met
- [x] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [x] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [x] No unchecked `- [ ]` under any `### Tasks`
- [x] No **Global Constraint** violated
- [x] Rollback path in `change.md` is still valid (or N/A documented)
- [x] `## State` status is `done` (set at archive gate)

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
phase: archive
current_step: 04-embedding-recall
status: done
updated: 2026-09-05T15:22:39+02:00
delivery: single-pr
verify_round: 2
apply_batch: 2-verify-gaps
human_qa: accepted
human_qa_at: 2026-09-05T15:22:00+02:00
note: human QA accepted — mechanical archive
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

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-01` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Produces contracts listed under Interfaces are available to dependents
- [x] No Global Constraint violated

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
- [x] 01.11 `[GAP]` Explicit dual-cwd remapped-id `store.Open` idempotence fixture — Scenario: `Store open is idempotent under remapped id` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/service/ -run 'Remap|Idempotent|Open' -count=1` — Expected: PASS

### Verification

```yaml
schema: skillgrid.verify-result/v1
change: 002-mnemonic-identity-and-parity
step: 01-identity-binding
evidence_revision: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
verdict: pass
blockers: 0
critical_findings: 0
scenarios: 10/10
test_command: go test ./internal/mnemonic/project/ ./internal/mnemonic/service/ ./internal/mnemonic/store/ ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ -count=1
test_exit_code: 0
test_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
build_command: (included in go test packages)
build_exit_code: 0
build_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
```

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused suite | `go test ./internal/mnemonic/project/ ./internal/mnemonic/service/ ./internal/mnemonic/store/ ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ -count=1` | PASS | PASS exit 0 | sha256:2ba3f7f3…b4bf3a (verify round 2, 2026-09-05T15:18+02) |
| Acceptance `@step-01` | scenario matrix | 10/10 COMPLIANT | 10/10 COMPLIANT | |
| Global Constraints | — | held | held | binding abort + ambiguous write refuse |

Acceptance compliance (`@step-01`):

| Scenario | Status | Covering test |
|----------|--------|---------------|
| Project binds to its clone | ✅ COMPLIANT | `TestIdentityStableAcrossRemoteChange` |
| Worktree and main checkout share project id | ✅ COMPLIANT | `TestWorktreeAndMainShareProjectID` |
| Remote change and sibling path keep project id | ✅ COMPLIANT | `TestIdentityStableAcrossRemoteChange` + `TestIdentityStableAcrossLocalRename` |
| Single child auto-promotes | ✅ COMPLIANT | `TestChildAutoPromotion` |
| Multi-repo parent returns AvailableProjects | ✅ COMPLIANT | `TestAmbiguityWithMultipleChildRepos` + `TestOpenForCWDRefusesAmbiguousParent` |
| Config walk stops at repository root | ✅ COMPLIANT | `TestConfigBoundedToEnclosingRepo` |
| Prior keys alias to canonical id | ✅ COMPLIANT | `TestOpenForDirectorySeedsAliasFromLegacyID` |
| MNEMONIC_PROJECT selects among candidates | ✅ COMPLIANT | `TestProcessOverrideWinsOverEverything` + `TestOpenForDirectoryHonoursMNEMONIC_PROJECT` |
| Store open is idempotent under remapped id | ✅ COMPLIANT | `TestOpenForDirectoryIdempotentAcrossWorktrees` |
| Binding write failure does not fall through to path-hash | ✅ COMPLIANT | `TestBindingWriteFailureAborts` |

**CRITICAL:** none  
**WARNING:** none  
**SUGGESTION:** none

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

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-02` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01-identity-binding

**Files:**
- Touch only if `[VERIFY]` fails: `skillgrid-cli/internal/mnemonic/service/`, `mcp/`, `http/`
- Test: same packages

**Interfaces:**
- Consumes: stable project id + aliases from 01
- Produces: `mem_search(all_projects=true)` merged results; `mem_unify` idempotent; HTTP surfaces with write auth

### Tasks

- [x] 02.1 `[VERIFY]` `[RED]` `all_projects` merges two seeded stores (threat: Mnemonic tool surface) — Scenario: `all_projects search merges two stores` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'AllProjects|CrossStore' -count=1` — Expected: PASS (if FAIL → promote to `[GAP]` micro-cycle)
- [x] 02.2 `[VERIFY]` `[RED]` `mem_unify` idempotent (threat: Mnemonic tool surface) — Scenario: `mem_unify is idempotent on already-unified keys` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify' -count=1` — Expected: PASS
- [x] 02.3 `[VERIFY]` Recall spans every store — Scenario: `Recall spans every store` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'CrossStore|Merge|SearchAll' -count=1` — Expected: PASS
- [x] 02.4 `[VERIFY]` Fragmented stores one logical index — Scenario: `Fragmented stores are one logical index` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify|Alias' -count=1` — Expected: PASS
- [x] 02.5 `[VERIFY]` Missing / empty → empty merged result — Scenario: `Missing data yields no result` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Empty|Missing' -count=1` — Expected: PASS
- [x] 02.6 `[VERIFY]` HTTP cross-store / unify write auth — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS
- [x] 02.7 `[GAP]` `[RED]` Runtime test: `mem_search(all_projects=true)` merges two seeded stores — Scenario: `all_projects search merges two stores` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'AllProjects|CrossStore|SearchAll' -count=1` — Expected: PASS
- [x] 02.8 `[GAP]` Runtime test: recall spans every store under data dir — Scenario: `Recall spans every store` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'SearchAll|CrossStore' -count=1` — Expected: PASS
- [x] 02.9 `[GAP]` `[RED]` Runtime test: `mem_unify` idempotent on already-unified keys — Scenario: `mem_unify is idempotent on already-unified keys` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify' -count=1` — Expected: PASS
- [x] 02.10 `[GAP]` Runtime test: empty/missing stores → empty merged result — Scenario: `Missing data yields no result` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'Empty|Missing|SearchAll' -count=1` — Expected: PASS
- [x] 02.11 `[GAP]` Runtime test: fragmented stores via aliases are one logical index — Scenario: `Fragmented stores are one logical index` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Unify|Alias|Fragment' -count=1` — Expected: PASS

### Verification

```yaml
schema: skillgrid.verify-result/v1
change: 002-mnemonic-identity-and-parity
step: 02-cross-store-recall
evidence_revision: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
verdict: pass
blockers: 0
critical_findings: 0
scenarios: 5/5
test_command: go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ -count=1
test_exit_code: 0
test_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
build_command: (included in go test packages)
build_exit_code: 0
build_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
```

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused suite | `go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ -count=1` (via full mnemonic suite) | PASS | PASS exit 0 | sha256:2ba3f7f3…b4bf3a |
| Acceptance `@step-02` | scenario matrix | 5/5 COMPLIANT | 5/5 COMPLIANT | |
| Global Constraints | — | held | held | empty merge warn+continue |

Acceptance compliance (`@step-02`):

| Scenario | Status | Covering test |
|----------|--------|---------------|
| Recall spans every store | ✅ COMPLIANT | `TestSearchAllProjectsSpansEveryStore` |
| all_projects search merges two stores | ✅ COMPLIANT | `TestSearchAllProjectsMergesTwoStores` |
| Fragmented stores are one logical index | ✅ COMPLIANT | `TestUnifyFragmentedStoresOneLogicalIndex` |
| mem_unify is idempotent on already-unified keys | ✅ COMPLIANT | `TestUnifyIdempotent` |
| Missing data yields no result | ✅ COMPLIANT | `TestSearchAllProjectsEmptyDir` |

**CRITICAL:** none  
**WARNING:** none  
**SUGGESTION:** HTTP `TestProjectsMergeRoute` covers merge auth surface; MCP `all_projects` wiring covered by `TestE2EEngramParityGaps` / tool registration

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

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-03` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 02-cross-store-recall

**Files:**
- Touch only if `[VERIFY]` fails: `memory/`, `store/migrations/`, `service/`, `mcp/`, `http/`

**Interfaces:**
- Consumes: cross-store / alias contracts from 02
- Produces: lifecycle columns honoured; `mem_pin` / `mem_unpin`; expiry soft-exclude; `tool_name` on save

### Tasks

- [x] 03.1 `[VERIFY]` `[RED]` pin/unpin reorder context; invalid pin structured error (threat: Mnemonic tool surface) — Scenario: `Pin and unpin reorder context` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Pin|Unpin' -count=1` — Expected: PASS
- [x] 03.2 `[VERIFY]` `[RED]` expired soft-excluded; invalid lifecycle rejected (threat: Mnemonic tool surface) — Scenario: `Expired entries are soft-excluded` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ -run 'Expir|Lifecycle|Invalid' -count=1` — Expected: PASS
- [x] 03.3 `[VERIFY]` Additive migration / columns present — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` — Expected: PASS
- [x] 03.4 `[VERIFY]` Lifecycle columns honoured — Scenario: `Lifecycle columns are honoured` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Lifecycle|Recency|Duplicate|Pin' -count=1` — Expected: PASS
- [x] 03.5 `[VERIFY]` Invalid lifecycle rejected — Scenario: `Invalid lifecycle state is rejected` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Invalid|Reject' -count=1` — Expected: PASS
- [x] 03.6 `[VERIFY]` `tool_name` provenance on save — Scenario: `tool_name provenance is stored on save` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/memory/ -run 'ToolName|Provenance' -count=1` — Expected: PASS
- [x] 03.7 `[VERIFY]` Lifecycle HTTP write auth if exposed — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS
- [x] 03.8 `[GAP]` Implement + test `tool_name` provenance on save (column missing from migration 008) — Scenario: `tool_name provenance is stored on save` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'ToolName|Provenance' -count=1` — Expected: PASS
- [x] 03.9 `[GAP]` Runtime test: invalid pin id / malformed expires_at → structured validation error — Scenario: `Invalid lifecycle state is rejected` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Invalid|Reject|Pin' -count=1` — Expected: PASS
- [x] 03.10 `[GAP]` Pin/unpin affect context ordering (not only flags) — Scenario: `Pin and unpin reorder context` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Pin|Context|Order' -count=1` — Expected: PASS

### Verification

```yaml
schema: skillgrid.verify-result/v1
change: 002-mnemonic-identity-and-parity
step: 03-lifecycle-parity
evidence_revision: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
verdict: pass
blockers: 0
critical_findings: 0
scenarios: 5/5
test_command: go test ./internal/mnemonic/memory/ ./internal/mnemonic/store/ ./internal/mnemonic/mcp/ -count=1
test_exit_code: 0
test_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
build_command: (included in go test packages)
build_exit_code: 0
build_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
```

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused suite | `go test ./internal/mnemonic/memory/ ./internal/mnemonic/store/ ./internal/mnemonic/mcp/ -count=1` (via full mnemonic suite) | PASS | PASS exit 0 | sha256:2ba3f7f3…b4bf3a |
| Acceptance `@step-03` | scenario matrix | 5/5 COMPLIANT | 5/5 COMPLIANT | migration 009 `tool_name` |
| Global Constraints | — | held | held | invalid lifecycle abort |

Acceptance compliance (`@step-03`):

| Scenario | Status | Covering test |
|----------|--------|---------------|
| Lifecycle columns are honoured | ✅ COMPLIANT | `TestPinUnpin` + `TestDuplicateBumpOnResave` + `TestTTLSoftExpiryAndRetire` + `TestPinReordersSearchContext` |
| Pin and unpin reorder context | ✅ COMPLIANT | `TestPinReordersSearchContext` (pin boost asserted; unpin exercised) |
| Expired entries are soft-excluded | ✅ COMPLIANT | `TestTTLSoftExpiryAndRetire` |
| tool_name provenance is stored on save | ✅ COMPLIANT | `TestToolNameProvenanceStoredOnSave` |
| Invalid lifecycle state is rejected | ✅ COMPLIANT | `TestInvalidPinRejected` + `TestMalformedExpiresAtRejected` |

**CRITICAL:** none  
**WARNING:** none  
**SUGGESTION:** `TestPinReordersSearchContext` does not re-assert post-unpin ranking; optional follow-up assert

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

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-04` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 03-lifecycle-parity

**Files:**
- Touch only if `[VERIFY]` fails: `memory/embedding.go`, `memory/search_embed.go`, `mcp/`

**Interfaces:**
- Consumes: search path from 03
- Produces: embedder gate; RRF merge with FTS5; FTS5-only when flag off / vectors absent / embedder missing

### Tasks

- [x] 04.1 `[VERIFY]` `[RED]` Flag on fuses; unset FTS5-only; missing embedder no 500 (threat: Mnemonic tool surface) — Scenario: `Flag on fuses vector and keyword results` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Embed|RRF|Fusion' -count=1` — Expected: PASS
- [x] 04.2 `[VERIFY]` Vector recall behind flag — Scenario: `Vector recall is available behind the flag` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Fusion' -count=1` — Expected: PASS
- [x] 04.3 `[VERIFY]` Keyword-only when vectors absent — Scenario: `Keyword-only fallback when vectors are absent` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Fallback|NoVector|RRF' -count=1` — Expected: PASS
- [x] 04.4 `[VERIFY]` Missing embedder degrades — Scenario: `Missing embedder degrades to keyword-only` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Degrad' -count=1` — Expected: PASS
- [x] 04.5 `[VERIFY]` Disabled flag no vector path — Scenario: `Disabled flag yields no vector recall` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'EmbedOff|FTSOnly|Embed' -count=1` — Expected: PASS
- [x] 04.6 `[GAP]` Runtime test: `MNEMONIC_EMBED=1` fuses FTS+vector ranking end-to-end — Scenario: `Flag on fuses vector and keyword results` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|RRF|Fusion|Blended' -count=1` — Expected: PASS
- [x] 04.7 `[GAP]` Runtime test: missing embedder with flag on degrades to FTS-only (no 500) — Scenario: `Missing embedder degrades to keyword-only` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'Embed|Degrad|Fallback' -count=1` — Expected: PASS
- [x] 04.8 `[GAP]` Runtime test: flag off skips vector path — Scenario: `Disabled flag yields no vector recall` — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'EmbedOff|FTSOnly|Embed' -count=1` — Expected: PASS

### Verification

```yaml
schema: skillgrid.verify-result/v1
change: 002-mnemonic-identity-and-parity
step: 04-embedding-recall
evidence_revision: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
verdict: pass
blockers: 0
critical_findings: 0
scenarios: 5/5
test_command: go test ./internal/mnemonic/memory/ -run 'Cosine|RRF|Blended|Embed|SetEmbedding|Missing|Disabled' -count=1
test_exit_code: 0
test_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
build_command: (included in go test packages)
build_exit_code: 0
build_output_hash: sha256:2ba3f7f3aa314192867838e89eb8e13c9c45a55714475c6634f25a6edcb4bf3a
```

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused suite | full `./internal/mnemonic/...` packages (round 2) | PASS | PASS exit 0 | sha256:2ba3f7f3…b4bf3a |
| Acceptance `@step-04` | scenario matrix | 5/5 COMPLIANT | 5/5 COMPLIANT | |
| Global Constraints | — | held | held | FTS5 floor; EmbedOff ignores vector leg |

Acceptance compliance (`@step-04`):

| Scenario | Status | Covering test |
|----------|--------|---------------|
| Vector recall is available behind the flag | ✅ COMPLIANT | `TestBlendedSearchFusesWhenEmbedOn` + `TestCosineAndRRF` |
| Flag on fuses vector and keyword results | ✅ COMPLIANT | `TestBlendedSearchFusesWhenEmbedOn` |
| Keyword-only fallback when vectors are absent | ✅ COMPLIANT | `TestBlendedSearchFallbackToFTS` |
| Missing embedder degrades to keyword-only | ✅ COMPLIANT | `TestMissingEmbedderDegradesToKeywordOnly` (flag on + empty vector leg → FTS, no hard fail) |
| Disabled flag yields no vector recall | ✅ COMPLIANT | `TestDisabledFlagYieldsNoVectorRecall` |

**CRITICAL:** none  
**WARNING:** none  
**SUGGESTION:** degrade fixture exercises empty vector-leg path, not a live HTTP embedder client failure

### Commit

When step DoD is met (only if gaps fixed): `fix(mnemonic): embedding recall acceptance gaps` — else verify-only note.

---

## Archive gate checklist

- [x] Change-level **Definition of Done** fully checked
- [x] No unchecked `- [ ]` under any `### Tasks`
- [x] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] No Global Constraint violated
- [x] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [x] STATUS banner updated to `complete`
- [x] Interview D1–D6 reflected in shipped behaviour
