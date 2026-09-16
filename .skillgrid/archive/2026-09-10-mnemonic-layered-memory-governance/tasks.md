# Tasks: 013-mnemonic-layered-memory-governance

> **STATUS:** `in-progress` — 0/3 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Upgrade Mnemonic's flat observation store into a layered (L0→L3), governed, and budgeted memory — with owner/version/status/usage/visibility governance (private-by-default), a provenance-linked session-close distillation pass (LLM-cached + no-LLM floor), and layered L2/L3-first retrieval with item+char+timeout budgets — while keeping existing `mem_*` tool names + required params stable.

**Architecture:** Three additive layers on the 005 observation store, one vertical slice each (change.md "Technical approach"): step 01 = governance columns/tables + `mem_share`/`mem_governance` on the additive `015_*` schema; step 02 = `observation_layers` + `personas` + the session-close distillation pass + `mem_layers`; step 03 = layered retrieval + a budget wrapper at the `mem_*` read seam. Additive, CGo-free, LLM-optional; 005's flat store stays intact on rollback.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`, CGo-free), existing `internal/mnemonic/memory` + `store` (005), optional LLM for distillation (cached by content-hash; 005 `CapturePassive` heuristics as the deterministic no-LLM floor), MCP (`mcp-go`), CLI.

**Spec:** `docs/skillgrid/changes/013-mnemonic-layered-memory-governance/change.md`

**Acceptance:** `docs/skillgrid/changes/013-mnemonic-layered-memory-governance/acceptance.feature` (`@step-NN`)

---

## Goal

Mnemonic's memory stops being a flat pile: a session's work **distills upward** (raw record → atoms → scenario → persona) so the next session bootstraps from stable L2/L3 and only falls back to L1/L0 for a fact; every read is **budgeted** (item + char + timeout) so a 20-hit search can't drown the context window; and every observation becomes a **governed asset** (owner, version history, status, retrieval usage, visibility) that is **private by default** and shared only by an explicit action. Trace: change.md `## Definition of Done`.

## Out of scope / Non-Goals

- Code-index / graph work — 005/008/010/011 own the code graph; this change touches the **memory** store and `mem_*` surface only
- Multi-tenant **teams + role layers** (System/Team Admin, Member) and the LLM **proxy** — single-operator fields only; no multi-tenant auth, no base-URL LLM proxy
- **Agent loadout** as a first-class binding engine — modeled only as `agent` visibility + a per-agent view (009 renders it); no Fixed-Binding / priority engine
- Replacing the separate L0/L1/L2 **file-tiering** side system (`tiered_contents`) — that tiers markdown docs; 013 tiers **observations**; they coexist
- New MCP tools beyond the additive `mem_*` fields + the new `mem_layers` / `mem_governance` / `mem_share` tools
- Skills-as-governed-assets (versions/ACL/registry) — 004 / the skill registry; not this change

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] A **distillation pass** refines a session's L0 record into L1 atoms + L2 scenario(s) + an L3 persona delta, each **linked** to its source (provenance), LLM-cached by content-hash with a deterministic no-LLM floor
- [ ] Retrieval is **layered**: `mem_context` / `mem_search` bootstrap from L2/L3 and fall back to L1/L0 (RRF) only for a specific fact; a `mem_layers` tool inspects the L0→L3 chain for a session/topic
- [ ] **Every `mem_*` read path is budgeted**: an item-count cap, a character budget (snippets truncated, full content only via an explicit `mem_get_observation`), and a context timeout — a 20-hit search never returns 20× untruncated full-observation JSON
- [ ] Every observation carries **owner**, **version history** (`mem_update` appends a version; prior content recoverable), **status** (`active`/`superseded`/`archived`), and a **retrieval usage count** (distinct from the existing `duplicate_count`)
- [ ] Observations have **visibility** `private`/`team`/`restricted`/`agent` with **private-by-default**; `restricted` grants via a User/Role/Agent ACL; a `private` observation is invisible to other owners even to an admin's read list
- [ ] Sharing is an **explicit action** (`mem_share`); the `mem_*` tools surface owner/version/status/usage/visibility
- [ ] Existing `mem_*` tool names + required params are unchanged (new fields + tools are additive); `go test ./...` passes for touched packages
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] Applicable threat-matrix rows (Mnemonic tool surface; Data leak/visibility; Context-window flood; Provenance integrity) have passing RED coverage
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

- **Private-by-default:** a new observation is `private` until an explicit `mem_share`; `private` is invisible to *other owners* (the creating owner/agent always sees its own). Sharing is an explicit mutation, never a default leak.
- **Versioning is append, not overwrite:** `mem_update` appends a row to `observation_versions` (prior content recoverable via `mem_governance`); the latest version is the read path; `revision_count` advances; a `superseded`/`archived` status is set explicitly, never inferred.
- **Layering is provenance-linked:** every L1/L2/L3 record links to a resolvable L0 source (session/topic); a distilled record with no resolvable source is **not created**; a session with no new L1-able content is a no-op (no empty layers fabricated).
- **Distillation is LLM-optional with a no-LLM floor:** the LLM pass is cached by content-hash (re-distill only on change); with no LLM, the deterministic extractor (005 `CapturePassive` Key-Learnings/Lesson/Discovery heuristics) still produces L1 atoms so the ladder works offline.
- **Budgeted reads:** every `mem_*` read path enforces an item-count cap, a character budget (in-list snippets truncated with an explicit "N chars omitted"), and a context timeout (returns a `truncated: true` partial with a reason, never hangs). The budget is tunable (config).
- **Full content only via `mem_get_observation`:** in-list results are truncated; full, untruncated content is available only via the explicit `mem_get_observation`; every in-list result carries its `mem_get_observation` id.
- **CGo-free:** SQLite via `modernc.org/sqlite`; no `mattn/go-sqlite3`.
- **Existing `mem_*` names + required params unchanged:** all new tools use distinct `mem_*` names; new response fields are additive; 005 tool contracts do not regress.
- **Additive schema:** never rewrite existing `observations` columns; new fields + tables are additive in `015_layered_memory_governance.sql` (leave 011/012/013/014 as-is).
- **No CGo, LLM-optional, single-operator:** no multi-tenant teams/roles, no LLM proxy; `team` ≈ the project's visible set.

---

## State

```yaml
phase: archive          # spec | apply | verify | archive
current_step: 03-layered-retrieval-budgets
status: done  # in_progress | blocked | done
updated: 2026-09-10
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `governance-fields` | `@step-01` | — (005 done) | Feature tagged `@step-01` |
| 02 | `layered-distill` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `layered-retrieval-budgets` | `@step-03` | 02 | Feature tagged `@step-03` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1600 (schema + governance + layer + budget/retrieve + mcp + service wiring + tests) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | auto-chain |
<!-- single-pr = one PR for the change; each ## NN step still commits separately when DoD is met (see work-unit-commits / commits.md). Each step is independently rollable via the additive 015_* boundary. -->

---

## 01-governance-fields

### Goal

Every memory is a governed asset — owner, version history (append), status, retrieval usage, visibility (private-by-default) — shared only by an explicit `mem_share`, surfaced by `mem_governance`, with existing `mem_*` tools unchanged.

### Out of scope / Non-Goals

- Layering / `observation_layers` / `personas` / `mem_layers` (step 02)
- Retrieval budgets / layered retrieval (step 03)
- Multi-tenant teams/roles/proxy; changing `mem_*` tool names/required params

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/015_layered_memory_governance.sql`
- Create: `skillgrid-cli/internal/mnemonic/memory/governance.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/governance_test.go`, `skillgrid-cli/internal/mnemonic/store/migrations_test.go`, `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go`, `skillgrid-cli/internal/mnemonic/service/governance_test.go`

**Interfaces:**
- Consumes: 005 observation store + `mem_save`/`mem_update`/`mem_search`/`mem_get_observation` (names + required params unchanged)
- Produces: `015_*` governance columns (owner, status, visibility, retrieval_usage) + `observation_versions` + `acl_grants`; `mem_share <id> <team|restricted|agent> [acl]`; `mem_governance <id>`; additive governance fields on existing `mem_*` responses — relied on by steps 02/03

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Data leak/visibility), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-01. -->

- [x] 01.1 `[RED]` Threat "Mnemonic tool surface" — governance tools registered + private-by-default + append-version + existing `mem_*` schema stable + bad args rejected (Scenarios: `governance-tools-and-existing-schema-stable`, `new-observation-private-by-default-with-owner`, `mem-update-appends-recoverable-version`, `bad-governance-args-rejected`)
  - [x] 01.1.a Write failing test
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run GovernanceTools -count=1` — Expected: FAIL
  - [x] 01.1.c Minimal implementation
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run GovernanceTools -count=1` — Expected: PASS
  - [x] 01.1.e Commit — `feat(memory): register mem_share/mem_governance + stable mem_* schema`
- [x] 01.2 `[RED]` Threat "Data leak / visibility" — private invisible to 2nd owner until share; restricted ACL enforced; no-grant restricted is owner-only; private absent from admin cross-owner list (Scenarios: `private-observation-invisible-to-second-owner-until-share`, `restricted-acl-grant-enforced`, `restricted-with-no-grants-is-owner-only`, `private-observation-absent-from-admin-cross-owner-list`)
  - [x] 01.2.a Write failing test
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Visibility -count=1` — Expected: FAIL
  - [x] 01.2.c Minimal implementation
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Visibility -count=1` — Expected: PASS
  - [x] 01.2.e Commit — `feat(memory): private-by-default visibility + restricted ACL enforcement`
- [x] 01.3 `[AFK]` Additive `015_*` governance schema (governance columns + `observation_versions` + `acl_grants`) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
- [x] 01.4 `[AFK]` `mem_save` private-by-default + owner; second owner's search excludes until shared (Scenarios: `new-observation-private-by-default-with-owner`, `private-observation-invisible-to-second-owner-until-share`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run PrivateDefault -count=1` — Expected: PASS
- [x] 01.5 `[AFK]` `mem_share <id> <team|restricted|agent> [acl]` is the only explicit widen; unknown owner/agent/role rejected, visibility unchanged (Scenario: `mem-share-unknown-target-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run MemShare -count=1` — Expected: PASS
- [x] 01.6 `[AFK]` `mem_update <id>` appends a row to `observation_versions`; prior content recoverable via `mem_governance`; latest version is the read path; `revision_count` advances (Scenario: `mem-update-appends-recoverable-version`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run UpdateVersion -count=1` — Expected: PASS
- [x] 01.7 `[AFK]` `superseded`/`archived` status set explicitly, never inferred (Scenario: `superseded-status-set-explicitly`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run StatusExplicit -count=1` — Expected: PASS
- [x] 01.8 `[AFK]` Search hit increments retrieval usage count, distinct from `duplicate_count` (Scenario: `retrieval-usage-count-increments-on-search`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run UsageCount -count=1` — Expected: PASS
- [x] 01.9 `[AFK]` `mem_governance <id>` returns owner/version history/status/usage/visibility (Scenario: `mem-governance-surfaces-asset-fields`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run GovernanceQuery -count=1` — Expected: PASS
- [x] 01.10 `[AFK]` Existing `mem_*` names + required params unchanged; additive fields only; bad governance args rejected clearly (Scenarios: `governance-tools-and-existing-schema-stable`, `bad-governance-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | PASS | governance core (visibility/ACL, append-version, status, usage) + 017 schema tests |
| Acceptance `@step-01` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` + BDD `@step-01 @p0` | PASS | PASS | GovernanceTools + Visibility (RED); end-to-end tool-handler wiring test (owner-B gated before share, visible after team-share) |
| Runtime harness | `go test ./skillgrid-cli/... -count=1` | PASS | PASS | full `./internal/mnemonic/...` (23 pkgs) green |
| Rollback boundary | drop `017_*` governance portion + `memory/governance.go` + `tools_memory_governance.go`; 005 store + `mem_*` intact | PASS | PASS | 017 purely additive (ALTER ADD COLUMN ×4 + CREATE TABLE IF NOT EXISTS); legacy store upgrades additively, rows kept + defaulted private/active |
| Global Constraints | — | held | held | private-by-default, append-version (latest=read path), status explicit, additive surface (24-tool name+required-param lock 75→77), bad-args rejected, per-owner read enforcement wired end-to-end |

Review: task reviewer `approved with fixes`. Fix commits c93b87d + 8563453 (the per-owner read-enforcement seam `canRead`/`ReadAs`/`SearchOwner` was tested but UNWIRED to the live tools — now `mem_search`→`SearchOwnerScoped`, `mem_get_observation`→`ReadAs` via optional `reader_owner`/`reader_agent`, additive not-found-for-reader on gate; + empty-owner `ErrVisibilityNotSet` consistency with `visibilityFilter`). Scoped re-review: end-to-end test `TestMemSearchGetOwnerEnforcedWiring` drives the real handlers (owner B excluded before share via both search+get, included after team-share) — non-vacuous.

Commits (step 01): 28467ec (017 schema), 7caa2b7 (governance core), dfbea4e (mem_share/mem_governance tools, 75→77), c93b87d + 8563453 (wire read enforcement + empty-owner fix). Migration `017_layered_memory_governance.sql` (brief's 015 was taken by 008).

### Commit

When step DoD is met: `feat(memory): governance fields (owner/version/status/usage/visibility) + mem_share/mem_governance`

---

## 02-layered-distill

### Goal

A session's raw record distills upward (L0→L1 atoms→L2 scenario→L3 persona), provenance-linked, LLM-cached by content-hash with a deterministic no-LLM floor — so the next session bootstraps from stable layers; inspectable via `mem_layers`.

### Out of scope / Non-Goals

- Retrieval budgets / layered retrieval (step 03)
- The file-tiering side system (`tiered_contents`); multi-tenant; changing 005 tools

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-governance-fields

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/store/migrations/015_layered_memory_governance.sql`
- Create: `skillgrid-cli/internal/mnemonic/memory/layer/distill.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/layer/layers.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/layer/distill_test.go`, `skillgrid-cli/internal/mnemonic/memory/layer/layers_test.go`, `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go`, `skillgrid-cli/internal/mnemonic/service/distill_test.go`

**Interfaces:**
- Consumes: 01 governance (version-append for correctable L1 atoms; 015 schema seam); 005 session record + `CapturePassive` heuristics; 005 `mem_*` (stable)
- Produces: `observation_layers` (L0→L1→L2→L3 links + layer type) + `personas` (L3); the opt-in async session-close distillation hook; `mem_layers <session_id|topic_key>` — relied on by step 03 (L2/L3 bootstrap sources)

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Provenance integrity), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-02. -->

- [x] 02.1 `[RED]` Threat "Provenance integrity" — distilled L1/L2/L3 carries a resolvable L0 link; a record with no resolvable source is **not created**; `mem_layers` surfaces the chain; no-LLM floor produces a provenance-linked ladder offline (Scenarios: `distilled-layer-carries-resolvable-l0-provenance`, `layer-with-unresolvable-source-not-created`, `mem-layers-inspects-l0-to-l3-chain`, `no-llm-floor-produces-provenance-ladder-offline`)
  - [x] 02.1.a Write failing test
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run Provenance -count=1` — Expected: FAIL
  - [x] 02.1.c Minimal implementation
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run Provenance -count=1` — Expected: PASS
  - [x] 02.1.e Commit — `feat(memory/layer): provenance-linked distillation (no orphan layers)`
- [x] 02.2 `[RED]` Threat "Mnemonic tool surface" — `mem_layers` registered + 005 tools still stable + bad layer args rejected (Scenarios: `mem-layers-registered-005-stable`, `bad-layer-args-rejected`)
  - [x] 02.2.a Write failing test
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run LayerTools -count=1` — Expected: FAIL
  - [x] 02.2.c Minimal implementation
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run LayerTools -count=1` — Expected: PASS
  - [x] 02.2.e Commit — `feat(memory/layer): register mem_layers + keep 005 tools stable`
- [x] 02.3 `[AFK]` Additive `015_*` layer schema — `observation_layers` (L0→L1→L2→L3 links + layer type) + `personas` (L3) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
- [x] 02.4 `[AFK]` Session-close distillation (opt-in, async) refines L0 → L1 atoms + L2 scenario(s) + L3 persona delta, each linked to its L0 source (Scenario: `session-close-distill-l0-to-l1-l2-l3`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run DistillHook -count=1` — Expected: PASS
- [x] 02.5 `[AFK]` Distillation LLM-cached by content-hash; re-distill only on source change (Scenario: `distill-llm-cached-by-content-hash`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run CacheHash -count=1` — Expected: PASS
- [x] 02.6 `[AFK]` No-LLM deterministic floor (005 `CapturePassive` Key-Learnings/Lesson/Discovery heuristics) produces L1 atoms so the ladder works offline (Scenarios: `no-llm-floor-produces-provenance-ladder-offline`, `no-llm-floor-produces-l1-atoms`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run NoLLMFloor -count=1` — Expected: PASS
- [x] 02.7 `[AFK]` A session with no new L1-able content is a no-op (no empty atoms/scenarios/persona-delta fabricated) (Scenario: `session-with-no-l1-content-is-noop`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run NoOp -count=1` — Expected: PASS
- [x] 02.8 `[AFK]` L1 atoms are correctable, not just deletable — an atom update appends a version (step 01) and the provenance link traces it back to its L0 source (Scenario: `l1-atom-correctable-with-traceable-provenance`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run AtomCorrectable -count=1` — Expected: PASS
- [x] 02.9 `[AFK]` `mem_layers <session_id|topic_key>` returns the L0→L1→L2→L3 chain with each layer's provenance link (Scenario: `mem-layers-inspects-l0-to-l3-chain`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run MemLayers -count=1` — Expected: PASS
- [x] 02.10 `[AFK]` 005 `mem_*` tools unchanged; bad layer args rejected clearly (Scenarios: `mem-layers-registered-005-stable`, `bad-layer-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/layer/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | PASS | layer (provenance, cache-hash, no-LLM floor, no-op, chain) + store 018 ok; RED 02.1/02.2 captured |
| Acceptance `@step-02` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` + BDD `@step-02 @p0` | PASS | PASS | Provenance, LayerTools, DistillHook (incl. end-to-end session-close wiring), CacheHash, NoLLMFloor, NoOp, AtomCorrectable, MemLayers, BadLayerArgs all green |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/... -count=1` (+ `-race`) | PASS | PASS | all 22 mnemonic packages ok incl. `-race`; route/affected/community/pdg/codeindex baselines included |
| Rollback boundary | drop `018_*` layer portion + `memory/layer` package + `tools_memory_layer.go` + distill hook; 005 + step-01 governance intact | PASS | PASS | 017 untouched; 018 + layer pkg + tool + hook are the additive surface |
| Global Constraints | — | held | held | provenance-linked (no orphan layers / no-op on empty); no-LLM floor reuses 005 CapturePassive (exported wrappers); LLM cached by content-hash (both directions); 005 mem_* names+params unchanged (77→78); opt-in + async (detached goroutine) + best-effort (error swallowed, never a close failure) |

Review: task reviewer `approved with fixes`. Fix commit c86414c (the session-close distill hook was tested but UNWIRED — `DistillSession` had zero non-test callers, so production session close never distilled and the "async fire-and-forget" claim had no committed goroutine. Now `memory.SessionEnd` — the single choke point both `handleMemSessionEnd` and the HTTP handler go through — launches the opt-in hook in a detached `go` goroutine, armed on `*Service` by `EnableDistill` (read at close time), error swallowed into `DistillResult.Err` (never a close failure). Scoped re-review: end-to-end tests drive the real close handler — enabled → hook fires + layers + close succeeds; distill-error → close still succeeds (best-effort); disabled → no hook. `-race` clean.

Commits (step 02): 547c5d0 (018 schema), 66f9b6c (mem_layers + 77→78), 5ede761 (cache-hash/no-op/chain tests), 2829fce (session-close hook + mem_layers seam + atom-correctable), 7dc3441 (bad-args + 005-stable), a4c6202 (DistillResult.Err), c86414c (wire session-close distill into real close path). Migration `018_layered_distill.sql` (brief's 015 was taken by 008; 017 is step-01's, untouched).

### Commit

When step DoD is met: `feat(memory/layer): session-close distillation (L0→L1/L2/L3, provenance-linked, LLM-cached + no-LLM floor) + mem_layers`

---

## 03-layered-retrieval-budgets

### Goal

Retrieval bootstraps from L2/L3 and falls back to L1/L0 (RRF) for a fact — and every `mem_*` read is budgeted (item + char + timeout) so memory can never overwhelm the context window; `mem_get_observation` stays the only full-content path; CLI parity.

### Out of scope / Non-Goals

- Distillation (step 02); governance (step 01); changing `mem_*` tool names/required params

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-layered-distill

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/budget.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/retrieve.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/budget_test.go`, `skillgrid-cli/internal/mnemonic/memory/retrieve_test.go`, `skillgrid-cli/internal/mnemonic/service/retrieval_test.go`, `skillgrid-cli/cmd/skillgrid/main_test.go`

**Interfaces:**
- Consumes: 02 layered store (`observation_layers`/`personas` as L2/L3 bootstrap sources) + 01 governance fields; 005 RRF read path
- Produces: layered retrieval (L2/L3-first, L1/L0 RRF fallback) + a uniform budget wrapper (item + char + timeout) on `mem_search`/`mem_context`/`mem_timeline`; `mem_get_observation` as the only full-content path; tunable budget config; CLI parity for `mem_layers`/`mem_governance`/`mem_share`

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Context-window flood), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-03. -->

- [x] 03.1 `[RED]` Threat "Context-window flood" — 20-hit search budgeted (truncated in-list + `mem_get_observation` id present + timeout honored with `truncated: true` partial) (Scenarios: `twenty-hit-search-is-budgeted`, `char-budget-truncates-with-explicit-omitted-count`, `context-timeout-returns-truncated-partial`)
  - [x] 03.1.a Write failing test
  - [x] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: FAIL
  - [x] 03.1.c Minimal implementation
  - [x] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: PASS
  - [x] 03.1.e Commit — `feat(memory): item+char+timeout budget wrapper on mem_* reads`
- [x] 03.2 `[RED]` Threat "Mnemonic tool surface" — layered retrieval + `mem_get_observation` only-full-content + 005 tools still stable + bad args rejected (Scenarios: `layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback`, `mem-get-observation-is-only-full-content-path`, `budgeted-reads-005-stable`)
  - [x] 03.2.a Write failing test
  - [x] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Retrieve -count=1` — Expected: FAIL
  - [x] 03.2.c Minimal implementation
  - [x] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Retrieve -count=1` — Expected: PASS
  - [x] 03.2.e Commit — `feat(memory): layered L2/L3-first retrieval + mem_get_observation full-content path`
- [x] 03.3 `[AFK]` `mem_context` / `mem_search` return L2/L3 first; specific-fact query falls back to L1/L0 via the existing RRF (Scenario: `layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/retrieve_test.go -run Layered -count=1` — Expected: PASS
- [x] 03.4 `[AFK]` Every `mem_*` read enforces the item-count cap, char budget (explicit "N chars omitted"), and context timeout (`truncated: true` partial + reason, never hangs) (Scenarios: `twenty-hit-search-is-budgeted`, `char-budget-truncates-with-explicit-omitted-count`, `context-timeout-returns-truncated-partial`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: PASS
- [x] 03.5 `[AFK]` A 20-hit search returns 20 budgeted snippets (not 20× full JSON); every in-list result carries its `mem_get_observation` id (Scenarios: `twenty-hit-search-is-budgeted`, `every-inlist-result-carries-get-observation-id`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run BudgetedSearch -count=1` — Expected: PASS
- [x] 03.6 `[AFK]` `mem_get_observation` remains the only full-content path; the budget is tunable (config); truncation never silent (Scenarios: `mem-get-observation-is-only-full-content-path`, `budget-is-tunable-via-config`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run FullContent -count=1` — Expected: PASS
- [x] 03.7 `[AFK]` Route `mem_context`/`mem_search`/`mem_timeline` through the layered + budgeted path; `mem_get_observation` stays the only full-content path (Scenario: `mem-context-search-timeline-budgeted`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run RouteReads -count=1` — Expected: PASS
- [x] 03.8 `[AFK]` CLI parity for `mem_layers`/`mem_governance`/`mem_share` (Scenario: `cli-parity-for-layer-governance-share`) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS
- [x] 03.9 `[AFK]` 005 `mem_*` names + required params unchanged; bad args rejected clearly (Scenarios: `budgeted-reads-005-stable`, `bad-retrieval-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | PASS | budget (item+char+timeout, enforced via Bound+ApplyRead), retrieve (L2/L3-first + RRF fallback), mcp (budgeted in-list + only-full-content + bad-args) |
| Acceptance `@step-03` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/service/... ./skillgrid-cli/cmd/skillgrid/... -count=1` + BDD `@step-03 @p0` | PASS | PASS | Layered, BudgetedSearch, FullContent, RouteReads, CLI parity (search/context/timeline budget flags), config-driven tunability |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/... ./skillgrid-cli/cmd/skillgrid/... -count=1` (+ `-race`) | PASS | PASS | all mnemonic pkgs + cmd green incl. `-race` |
| Rollback boundary | drop `memory/budget` + `memory/retrieve` + budget wiring + CLI additions; 005 read path + 01/02 additive fields intact (unused) | PASS | PASS | 017 (step-01) + 018 (step-02) untouched; budget/retrieve + wiring + CLI flags are the additive surface |
| Global Constraints | — | held | held | every mem_* read budgeted (item+char+timeout, enforced, never hangs); mem_get_observation is the ONLY full-content path; layered L2/L3-first + existing-RRF L1/L0 fallback (real production caller, owner-gated); 005 mem_* names+params unchanged (78-tool lock); step-01 per-owner visibility gate preserved on the budgeted path |

Review: task reviewer `approved with fixes`. Fix commits c25e122, 0cfa827, c19789a: (1) the budget timeout was a no-op (`TimeoutNs` stored but never consulted by `Apply`) — now `Budget.Bound` derives a deadline-bound ctx before the query and `ApplyRead` cuts a slow read to a `truncated:true`/`reason:timeout` partial (injectable, fast test); (2) `BudgetedRetrieval` had no production caller — now CLI `mem search` → `BudgetedRetrievalAs(AsRoot)` → `SearchOwnerScopedRetrieve` (the existing BlendedSearch RRF leg + step-01 `canRead` gate), with an end-to-end test asserting L2/L3-first + the gate; (3) CLI `mem context`/`timeline` now honor `--item/--char/--timeout` (uniform budget across all three reads). Scoped re-review: `TestMemSearchGetOwnerEnforcedWiring` still passes through the budgeted path; `-race` clean; 005/008/010/011 baselines green.

Commits (step 03): 6fce689 (budget wrapper), 91fd4a4 (layered retrieve), 9142064 (route reads), 7508faa (budgeted in-list + only-full-content + bad-args), 7e29c3d (CLI parity), c25e122 (enforce timeout), 0cfa827 (owner-gated layered + production entry point), c19789a (uniform CLI budget).

### Commit

When step DoD is met: `feat(memory): layered retrieval (L2/L3-first) + item/char/timeout budgets on every mem_* read + CLI parity`

---

## Verification (change-level)

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

**Change**: 013-mnemonic-layered-memory-governance
**Per-step verdicts**: 01-governance-fields PASS · 02-layered-distill PASS · 03-layered-retrieval-budgets PASS
**Runtime proof** (all run at verify time, `-count=1`, exit 0):
- Full 013 suite: `go test ./internal/mnemonic/memory/... ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... ./cmd/skillgrid/... -count=1` → all `ok` (also `-race` clean on memory/mcp/service/cmd)
- 005/008/010/011 baselines intact: `go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1` → all `ok`

### Scenario traceability (34 scenarios; @step-01 = 12, @step-02 = 11, @step-03 = 11)

Every scenario is COMPLIANT — a covering test passed at runtime in the suite above.

| @step-01 Scenario | Covering test | Result |
|---|---|---|
| governance-tools-and-existing-schema-stable | `mcp.TestMemGovernanceTools` (+ 77→78 surface lock) | COMPLIANT |
| bad-governance-args-rejected | `mcp.TestBadGovernanceArgs` | COMPLIANT |
| new-observation-private-by-default-with-owner | `service.PrivateDefault` + `memory.governance` | COMPLIANT |
| private-observation-invisible-to-second-owner-until-share | `memory.TestVisibility` + `mcp.TestMemSearchGetOwnerEnforcedWiring` (end-to-end) | COMPLIANT |
| restricted-acl-grant-enforced | `memory.TestVisibility` (acl_grants) | COMPLIANT |
| restricted-with-no-grants-is-owner-only | `memory.TestVisibility` (RestrictedNoGrants) | COMPLIANT |
| private-observation-absent-from-admin-cross-owner-list | `memory.AdminCrossOwnerList` clause test | COMPLIANT |
| mem-share-unknown-target-rejected | `service.MemShare` (unknown target → rejected, unchanged) | COMPLIANT |
| mem-update-appends-recoverable-version | `service.UpdateVersion` (prior content recoverable, latest=read path) | COMPLIANT |
| superseded-status-set-explicitly | `service.StatusExplicit` | COMPLIANT |
| retrieval-usage-count-increments-on-search | `service.UsageCount` (distinct from duplicate_count) | COMPLIANT |
| mem-governance-surfaces-asset-fields | `service.GovernanceQuery` + `mcp.TestMemGovernanceRoundTrip` | COMPLIANT |

| @step-02 Scenario | Covering test | Result |
|---|---|---|
| distilled-layer-carries-resolvable-l0-provenance | `layer.TestDistillProvenance` (L0 link resolves) | COMPLIANT |
| layer-with-unresolvable-source-not-created | `layer.TestDistillUnresolvableSourceNotCreated` (zero rows) | COMPLIANT |
| mem-layers-inspects-l0-to-l3-chain | `layer.TestInspectChain` + `service.TestMemLayers` | COMPLIANT |
| no-llm-floor-produces-provenance-ladder-offline | `layer.TestDistillNoLLMFloor` | COMPLIANT |
| mem-layers-registered-005-stable | `mcp.TestMemLayersRegistered` + 78-tool lock | COMPLIANT |
| bad-layer-args-rejected | `mcp` bad-layer-args test | COMPLIANT |
| session-close-distill-l0-to-l1-l2-l3 | `service.DistillHook` + `mcp` session-close wiring (Wired/BestEffort/Disabled) | COMPLIANT |
| distill-llm-cached-by-content-hash | `layer.TestDistillCacheHash` (both directions) | COMPLIANT |
| no-llm-floor-produces-l1-atoms | `layer.NoLLMFloor` (reuses 005 CapturePassive) | COMPLIANT |
| session-with-no-l1-content-is-noop | `layer.TestDistillNoOp` (zero rows) | COMPLIANT |
| l1-atom-correctable-with-traceable-provenance | `service.TestAtomCorrectable` (version-append + L0 trace) | COMPLIANT |

| @step-03 Scenario | Covering test | Result |
|---|---|---|
| twenty-hit-search-is-budgeted | `memory.Budget` + `service.BudgetedSearch` (10 budgeted, not 20 full) | COMPLIANT |
| char-budget-truncates-with-explicit-omitted-count | `memory.Budget` (marker + count) | COMPLIANT |
| context-timeout-returns-truncated-partial | `memory.TestBudgetTimeoutEnforced` (Bound+ApplyRead, injectable slow read) | COMPLIANT |
| layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback | `memory/retrieve` + `service.TestBudgetedRetrievalProductionEntryPoint` (CLI mem search, owner-gated) | COMPLIANT |
| mem-get-observation-is-only-full-content-path | `mcp` only-full-content test (in-list truncated, get full) | COMPLIANT |
| budgeted-reads-005-stable | `mcp` 78-tool lock + required-param map | COMPLIANT |
| bad-retrieval-args-rejected | `mcp` bad-args + `cmd` CLI bad-args | COMPLIANT |
| mem-context-search-timeline-budgeted | `mcp` RouteReads + uniform budget across 3 reads | COMPLIANT |
| every-inlist-result-carries-get-observation-id | `mcp`/`service` in-list id assertion | COMPLIANT |
| budget-is-tunable-via-config | `service.TestBudgetConfigDrivenTunability` (loads retrieval_budget YAML) | COMPLIANT |
| cli-parity-for-layer-governance-share | `cmd` mem layers/governance/share + search/context/timeline budget flags | COMPLIANT |

### Global Constraints — held
- Private-by-default + per-owner read enforcement (wired into mem_search/mem_get_observation end-to-end); restricted ACL; mem_share is the only widen.
- Append-versioning (latest = read path; revision_count advances; status explicit, never inferred); retrieval_usage distinct from duplicate_count.
- Provenance-linked layering (no orphan layers; no-op on empty); no-LLM floor reuses 005 CapturePassive; LLM cached by content-hash.
- Budgeted reads (item+char+timeout enforced, never hangs); mem_get_observation is the ONLY full-content path; layered L2/L3-first + existing-RRF L1/L0 fallback (real production caller, owner-gated).
- Additive: 005 mem_* names + required params unchanged (75→78 tool surface, name+required-param lock); 017/018 migrations additive; no CGo boundary added (pure Go over the existing store).

### Review
- Per-step task reviews: 01 `approved with fixes` (c93b87d+8563453 — wire read enforcement), 02 `approved with fixes` (c86414c — wire session-close hook), 03 `approved with fixes` (c25e122+0cfa827+c19789a — enforce timeout, production layered caller, uniform CLI budget). All re-reviewed clean; step-01 visibility gate preserved through the step-03 budgeted path.
- Non-blocking follow-ups (do not gate archive): (a) the in-list "N chars omitted" marker is asserted but the exact omitted count is not pinned in every path; (b) mem_context/mem_timeline item-cap is via their own `limit` param (char+timeout uniform) — documented divergence, not a constraint violation.

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
