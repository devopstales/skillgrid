# Tasks: 003-mnemonic-self-evolving-context-database

> **STATUS:** `in-progress` (2026-09-05) — 0/5 steps PASS — revised after questioning
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Upgrade Mnemonic from flat full-token loads to tiered L0/L1/L2 storage, semantic retrieval, explicit session compaction, and trail observability — without rewriting Go MCP + SQLite+FS.

**Architecture:** Keep Go MCP + SQLite+FS; add deep `tiered` module + content-write seam (wired from `mnemonic_commit`), Pure Go `Embedder` behind `MNEMONIC_EMBED` (no CGO), separate L1 `semantic_search` with corpus filter (default LTM), and explicit async-tier `mnemonic_commit`. See `change.md` + `interview.md`. SQL at `010_*`.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), FS sidecars (`.abstract` / `.overview`), optional Pure Go embeddings behind `MNEMONIC_EMBED`.

**Spec:** `docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/change.md`

**Acceptance:** `docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/acceptance.feature` (`@step-NN`)

---

## Goal

Agents get overview-first recall with full detail on demand, operators can inspect retrieval trails when recall fails, and durable long-term memory is written only via explicit commit — all on the existing Go MCP + SQLite+FS stack.

## Out of scope / Non-Goals

- OpenCode plugin integration
- Python embedding service (Option B)
- CGO / onnxruntime (or other native) local embedders
- Rewriting FTS5, the code index, or the web research cache
- Cloud sync
- Auto-commit on every session end (only explicit `mnemonic_commit`; auto only as a future 001 completion hook)
- Tiering every `mem_save`
- Live external content-write producers this change — seam ready; producers are `mnemonic_commit` + `migrate --tier` only
- Merging `semantic_search` into `mem_search`
- Hard UAT token-% targets (behavioral UAT only)
- Chained PRs (one PR for this change)

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

- No OpenCode plugin; no Python embedding service; no CGO/onnxruntime embedder; no cloud sync
- Do not rewrite FTS5, the code index, or the web research cache
- Do not auto-commit on every session end; do not tier every `mem_save`
- No live external content writers this change (seam via commit + migrate only)
- Keep `semantic_search` separate from `mem_search`
- No hard UAT token-% targets (behavioral UAT only)
- Modules only; keep Go MCP + SQLite+FS hybrid
- Pure Go embeddings (Option A); feature-flagged; local preferred; optional remote HTTP
- Additive SQL only; leave `009_*` for 001; this change uses `010_*`
- Default search returns L1; L2 only via `load_full_details`
- `semantic_search` corpus filter defaults to LTM-only; optional all `tiered_contents`
- Auto-tiering must not block writes; commit success does not await tiers
- Migration fails mid-way → `abort`; prior observations/FTS/code index remain readable
- Summarizer failure during tier generation → `warn+continue`; L2 intact; failure logged
- Embedder unavailable / `MNEMONIC_EMBED` off → `warn+continue`; title/L0 fallback; trail still recorded
- `load_full_details` unknown path → `abort` (not-found)
- `mnemonic_commit` missing required sources → `abort`; no partial row
- Empty retrieval trails store → `warn+continue`; empty list, not error
- Unknown trail id → `abort` (not-found)
- Content-write hook / background tiering → `warn+continue`; never block L2 write
- Ship as one PR (per-step commits still OK on the branch)

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-schema-extensions
status: in_progress  # in_progress | blocked | done
updated: 2026-09-05T16:14:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `schema-extensions` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `tiered-storage` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `semantic-retrieval` | `@step-03` | 02 | Feature tagged `@step-03` |
| 04 | `session-compaction` | `@step-04` | 03 | Feature tagged `@step-04` |
| 05 | `trail-observability` | `@step-05` | 04 | Feature tagged `@step-05` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Delivery strategy | single PR (per-step commits on branch) |

---

## 01-schema-extensions

### Goal

Additive SQL for tier paths, long-term memories, trails, and embeddings so store open upgrades without rewriting existing rows.

### Out of scope / Non-Goals

- FS sidecars, summarizer, migrate CLI (step 02)
- MCP retrieval/compaction tools (steps 03–04)
- Trail CLI (step 05)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/010_tiered_context.sql`
- Modify: `docs/skillgrid/agents/glossary/technical.md` (apply-time)
- Modify: `docs/skillgrid/agents/glossary/business.md` (apply-time)
- Test: `skillgrid-cli/internal/mnemonic/store/` (migration / open tests)

**Interfaces:**
- Consumes: none
- Produces: tables `tiered_contents`, `long_term_memories`, `retrieval_trails`, `path_embeddings` via additive `010_*` migration; existing observations/FTS/code index intact

### Tasks

- [ ] 01.1 `[AFK]` Create `010_tiered_context.sql` with `tiered_contents`, `long_term_memories`, `retrieval_trails`, `path_embeddings` (leave `009_*` for 001)
- [ ] 01.2 `[AFK]` Add Tiered Storage / Semantic Search / Retrieval Trail terms to `docs/skillgrid/agents/glossary/technical.md`
- [ ] 01.3 `[AFK]` Add Long-term Memory to `docs/skillgrid/agents/glossary/business.md`
- [ ] 01.4 `[AFK]` Cover Scenario: `Store open adds tables without rewriting rows` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Schema|Migrate|Open|Tier' -count=1` — Expected: PASS
- [ ] 01.5 `[AFK]` Cover Scenario: `Upgrade from schema 008 is idempotent` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Idempotent|008|010' -count=1` — Expected: PASS
- [ ] 01.6 `[AFK]` Cover Scenario: `Failed migration leaves prior data intact` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'MigrationFail|Rollback|Intact' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped store tests for `@step-01` | PASS | | |
| Runtime harness | store open on fixture DB | PASS | | |
| Rollback boundary | failed migrate leaves prior rows | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): add 010 tiered context schema`

---

## 02-tiered-storage

### Goal

L0/L1/L2 filesystem layout, tiered module, `migrate --tier`, and non-blocking content-write seam (product wire in step 04).

### Out of scope / Non-Goals

- Semantic search / load tools (step 03)
- `mnemonic_commit` product wire (step 04)
- Trail CLI (step 05)
- Live external writers

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-schema-extensions

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/tiered/summarizer.go`
- Create: `skillgrid-cli/internal/mnemonic/tiered/tiered.go`
- Create: `skillgrid-cli/internal/mnemonic/tiered/hook.go`
- Create: `skillgrid-cli/cmd/skillgrid/migrate.go`
- Test: `skillgrid-cli/internal/mnemonic/tiered/`

**Interfaces:**
- Consumes: `tiered_contents` schema from 01
- Produces: `Summarizer` (`Abstract`, `Overview`); Generate/Read L0/L1/L2; `ContentWriteHook.AfterContentWrite` (non-blocking); `skillgrid migrate --tier`

### Tasks

- [ ] 02.1 `[AFK]` Create `summarizer.go` with `Summarizer` interface + stub/heuristic adapters (`Abstract`, `Overview`)
- [ ] 02.2 `[AFK]` Create `tiered.go` to generate/read L0 (`.abstract`) / L1 (`.overview`) / L2 and register paths in `tiered_contents`
- [ ] 02.3 `[AFK]` Create `hook.go` implementing non-blocking `ContentWriteHook.AfterContentWrite` (does not await summarization)
- [ ] 02.4 `[AFK]` Create `migrate.go` with `runMigrate` / `--tier` backfill of L0/L1 from existing L2
- [ ] 02.5 `[AFK]` Cover Scenario: `Content write yields sidecars without blocking` via test harness calling the seam — `Run: go test ./skillgrid-cli/internal/mnemonic/tiered/ -run 'Write|Hook|Sidecar|NonBlocking' -count=1` — Expected: PASS
- [ ] 02.6 `[AFK]` Cover Scenario: `Tier migrate backfills without changing full detail` — `Run: go test ./skillgrid-cli/cmd/skillgrid/ ./skillgrid-cli/internal/mnemonic/tiered/ -run 'Migrate|Tier|Backfill' -count=1` — Expected: PASS
- [ ] 02.7 `[AFK]` Cover Scenario: `Summarizer failure preserves full detail` — `Run: go test ./skillgrid-cli/internal/mnemonic/tiered/ -run 'SummarizerFail|Preserve' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/tiered/ -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped tiered tests for `@step-02` | PASS | | |
| Runtime harness | `skillgrid migrate --tier` on fixture | PASS | | |
| Rollback boundary | summarizer fail leaves L2 | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): add tiered L0/L1/L2 storage and migrate --tier`

---

## 03-semantic-retrieval

### Goal

Pure Go embeddings plus `semantic_search` / `load_full_details` tools with retrieval-trail logging, L1-default results.

### Out of scope / Non-Goals

- `mnemonic_commit` (step 04)
- Trail CLI (step 05)
- Changing existing `mem_*` contracts

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-tiered-storage

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/embedder/embedder.go`
- Modify: `skillgrid-cli/internal/mnemonic/memory/embedding.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_retrieval.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/` / `embedder/` / `service/`

**Interfaces:**
- Consumes: tiered paths + `path_embeddings` / `retrieval_trails` from 01–02; shared Vector helpers from 002 via `memory/embedding.go`
- Produces: `Embedder` (`Embed`, `Model`) Pure Go only; `semantic_search` → `{results:[{overview,abstract,full_path}], trail_id}` with corpus filter (`ltm` default | `all`); `load_full_details{path}` → `{content}`

### Tasks

- [ ] 03.1 `[RED]` Mnemonic tool surface: `semantic_search` response body is L1-only (overview + abstract; no L2 markdown body) — Scenario: `Semantic search returns ranked overviews only`
  - [ ] 03.1.a Write failing test
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'SemanticSearch|L1Only|Overview' -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'SemanticSearch|L1Only|Overview' -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): return L1-only semantic_search results`
- [ ] 03.2 `[RED]` Mnemonic tool surface: L2 content reachable only via `load_full_details{path}` — Scenario: `Explicit load returns full markdown`
  - [ ] 03.2.a Write failing test
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'LoadFullDetails|L2' -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'LoadFullDetails|L2' -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): load L2 only via load_full_details`
- [ ] 03.3 `[AFK]` Create `embedder.go` (`Embedder` + Pure Go local and/or stub + optional remote HTTP; `MNEMONIC_EMBED`; no CGO/onnxruntime)
- [ ] 03.4 `[AFK]` Modify `memory/embedding.go` to share Vector/blob/cosine helpers with the embedder
- [ ] 03.5 `[AFK]` Modify `service/service.go` for ranked L1 search with corpus filter (default LTM), `load_full_details`, and retrieval-trail persistence
- [ ] 03.6 `[AFK]` Create `tools_retrieval.go` with `semantic_search` and `load_full_details` handlers (JSON-only); make 03.1–03.2 pass
- [ ] 03.7 `[AFK]` Cover Scenario: `Embeddings off falls back with trail` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'EmbedOff|Fallback|Trail' -count=1` — Expected: PASS
- [ ] 03.8 `[AFK]` Cover Scenario: `Unknown path rejects full-detail load` — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'UnknownPath|NotFound' -count=1` — Expected: PASS
- [ ] 03.9 `[AFK]` Cover Scenario: `Default corpus is long-term memory only` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Corpus|LTM|Filter' -count=1` — Expected: PASS
- [ ] 03.10 `[AFK]` Cover Scenario: `Widened corpus includes all tiered paths` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Corpus|AllTiered|Filter' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped retrieval tests for `@step-03` | PASS | | |
| Runtime harness | MCP semantic_search + load_full_details | PASS | | |
| Rollback boundary | embeddings off → title/L0 fallback | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): add semantic_search and load_full_details`

---

## 04-session-compaction

### Goal

Explicit `mnemonic_commit` persists Long-term Memory with L0/L1/L2 without auto-commit on session end, keeping existing `mem_save` registered.

### Out of scope / Non-Goals

- Trail CLI (step 05)
- Auto-commit on session end
- Teams completion-hook wiring (future, after 001)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 03-semantic-retrieval

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/compaction.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/` / `service/`

**Interfaces:**
- Consumes: tiered Generate/Read + ContentWriteHook from 02; retrieval registration path from 03
- Produces: `mnemonic_commit{…}` → `{memory_id,paths}` with L2 durable on success and async L0/L1 via seam; existing `mem_save` still registered

### Tasks

- [ ] 04.1 `[RED]` Mnemonic tool surface: after registering new tools, existing `mem_save` remains registered and callable — Scenario: `Existing memory save remains registered`
  - [ ] 04.1.a Write failing test
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|Registered|Additive' -count=1` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|Registered|Additive' -count=1` — Expected: PASS
  - [ ] 04.1.e Commit — `test(mnemonic): assert mem_save remains registered with new tools`
- [ ] 04.2 `[AFK]` Create `compaction.go` for explicit commit → `long_term_memories` with durable L2 + optional source link; invoke `ContentWriteHook` without awaiting tiers
- [ ] 04.3 `[AFK]` Create `tools_compaction.go` with `mnemonic_commit` (`task_id?`, `lessons_learned?`, `title?` → `{memory_id,paths}`)
- [ ] 04.4 `[AFK]` Modify `server.go` to register retrieval + compaction tools; make 04.1 pass (no auto-commit on session end)
- [ ] 04.5 `[AFK]` Cover Scenario: `Explicit commit persists tiered long-term memory` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MnemonicCommit|LongTerm' -count=1` — Expected: PASS
- [ ] 04.6 `[AFK]` Cover Scenario: `Session end does not auto-commit` — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -run 'NoAutoCommit|SessionEnd' -count=1` — Expected: PASS
- [ ] 04.7 `[AFK]` Cover Scenario: `Missing sources reject without partial write` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MissingSources|Partial' -count=1` — Expected: PASS
- [ ] 04.8 `[AFK]` Cover Scenario: `Commit succeeds without waiting for tiers` — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'CommitAsync|NoAwaitTier' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | | |
| Acceptance `@step-04` / `@p0` | BDD / mapped compaction tests for `@step-04` | PASS | | |
| Runtime harness | MCP mnemonic_commit + mem_save still callable | PASS | | |
| Rollback boundary | missing sources → no partial row | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): add explicit mnemonic_commit compaction`

---

## 05-trail-observability

### Goal

`skillgrid trail show|recent` CLI so operators can inspect retrieval trails (query, directories, files, result path).

### Out of scope / Non-Goals

- New MCP tools
- Schema changes
- Changing search ranking behaviour

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 04-session-compaction

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/trail.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/cmd/skillgrid/`

**Interfaces:**
- Consumes: `retrieval_trails` rows written by step 03 search path
- Produces: `trail recent` and `trail show <id>` CLI surfaces

### Tasks

- [ ] 05.1 `[AFK]` Create `trail.go` with `trail recent` and `trail show <id>` reading `retrieval_trails` (query, directories, files, result path)
- [ ] 05.2 `[AFK]` Modify `main.go` to dispatch `migrate` and `trail` subcommands
- [ ] 05.3 `[AFK]` Cover Scenario: `Trail recent and show expose query paths` — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TrailRecent|TrailShow' -count=1` — Expected: PASS
- [ ] 05.4 `[AFK]` Cover Scenario: `Empty store lists nothing without error` — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TrailEmpty' -count=1` — Expected: PASS
- [ ] 05.5 `[AFK]` Cover Scenario: `Unknown trail id is not found` — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TrailNotFound|Unknown' -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | | |
| Acceptance `@step-05` / `@p0` | BDD / mapped trail tests for `@step-05` | PASS | | |
| Runtime harness | `skillgrid trail recent` / `show` | PASS | | |
| Rollback boundary | unknown id → not-found | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): add trail show and recent CLI`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
