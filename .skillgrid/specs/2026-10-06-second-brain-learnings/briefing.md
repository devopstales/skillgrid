# Second-Brain Learnings — Design Briefing

> **STATUS:** `draft` (2026-10-06)

**Topic:** 2026-10-06-second-brain-learnings
**Date:** 2026-10-06
**Classification:** risky (DB migration) — T2
**Build shape:** Journey (each feature is a complete user-facing path)
**Reference design:** `~/git/ai-test/claude-os` (brobertsaz/claude-os)

## Problem / Intent

Mnemonic's "second brain" lacks several capabilities present in `claude-os` that improve memory recall, maintenance, and cost-efficiency: a visual growth timeline, actionable health recommendations, a durability dimension for knowledge, an efficient embedding strategy for code, and a confidence-gated capture mechanism. We want to adopt these five concepts to make the second brain more useful and easier to maintain for the user.

## Purpose & Success Criteria

- **Purpose:** Implement 5 learnable concepts to enhance mnemonic's second-brain capabilities.
- **Success criteria (verifiable):**
    1. `mem_lifecycle growth` returns a cumulative timeline of added/total observations per period.
    2. `mem_lifecycle health` returns actionable recommendations for high duplicate density, low embedding coverage, and stale data.
    3. `mem_save` infers `knowledge_class` from `type` if not provided (e.g., decision → convention).
    4. `mem_search` with `knowledge_class=convention` returns only convention-classed observations.
    5. `code_index` with `--embed-top 50` embeds only the top 50% of symbols by centrality.
    6. `mem_capture_suggest` with high-confidence text returns a save suggestion but does not persist it.
    7. `go test ./skillgrid-cli/internal/mnemonic/...` passes.
- **Out of scope:** Replacing existing dedup/consolidation, changing the AKL decay model, or implementing a separate KB-per-class.

## Context

- **Existing flows:** The second brain is implemented in `skillgrid-cli/internal/mnemonic/`.
- **Files:** `secondbrain/lifecycle.go` (health/growth), `memory/service.go` (save/search), `codeindex/indexer.go` (embedding).
- **Constraints:**
    - `per 05-locked-constraints.md`: No new dependencies without an ADR.
    - `per 04-adr-0012-sqlite-as-second-brain.md`: Mnemonic uses a single source of truth (SQLite).
    - `per 04-adr-0009-vector-search-in-sql-latency-viant-deferred.md`: Vector search degrades to BM25 if no embedder.

## Approaches Considered

- **Chosen:** Additive dimensions and advisory gates (Knowledge class, Embed gate). This preserves the single-source-of-truth model and allows for incremental adoption.
- **Rejected:** KB-per-class (would break single source of truth).
- **Rejected:** Replacing the entire capture process with a new engine (too high risk).

## Requirements

1. **Growth Timeline:** Add a time-series growth view to the second brain.
    - **Current:** No time-series growth/period grouping in the memory store.
    - **Target:** `GrowthTimeline(project, granularity)` returns `[]{period, added, total}`.
    - **Acceptance:** `growth` action in `mem_lifecycle` returns cumulative counts for synthetic rows across 3 periods.
    - **Acceptance scenario:** `happy path growth timeline cumulative` → `acceptance.feature`

2. **Health Recommendations:** Expand health report rules to be actionable.
    - **Current:** `healthRecommendations` exists but only has basic rules.
    - **Target:** 5 rules (dedup, embeddings, stale, cleanup, review) with HIGH/MEDIUM/LOW severities.
    - **Acceptance:** Fixtures with high duplicate density or 0 embedding coverage produce the corresponding recommendation.
    - **Acceptance scenario:** `happy path health recommendations actionable` → `acceptance.feature`

3. **Knowledge Class Dimension:** Add a durability class to observations.
    - **Current:** `observations` has `type` and `scope` but no `knowledge_class`.
    - **Target:** New nullable column `knowledge_class` (`episodic`, `convention`, `doc`, `unclassified`).
    - **Acceptance:** After backfill, a `type:convention` observation reads back `knowledge_class=convention`.
    - **Acceptance scenario:** `happy path knowledge class infer and search` → `acceptance.feature`

4. **PageRank-guided Embedding:** Use centrality to select code chunks for embedding.
    - **Current:** Eager, per-chunk embedding in `codeindex/indexer.go:515`.
    - **Target:** `--embed-top <pct>` flag; only top-centrality symbols are embedded.
    - **Acceptance:** `--embed-top 50` on N symbols embeds ≤ ceil(N/2) code symbols.
    - **Acceptance scenario:** `happy path pagerank embed gate` → `acceptance.feature`

5. **Confidence-gated Capture:** Add a deterministic filter for real-time capture.
    - **Current:** Agent-proactive capture; no confidence scoring.
    - **Target:** `ClassifyCapture(text)` and `mem_capture_suggest` tool.
    - **Acceptance:** "We're switching from X to Y" returns `suggested:true` with no DB row created.
    - **Acceptance scenario:** `happy path confidence gated capture` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
    - `secondbrain/lifecycle.go`: Add `GrowthTimeline` and expand `healthRecommendations`.
    - `memory/service.go`: Handle `knowledge_class` in `Save`, `Update`, and `Search`.
    - `codeindex/indexer.go`: Implement `--embed-top` and centrality ranking.
    - `secondbrain/infer.go`: Add `ClassifyCapture`.
    - `store/migrations/00N_add_knowledge_class.sql`: Add the new column.
- **Interfaces:**
    - `func GrowthTimeline(project string, granularity string) ([]PeriodCount, error)`
    - `func ClassifyCapture(text string) (trigger string, confidence float64)`
    - `KnowledgeClass` field on `observation` structs.
- **Data flow:**
    - `mem_save` → `Save` → `inferKnowledgeClass` → SQLite.
    - `code_index` → `Index` → `CentralityRank` → `EmbedGate` → Vectors.
- **Error handling:**
    - All new legs are advisory and fail-open (per ADR-0009).
    - `mem_capture_suggest` never fails; it returns `suggested:false` if below threshold.

## Testing Decisions

- **What makes a good test:** Only test external behavior (e.g., the output of `mem_lifecycle growth`).
- **Modules to test:** `secondbrain`, `memory`, `codeindex`.
- **Prior art:** `lifecycle_test.go`, `service_test.go`.
- **Edge cases:** Empty stores, soft-deleted rows, centrality ties, and below-threshold captures.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: Add "Second-Brain Learnings" to the feature list.
- `.skillgrid/ASSUMPTIONS.md`: None.
- `.skillgrid/ARCHITECTURE.md`: Note the `knowledge_class` dimension in the `observations` table description.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.95  | 0.75 | OK     | Very clear, based on claude-os.    |
| Boundary Clarity    | 0.85  | 0.70 | OK     | 5 distinct features.               |
| Constraint Clarity  | 0.80  | 0.65 | OK     | Additive, no new deps.             |
| Acceptance Criteria | 0.90  | 0.70 | OK     | Specific behaviors for each feature.|
| **Clarity**         | 0.05  | ≤0.20| OK     |                                    |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Which claude-os features?| 5 concepts: Growth, Health, KC, PR, Confidence Gate. |
| 2     | Is it a new DB?          | No, additive dimension on SQLite.  |

## Open Questions & Assumptions

- **Assumption:** The `knowledge_class` backfill heuristic is non-load-bearing.
- **Assumption:** Centrality (PageRank) is sufficient for the embedding gate; no new graph engine needed.

## Decisions (ADR)

- `04-adr-0025-knowledge-class-dimension.md` (Planned): Document the addition of the `knowledge_class` dimension.

## Terms

- **Second Brain:** See `.skillgrid/artifacts/01-business-terms.md`.
- **Knowledge Class:** A new dimension added in this change.
