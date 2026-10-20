---
type: Spec
title: "2026-10-06-second-brain-learnings — briefing"
status: draft

---

# Second-Brain Learnings — Design Briefing (requirements & intent)

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
    7. `cd mnemonic && go test ./internal/...` passes.
    8. `mem_search` returns labeled `web_cache` hits alongside observations; expired cache rows never surface.
    9. `reflect` proposes web-cache promotion candidates in `report.md` without writing concepts.
    10. Onboarding re-runs consult the knowledge index + memory before re-detecting; brownfield enroll yields draft candidates.
    11. The knowledge bundle renders as a read-only concept graph (orphans and deprecated distinct; broken links tolerated).
    12. Multi-bucket search carries an explicit scope block; default single-bucket output is byte-identical to today.
    13. One graph page teaches the repo: synced tree/canvas/inspector, guided tour, concept toggle, on-demand locals.
    14. Chunks carry deterministic context prefixes; two-layer rank formalized; MMR-vs-RRF decided by ablation numbers.
- **Out of scope:** Replacing existing dedup/consolidation, changing the AKL decay model, implementing a separate KB-per-class, adding openwiki-style generation skills (skills stay follow-up; see below), or adopting repocodex's standalone `repocodex write` 7-check write gate as a separate CLI subcommand (the gate *checks* — distinctiveness, identity prefix — inform our `okf-lint.mjs` rules; our OKF writes go through the existing skill pipeline). Note: R6 (this change) DOES adopt the repocodex `.context/`-equivalent layout as `.skillgrid/`-native OKF folders — the exclusion applies to importing the repocodex Python package as a dependency, not to the layout/pattern.

## Context

- **Existing flows:** The second brain is implemented in `mnemonic/internal/` (standalone module; was `skillgrid-cli/internal/mnemonic/` pre-restructure).
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

7. **Fused Recall:** `mem_search` always searches observations + web_cache.
    - **Current:** `mem_search` queries `observations_fts` only (vector leg optional, observations-only); web_cache reachable solely via `web_cache_search`/`web_cache_get`.
    - **Target:** Every `mem_search` also MATCHes `web_cache_fts` (project-scoped, expired rows excluded), fused via the existing RRF; web hits labeled `corpus: web_cache` + source/url/fetched_at/expires_at + retrieve pointer; vector leg stays observations-only; tool contract recorded in a new ADR (additive).
    - **Acceptance:** Seeded web row appears labeled in fused results; expired rows never surface; observations-only behavior otherwise unchanged.
    - **Acceptance scenario:** `fused recall labeled` → `acceptance.feature`

8. **Reflect-time Promotion:** `reflect` distills this change's web_cache entries into concept candidates for human confirmation.
    - **Current:** Web findings enter only via TTL'd cache rows; per-change findings stay in `specs/` until `reflect` lifts durable bits to `06-research-findings.md`.
    - **Target:** New `reflect` step: select this change's cache entries (`session_id` match or fetched-at window + topic tags), distill each into a candidate card (concept home, why, `sources:` entry, claim text) under `report.md` `## Web-cache promotion candidates` + Open Items; `reflect` never writes concepts itself; confirmation explicit before any write (same-edit rule on confirm); unconfirmed candidates expire harmlessly.
    - **Acceptance:** `reflect` skill contains the promotion step + template section; every candidate cites cache id + URL.
    - **Acceptance scenario:** `promotion candidates proposed` → `acceptance.feature`

9. **Onboarding Recall + Bootstrap:** Onboarding invokes knowledge-base and mnemonic before detecting facts.
    - **Current:** Merge re-runs re-detect without reading recorded knowledge; brownfield enroll captures conventions into block + glossary but authors no pinned concepts.
    - **Target:** Step 1 recall-before-detect (`index.md` + `log.md` tail + `mem_context`/`mem_search` on re-runs; noted skip on fresh); Step 1.5b bootstrap (enrolled conventions → `status: draft` candidates with `verification.anchors`, confirm-one-at-a-time, unconfirmed dropped); `skillgrid init --docs` ingest unchanged.
    - **Acceptance:** Onboarding skill contains recall + bootstrap steps and verification items; skill hygiene check clean for touched skills.
    - **Acceptance scenario:** `onboarding recall and bootstrap` → `acceptance.feature`

10. **Knowledge-concept graph view:** The bundle renders as a derived, read-only concept graph in the dashboard.
    - **Current:** D3 force-layout code graph exists (`GraphPage.tsx` over `GET /mnemonic/graph/data`); concept links live only as Markdown text with no rendered view.
    - **Target:** New derive-on-demand function over the post-R6 knowledge folders (frontmatter + Markdown links → nodes/edges/backlinks; `index.md`/`log.md`/reverse-index excluded; broken links tolerated); `GET /knowledge/graph` open read route (existing ServeMux + loopback conventions); new `KnowledgeGraphPage` reusing the code-graph renderer with concept type colors, orphan + deprecated distinction, node-select detail. No store, no export, no editable graph.
    - **Acceptance:** Fixture bundle renders the expected nodes/edges; orphan flagged; broken link tolerated; empty bundle renders empty state, not error.
    - **Acceptance scenario:** `knowledge graph view` → `acceptance.feature`

11. **MCP hardening (descriptions + explicit scope):** Recall tools contract progressive disclosure; multi-bucket search declares its scope.
    - **Current:** `mem_search`/`mem_get_observation`/`code_search` descriptions don't forbid preloading; `all_projects=true` silently merges cross-project ranks (federated `project:"all"` contract only).
    - **Target:** Description wording (compact labeled hits; full bodies via get-calls only; stop once grounded); explicit `scope` block (`buckets_hit[]`, per-bucket counts) ahead of merged ranks when ≥2 non-empty buckets hit; default single-bucket output unchanged; no new tools, no lifecycle tools.
    - **Acceptance:** Existing federated tests pass unmodified; multi-bucket response carries the scope block; descriptions carry the wording.
    - **Acceptance scenario:** `explicit search scope` → `acceptance.feature`

12. **Teaching graph page:** One dashboard page combines the code graph, the concept graph, and repo-layout browsing so a newcomer understands the project. Locked UI decisions: single page with toggles; tour as client-side ordering; locals-expansion via on-demand parse.
    - **Current:** `GraphPage.tsx` renders a 500-node force graph with a thin inspector (label/type/file/caller-lists); the served file list is never displayed; no search-to-focus, no layers, no concept nodes, no tour.
    - **Target:** Tree panel (served file list as dir→file→symbols, synced both ways with the canvas) + enriched inspector (identity with lines + breadcrumb, connections grouped by edge kind with per-edge source file:line + code preview, concept links) + search-to-focus (existing `/code/search`) + layer/newcomer-cap filters + concept toggle with anchor-edge overlay (R10 payload) + tour mode (workflow → decisions → top-centrality symbols, client-side order, doubles as injectable L1 context) + locals expansion (read-only on-demand parse per symbol: params/locals/callsites as ephemeral children, never stored). Backend: node += lines (+ signature/doc if cheap), bridge edges derived at compose time, optional `GET /code/file-symbols` only if client-side grouping strains.
    - **Acceptance:** Fixture repo exercises every behavior incl. ephemeral locals (store diff empty) and all degraded states.
    - **Acceptance scenario:** `teaching graph page` → `acceptance.feature`

13. **Contextualized retrieval:** Deterministic chunk prefixes + two-layer rank + MMR ablation (BYO ingestion lesson, minus the LLM).
    - **Current:** Chunks are raw joined lines (`indexer.go:281`, no prefix); single-level RRF fusion (RRFK=60); pure-Go HNSW adapter landed (`vectorstore/hnsw.go`, migration 057, ADR-0037) but the rank path is unevaluated against alternatives.
    - **Target:** Prepend `path · symbol · scope` to chunk embedding text (deterministic, no LLM, near-zero cost); formalize two-layer rank in the search router (symbols/files first, drill into chunks); run an MMR-vs-RRF ablation on the eval harness and change the ranker only on winning numbers (recorded in `findings.md`); LLM-written section summaries and OpenAI embeddings explicitly out (cost + local-first).
    - **Acceptance:** Prefixed chunks beat raw on a code-name-heavy fixture; ablation numbers recorded; no ranker change without them.
    - **Acceptance scenario:** `contextualized retrieval` → `acceptance.feature`

6. **Knowledge-base restructure (.skillgrid as OKF wiki):** Dissolve the flat
   `artifacts/` folder into top-level knowledge folders under OKF v0.2, and
   rename `archive/` to `xarchive/`.
    - **Current:** Flat `artifacts/` — PRD, glossaries, 36 ADR files,
      constraints, research, and roadmaps mixed under numeric prefixes.
    - **Target:** Per `okf-wiki-plan.md` (same dir): `decisions/` (was
      `artifacts/04-adr-*.md`), `invariants/` (split from
      `artifacts/05-locked-constraints.md`), `workflows/`, `glossary/`,
      `reference/` at `.skillgrid/` root; root `index.md` (was
      `artifacts/README.md`), `log.md`, generated `reverse-index.md`;
      `artifacts/` removed; `archive/` renamed to `xarchive/` (`git mv`,
      history preserved); OKF v0.2 frontmatter for all `.skillgrid/**/*.md`
      except YAML (`config.yaml`, `state.yaml` untouched); hybrid SoT (OKF
      bundle durable, SQLite retrieval index) via a new ADR amending ADR-0012;
      pins via borrowed pattern enforced by `okf-lint.mjs` (TICKET-11),
      no new dependency (Node `yaml` package only, per ADR-0008);
      reference sweep over every
      in-repo `artifacts/`/`archive/` path (config, sdd-structure, skills,
      hooks, dashboard).
      **Telemetry (borrowed from repocodex `metrics.jsonl`):** shadow-posture
      telemetry sink — append-only JSONL of lint/validate events (drift
      detected, anchors re-anchored, false positives resolved) with a
      computed `false_drift_rate` field. Gitignored, overridable via env
      var, not memory. Purpose: data trail that justifies promoting a
      shadow gate (WARNING) to ratchet (blocking) based on measured
      precision, not gut feeling.
    - **Acceptance:** TBD — requires a spec-zone delta (blueprint task +
      acceptance Gate + sliced ticket) and a re-gate before execution. Must not
      break the R1–R5 traceability chain or the ADR IRON RULE (`git mv` only).

## Implementation Decisions

- **Modules to build/modify:**
    - `secondbrain/lifecycle.go`: Add `GrowthTimeline` and expand `healthRecommendations`.
    - `memory/service.go`: Handle `knowledge_class` in `Save`, `Update`, and `Search`.
    - `codeindex/indexer.go`: Implement `--embed-top` and centrality ranking.
    - `secondbrain/infer.go`: Add `ClassifyCapture`.
    - `store/migrations/051_add_knowledge_class.sql`: Add the new column.
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

- `.skillgrid/reference/00-prd.md`: Add "Second-Brain Learnings" to the feature list.
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

## Follow-up Proposals (out of scope for this change)

Recorded here so the follow-up work starts from pointers, not re-derivation.
The restructure (R6 above) is a GOAL of this change — it is listed here only
until its spec-zone delta (blueprint task + acceptance Gate + sliced ticket)
lands and the change re-gates. TICKET-01..05 stay frozen.

- **OKF wiki restructure — GOAL (R6), spec-zone delta pending:**
  `okf-wiki-plan.md` (same dir) — dissolve `artifacts/` into top-level
  `decisions/`/`invariants/`/`workflows/`/`glossary/`/`reference/`,
  `archive/` → `xarchive/`, hybrid SoT amending ADR-0012
  via the next free ADR number, borrow-pattern pins with no new dependency.
  User-confirmed Q&A (2026-10-10, layout revised same day): OKF v0.2,
  hybrid SoT, dissolve-`artifacts/` scope, native pins.
- **Consolidated evidence:** `findings.md` — `## Research: repocodex`
  (anchor liveness, file-path retrieval, coverage ratchet, shadow posture,
  closed blocking set, posture semantics, write gate, metrics.jsonl
  telemetry, validate envelope shape — per `docs/architecture.md`),
  `## Research: vector-DB-demotion video` (BM25-first verdict; compiled-wiki
  cost caution: ~100× build, ~21×/query — bundle stays small/curated),
  `## Research: openwiki deep-dive` (page-job pipeline, ownership split,
  disposable FTS ranker, 4 proposed skills).
- **Proposed follow-up skills:** `knowledge-authoring`, `knowledge-retrieval`
  (via `mnemonic`), `knowledge-lint` (shadow/WARNING-first), `knowledge-diagrams`.
- **Open point for the follow-up briefing:** claim sidecars (openwiki-style)
  vs frontmatter-embedded `verification.anchors` (repocodex-style); default
  frontmatter-embedded unless the bundle exceeds ~100 concepts.
- **Design constraint carried forward:** knowledge bundle size cap — curated
  tens of files, no auto-compiled corpus dumps (per-query cost stays flat via
  index-lookup + one-hop reads).

## Decisions (ADR)

- `04-adr-0038-knowledge-class-dimension.md` (Planned): Document the addition of the `knowledge_class` dimension.

## Terms

- **Second Brain:** See `.skillgrid/glossary/01-business-terms.md`.
- **Knowledge Class:** A new dimension added in this change.
