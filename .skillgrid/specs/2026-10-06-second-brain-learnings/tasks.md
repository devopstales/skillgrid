# Tasks — 2026-10-06-second-brain-learnings (sliced)

> **STATUS:** `sliced` (2026-10-10)

**Change:** 2026-10-06-second-brain-learnings
**Tier:** T2 · **Build shape:** Journey · **Delivery:** single PR, 5 tickets, Medium ~600-line risk
**Chain strategy:** single-branch (current branch; no stacked PRs)
**Traceability:** briefing.md R1–R5 ↔ blueprint.md Tasks 1–5 ↔ acceptance.feature @step-01..06 (G1–G7)

## Waves

| Wave | Tickets | Rationale |
|------|---------|-----------|
| 1 | TICKET-01 (growth) + TICKET-04 (embed gate, independent codeindex) | No deps, parallel-safe |
| 2 | TICKET-02 (health rules, builds on TICKET-01 metrics) | Needs growth metrics shape stable |
| 3 | TICKET-03 (knowledge-class migration 051 + ADR 0037) ⚠ one-way | STOP for user approval before Step 3 (migration + backfill; snapshot first) |
| 4 | TICKET-05 (capture gate, reuses TICKET-03 class) + full regression G7 | Sequential after TICKET-03 |

## Tickets

### TICKET-01: Growth timeline
- **Files:** `mnemonic/internal/secondbrain/lifecycle.go`, `mnemonic/internal/mcp/tools_memory.go`, `mnemonic/internal/secondbrain/lifecycle_test.go`
- **Depends:** none
- **SATISFIES:** @step-01 `growth-timeline` (G1)
- **Do:** `GrowthTimeline(project, granularity day|week|month)` via `strftime` bucketing over `created_at`; exclude `deleted_at`/`archived_at`; `mem_lifecycle growth` action; empty store → `[]`.
- **Verify:** `cd mnemonic && go test ./internal/secondbrain/... -run TestGrowthTimeline_Monthly -v` PASS
- [ ] done

### TICKET-02: Health recommendations (5 rules)
- **Files:** `mnemonic/internal/secondbrain/lifecycle.go`, `mnemonic/internal/secondbrain/lifecycle_test.go`
- **Depends:** TICKET-01 (metrics shape)
- **SATISFIES:** @step-02 `health-recommendations` (G2)
- **Do:** Extend pure `healthRecommendations` (call directly in tests, not `Health()`, to avoid 24h cache): dedup HIGH density>0.05, embeddings MEDIUM with embedder-off no-nag gate, stale MEDIUM total>10 + >90d fraction, cleanup LOW archived>30%, review LOW `review_after<=now`. Stable ordering, no new DB queries.
- **Verify:** `cd mnemonic && go test ./internal/secondbrain/... -run TestHealthRecommendations -v` PASS
- [ ] done

### TICKET-03: Knowledge-class dimension + ADR ⚠ ONE-WAY
- **Files:** `mnemonic/internal/store/migrations/051_add_knowledge_class.sql` (NEW), `mnemonic/internal/memory/service.go`, `mnemonic/internal/memory/search_blend.go`, `mnemonic/internal/mcp/tools_memory.go`, `.skillgrid/artifacts/04-adr-0037-knowledge-class-dimension.md` (NEW), `mnemonic/internal/memory/service_test.go`
- **Depends:** none (migration reservation 051; CLM 053–055 / CTX 056+ untouched)
- **SATISFIES:** @step-03 `knowledge-class` (G3, G4)
- **Do:** Nullable `knowledge_class episodic|convention|doc|unclassified` + index `(project, knowledge_class)`; `inferKnowledgeClass(type)` on save/update; re-runnable heuristic backfill (decision/convention/architecture/config/standing→convention; session_log/learning/discovery/bugfix→episodic); `mem_search` optional filter + per-class labels; ADR-0037 (why dimension not KB-per-class).
- **Reversibility:** one-way (ADD COLUMN, no down-migrations; runner applies once). Soft one-way: backfill mutates rows — snapshot/backup before applying.
- **Verify:** `cd mnemonic && go test ./internal/memory/... -run TestSave_KnowledgeClassInfer -v` PASS; migration file present; unfiltered search unchanged
- [ ] done — **STOP: user approval required before migration apply**

### TICKET-04: PageRank-guided embed gate
- **Files:** `mnemonic/internal/codeindex/indexer.go`, `mnemonic/internal/codeindex/graph.go` (expose degree/PageRank; no `graph_index.go`), `mnemonic/internal/codeindex/indexer_test.go`
- **Depends:** none (codeindex-only, parallel with 01–03)
- **SATISFIES:** @step-04 `pagerank-embed` (G5)
- **Do:** Centrality rank from existing call/reference graph; `--embed-top <pct>` / `WithEmbedGate`, default 100 (today's behavior); gate=50 → ≤ceil(N/2) centrality-selected + docs + recent-changes carve-out; NULL vector for skipped (degrade per ADR-0009); report embedded/skipped counts.
- **Reversibility:** fully reversible (runtime flag, no schema).
- **Verify:** `cd mnemonic && go test ./internal/codeindex/... -run TestIndex_EmbedTop -v` PASS
- [ ] done

### TICKET-05: Confidence-gated capture + regression
- **Files:** `mnemonic/internal/secondbrain/infer.go` (or `capture.go`), `mnemonic/internal/mcp/tools_memory.go`, `mnemonic/internal/secondbrain/capture_test.go`
- **Depends:** TICKET-03 (suggested `knowledge_class`)
- **SATISFIES:** @step-05 `confidence-capture` (G6) + @step-06 regression (G7)
- **Do:** Deterministic `ClassifyCapture(text)->{trigger,confidence}` (10 named patterns, keyword+position score, no LLM); `mem_capture_suggest` threshold 0.75 default, returns draft without persisting; agent confirms via `mem_save`; idempotent never-writes. Then full `cd mnemonic && go test ./internal/...` green.
- **Verify:** `cd mnemonic && go test ./internal/secondbrain/... -run TestCaptureSuggest -v` PASS; `cd mnemonic && go test ./internal/...` PASS
- [ ] done

## Execution notes
- TDD per ticket: failing test → minimal impl → green → commit `feat(mnemonic): ...` (no AI trailers).
- Edit scope guard: files outside the ticket's Files set → halt + checkpoint.
- Blocked format: record `blocked(<phase>): <reason>` in state.yaml notes.
