# Second-Brain Learnings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Journey

**Goal:** Implement 5 learnable concepts (Growth, Health, Knowledge Class, PageRank, Confidence Gate) to improve mnemonic's "second brain" capabilities.

**Architecture:** This plan builds five independent features that are integrated into the existing `mnemonic` Go module. Feature 1 (Growth) and Feature 2 (Health) extend the existing `secondbrain` lifecycle. Feature 3 (Knowledge Class) adds a new database dimension for durability. Feature 4 (PageRank) optimizes code indexing. Feature 5 (Confidence Gate) adds a deterministic filter for real-time capture.

**Tech Stack:** Go 1.25, SQLite, Tree-sitter, BM25/FTS5

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
- `code_index` with `--embed-top 50` embeds only the top 50% of symbols by centrality.
- `mem_capture_suggest` with high-confidence text returns a save suggestion but does not persist it.

**Artifacts** (files that must exist with real implementation, not stubs):
- `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go` — Contains `GrowthTimeline` and extended `healthRecommendations`.
- `skillgrid-cli/internal/mnemonic/store/migrations/00N_add_knowledge_class.sql` — Database migration for Feature 3.
- `skillgrid-cli/internal/mnemonic/memory/service.go` — Handles `knowledge_class` logic in `Save` and `Update`.
- `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — Implements the centrality-based embedding gate.
- `skillgrid-cli/internal/mnemonic/secondbrain/infer.go` — Contains `ClassifyCapture` for Feature 5.

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
- Modify: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Test: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle_test.go`

**Interfaces:**
- Produces: `func GrowthTimeline(project string, granularity string) ([]PeriodCount, error)`

**SATISFIES:** `growth-timeline` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestGrowthTimeline_Monthly(t *testing.T) {
    // Setup DB with synthetic rows for Jan, Feb, Mar
    // Call GrowthTimeline
    // Verify cumulative total
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/... -v -run TestGrowthTimeline_Monthly`
Expected: FAIL with "undefined: GrowthTimeline"

- [ ] **Step 3: Write minimal implementation**

Implement `GrowthTimeline` using a `strftime` based bucketing query in `lifecycle.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/... -v -run TestGrowthTimeline_Monthly`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/
git commit -m "feat(mnemonic): add growth timeline to secondbrain lifecycle"
```

---

## Task 2: Health Recommendations

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go`
- Test: `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle_test.go`

**Interfaces:**
- Consumes: `HealthReport` struct from `lifecycle.go`

**SATISFIES:** `health-recommendations` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestHealthRecommendations_HighDensity(t *testing.T) {
    // Create fixture with high duplicate density
    // Call healthRecommendations
    // Assert MEDIUM severity dedup recommendation
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/... -v -run TestHealthRecommendations_HighDensity`
Expected: FAIL (recommendation missing)

- [ ] **Step 3: Write minimal implementation**

Extend `healthRecommendations` with the 5 rules defined in the spec.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/secondbrain/
git commit -m "feat(mnemonic): expand health recommendations for secondbrain"
```

---

## Task 3: Knowledge Class Dimension

> ⚠ **one-way:** Adding a database column `knowledge_class` to `observations`. STOP for user approval.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/00N_add_knowledge_class.sql`
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go`
- Modify: `skillgrid-cli/internal/mnemonic/memory/search_blend.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/service_test.go`

**Interfaces:**
- Produces: `KnowledgeClass` field on `observation` structs.
- Produces: `inferKnowledgeClass` helper function.

**SATISFIES:** `knowledge-class` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestSave_KnowledgeClassInfer(t *testing.T) {
    // Save a 'decision' type observation without knowledge_class
    // Verify it is read back as 'convention'
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/... -v -run TestSave_KnowledgeClassInfer`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Add the migration and the `inferKnowledgeClass` logic in `service.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/
git commit -m "feat(mnemonic): add knowledge class dimension to observations"
```

---

## Task 4: PageRank-guided Embedding

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go`
- Test: `skillgrid-cli/internal/mnemonic/codeindex/indexer_test.go`

**Interfaces:**
- Produces: `--embed-top` CLI flag.

**SATISFIES:** `pagerank-embed` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestIndex_EMBEDTop(t *testing.T) {
    // Index a fixture with N symbols
    // Verify only top-centrality symbols have vectors
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/codeindex/... -v -run TestIndex_EMBEDTop`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement centrality ranking and the embedding gate in `indexer.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/codeindex/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/codeindex/
git commit -m "feat(mnemonic): add PageRank-guided embedding selection"
```

---

## Task 5: Confidence-gated Capture

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/secondbrain/infer.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Test: `skillgrid-cli/internal/mnemonic/secondbrain/capture_test.go`

**Interfaces:**
- Produces: `ClassifyCapture(text string) (trigger string, confidence float64)`
- Produces: `mem_capture_suggest` MCP tool.

**SATISFIES:** `confidence-capture` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestCaptureSuggest_HighConfidence(t *testing.T) {
    // Call suggest with "We're switching from X to Y"
    // Verify suggested:true and NO DB row created
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/secondbrain/... -v -run TestCaptureSuggest_HighConfidence`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Add `ClassifyCapture` and the `mem_capture_suggest` tool registration.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/
git commit -m "feat(mnemonic): add confidence-gated real-time capture"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** All 5 features from the spec have corresponding tasks.
- [ ] **Must-haves coverage:** Every spec requirement is mapped to a truth/artifact/link.
- [ ] **One-way-door:** The DB migration in Task 3 is tagged.
- [ ] **Placeholder scan:** No "TBD" or "implement later" found.
- [ ] **Type consistency:** Signatures match across tasks.

## Execution Handoff

**"Blueprint written and committed. Two execution options:**

**1. Subagent-Driven (recommended)** - Dispatch a fresh subagent per task.

**2. Inline Execution** - Execute tasks in this session using skillgrid:simple-execution.

**Which approach?"**
