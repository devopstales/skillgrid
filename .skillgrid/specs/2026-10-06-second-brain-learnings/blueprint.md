# Second-Brain Learnings — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** VALIDATED (critic 2026-10-10: VALIDATED-WITH-FIXES, all Critical applied)

**Tier:** T2

**Build shape:** Journey

**Goal:** Implement 5 learnable concepts (Growth, Health, Knowledge Class, PageRank, Confidence Gate) to improve mnemonic's "second brain" capabilities.

**Architecture:** This plan builds five independent features that are integrated into the existing `mnemonic` Go module. Feature 1 (Growth) and Feature 2 (Health) extend the existing `secondbrain` lifecycle. Feature 3 (Knowledge Class) adds a new database dimension for durability. Feature 4 (PageRank) optimizes code indexing. Feature 5 (Confidence Gate) adds a deterministic filter for real-time capture.

**Tech Stack:** Go 1.26 (`mnemonic/go.mod`), SQLite, Tree-sitter, BM25/FTS5

**Spec:** `.skillgrid/artifacts/06-spec-second-brain-learnings.md`

## Terms

- **Second Brain:** Mnemonic's persistent memory system for observations.
- **Knowledge Class:** A new dimension (episodic/convention/doc/unclassified) added in Feature 3.
- **Centrality:** A PageRank/degree score used in Feature 4 to select code chunks for embedding.
- **Capture Gate:** The confidence threshold used in Feature 5 to suggest memory saves.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `mem_lifecycle growth` returns a cumulative timeline of added/total observations per period.
- `mem_lifecycle health` returns actionable recommendations for high duplicate density, low embedding coverage, and stale data.
- `mem_save` infers `knowledge_class` from `type` if not provided (e.g., decision → convention).
- `mem_search` with `knowledge_class=convention` returns only convention-classed observations.
- `code_index` with `--embed-top 50` embeds ≤ ceil(N/2) centrality-selected code symbols *plus* the doc-bearing + recently-changed carve-out; gate=100 (default) reproduces today's vector set; index output reports embedded/skipped counts.
- `mem_capture_suggest` with high-confidence text returns a save suggestion but does not persist it.

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/secondbrain/lifecycle.go` — Contains `GrowthTimeline` and extended `healthRecommendations`.
- `mnemonic/internal/store/migrations/051_add_knowledge_class.sql` — Database migration for Feature 3 (051 claimed by this change; CLM 053–055 / CTX 056+ untouched).
- `mnemonic/internal/memory/service.go` — Handles `knowledge_class` logic in `Save` and `Update`.
- `mnemonic/internal/memory/search_blend.go` — Optional `knowledge_class` filter + per-class labeling.
- `mnemonic/internal/codeindex/indexer.go` — Implements the centrality-based embedding gate (centrality from existing `codeindex/graph.go`; there is no `graph_index.go` on disk).
- `mnemonic/internal/secondbrain/infer.go` — Contains `ClassifyCapture` for Feature 5.
- `mnemonic/internal/mcp/tools_memory.go` — `mem_lifecycle growth`, `mem_save`/`mem_search` `knowledge_class` params, `mem_capture_suggest` registration.
- `artifacts/04-adr-0037-knowledge-class-dimension.md` — ADR for the additive durability dimension (reserve 0037; highest in use is 0036).

**Key links** (critical connections between artifacts that must work together):
- `GrowthTimeline` must exclude soft-deleted and archived rows from the cumulative total.
- `Save` in `memory/service.go` must call the heuristic backfill for `knowledge_class` if the field is empty.
- The `codeindex` embed gate must ensure that doc-bearing and recently-changed chunks are always embedded.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Task 3.1:** Adding a new database column `knowledge_class` to the `observations` table (requires migration).

## Global Constraints

- **Go 1.22+:** Minimum Go version to build (per `05-locked-constraints.md`).
- **No new dependencies:** No new libraries without an ADR.
- **Serial development:** One change at a time (per `05-locked-constraints.md`).
- **Conventional commits:** No AI-attribution trailers.

---

## Task 1: Growth Timeline

**Files:**
- Modify: `mnemonic/internal/secondbrain/lifecycle.go`
- Modify: `mnemonic/internal/mcp/tools_memory.go`
- Test: `mnemonic/internal/secondbrain/lifecycle_test.go`

**Interfaces:**
- Produces: `func GrowthTimeline(project string, granularity string) ([]PeriodCount, error)`

**SATISFIES:** `growth-timeline` scenario in `acceptance.feature` (pending acceptance-test-authoring)

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

**SATISFIES:** `health-recommendations` scenario in `acceptance.feature` (pending acceptance-test-authoring)

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

**SATISFIES:** `knowledge-class` scenario in `acceptance.feature` (pending acceptance-test-authoring)

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

**SATISFIES:** `pagerank-embed` scenario in `acceptance.feature` (pending acceptance-test-authoring)

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

**SATISFIES:** `confidence-capture` scenario in `acceptance.feature` (pending acceptance-test-authoring)

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

## Self-Review Checklist

- [x] **Spec coverage:** All 5 features from the spec have corresponding tasks.
- [x] **Must-haves coverage:** Every spec requirement is mapped to a truth/artifact/link. Briefing criteria 1–6 ↔ 6 truths; criterion 7 (test command) covered by Task Steps 2/4.
- [x] **One-way-door:** The DB migration in Task 3 is tagged (+ soft backfill note).
- [x] **Placeholder scan:** No "TBD" or "implement later" found (`00N` → `051` resolved).
- [x] **Type consistency:** Signatures match across tasks.
- [x] **Validation:** Critic VALIDATED-WITH-FIXES 2026-10-10, all Critical applied (paths → `mnemonic/`, commands → `cd mnemonic && go test ./internal/...`, migration `051`, Task 4 truth carve-out).

## Execution Handoff

**"Blueprint written and committed. Two execution options:**

**1. Subagent-Driven (recommended)** - Dispatch a fresh subagent per task.

**2. Inline Execution** - Execute tasks in this session using skillgrid:simple-execution.

**Which approach?"**
