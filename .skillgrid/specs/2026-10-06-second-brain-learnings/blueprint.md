# Second-Brain Learnings — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** VALIDATED (critic re-review 2026-10-11: VALIDATED-WITH-FIXES, all 10 applied (F1–F10); user re-gate 2026-10-11: APPROVED, execution HELD for a later session — no tickets claimed, no code touched)

**Tier:** T2

**Build shape:** Journey

**Numbering:** blueprint Tasks 12/13/14 = tasks.md TICKET-11/12/13; blueprint Task 6 (moves) = tasks.md TICKET-14 (lint tooling = TICKET-11). `TICKET-NN` references inside task headers below use ticket numbers (matching tasks.md); everything else uses task numbers.

**Goal:** Implement 5 learnable concepts (Growth, Health, Knowledge Class, PageRank, Confidence Gate) to improve mnemonic's "second brain" capabilities, plus restructure the knowledge zone as an OKF v0.2 wiki (R6), fuse web_cache into mem_search recall (R7), promote cache findings at reflect time (R8), wire recall + bootstrap into onboarding (R9), render the bundle as a read-only concept graph (R10), harden the recall MCP contract (R11), and teach the repo through one combined graph page (R12).

**Architecture:** This plan builds five independent features that are integrated into the existing `mnemonic` Go module. Feature 1 (Growth) and Feature 2 (Health) extend the existing `secondbrain` lifecycle. Feature 3 (Knowledge Class) adds a new database dimension for durability. Feature 4 (PageRank) optimizes code indexing. Feature 5 (Confidence Gate) adds a deterministic filter for real-time capture. Task 6 (Knowledge-base restructure) is docs-only: it dissolves `.skillgrid/artifacts/` into top-level OKF folders, renames `archive/` to `xarchive/`, and records the hybrid-SoT decision in a new ADR. Task 6 makes no Go *behavior* change (path-string sweep only, verified via `git show --stat` per Step 6) and runs after (or parallel to, disjoint files) Tasks 1–5.

**Tech Stack:** Go 1.26 (`mnemonic/go.mod`), SQLite, Tree-sitter, BM25/FTS5; Task 6 is Markdown + `git mv` + Node lint scripts only.

**Spec:** `.skillgrid/artifacts/06-spec-second-brain-learnings.md`

## Terms

- **Second Brain:** Mnemonic's persistent memory system for observations.
- **Knowledge Class:** A new dimension (episodic/convention/doc/unclassified) added in Feature 3.
- **Centrality:** A PageRank/degree score used in Feature 4 to select code chunks for embedding.
- **Capture Gate:** The confidence threshold used in Feature 5 to suggest memory saves.
- **OKF:** Open Knowledge Format v0.2 — a directory of Markdown concepts with YAML frontmatter (`type` required; `index.md`/`log.md` reserved).
- **Concept:** One unit of knowledge = one `.md` file; identity folders `decisions/`/`invariants/`/`workflows/` per repocodex pattern.
- **Reverse index:** Generated, committed `path → concept` map (repocodex `.repocodex/reverse-index.md` pattern, reimplemented natively).
- **OKF metrics:** Append-only JSONL telemetry sink (`.skillgrid/okf-metrics.jsonl`, gitignored) recording lint/validate events + `false_drift_rate`; borrowed from repocodex `metrics.jsonl` pattern; operational artifact, not memory, not indexed.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `mem_lifecycle growth` returns a cumulative timeline of added/total observations per period.
- `mem_lifecycle health` returns actionable recommendations for high duplicate density, low embedding coverage, and stale data.
- `mem_save` infers `knowledge_class` from `type` if not provided (e.g., decision → convention).
- `mem_search` with `knowledge_class=convention` returns only convention-classed observations.
- `code_index` with `--embed-top 50` embeds ≤ ceil(N/2) centrality-selected code symbols *plus* the doc-bearing + recently-changed carve-out; gate=100 (default) reproduces today's vector set; index output reports embedded/skipped counts.
- `mem_capture_suggest` with high-confidence text returns a save suggestion but does not persist it.
- `decisions/04-adr-*.md` exist with OKF frontmatter; every `ASSUMPTIONS.md` LOCKED row path resolves to a file on disk.
- `.skillgrid/index.md` lists every concept file; no concept file is missing from the index and no index entry points at a missing file.
- `.skillgrid/reverse-index.md` is in sync with `verification.anchors` across concepts.
- `code_search` for a relocated ADR title returns the new `decisions/` path (retrieval index covers the moved files).
- No stale `artifacts/` or `archive/` references remain anywhere in the repo (docs, skills, hooks, config, dashboard).
- `mem_search` returns labeled `web_cache` hits alongside observations; expired cache rows never surface.
- `reflect` proposes web-cache promotion candidates in `report.md` (with cache id + URL each) without writing concepts.
- Onboarding re-runs consult the knowledge index + memory before re-detecting; brownfield enroll yields anchored draft candidates.
- Fixture bundle renders the expected concept nodes/edges; orphans flagged; broken links tolerated; empty bundle renders empty state.
- Multi-bucket `mem_search` carries an explicit scope block; default single-bucket output unchanged (federated tests green unmodified).
- Fixture repo browses both ways (tree↔canvas), inspector shows identity + typed connections with source lines + preview, tour completes ≥3 stops, locals expand ephemerally with an empty store diff.
- Prefixed chunks carry path/symbol/scope context; MMR-vs-RRF ablation numbers recorded before any ranker change.
- `okf-lint.mjs` appends a JSONL event to `.skillgrid/okf-metrics.jsonl` (or `OKF_METRICS_SINK` override) per run; the file is gitignored; `false_drift_rate` is 0.0 on first run and computed from prior-run findings on subsequent runs; a telemetry-write failure does not change the exit code.

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/secondbrain/lifecycle.go` — Contains `GrowthTimeline` and extended `healthRecommendations`.
- `mnemonic/internal/store/migrations/051_add_knowledge_class.sql` — Database migration for Feature 3 (051 claimed by this change — 051/052 free, verified 2026-10-11; 053–058 occupied: tool_outputs/obs_split/ivf/hnsw, untouched).
- `mnemonic/internal/memory/service.go` — Handles `knowledge_class` logic in `Save` and `Update`.
- `mnemonic/internal/memory/search_blend.go` — Optional `knowledge_class` filter + per-class labeling.
- `mnemonic/internal/codeindex/indexer.go` — Implements the centrality-based embedding gate (centrality from existing `codeindex/graph.go`; there is no `graph_index.go` on disk).
- `mnemonic/internal/secondbrain/infer.go` — Contains `ClassifyCapture` for Feature 5.
- `mnemonic/internal/mcp/tools_memory.go` — `mem_lifecycle growth`, `mem_save`/`mem_search` `knowledge_class` params, `mem_capture_suggest` registration.
- `artifacts/04-adr-0038-knowledge-class-dimension.md` — ADR for the additive durability dimension (reserve 0038; 0037 taken by the embedded-ANN index, highest in use is 0037).
- `decisions/04-adr-*.md` (relocated via `git mv` from `artifacts/`, history preserved) — OKF frontmatter added (`type: TechnicalDecision`), `status/supersedes/date` untouched.
- `invariants/<slug>.md` — one per locked constraint, `type: InvariantContract` + `claims[]` where a checkable literal exists.
- `workflows/sdd-pipeline.md` (+ per-domain flows) — `type: BusinessWorkflow`, one anchor per participating site.
- `glossary/` + `reference/` — relocated terms/PRD/research/roadmaps, `type: Reference`.
- `.skillgrid/index.md` (OKF root catalog, `okf_version: "0.2"`; was `artifacts/README.md`), `.skillgrid/log.md`, `.skillgrid/reverse-index.md` (generated, committed); per-dir `index.md` catalogs (incl. `decisions/index.md` absorbing `03-adr-index.md`).
- `xarchive/` (renamed from `archive/` via `git mv`, history preserved; closed changes immutable).
- `artifacts/` REMOVED (dissolved; verified empty before removal).
- New ADR amending ADR-0012 (hybrid SoT: OKF bundle durable, SQLite retrieval index) — next free number at authoring time (NOT 0037/0038; expect 0039).
- `mnemonic/internal/memory/search_blend.go` (+ `service.go` query sites) — fused RRF path over `observations_fts` + `web_cache_fts`; `mcp/tools_memory.go` `mem_search` description update.
- New ADR recording the always-fused `mem_search` contract (behavior shift for every caller) — next free number at authoring time (NOT 0037/0038/0039; expect 0040; verify `ASSUMPTIONS.md` "Highest sequence in use" first).
- `.agents/skills/lifecycle/reflect/SKILL.md` (+ report template section) — promotion-candidates step.
- `.agents/skills/lifecycle/onboarding/SKILL.md` (+ `references/process.md`) — recall-before-detect + bootstrap steps.
- `mnemonic/internal/knowledge/graph.go` (+ test) — derive nodes/edges/backlinks from concept frontmatter + Markdown links; `mnemonic/internal/http/server.go` — one open `GET /knowledge/graph` route.
- `skillgrid-ui/src/features/mnemonic/KnowledgeGraphPage.tsx` (+ route registration, vitest render test) — reuses the code-graph D3 renderer with concept type colors.
- `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` (+ tree panel, enriched inspector, search box, layer filters, tour mode, concept toggle, locals expansion) — R12 teaching upgrade.
- `mnemonic/internal/mcp/tools_memory.go` (descriptions + scope block) + description-only touch on `tools_secondbrain.go`/`tools_session_inject.go`.
- `mnemonic/internal/codeindex/indexer.go` — deterministic `path · symbol · scope` chunk prefixes; `hybrid/rank.go` — ablation-only (no ranker change without winning numbers, recorded in `findings.md`).
- `.gitignore` — `.skillgrid/okf-metrics.jsonl` entry (telemetry sink, operational artifact).

**Key links** (critical connections between artifacts that must work together):
- `GrowthTimeline` must exclude soft-deleted and archived rows from the cumulative total.
- `Save` in `memory/service.go` must call the heuristic backfill for `knowledge_class` if the field is empty.
- The `codeindex` embed gate must ensure that doc-bearing and recently-changed chunks are always embedded.
- Every `ASSUMPTIONS.md` § LOCKED row must resolve to its relocated `decisions/` file (index ↔ file consistency).
- `.skillgrid/index.md` entries must match files on disk both directions (no orphans, no dangling entries).
- The new amending ADR must name ADR-0012 without editing the ADR-0012 file or its row's history (additive only).
- The ship target is `xarchive/` — `ship`/`reflect`/`resume`/`qa`, hooks, `sdd-structure.md`, `config.yaml`, and dashboard paths must all agree (reference sweep, reviewed not blind).
- Fusion must exclude `expires_at < now` rows — expired web content never surfaces in default recall.
- Promotion candidates must cite cache id + URL each; no concept is written unconfirmed (same-edit rule applies on confirm).
- Bootstrap drafts carry `verification.anchors` + `status: draft`; unconfirmed drafts are dropped, never kept.
- Graph derivation reads current Markdown per call (disposable view, never stored); `index.md`/`log.md`/reverse-index excluded from nodes; write path nonexistent by construction (read-only route + reader page).
- No ranker change without winning ablation numbers; LLM-written summaries and OpenAI embeddings stay out (cost + local-first).

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Task 3.1:** Adding a new database column `knowledge_class` to the `observations` table (requires migration).
- **Task 6.1:** Rewriting the LOCKED index paths (37 ADR relocations touch the decision record) + authoring the ADR-0012 amendment (LOCKED rule: adding a row requires explicit user OK). STOP for user approval before Step 3. `git mv` preserves history; the risk is index inconsistency, not data loss.

## Global Constraints

- **Go 1.22+:** Minimum Go version to build (per `05-locked-constraints.md`).
- **No new dependencies:** No new libraries without an ADR (Task 6 adds none — Node scripts only, same runtime as existing drift guards).
- **Serial development:** One change at a time (per `05-locked-constraints.md`).
- **Conventional commits:** No AI-attribution trailers.
- **IRON RULE:** Never delete or edit an accepted ADR file in place — `git mv` + new superseding/amending files only; rows flip, files stay.
- **Spec zone before code zone:** `.skillgrid/specs/` commits separate from code commits (pre-commit zone guard).

---

## Task 1: Growth Timeline

**Files:**
- Modify: `mnemonic/internal/secondbrain/lifecycle.go`
- Modify: `mnemonic/internal/mcp/tools_memory.go`
- Test: `mnemonic/internal/secondbrain/lifecycle_test.go`

**Interfaces:**
- Produces: `func GrowthTimeline(project string, granularity string) ([]PeriodCount, error)`

**SATISFIES:** `growth-timeline` scenario in `acceptance.feature` (@step-01, G1)

- [ ] **Step 1: Write the failing test**

```go
func TestGrowthTimeline_Monthly(t *testing.T) {
    // Setup DB with synthetic rows for Jan, Feb, Mar
    // Call GrowthTimeline
    // Verify cumulative total
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/secondbrain/... -v -run TestGrowthTimeline_Monthly`
Expected: FAIL with "undefined: GrowthTimeline"

- [ ] **Step 3: Write minimal implementation**

Implement `GrowthTimeline` using a `strftime` based bucketing query in `lifecycle.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/secondbrain/... -v -run TestGrowthTimeline_Monthly`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/secondbrain/
git commit -m "feat(mnemonic): add growth timeline to secondbrain lifecycle"
```

---

## Task 2: Health Recommendations

**Files:**
- Modify: `mnemonic/internal/secondbrain/lifecycle.go`
- Test: `mnemonic/internal/secondbrain/lifecycle_test.go`

**Interfaces:**
- Consumes: `HealthReport` struct from `lifecycle.go`

**SATISFIES:** `health-recommendations` scenario in `acceptance.feature` (@step-02, G2)

> NOTE: unit-test the pure `healthRecommendations` function directly (not `Health()`) to avoid the 24h health file cache. Target severities per spec: dedup HIGH (density>0.05, keep current code), embeddings MEDIUM with embedder-off no-nag gate, stale MEDIUM with total>10 guard, cleanup LOW, review LOW.

- [ ] **Step 1: Write the failing test**

```go
func TestHealthRecommendations_HighDensity(t *testing.T) {
    // Call healthRecommendations pure function directly (not Health(), avoids 24h cache)
    // Create fixture with high duplicate density
    // Assert HIGH severity dedup recommendation present
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/secondbrain/... -v -run TestHealthRecommendations_HighDensity`
Expected: FAIL (recommendation missing)

- [ ] **Step 3: Write minimal implementation**

Extend `healthRecommendations` with the 5 rules defined in the spec.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/secondbrain/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/secondbrain/
git commit -m "feat(mnemonic): expand health recommendations for secondbrain"
```

---

## Task 3: Knowledge Class Dimension

> ⚠ **one-way:** Adding a database column `knowledge_class` to `observations`. STOP for user approval. Soft one-way: the heuristic backfill rewrites existing rows (re-runnable but data-mutating) — snapshot/backup before Task 3 Step 3.

**Files:**
- Create: `mnemonic/internal/store/migrations/051_add_knowledge_class.sql`
- Modify: `mnemonic/internal/memory/service.go`
- Modify: `mnemonic/internal/memory/search_blend.go`
- Test: `mnemonic/internal/memory/service_test.go`

**Interfaces:**
- Produces: `KnowledgeClass` field on `observation` structs.
- Produces: `inferKnowledgeClass` helper function.

**SATISFIES:** `knowledge-class` scenario in `acceptance.feature` (@step-03, G3+G4)

- [ ] **Step 1: Write the failing test**

```go
func TestSave_KnowledgeClassInfer(t *testing.T) {
    // Save a 'decision' type observation without knowledge_class
    // Verify it is read back as 'convention'
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/memory/... -v -run TestSave_KnowledgeClassInfer`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Add the migration and the `inferKnowledgeClass` logic in `service.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/memory/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/
git commit -m "feat(mnemonic): add knowledge class dimension to observations"
```

---

## Task 4: PageRank-guided Embedding

**Files:**
- Modify: `mnemonic/internal/codeindex/indexer.go`
- Test: `mnemonic/internal/codeindex/indexer_test.go`

**Interfaces:**
- Produces: `--embed-top` CLI flag.

**SATISFIES:** `pagerank-embed` scenario in `acceptance.feature` (@step-04, G5)

- [ ] **Step 1: Write the failing test**

```go
func TestIndex_EmbedTop(t *testing.T) {
    // Index a fixture with N symbols
    // With gate=50: assert ≤ ceil(N/2) centrality-selected code symbols embedded,
    //   doc-bearing + recently-changed always embedded, output reports embedded/skipped counts
    // With gate=100: assert identical vector set to ungated behavior
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/codeindex/... -v -run TestIndex_EmbedTop`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement centrality ranking and the embedding gate in `indexer.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/codeindex/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/codeindex/
git commit -m "feat(mnemonic): add PageRank-guided embedding selection"
```

---

## Task 5: Confidence-gated Capture

**Files:**
- Modify: `mnemonic/internal/secondbrain/infer.go`
- Modify: `mnemonic/internal/mcp/tools_memory.go`
- Test: `mnemonic/internal/secondbrain/capture_test.go`

**Interfaces:**
- Produces: `ClassifyCapture(text string) (trigger string, confidence float64)`
- Produces: `mem_capture_suggest` MCP tool.

**SATISFIES:** `confidence-capture` scenario in `acceptance.feature` (@step-05, G6)

- [ ] **Step 1: Write the failing test**

```go
func TestCaptureSuggest_HighConfidence(t *testing.T) {
    // Call suggest with "We're switching from X to Y"
    // Verify suggested:true and NO DB row created
    // Strong pattern ("don't do that") fires; weak ("no") + directive ("use X") fires;
    // weak alone does not; negative ("no worries") suppresses
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/secondbrain/... -v -run TestCaptureSuggest_HighConfidence`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Add `ClassifyCapture` (two-pass structure per spec: strong patterns always fire; weak patterns require a directive word; negative patterns suppress; sets user-overridable via config) and the `mem_capture_suggest` tool registration.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/
git commit -m "feat(mnemonic): add confidence-gated real-time capture"
```

---

## Task 6: Knowledge-base restructure (.skillgrid as OKF wiki)

> ⚠ **one-way (index-level):** Rewriting the LOCKED index paths + authoring the ADR-0012 amendment requires explicit user OK (LOCKED rule). STOP for user approval before Step 3. Docs-only task: no Go *behavior* change (path strings only), no migration, no new dependency. Disjoint file set from Tasks 1–5 (parallel-safe once approved). Executes as tasks.md TICKET-14 (lint tooling is TICKET-11).

**Files:**
- Move (`git mv` only): `.skillgrid/artifacts/04-adr-*.md` → `.skillgrid/decisions/04-adr-*.md` (history preserved; `0014/0015-removed.md` rows stay frozen)
- Move (`git mv` only): `.skillgrid/archive/` → `.skillgrid/xarchive/` (history preserved; closed changes immutable)
- Create: `.skillgrid/decisions/` + `invariants/` + `workflows/` + `glossary/` + `reference/` contents (splits of `artifacts/05-locked-constraints.md`, `artifacts/01/02-*-terms.md`, `artifacts/00-prd.md`, `artifacts/06/07/08/09-*.md`)
- Create: `.skillgrid/index.md` (`okf_version: "0.2"` root catalog; was `artifacts/README.md`) — doubles as the Obsidian vault home (sections per folder + per-audience start-here pointers; `.skillgrid/` itself is the vault with committed `.obsidian/app.json` shared filters, local workspace state gitignored inside it), `.skillgrid/log.md`, `.skillgrid/reverse-index.md` (generated, committed), per-dir `index.md` catalogs
- Convention (vault-compatible, lint-enforced): links are relative Markdown `[text](path.md)` only — no `[[wikilinks]]` (one syntax renders in Obsidian, GitHub, and the linter alike; unresolvable `[[ ]]` is an `okf-lint.mjs` finding)
- Remove: `.skillgrid/artifacts/` (only after verified empty)
- Create: new ADR amending ADR-0012 in `decisions/` — next free number at authoring time (NOT 0037/0038; expect 0039; verify `ASSUMPTIONS.md` "Highest sequence in use" first)
- Modify: `.skillgrid/ASSUMPTIONS.md` (LOCKED rows → new paths), `ARCHITECTURE.md` §9 state-zone diagram, `config.yaml` (`conventions.artifacts`), `_shared/rules/sdd-structure.md` + all skills citing `artifacts/…` or `specs/…→archive/`, `ship`/`reflect`/`resume`/`qa` scripts + hooks, dashboard `/plans` + docs endpoints if they hardcode paths
- Modify (frontmatter only): relocated ADR files (add `type/title/description/generated`; `status/supersedes/date` untouched), `ARCHITECTURE.md`, `specs/*` + `xarchive/*` pages (`status: draft/stable/deprecated`)
- Never touch: `.skillgrid/config.yaml` values other than the path keys above, `.skillgrid/state.yaml` (machine state, excluded)

**Interfaces:**
- Consumes: `okf-wiki-plan.md` (same dir) as the move/frontmatter specification.
- Produces: OKF v0.2 bundle layout verifiable by `index.md` completeness both directions.

**SATISFIES:** R6 scenario in `acceptance.feature` (TBD — pending acceptance delta + re-gate; do not execute before the Gate exists)

- [ ] **Step 1: Snapshot the before-state**

```bash
find .skillgrid/artifacts .skillgrid/archive -name '*.md' | sort > /tmp/okf-before.txt
grep -rln 'skillgrid/artifacts\|\.skillgrid/archive\|specs/.*archive' --include='*.md' --include='*.yaml' --include='*.mjs' --include='*.js' --include='*.go' --include='*.ts' --include='*.tsx' .agents/ hooks/ git-hooks/ scripts/ config.d/ mnemonic/ skillgrid-ui/src/ .skillgrid/ | sort > /tmp/okf-refs-before.txt
git status --short .skillgrid/ > /tmp/okf-clean-check.txt
```

Expected: clean `.skillgrid/` worktree (no uncommitted spec-zone changes — commit those first per the zone guard); `/tmp/okf-refs-before.txt` is the reference-sweep checklist.

- [ ] **Step 2: Dry-run the move list and verify it fails clean**

```bash
git mv --dry-run .skillgrid/artifacts/04-adr-0001-prd-scope-whole-hub.md .skillgrid/decisions/04-adr-0001-prd-scope-whole-hub.md
git mv --dry-run .skillgrid/archive .skillgrid/xarchive
```

Expected: dry-runs succeed (proves paths/permissions); nothing moved yet. Repeat the ADR pattern for all 37 files in Step 3 (recount at execution — 37 as of 2026-10-11, including any ADR landed since, e.g. 0038 from Task 3).

- [ ] **Step 3: Execute moves + splits + frontmatter + sweep (STOP — user approval required before this step)**

`git mv` all `04-adr-*.md` → `decisions/`; `git mv archive` → `xarchive`; split constraints/terms/PRD/research into `invariants/`/`glossary/`/`reference/`; draft `workflows/sdd-pipeline.md`; add OKF frontmatter per `okf-wiki-plan.md` profile (never touch `status/supersedes/date` on accepted ADRs); write root `index.md`/`log.md`/generate `reverse-index.md` + per-dir catalogs; remove `artifacts/` only after `find .skillgrid/artifacts -type f | wc -l` returns 0; author the ADR-0012 amendment; update `ASSUMPTIONS.md` LOCKED rows; sweep every path in `/tmp/okf-refs-before.txt` (`config.yaml`, `sdd-structure.md`, skills, ship/reflect/resume/qa scripts + hooks, dashboard endpoints).

- [ ] **Step 4: Verify index ↔ file consistency both directions**

```bash
node .agents/skills/verification/qa/scripts/okf-lint.mjs .   # TICKET-11 lands first
node .agents/skills/verification/qa/scripts/state-drift-check.mjs .
# every LOCKED row resolves:
grep -o '\.skillgrid/[a-z]*/[^)]*\.md' .skillgrid/ASSUMPTIONS.md | sort -u | while read f; do test -f "$f" || echo "MISSING: $f"; done
# every concept indexed, no dangling entries, no leftovers:
find .skillgrid/decisions .skillgrid/invariants .skillgrid/workflows .skillgrid/glossary .skillgrid/reference -name '*.md' | sort > /tmp/okf-concepts.txt
test -d .skillgrid/artifacts && echo "LEFTOVER: artifacts/ still exists" || echo "artifacts/ dissolved OK"
test -d .skillgrid/archive && echo "LEFTOVER: archive/ still exists" || echo "archive/ renamed OK"
# reference sweep complete:
grep -rln 'skillgrid/artifacts\|\.skillgrid/archive' --include='*.md' --include='*.yaml' --include='*.mjs' --include='*.js' --include='*.go' --include='*.ts' --include='*.tsx' .agents/ hooks/ git-hooks/ scripts/ config.d/ mnemonic/ skillgrid-ui/src/ .skillgrid/ | sort > /tmp/okf-refs-after.txt
```

Expected: drift-check exit 0; zero MISSING lines; every file in `/tmp/okf-concepts.txt` appears in `index.md` and vice versa; `artifacts/` gone, `archive/` gone; `/tmp/okf-refs-after.txt` empty; `okf-lint.mjs` exit 0 with `DRIFT: clean` + `SCOPE: COMPLETE` (it subsumes the manual grep checks above — keep both until the script is proven, then the greps become its cross-check).

- [ ] **Step 5: Verify retrieval + history**

```bash
git log --follow --oneline -- .skillgrid/decisions/04-adr-0012-sqlite-as-second-brain.md | head -3
git log --follow --oneline -- .skillgrid/xarchive/2026-10-06-kanban-milestones/report.md | head -3
```

plus a `code_search` for a relocated ADR title, expecting the new `decisions/` path. Expected: history intact through both moves; search hits the new path.

- [ ] **Step 6: Commit in two zone-clean commits (spec-zone first)**

```bash
git add .skillgrid/
git commit -m "docs(skillgrid): dissolve artifacts into OKF folders, archive -> xarchive"
# then the reference sweep (skills/hooks/config/dashboard — no Go behavior change):
git add .agents/ hooks/ git-hooks/ scripts/ config.d/ config.yaml skillgrid-ui/src/ mnemonic/
git commit -m "chore(skillgrid): sweep artifacts/archive references to OKF layout"
```

Expected: first commit contains `.skillgrid/` paths only (zone guard: no code mixed in); second commit contains no Go behavior change (path strings only — verify with `git show --stat`).

---

## Task 7: Fused mem_search recall (observations + web_cache)

> Depends on Task 3 (shares `search_blend.go` — implement after TICKET-03 lands to avoid conflicts). Behavior shift for every `mem_search` caller: record the contract in a new ADR in the same task.

**Files:**
- Modify: `mnemonic/internal/memory/search_blend.go` (second MATCH over `web_cache_fts`, RRF fuse, provenance labels)
- Modify: `mnemonic/internal/memory/service.go` (query sites feeding `mem_search`)
- Modify: `mnemonic/internal/mcp/tools_memory.go` (`mem_search` description: two-corpus results)
- Test: `mnemonic/internal/memory/search_blend_test.go` (or `service_test.go`)
- Create: `.skillgrid/decisions/04-adr-NNNN-fused-mem-search-contract.md` (post-R6 path; pre-R6 `artifacts/`) — next free number at authoring (NOT 0037/0038/0039; expect 0040)

**Interfaces:**
- Consumes: existing RRF (`hybrid/rank.go`, RRFK=60); `web_cache_fts` schema (title, content, query, url, source, library_id).
- Produces: web hits labeled `corpus: web_cache` + source/url/fetched_at/expires_at + `web_cache_get id=` pointer. Vector leg unchanged (observations-only).

**SATISFIES:** `fused-recall` scenario in `acceptance.feature` (@step-07, G8)

- [ ] **Step 1: Write the failing test**

```go
func TestMemSearch_FusedWebCache(t *testing.T) {
    // Seed one observation + one web_cache row (unexpired) + one expired web_cache row
    // mem_search over a term matching all three
    // Assert: observation hit + labeled web hit present; expired row absent;
    //   web hit carries corpus/source/url/fetched_at/expires_at labels
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/memory/... -v -run TestMemSearch_FusedWebCache`
Expected: FAIL (no web leg)

- [ ] **Step 3: Write minimal implementation**

Add the project-scoped `web_cache_fts` MATCH with `expires_at` exclusion, fuse via RRF, attach labels + retrieve pointer. Update the `mem_search` tool description. Author the contract ADR (additive; LOCKED row needs user OK — bundle the approval with the re-gate).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd mnemonic && go test ./internal/memory/... -v`
Expected: PASS; prior observations-only assertions re-baselined only where they pinned exact result sets (record each re-baseline in the commit message).

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/
git commit -m "feat(mnemonic): fuse web_cache into mem_search recall"
# ADR separately (spec zone):
git add .skillgrid/
git commit -m "docs(skillgrid): record fused mem_search contract (ADR-NNNN)"
```

---

## Task 8: Reflect-time promotion candidates

> Docs-only task (no Go code). Disjoint files from Tasks 1–7 — parallel-safe.

**Files:**
- Modify: `.agents/skills/lifecycle/reflect/SKILL.md` (new Step 3.x: select → distill → propose; explicit-confirmation rule)
- Modify: `.agents/skills/lifecycle/reflect/templates/report.md` pointer target (add `## Web-cache promotion candidates` section shape) — or the merged qa template if that is where the shape lives; keep one canonical shape
- Test: structural — the gates below

**Interfaces:**
- Consumes: `web_cache_search` selection rule (`session_id` in the change's sessions OR fetched-at window + topic tags).
- Produces: candidate cards in `report.md` (concept home, why, `sources:` entry, claim text, cache id + URL) + Open Items. Never concepts.

**SATISFIES:** `promotion-candidates` scenario in `acceptance.feature` (@step-08, G9)

- [ ] **Step 1: Snapshot**

```bash
grep -c 'promotion candidates' .agents/skills/lifecycle/reflect/SKILL.md || echo "absent (expected before)"
```

- [ ] **Step 2: Draft the step + template section**

Write the selection rule, card shape, explicit-confirmation rule (interactive before close, else next session/change; same-edit rule on confirm; unconfirmed expire harmlessly), and the verification item (every candidate cites cache id + URL).

- [ ] **Step 3: Verify gates + hygiene**

```bash
node scripts/check-skillgrid-skills.mjs 2>&1 | grep -E 'reflect/' || echo "reflect clean"
```

Expected: G9 grep counts met; no new skill-check findings for `reflect`.

- [ ] **Step 4: Commit (docs zone)**

```bash
git add .agents/skills/lifecycle/reflect/
git commit -m "docs(skillgrid): reflect proposes web-cache promotion candidates"
```

---

## Task 9: Onboarding recall + bootstrap

> Docs-only task (no Go code). Disjoint files from Tasks 1–8 — parallel-safe.

**Files:**
- Modify: `.agents/skills/lifecycle/onboarding/SKILL.md` (recall-before-detect + bootstrap entries)
- Modify: `.agents/skills/lifecycle/onboarding/references/process.md` (Step 1 recall rule, Step 1.5b bootstrap procedure, verification items)
- Test: structural — the gates below

**Interfaces:**
- Consumes: `skillgrid:knowledge-base` (recall path, draft shape) + `skillgrid:mnemonic` (`mem_context`/`mem_search`).
- Produces: Step 1 rule (re-runs read `index.md` + `log.md` tail + memory before detecting; fresh projects skip with a note); Step 1.5b (enrolled conventions → `status: draft` candidates with `verification.anchors`, confirm-one-at-a-time, unconfirmed dropped); `skillgrid init --docs` ingest unchanged.

**SATISFIES:** `onboarding-recall-bootstrap` scenario in `acceptance.feature` (@step-09, G10)

- [ ] **Step 1: Snapshot**

```bash
grep -c 'recall-before-detect\|Step 1.5b' .agents/skills/lifecycle/onboarding/SKILL.md .agents/skills/lifecycle/onboarding/references/process.md || echo "absent (expected before)"
```

- [ ] **Step 2: Draft the steps + verification items**

Write the recall rule (merge re-runs only), the bootstrap procedure (brownfield only; anchors required; confirm-each; drop-unconfirmed), and verification additions (recall evidenced; every draft anchored; no draft survives unconfirmed).

- [ ] **Step 3: Verify gates + hygiene**

```bash
node scripts/check-skillgrid-skills.mjs 2>&1 | grep -E 'onboarding/' || echo "onboarding clean"
```

Expected: G10 grep counts met; no new skill-check findings for `onboarding`.

- [ ] **Step 4: Commit (docs zone)**

```bash
git add .agents/skills/lifecycle/onboarding/
git commit -m "docs(skillgrid): onboarding recalls knowledge base, bootstraps drafts"
```

---

## Task 10: Knowledge-concept graph view

> Vertical slice (derive + route + page) — small by construction: one Go file, one route, one page reusing the code-graph renderer. Depends on Task 6 only for real data; develops against a fixture bundle so it can build in parallel (verify against the real bundle at the end if Task 6 has landed, else fixture-only). Note: package `knowledge` already exists (doc/config/sql extractors) — new `graph.go` coexists; keep new symbols `Concept*`-prefixed (no `Node/Edge/Graph/Derive` collisions as of 2026-10-11).

**Files:**
- Create: `mnemonic/internal/knowledge/graph.go`
- Test: `mnemonic/internal/knowledge/graph_test.go` (fixture bundle)
- Modify: `mnemonic/internal/http/server.go` (one route: `GET /knowledge/graph`, open read, existing `?project=` scoping)
- Create: `skillgrid-ui/src/features/mnemonic/KnowledgeGraphPage.tsx` (+ SPA route registration)
- Test: vitest render test for the page

**Interfaces:**
- Consumes: concept Markdown (frontmatter `type/title/description/tags/status` + Markdown links); D3 force renderer + `apiGet` project injection from the code-graph page.
- Produces: `{nodes:[{id,title,type,description,tags,status}], edges:[{source,target}], backlinks}` per call; node-select detail; orphan + deprecated styling.

**SATISFIES:** `knowledge-graph-view` scenario in `acceptance.feature` (@step-10, G11)

- [ ] **Step 1: Write the failing tests**

```go
func TestKnowledgeGraph_Fixture(t *testing.T) {
    // Fixture bundle: 4 concepts (decision/invariant/workflow/reference),
    //   one orphan, one broken link, one deprecated page, index.md + log.md present
    // Assert: 4 nodes with frontmatter fields; edges only for resolving links;
    //   orphan flagged; broken link tolerated (no error); index/log excluded
}
```

```bash
# UI: render test asserts nodes/edges render and empty bundle renders EmptyState
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic && go test ./internal/knowledge/... -run TestKnowledgeGraph_Fixture -v` (test/derive absent → FAIL); UI test FAIL (page absent).
Expected: FAIL both (name-scoped — the `knowledge` package itself exists for doc/config/sql extractors, so a bare package run would pass).

- [ ] **Step 3: Write minimal implementation**

Derive nodes/edges/backlinks per call (cap note: tens of files; document the bundle-size-cap assumption, paginate later if ever needed); mount the route; build the page reusing `GraphPage` simulation/zoom/select chrome with concept type colors.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic && go test ./internal/knowledge/... -v` + UI `vitest run KnowledgeGraphPage`
Expected: PASS; `go vet` clean on new package. If Task 6 has landed, re-run against the real bundle; else record `fixture-only` in the commit message so tail verification is scheduled, not silently dropped.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/knowledge/ mnemonic/internal/http/server.go
git commit -m "feat(mnemonic): derive read-only knowledge concept graph"
git add skillgrid-ui/src/features/mnemonic/KnowledgeGraphPage.tsx
git commit -m "feat(ui): render knowledge concept graph read-only"
```

---

## Task 11: MCP hardening (descriptions + explicit scope)

> Docs-envelope task (no ranking change; additive response field only). Disjoint files from Task 10 — parallel-safe. No new tools, no lifecycle tools.

**Files:**
- Modify: `mnemonic/internal/mcp/tools_memory.go` (`mem_search` description + scope block on multi-bucket hits)
- Modify (descriptions only): `mnemonic/internal/mcp/tools_secondbrain.go`, `mnemonic/internal/mcp/tools_session_inject.go`
- Test: extend federated/shape tests (scope-block presence + default-unchanged)

**Interfaces:**
- Consumes: existing federated path (`SearchAllProjects`, `project:"all"` contract).
- Produces: scope block (`buckets_hit[]`, per-bucket counts) ahead of merged ranks when ≥2 non-empty buckets hit; progressive-disclosure wording in descriptions.

**SATISFIES:** `explicit-search-scope` scenario in `acceptance.feature` (@step-11, G12)

- [ ] **Step 1: Write the failing test**

```go
func TestMemSearch_ScopeBlock(t *testing.T) {
    // Seed two non-empty project buckets sharing a term
    // all_projects=true → response carries scope block naming both buckets with counts
    // all_projects=false → response byte-identical shape to today (no scope block)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/mcp/... -v -run TestMemSearch_ScopeBlock`
Expected: FAIL (no scope block)

- [ ] **Step 3: Write minimal implementation**

Additive scope block only; ranking untouched; description wording (compact labeled hits; full bodies via get-calls; stop once grounded).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic && go test ./internal/mcp/... -v`
Expected: PASS, existing federated tests (`TestMemSearchAllProjectsFederated` et al.) green UNMODIFIED — any edit to those tests fails this step (re-baseline is forbidden here; shape must be purely additive).

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/mcp/
git commit -m "feat(mnemonic): explicit scope block on multi-bucket search"
```

---

## Task 12: okf-lint.mjs knowledge lint script (+ shadow-posture telemetry)

> Node-only task (no Go, no migration, no new dependency — `yaml` npm package only, per ADR-0008). Follows the existing guard conventions (`state-drift-check.mjs`: read-only, exit 0/1/2, `SCOPE:` line per ADR-0010, `[project-root]` arg). Disjoint files from all other tasks — parallel-safe. Lands before Task 6 executes (Task 6 Step 4 consumes it).
>
> **Telemetry (borrowed from repocodex `metrics.jsonl`):** the lint script appends a JSONL event per run to `.skillgrid/okf-metrics.jsonl` (gitignored; overridable via `OKF_METRICS_SINK` env var). Event shape: `{"event": "lint", "at": <iso-utc>, "drift": <n>, "findings": [...], "false_drift_rate": <0.0-1.0>}`. `false_drift_rate` = share of previously-reported drift findings that were absent on the next run (i.e., false positives). Purpose: measured data trail for promoting the lint gate from WARNING (shadow) to blocking (ratchet) — mirrors repocodex's `shadow`→`ratchet` posture promotion. The telemetry file is NOT memory, NOT a concept, NOT indexed by SQLite — it is an operational artifact that lives or dies with the working tree.

**Files:**
- Create: `.agents/skills/verification/qa/scripts/okf-lint.mjs`
- Create: fixture bundles + runner evidence (e.g. `test/fixtures/okf-clean/`, `test/fixtures/okf-drift/` — co-locate with the script if the repo convention prefers script-local fixtures; keep both fixtures committed)
- Modify: `.gitignore` (add `.skillgrid/okf-metrics.jsonl`)

**Interfaces:**
- Consumes: knowledge roots post-R6 (`decisions/ invariants/ workflows/ glossary/ reference/` + root `index.md`/`log.md`/`reverse-index.md`) with pre-R6 `artifacts/` fallback; working-tree file reads for anchor matching.
- Produces: `DRIFT: clean|drift` + findings table (`check | file | detail`) + `SCOPE: COMPLETE|TRUNCATED|UNSCOPED|UNREADABLE`. Exit 0 clean / 1 drift / 2 usage-parse. Read-only always (no `--fix` flag exists; callers apply repairs).

**SATISFIES:** R6 (tooling half) ↔ @step-12 `okf-lint` (G13)

- [ ] **Step 1: Write the fixture bundles first (red)**

Create `okf-clean` (2–3 concepts, valid frontmatter, complete index, resolving links, live anchors, synced reverse-index) and `okf-drift` (one broken link, one unindexed concept, one broken claim literal, one `stable` page missing `generated`). Record the expected verdicts in a comment header. No script yet — `node .../okf-lint.mjs` errors (missing file), which is the failing state.

- [ ] **Step 2: Implement the seven checks + telemetry sink (green)**

frontmatter-valid → index-complete → links-unbroken (in-bundle targets only, no network) → no-`[[wikilink]]`-syntax (relative Markdown links only; unresolvable `[[ ]]` is a finding) → provenance-required (`status: stable` ⇒ `generated` present, plus `sources` when anchored; `draft` exempt — the agent never self-stamps) → claims-require-verification → anchors-match liveness (`all_of` honoring `near`/`scope_lines`, literals in matched region; report LIVE/DRIFT/CLAIM_BROKEN per anchor) → reverse-index-sync → orphans advisory-only. Layout detection first (post-R6 roots, else `artifacts/` fallback, else UNSCOPED).

**Telemetry:** after the checks complete, append one JSONL event to `.skillgrid/okf-metrics.jsonl` (path overridable via `OKF_METRICS_SINK`). Before appending, read the previous file (if any) and compute `false_drift_rate`: findings reported in the previous run that are ABSENT in this run are false positives; rate = false / total_previous_drift (0.0 if no previous run). The telemetry write is the ONLY side-effect of the script (it is otherwise read-only); a telemetry-write failure is non-fatal (warn to stderr, still exit with the correct code).

- [ ] **Step 3: Run G13 + prove read-only + prove telemetry**

```bash
node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-clean
node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-drift; test $? -eq 1
git status --short test/fixtures/ | wc -l   # must be 0 (no fixture writes)
# telemetry: run twice on the drifted fixture, verify JSONL appends
OKF_METRICS_SINK=/tmp/okf-test-metrics.jsonl node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-drift 2>/dev/null; true
OKF_METRICS_SINK=/tmp/okf-test-metrics.jsonl node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-drift 2>/dev/null; true
wc -l /tmp/okf-test-metrics.jsonl   # must be 2 (two appends)
cat /tmp/okf-test-metrics.jsonl | head -1 | node -e "const d=JSON.parse(require('fs').readFileSync(0)); if(d.event!=='lint'||typeof d.false_drift_rate!=='number') process.exit(1)"
```

Expected: clean exits 0 (`DRIFT: clean`, `SCOPE: COMPLETE`); drifted exits 1 naming all three finding classes; fixtures unmodified; telemetry file has exactly 2 lines, each a valid JSON object with `event: "lint"` and a numeric `false_drift_rate` (0.0 on the first run).

- [ ] **Step 4: Commit (scripts zone)**

```bash
git add .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-*/
git commit -m "feat(skillgrid): add okf-lint knowledge drift guard"
```

---

## Task 13: Teaching graph page (single page, tour, locals)

> UI-heavy vertical slice. Depends on Task 10 payloads (code enrich + R10 concept route) for the full picture; develops fixture-first so it can build in parallel (verify against live routes at the end). Locked: single page with toggles; tour as client-side ordering; locals-expansion via on-demand parse. No new route (the only planned route is Task 10's `GET /knowledge/graph`; node enrich is a payload extension).

**Files:**
- Modify: `mnemonic/internal/http/mnemonic_graph.go` (nodes += `start_line/end_line` + signature/doc if extractor-cheap)
- Create: locals resolver (read-only on-demand parse per symbol → ephemeral params/locals/callsites; co-locate with graph derive or codeintel per author judgment — record the choice in the commit message)
- Modify: `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` (tree panel, enriched inspector, search-to-focus, kind/community/layer filters + newcomer cap, concept toggle with anchor overlay, tour mode)
- Test: Go tests (enrich fields present, locals resolver pure + store-untouched) + vitest render/interaction tests

**Interfaces:**
- Consumes: `/mnemonic/graph/data` (enriched), `/knowledge/graph` (Task 10), `/code/search`, `/code/read`, `?project=` injection, PageRank centrality (Task 4 machinery) for tour order.
- Produces: synced tree/canvas/inspector; client-side tour order (workflow → decisions → top-centrality); ephemeral locals clusters.

**SATISFIES:** `teaching-graph-page` scenario in `acceptance.feature` (@step-13, G14)

- [ ] **Step 1: Write the failing tests**

```go
func TestGraphEnrich_Lines(t *testing.T) {
    // Fixture symbols with known lines; assert payload carries start/end lines
}
func TestLocalsResolver_Ephemeral(t *testing.T) {
    // Resolve locals for a fixture function; assert params/locals/callsites + lines;
    // assert store diff empty (snapshot row counts before/after)
}
```

```bash
# UI: render fixture graph; assert tree↔canvas sync, inspector identity block,
# tour completes ≥3 stops, locals expand/collapse, degraded/empty states render
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic && go test ./internal/... -run 'TestGraphEnrich_Lines|TestLocalsResolver_Ephemeral' -v` + UI test
Expected: FAIL (fields/resolver/page absent)

- [ ] **Step 3: Write minimal implementation**

Enrich payload; pure resolver (parse-on-demand, never writes); tree panel from served file list; typed connection groups with per-edge source lines + preview; search box; filters + newcomer cap; concept toggle + anchor overlay; client-side tour.

- [ ] **Step 4: Run tests to verify they pass**

Run: Go tests + `vitest run GraphPage`
Expected: PASS; `go vet` clean; no new endpoint (route-table diff shows at most the one planned route). If Task 10's payloads have landed, re-run against live routes; else record `fixture-only` in the commit message so tail verification is scheduled, not silently dropped.

- [ ] **Step 5: Commit (zone-split)**

```bash
git add mnemonic/internal/
git commit -m "feat(mnemonic): graph node lines plus ephemeral locals resolver"
git add skillgrid-ui/src/features/mnemonic/GraphPage.tsx
git commit -m "feat(ui): teaching graph page with tree, tour, and concept toggle"
```

---

## Task 14: Contextualized retrieval (prefixes + two-layer + ablation)

> Shares `indexer.go` with Task 4 — implement after TICKET-04 lands. Deterministic only: no LLM, no new dependency, no migration. The ranker is MEASURED, not changed, in this task. Prefix change stales existing vectors — re-index after landing (no migration, no backfill).

**Files:**
- Modify: `mnemonic/internal/codeindex/indexer.go` (prepend `path · symbol · scope` to chunk embedding text; raw lines retained for display)
- Test: `mnemonic/internal/codeindex/indexer_test.go` (prefix presence + raw-text unchanged)
- Read-only: `mnemonic/internal/hybrid/rank.go` (ablation harness runs RRF vs MMR over the eval fixtures; code untouched unless numbers win — a ranker change would be a new ticket, not smuggled in here)

**Interfaces:**
- Consumes: existing chunk window (80/10), tree-sitter symbol scope at chunk time, landed HNSW adapter (`vectorstore/hnsw.go`, ADR-0037) as measured infrastructure.
- Produces: prefixed embedding text; ablation numbers recorded in `findings.md` (code-name-heavy fixture: prefixed vs raw hit@k + latency).

**SATISFIES:** `contextualized-retrieval` scenario in `acceptance.feature` (@step-14, G15)

- [ ] **Step 1: Write the failing test**

```go
func TestChunkPrefix_Context(t *testing.T) {
    // Index a fixture with colliding code names across files
    // Assert embedded text carries path/symbol/scope prefix while display text stays raw
    // Assert prefixed retrieval beats raw on the fixture's code-name queries (eval harness)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mnemonic && go test ./internal/codeindex/... -v -run TestChunkPrefix_Context`
Expected: FAIL (no prefix)

- [ ] **Step 3: Write minimal implementation + record ablation**

Prepend the deterministic prefix at embed time only (stored display text unchanged); run the MMR-vs-RRF ablation over eval fixtures; append the numbers table to `findings.md` (win or lose — negative results are recorded, ranker untouched either way in this ticket).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic && go test ./internal/codeindex/... -v`
Expected: PASS; `go vet` clean; `rank.go` diff empty (prove with `git diff --stat`).

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/codeindex/
git commit -m "feat(mnemonic): deterministic chunk context prefixes plus retrieval ablation"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** All 13 requirements from the briefing have corresponding tasks (R1–R5 ↔ Tasks 1–5; R6 ↔ Tasks 6+12; R7 ↔ Task 7; R8 ↔ Task 8; R9 ↔ Task 9; R10 ↔ Task 10; R11 ↔ Task 11; R12 ↔ Task 13; R13 ↔ Task 14).
- [x] **Must-haves coverage:** Every requirement is mapped to a truth/artifact/link. Briefing criteria 1–6 ↔ behavior truths; criteria 8–14 ↔ recall/promotion/onboarding/graph/scope/teaching/retrieval truths; criterion 7 (test command) covered by Task Steps 2/4; R6 + R8/R9/R10/R11/R12/R13 acceptance scenarios drafted (@step-07..14), executable only after re-gate.
- [x] **One-way-door:** The DB migration in Task 3 is tagged (+ soft backfill note); the LOCKED-index rewrite + ADR-0012 amendment in Task 6 is tagged with a STOP; Task 7's contract ADR needs LOCKED user OK (bundled with re-gate).
- [x] **Placeholder scan:** No "TBD" or "implement later" in Tasks 1–5 (`00N` → `051` resolved); Task 6's TBD acceptance now has @step-07..14 scenarios pending re-gate, not executable before it. Task 12's fixture co-location + Task 13's resolver co-location are decided at authoring with the choice recorded in the commit message.
- [x] **Type consistency:** Signatures match across tasks; Tasks 6/8/9 change no function signatures (Task 6's sweep is path strings only); Task 7 reuses the existing RRF (no new ranking contract); Task 10 adds one route + one pure derive function (no new transport); Task 11 is envelope-additive only; Task 12 follows guard exit-code/output conventions (no new contract); Task 13 adds no endpoint (payload extension + pure resolver + client ordering); Task 14 changes embedding input only (display text + schema untouched, ranker measured-not-changed).
- [x] **Validation:** Tasks 1–5 preserved from the 2026-10-10 VALIDATED version (re-verify byte-identical against HEAD at execution start). Full blueprint (incl. Tasks 6–14) requires critic re-validation + user re-gate before execution.

## Execution Handoff

**"Blueprint rewritten 2026-10-10 (Task 6 added for R6). Re-validation required before execution:**

**0. Re-validate** — critic review + user re-gate (Task 6 is new; acceptance TBD).

**Then two execution options:**

**1. Subagent-Driven (recommended)** - Dispatch a fresh subagent per task.

**2. Inline Execution** - Execute tasks in this session using skillgrid:simple-execution.

**Which approach?"**
