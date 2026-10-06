# Spec — Second-Brain Learnings (from claude-os)

Status: DRAFT · Date: 2026-10-06 · Source: `~/git/ai-test/claude-os` (brobertsaz/claude-os, in sync with origin)

## Context

We compared mnemonic's second-brain architecture against claude-os's and isolated four
concepts that are genuinely adoptable — i.e. things mnemonic does **not** already have and
that improve a "second brain." This spec plans all four. It deliberately **excludes** the
concepts where the clone and mnemonic have already converged (two-phase `lifecycle_log`
audit, union-find dedup clustering, hybrid structural + selective-embedding index) and the
places where mnemonic already leads (AKL decay, bitemporal supersede, typed
`observation_relations`, dedup/consolidation engine). Those are not work items here.

## Ground truth (verified against code)

- `HealthReport` already carries `CountsByType`, `EmbeddingCoverage`, `AgeDistribution`,
  `DuplicateDensity`, and `Recommendations []HealthRecommendation{Severity, Message}` —
  `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go:52`. Recommendation
  *generation* exists (`healthRecommendations`, `lifecycle.go:251`).
- There is **no** time-series growth/period grouping anywhere in the memory store — the only
  `strftime` usages are `'%s'` epoch math for expiry (`memory/lifecycle.go:198`,
  `governance.go:464`). This is the one clear gap vs. claude-os `get_growth_timeline`.
- Embedding in the code index is **eager, per-chunk** (`codeindex/indexer.go:515`
  `WithEmbedder`, "eager dual-granularity embedding pass"). There is **no** centrality /
  PageRank / degree gate deciding *what* to embed.
- Capture has two entry points only: `CapturePassive` + `shapePassiveItem`
  (`memory/service.go:1797`, `:1954`) and `InferType`/`ApplyInfer`
  (`secondbrain/infer.go`). There is **no** confidence scoring and **no**
  prompt-then-confirm gate.
- `observations` carries `type` (13 kinds) + `scope` (project/user/global) but **no**
  durability/knowledge-class dimension. Cross-project recall exists via `all_projects`;
  cross-*class* recall does not.

## Out of scope (converged or mnemonic-led — do not regress)

- Forgetting: AKL importance decay + maturity tiers + dream prune + bitemporal supersede.
- Knowledge-graph linking: `observation_relations` (related / compatible / scoped /
  conflicts_with / supersedes).
- Dedup/consolidation: `mem_lifecycle` `dedup_scan`/`dedup_merge`/`consolidate` +
  `lifecycle_log` two-phase audit.
- Cited answers: `mem_ask` (deterministic cited floor, fail-open LLM).

---

## Feature 1 — Growth timeline (cheapest, clearest win)

**Intent.** A "second brain" should show its own memory *growth over time* — the per-period
`{added, total}` cumulative view claude-os exposes (`get_growth_timeline`), which mnemonic
lacks entirely.

**Behavior.**
- New read path `GrowthTimeline(project, granularity)` where granularity ∈ `day | week |
  month` (default `month`). Returns `[]{period string, added int, total int}` where `total`
  is the running cumulative count of live (non-soft-deleted, non-archived) observations.
- `period` is a stable string key: `%Y-%m-%d` (day), `%Y-Www` (week, ISO week), `%Y-%m`
  (month), derived from `observations.created_at` via SQLite `strftime` (same
  RFC3339-vs-space tolerance as the existing `'%s'` comparisons).
- Soft-deleted (`deleted_at`) and archived (`archived_at`) rows are excluded — growth counts
  *living* memory.

**Surface.**
- Add `mem_lifecycle` action `growth` (param `granularity`), mirroring the existing
  action-enum dispatch. Output errors-as-values per the C4 convention.
- Also fold a one-line trend into `mem_stats` (optional, low priority) so the dashboard
  `HealthReport` can surface "N added this month, total M."

**Files.**
- `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go` — `GrowthTimeline` method +
  `growth` action branch in the `mem_lifecycle` dispatcher.
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` — action enum + param docs.
- Test: `secondbrain/lifecycle_test.go` (synthetic observations across 3+ periods,
  verify cumulative `total` and soft-deleted exclusion).

**Acceptance.**
- `growth` with `granularity=month` over a fixture with rows in Jan/Feb/Mar returns
  `[{Jan, a1, a1}, {Feb, a2, a1+a2}, {Mar, a3, a1+a2+a3}]`.
- A soft-deleted March row does not count toward March `added` nor the cumulative `total`.
- Empty store returns `[]` (not null, not error).
- `pnpm test` / `go test ./skillgrid-cli/internal/mnemonic/...` pass.

---

## Feature 2 — Health-report recommendations (expand, not create)

**Intent.** claude-os's `get_health_report` emits *actionable* `recommendations`
(type + priority + message: dedup / cleanup / stale / embeddings). Mnemonic already has the
`Recommendations` field and a `healthRecommendations` rule function — this feature only
**completes** the rule set so every signal maps to a "what to do next," and adds a
staleness-based recommendation tied to the age buckets.

**Behavior.**
Extend `healthRecommendations` (lifecycle.go:251) with deterministic, ordered rules. Each
returns `{Severity, Message}`; severities are HIGH / MEDIUM / LOW. Rules (only when the
threshold trips):
1. **dedup** — `DuplicateDensity` above a small ε → MEDIUM (HIGH if many near-dup clusters):
   "Found N near-duplicate observations; run `mem_lifecycle dedup_scan`."
2. **embeddings** — `EmbeddingCoverage < 1.0` and embedder expected → MEDIUM: "M
   observations lack embeddings; run `code_index`/re-embed." (No-op when embedder is
   intentionally off — gate on embedder availability so it never naggs.)
3. **stale** — a large fraction of the age bucket `> 90 days` (the "older" bucket) AND total
   > 10 → MEDIUM: "N observations older than 90 days; review for archive."
4. **cleanup** — archived count > 30% of total AND total > 0 → LOW: "N archived observations;
   consider hard-delete."
5. **review** — (new, mnemonic-specific, uses existing `review_cycle`) observations with
   `review_after <= now` > 0 → LOW: "N observations are due for review (`mem_review`)."

Rules are pure functions of the computed metrics (no DB, no LLM) so they stay testable and
fail-open: on probe error `Recommendations` stays `[]` (already enforced by
`TestLifecycle_HealthNeverThrows`).

**Files.**
- `skillgrid-cli/internal/mnemonic/secondbrain/lifecycle.go` — `healthRecommendations` rules.
- Test: `secondbrain/lifecycle_test.go` — one case per rule, threshold-tripped and
  not-tripped; ordering stable.

**Acceptance.**
- A fixture with high duplicate density → a MEDIUM dedup recommendation present.
- A fixture with 0 embedding coverage and an active embedder → embeddings recommendation;
  with embedder off → none.
- A fixture with >50% rows in the >90d bucket and total>10 → stale recommendation.
- All rules off → `Recommendations` is `[]`.
- No new DB queries introduced in the health path beyond the existing probe.

---

## Feature 3 — Durability / knowledge-class dimension + cross-class search

**Intent.** The single most architecturally significant idea from claude-os: separate
knowledge by *temporal nature* (episodic decisions vs. timeless conventions vs. docs), not
just by type. Mnemonic has `type` (kind) and `scope` (project/user/global) but no
*durability class*, and cross-class recall/merge does not exist (only `all_projects`).

**Design.**
- New nullable column `knowledge_class` on `observations` ∈
  `episodic | convention | doc | unclassified` (default `unclassified`). Migration
  `00N_add_knowledge_class.sql`. **Orthogonal** to `type` and `scope`.
- **Backfill (heuristic, non-load-bearing):** `decision, convention, architecture, config,
  standing` → `convention`; `session_log, learning, discovery, bugfix` → `episodic`;
  everything else stays `unclassified`. Backfill is best-effort and re-runnable; a wrong
  backfill is corrected by later explicit sets (never load-bearing).
- `mem_save` / `mem_update` accept optional `knowledge_class` (default: infer from `type`
  using the same heuristic when the caller omits it, else `unclassified`).
- **Retrieval:** `mem_search` / `BlendedSearch` gain an optional `knowledge_class` filter
  (single or set). When a `knowledge_class` filter is set, results are grouped/labeled by
  class so the agent can reason "standing convention" vs. "one-off decision." This is the
  cross-class merge analog of claude-os `query_documents_multi_kb` — but implemented as a
  *dimension on the flat store*, not a separate KB (keeps single-source-of-truth per
  ADR-0012).
- **Boost (optional, behind `mnemonic.improve.enabled`):** on recall, `convention` rows get
  a small additive rank bonus over `episodic` for queries that read as policy/standard
  lookups. Default OFF; measure before enabling.

**Non-goals.**
- Not a separate DB / KB-per-class. Not a rename of `type`. Not a breaking change —
  `knowledge_class` is additive and nullable.

**Files.**
- `skillgrid-cli/internal/mnemonic/store/migrations/00N_add_knowledge_class.sql` — column +
  index `(project, knowledge_class)`.
- `skillgrid-cli/internal/mnemonic/memory/service.go` — Save/Update accept + infer
  `knowledge_class`; backfill helper.
- `skillgrid-cli/internal/mnemonic/memory/search_blend.go` — optional `knowledge_class`
  filter + per-class labeling in results.
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` — param on `mem_save`/`mem_update`/
  `mem_search`.
- ADR: `04-adr-002X-knowledge-class-dimension.md` (additive durability dimension; why not a
  KB-per-class; backfill is advisory).
- Tests: migration up/backfill, save-infer, search filter, result labeling.

**Acceptance.**
- After backfill, a `type:convention` observation reads back `knowledge_class=convention`.
- `mem_search` with `knowledge_class=convention` returns only convention rows, each labeled.
- Omitting `knowledge_class` on `mem_save` infers from `type` (decision→convention,
  session_log→episodic) and never fails.
- All prior `mem_search` results unchanged when no `knowledge_class` filter is passed
  (additive, zero behavior change for existing callers).
- `pnpm typecheck` + `pnpm test` pass.

---

## Feature 4 — PageRank-guided embedding selection (code index)

**Intent.** claude-os's standout efficiency idea: structural index (tree-sitter +
dependency graph + PageRank), then embed **only the top ~20% by PageRank + all docs +
recent changes** — an 80% embedding cut. Mnemonic's code index embeds every AST chunk
eagerly (`codeindex/indexer.go:515`) with no centrality gate.

**Design.**
- Compute a centrality score per symbol/chunk from the existing call/reference graph (in-+
  out-degree, or PageRank over the edges table — reuse whatever `graph_index` already
  builds; do not add a new graph). Rank symbols by score.
- Embedding gate (new `code_index` option `--embed-top <pct>` or
  `codeindex.WithEmbedGate`): when set, the eager embedding pass embeds only the top-`pct`
  symbols by centrality **plus** all doc-bearing chunks and **plus** recently-changed files
  (delta since last index). Default `100` (embed everything — current behavior, so the gate
  is opt-in and backward compatible).
- The gate is **advisory**: a chunk skipped by the gate simply has no vector; the BLOB
  tables remain source of truth and the semantic leg degrades (`degraded=true`) exactly as
  today (ADR-0009). No hard fail.
- Expose the gate decision in index output (e.g. "embedded X of Y symbols; skipped Z below
  centrality cut") so the cost tradeoff is visible.

**Non-goals.**
- Not a new PageRank engine if degree suffices — prefer the cheapest centrality the existing
  graph already exposes. Not a change to FTS5/lexical retrieval (unaffected).

**Files.**
- `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — centrality ranking + embed gate
  in the eager pass.
- `skillgrid-cli/internal/mnemonic/codeindex/graph_index.go` (or existing graph builder) —
  expose degree/PageRank per symbol.
- `codeindex/symbol_embeddings.go` — apply the gate when writing vectors.
- Test: `codeindex/eager_embed_test.go` — with gate=50, only top-centrality symbols get
  vectors; docs + recent changes always embedded; gate=100 reproduces current behavior.

**Acceptance.**
- `--embed-top 50` on a fixture with N symbols embeds ≤ ceil(N/2) code symbols, always
  including doc chunks and recently-changed files.
- Gate at 100 (default) produces the identical set of vectors as today (no regression).
- A symbol skipped by the gate has a NULL vector and its lookup degrades, not errors.
- Index output reports the embedded/skipped counts.
- `go test ./skillgrid-cli/internal/mnemonic/codeindex/...` passes.

---

## Feature 5 — Confidence-gated real-time capture

**Intent.** claude-os watches the conversation, detects 10 named trigger patterns at
75–95% confidence, and saves **only after explicit user confirmation**. Mnemonic's capture
is agent-proactive + passive `Key Learnings` extraction with no confidence score and no
prompt-gate. This adds a *filter*, not a replacement — the agent still leads.

**Design.**
- Deterministic trigger classifier `ClassifyCapture(text) -> {trigger, confidence}` over a
  named pattern set (mirror claude-os): `switching`, `decided_to_use`, `no_longer`,
  `now_using`, `implement_change`, `performance_issue`, `bug_fixed`,
  `architecture_change`, `rejected_idea`, `edge_case`. Confidence is a deterministic score
  (keyword strength + phrase position), **not** LLM — keeps it cheap and testable.
- New tool `mem_capture_suggest(text)`: runs the classifier; if `confidence >= threshold`
  (default 0.75) it returns a *suggested* save (type + draft title + `knowledge_class` from
  Feature 3) **without** persisting — the agent (or user) confirms by calling `mem_save`.
  Below threshold it returns `suggested:false` with no persistence.
- **Gate is advisory:** nothing is saved by `mem_capture_suggest` alone. This is the
  "only save after confirmation" contract made explicit as a tool boundary. The passive
  `Key Learnings:` auto-capture path is unchanged (it already runs at Task-completion and is
  idempotent).

**Non-goals.**
- Not a new Go regex engine that *replaces* the agent's proactive saves. Not an LLM call on
  the hot path. The skill (ADR-0016 "capture as a classifier, not a Go engine") remains the
  primary capture authority; this is an optional deterministic gate the agent may consult.

**Files.**
- `skillgrid-cli/internal/mnemonic/secondbrain/infer.go` (or new `capture.go`) —
  `ClassifyCapture` + threshold.
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` — `mem_capture_suggest` tool +
  registration in the tool table (`tools_memory.go:37`).
- Test: `secondbrain/capture_test.go` — each trigger pattern at high/low confidence;
  suggest-then-save flow; below-threshold returns no suggestion; no row written by suggest.

**Acceptance.**
- `"We're switching from Bootstrap to Tailwind"` → trigger `switching`, confidence ≥ 0.75,
  `suggested:true` with a draft; **no observation row created**.
- `"hey what's up"` → `suggested:false`, no row created.
- After the agent confirms with `mem_save`, exactly one observation exists, with the
  suggested `knowledge_class`.
- `mem_capture_suggest` is idempotent in that it never writes (repeated calls, same result).
- `go test ./skillgrid-cli/internal/mnemonic/...` passes.

---

## Cross-cutting

- **Degradation contract (unchanged):** every new leg is advisory and fail-open. Growth
  timeline, recommendations, the class boost, the embed gate, and the capture gate all
  degrade to today's behavior when their inputs are absent — never a hard fail, never a
  silent degrade (errors-as-values, `_`-prefixed meta per C4).
- **Pure-Go constraints:** no new SQL `regexp`/`sha1` functions (hashes/scores in Go);
  `strftime` only for date bucketing; WAL-locked opens retried with backoff.
- **No new DB:** all five features ride the existing per-project SQLite file; only one new
  nullable column (Feature 3) and one optional index.
- **Testing:** TDD — each feature ships with the listed tests red-then-green; `pnpm test`,
  `pnpm lint`, `pnpm typecheck` before completion.

## Suggested order (dependency-aware)

1. **Feature 1** (growth timeline) — no deps, smallest, highest clarity.
2. **Feature 2** (recommendations) — builds on Feature 1's metrics + existing field.
3. **Feature 3** (knowledge class) — the migration that Feature 5's suggested saves use.
4. **Feature 5** (confidence-gated capture) — reuses Feature 3 for suggested `knowledge_class`.
5. **Feature 4** (PageRank embed gate) — independent of the memory store; can run in
   parallel with 3/5 since it touches only `codeindex`.

Each feature is independently shippable and independently revertable.
