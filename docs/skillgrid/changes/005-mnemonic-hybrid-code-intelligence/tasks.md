# Tasks: 005-mnemonic-hybrid-code-intelligence

> **STATUS:** `in-progress` (2026-09-04) — 0/4 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Turn Mnemonic's chunk-FTS code index into a foundation hybrid/graph code-intelligence slice (symbols, edges, identifier FTS, Tier-1/2 tools, offline RRF) so agents navigate and blast-radius without burning tokens on grep/read loops.

**Architecture:** Additive `011_*` schema + Extractor Module hooked into `Indexer.Run`; Identifier-Aware FTS and graph resolve feed Tier-1/2 `code_*` tools; hybrid RRF with Null Adapter default. See `change.md` decisions. Existing four `code_*` tools stay stable.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), FTS5 `unicode61`, MCP (`mcp-go`); optional code embedder Adapter (off by default).

**Spec:** `docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md`

**Acceptance:** `docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/acceptance.feature` (`@step-NN`)

---

## Goal

Coding agents and operators get queryable Symbols and Edges, Identifier-Aware FTS, orientation and call-graph tools, and offline Hybrid Search with provenance — without changing existing chunk `code_*` contracts or requiring embeddings online.

## Out of scope / Non-Goals

- Full plan-07 platform: 42 tools, communities/impact, git-diff analysis, documents, `graph.sqlite.zst`, fsnotify watcher
- Languages beyond Go / TypeScript / TSX; CGo tree-sitter extractors in this Change
- Replacing `chunks` / `chunks_fts` or renaming existing `code_status` / `code_index` / `code_search` / `code_read`
- Memory rewrite (Changes 002–004); cloud sync; clash with memory `semantic_search` (003) via bare plan-07 tool names
- Requiring ONNX / bundled nomic-embed-code to ship the foundation slice

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

- No full plan-07 platform (42 tools, communities/impact, git, documents, `graph.sqlite.zst`, watcher)
- No languages beyond Go/TS/TSX; no CGo tree-sitter in this Change
- Do not replace `chunks` / `chunks_fts` or rename existing `code_status` / `code_index` / `code_search` / `code_read`
- Do not rewrite memory (002–004); no cloud sync; no bare plan-07 names that clash with memory `semantic_search`
- Do not require ONNX / bundled nomic-embed-code to ship
- All new SQL is additive (`011_hybrid_code_intel.sql`); leave `009`/`010` for 001/003
- CGo-free (`modernc.org/sqlite`); per-project; content-hash (+ mtime) sync
- Graph + FTS offline without embeddings; Null Adapter default
- Per-file extract failure → `warn+continue` (fallback); never abort whole index run
- Every Edge carries Confidence Label `EXTRACTED | INFERRED | AMBIGUOUS`
- Store open with new schema → `warn+continue`; `files`/`chunks` intact
- Unknown / missing symbol → `warn+continue`; empty/not-found; no fabricated symbols or edges
- Ambiguous edge resolution → `warn+continue`; return `AMBIGUOUS`, never silent drop
- Bad / missing args on new `code_*` tools → `abort` with clear validation error
- Embedder unavailable → `warn+continue`; degrade to FTS + signals
- Existing four `code_*` tools → unchanged name + required params

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-schema-extractors
status: in_progress  # in_progress | blocked | done
updated: 2026-09-04T22:05:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `schema-extractors` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `identifier-fts-orientation` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `call-graph-traversal` | `@step-03` | 02 | Feature tagged `@step-03` |
| 04 | `hybrid-search-core` | `@step-04` | 03 | Feature tagged `@step-04` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

Honest forecast: four vertical slices (schema+extract → FTS/orient → graph → hybrid), each likely its own stacked PR. Step 01 alone is already near/over the 400-line budget (~450–650). Do not attempt a single-PR delivery without explicit exception.

Suggested split: PR1 schema+extract → PR2 FTS/orient → PR3 graph → PR4 hybrid · Chain strategy: stacked-to-main

---

## 01-schema-extractors

### Goal

Additive `011_*` schema + Extractor Module + Go/TS/TSX Adapters + indexer graph hook so index runs produce queryable Symbols and Edges without rewriting chunks.

### Out of scope / Non-Goals

- Identifier FTS tools, orientation MCP/CLI (step 02)
- Call-graph Tier-2 tools (step 03)
- Hybrid ranking / embedders (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/011_hybrid_code_intel.sql`
- Create: `skillgrid-cli/internal/mnemonic/extract/extract.go`
- Create: `skillgrid-cli/internal/mnemonic/extract/go.go`
- Create: `skillgrid-cli/internal/mnemonic/extract/tsx.go`
- Create: `skillgrid-cli/internal/mnemonic/extract/fallback.go`
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go`
- Test: `skillgrid-cli/internal/mnemonic/extract/...`, `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/store/...`

**Interfaces:**
- Consumes: none (existing indexer + store migration runner)
- Produces: `Extractor` Interface, `FileGraph`, graph tables (symbols/edges/…), same-tx extract/prune hook in `Indexer.Run`

### Tasks

- [ ] 01.1 `[RED]` Indexed Go and TS yield symbols and edges (Scenario: Indexed Go and TS yield symbols and edges)
  - [ ] 01.1.a Write failing test — fixture Go+TS/TSX project; after index, assert queryable symbols/edges and new tables without rewriting files/chunks
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/extract/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `011_*` migration + Extractor registry + Go/TS/TSX Adapters + indexer hook
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/extract/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): additive graph schema and language extractors`
- [ ] 01.2 `[AFK]` Malformed file falls back and index continues (Scenario: Malformed file falls back and index continues) — `Run: go test ./skillgrid-cli/internal/mnemonic/extract/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [ ] 01.3 `[AFK]` Store open preserves existing chunk index (Scenario: Store open preserves existing chunk index) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/extract/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/...` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `skillgrid index` on fixture repo; query symbols/edges via SQL | PASS | | |
| Rollback boundary | Drop `011_*` + `extract/`; revert indexer hook | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): additive graph schema and language extractors`

---

## 02-identifier-fts-orientation

### Goal

Identifier-Aware FTS + Tier-1 orientation MCP/CLI so agents find camelCase/snake_case symbols chunk search misses, while existing `code_*` stay stable.

### Out of scope / Non-Goals

- Call-graph Tier-2 tools (step 03)
- Hybrid/semantic ranking (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-schema-extractors

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/search/symbol_fts.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_orient.go`
- Create: `skillgrid-cli/cmd/skillgrid/code_intel.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/search/...`, `skillgrid-cli/internal/mnemonic/mcp/...`

**Interfaces:**
- Consumes: graph tables + Extractor output from 01
- Produces: Identifier-Aware FTS search; Tier-1 orientation tools (`code_map`, `code_search_symbols`, `code_get_symbol`, `code_get_signature`, `code_symbols_in_file`, `code_list_projects`, `code_index_status`); CLI orient entrypoints

### Tasks

- [ ] 02.1 `[RED]` Mnemonic tool surface — `code_search` schema stable (Scenario: code_search stable and bad orient args fail) — threat: Mnemonic tool surface
  - [ ] 02.1.a Write failing test — assert `code_search` tool name + required `query` param schema unchanged
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run CodeSearchSchema -count=1` — Expected: FAIL (or red until baseline lock exists)
  - [ ] 02.1.c Minimal implementation — lock baseline registry assertions before new tools
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run CodeSearchSchema -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `test(mnemonic): lock code_search tool schema baseline`
- [ ] 02.2 `[RED]` Mnemonic tool surface — orient tools register and reject bad args (Scenario: code_search stable and bad orient args fail) — threat: Mnemonic tool surface
  - [ ] 02.2.a Write failing test — assert orientation tools will be registered; bad/missing args rejected clearly
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run OrientTools -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `tools_code_orient.go` + registrar without dropping existing four `code_*`
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run OrientTools -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): register Tier-1 code orientation tools`
- [ ] 02.3 `[AFK]` Identifier FTS and orientation locate a symbol (Scenario: Identifier FTS and orientation locate a symbol) — `Run: go test ./skillgrid-cli/internal/mnemonic/search/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS
- [ ] 02.4 `[AFK]` Unknown symbol returns empty or not-found (Scenario: Unknown symbol returns empty or not-found) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS
- [ ] 02.5 `[AFK]` CLI parity for orientation commands via `code_intel.go` — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/search/... ./skillgrid-cli/internal/mnemonic/mcp/...` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | MCP orient tools on indexed fixture | PASS | | |
| Rollback boundary | Drop orient tools + `symbol_fts.go` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): identifier-aware FTS and Tier-1 orientation`

---

## 03-call-graph-traversal

### Goal

Edge resolve + Confidence Labels + Tier-2 graph tools so agents can judge blast radius before edits.

### Out of scope / Non-Goals

- Hybrid RRF / embedders (step 04)
- Changing orientation contracts from step 02

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-identifier-fts-orientation

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/graph/resolve.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_graph.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go`
- Test: `skillgrid-cli/internal/mnemonic/graph/...`, `skillgrid-cli/internal/mnemonic/mcp/...`

**Interfaces:**
- Consumes: symbols/edges from 01; orientation resolve patterns from 02
- Produces: graph resolve with Confidence Labels; Tier-2 tools (`code_get_callers`, `code_get_callees`, `code_get_dependents`, `code_get_implementors`, `code_get_tests_for`, `code_get_type_hierarchy`); CLI graph parity

### Tasks

- [ ] 03.1 `[RED]` Known symbol returns graph views with confidence (Scenario: Known symbol returns graph views with confidence)
  - [ ] 03.1.a Write failing test — known symbol yields callers/callees/dependents/implementors/hierarchy/tests-for; every edge has Confidence Label
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/graph/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `graph/resolve.go` + `tools_code_graph.go` + service/CLI facade
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/graph/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): call-graph traversal with confidence labels`
- [ ] 03.2 `[AFK]` Ambiguous resolution is labeled not dropped (Scenario: Ambiguous resolution is labeled not dropped) — `Run: go test ./skillgrid-cli/internal/mnemonic/graph/... -count=1` — Expected: PASS
- [ ] 03.3 `[AFK]` Unknown symbol graph query invents no edges (Scenario: Unknown symbol graph query invents no edges) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/graph/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/graph/... ./skillgrid-cli/internal/mnemonic/mcp/...` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | callers/callees on known symbol | PASS | | |
| Rollback boundary | Drop `graph/` + graph tools | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): call-graph traversal with confidence labels`

---

## 04-hybrid-search-core

### Goal

Offline RRF hybrid (FTS+signals) + pluggable embedders + hybrid/semantic/status tools so search ships without embeddings online.

### Out of scope / Non-Goals

- Communities, git impact, watcher
- Requiring ONNX to ship

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 03-call-graph-traversal

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/hybrid/rank.go`
- Create: `skillgrid-cli/internal/mnemonic/embedder/` (code-unit Adapter; Null default)
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` / `cmd/skillgrid/code_intel.go` as needed
- Test: `skillgrid-cli/internal/mnemonic/hybrid/...`, `skillgrid-cli/internal/mnemonic/mcp/...`

**Interfaces:**
- Consumes: Identifier FTS (02), graph/symbols (01–03), optional 003 embedder patterns
- Produces: RRF ranker + provenance; `code_hybrid_search`, `code_semantic_search`, `code_embedding_status`; registered tool sets; CLI dispatch

### Tasks

- [ ] 04.1 `[RED]` Mnemonic tool surface — hybrid distinct from memory semantic_search (Scenario: Hybrid tool is distinct and rejects bad args) — threat: Mnemonic tool surface
  - [ ] 04.1.a Write failing test — assert `code_hybrid_search` registers distinct from memory `semantic_search`; `code_search` still stable; bad hybrid/semantic args rejected
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run HybridTools -count=1` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation — `tools_code_hybrid.go` + server registration without dropping existing `code_*` or memory `semantic_search`
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run HybridTools -count=1` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(mnemonic): register hybrid code search tools`
- [ ] 04.2 `[AFK]` Hybrid search ranks offline with provenance (Scenario: Hybrid search ranks offline with provenance) — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [ ] 04.3 `[AFK]` Down embedder degrades to FTS and signals (Scenario: Down embedder degrades to FTS and signals) — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -count=1` — Expected: PASS
- [ ] 04.4 `[AFK]` Null Adapter default + CLI dispatch for hybrid commands — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/hybrid/... ./skillgrid-cli/internal/mnemonic/mcp/...` | PASS | | |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `code_hybrid_search` embeddings-off | PASS | | |
| Rollback boundary | Drop `hybrid/` + hybrid tools | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): offline hybrid code search with pluggable embeddings`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
