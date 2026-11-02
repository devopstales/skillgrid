# 005 — Query (the deep retrieval, wiki-first)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `004-index` (the FTS cache it searches, the `pin` state it surfaces),
`001-structure` (the page format it returns), the memory change (`01-memory/005-lookup`
— the `mem_lookup` fallback it hands off to; the shared retrieval-trail store it records
into, INV-5), `006-validate` (the pin-check it can gate on).
**Reference:** pi-llm-wiki two-stage links-first recall, openwiki cited sections, the
wiki-first/memory-fallback decision (OPEN-12).

---

## 1. Capability

`kb_query` is the **deep retrieval** half of the retrieval model. Per the memory↔wiki
model (`000-architecture` §3), the agent's *understanding* retrieval — "what do we know
about X" — goes to the KB (wiki-first). `kb_query` answers with **cited markdown
sections**, not raw facts. The raw-fact half (`mem_lookup` / `mem_search`) is the
**fallback**, used only when the KB is thin/empty on the topic (OPEN-12).

The retrieval is **progressive and token-efficient** (pi's two-stage pattern):

```
1. kb_query "query"            → stage 1: ranked LINKS + 1-line snippets (protects context)
2. kb_read "type/slug"         → stage 2: the full page(s) that matter
```

Plus the two grounding tools: `kb_search` (the raw FTS, for power users) and the
pin-check gate (`kb_validate`, `006-validate`) that can run before the agent relies on
a grounded page.

## 2. Stage 1 — `kb_query` (ranked links + snippets)

`kb_query(query, filters)` runs FTS over `kb_pages_fts` (`004-index`) and returns
**stage-1 results** — the cheap, context-protecting layer:

- **Filters:** `type` (page type), `project`, `limit` (default 10), `match_mode`
  (`all`/`any`), `grounded_only` (only decision/invariant/workflow pages).
- **Rank:** BM25 over `(title, body, type, rel_path)` with KB-specific boosts:
  - **groundedness +0.15** (grounded pages — decisions/invariants/workflows — rank
    above pure context pages; the "why" is the point),
  - **freshness** (a small recency weight on `kb_pages.updated_at`),
  - **citation depth +0.05** (a page cited by many other pages — high in-degree backlink
    count — is more central). The in-degree is read from **`kb_pages.in_degree`** (a
    cached count rebuilt by `kb_index`, `004-index` §3) — a *count* cache, not a stored
    backlink *graph* (the listed backlinks in `kb_read` are still computed on the fly,
    §5).
- **Projection (stage 1):** per result — `rel_path` (the wikilink target, e.g.
  `decisions/auth-model`), `title`, `type`, `status`, `grounded` flag, a **1-line
  snippet** (~120 chars around the best match), and a **`pin`** field (present only for
  grounded pages: `fresh` / `stale` / `unresolved` / `null`). **Never hydrates the full
  body** at stage 1 (that is stage 2).
- **Excludes** soft-deleted pages (`kb_pages.deleted_at`) **and `deprecated` pages**
  (`kb_pages.status = 'deprecated'`; `draft` pages are included unless the caller
  filters them, `004-index` §3).

**Stage-1 row shape** (the machine contract; non-grounded pages omit `pin`):

```json
{
  "rel_path": "decisions/auth-model",
  "title": "Auth model is token-based, not session",
  "type": "decision",
  "status": "stable",
  "grounded": true,
  "snippet": "…token rotation is pinned invariants/token-rotation…",
  "pin": "fresh",                 // grounded only: fresh | stale | unresolved | null(never validated)
  "score": 42.7
}
```

> **The `pin` field's source (the read-time question resolved):** `kb_query` does **not**
> run the full pin-check per result (that would make query as expensive as validate).
> Instead it reports the **last recorded** pin state: `fresh`/`stale`/`unresolved` from
> the page's most recent `kb_validate` (recorded in the `verified:` frontmatter event +
> `kb_run_state`), or **`null`** if the page has never been validated. A `null` pin means
> "unknown — run `kb_validate` before relying on it," not "fresh." The management skill
> (`009-skill`) drives `kb_validate` before edits, so by the time the agent is working the
> pin is usually recorded; `kb_query` just surfaces the last-known state cheaply.

**The snippet is a teaser, not the page** (pi's discipline): the agent must not treat a
stage-1 snippet as the whole page. It is for *ranking* and *deciding what to open*. The
agent opens the page (`kb_read`) before relying on its content.

### 2.1 The semantic leg's seam (confirm-and-name; reuses the memory change's embedder)

`kb_query`'s **semantic leg** (when the embedder seam is wired) does **not** define its
own vector store or ranker — it **references the memory change's `Embedder`/ranker seam**
(`00-foundation/003-intelligence`), the same seam the code-index's semantic leg
(`02-code-index/006-search`) uses. One retrieval contract across the three functions
(D-KB's "share the machinery, not the interface" applied to retrieval). Three rules are
fixed here (the ScopeLab `VectorRanker` design, the injection-point contract):

- **Candidate-scoping (the legs compose, they don't each scan the vault).** The
  **FTS (keyword) leg narrows the candidate set**; the semantic leg ranks **within**
  that set (`candidateIds`), not over the whole vault. This is what keeps a large vault
  query O(candidates), not O(vault) — the FTS `preFilterLimit` semantics.
- **Single-blend-place.** The **final blended score** (the hybrid/RRF fusion of the
  FTS + semantic legs) is computed in **one place — the query service**, not inside the
  ranker. The ranker returns monotonic per-leg scores; the service owns the blend
  (the same shape as the code-index's RRF-K=60 living in the `search` package, not the
  legs). A ranker that bakes in the blend would diverge from the FTS-only path.
- **Explicit fallback, never silent.** When the ranker/embedder rejects, the leg degrades
  to an **explicit** policy — `keyword` (FTS-only) | `empty` (no semantic results) |
  `throw` — matching INV-3 (LLM-first, explicit degradation) and the code-index's
  `degraded: no_vector`. The **default with no embedder** is the portable hash-64 /
  cosine floor (matching the code-index's D-CI8 hash-64 default), so the semantic leg
  always has a deterministic floor and never a silent gap.

> **The default is FTS5 (not in-RAM fallback).** Unlike the ScopeLab design (which
> defaults to an in-RAM MiniSearch and flips to FTS5 via `auto`, because FTS5
> availability varies by JS adapter), the KB is Go/`modernc.org/sqlite` — FTS5 is always
> present, so the **keyword leg defaults to external-content FTS5** (disk-backed,
> memory-flat). Only the *parity-gate + threshold* lessons transfer (§5.1), not the
> default.

### 2.2 The optional steerable rerank (the System-1 decision-model leg — `010-decision` §3.1)

The fused FTS+semantic ordering (§2.1) is the **floor**. `010-decision` adds an optional
**second stage**: **search wide** (the candidate pool) → each candidate gets a decision
model's `new` ("does this page directly answer the query?") → sort by confidence →
**threshold-drop** (below `decision.rerank_threshold`, it never reaches the agent). Only
the candidates that clear the bar go to `kb_read` / the answer. This is the reference
pattern's **cross-encoder rerank** — *similarity is not relevance* (a table of contents
matches many queries but answers none).

**The steerable part (the KB-specific value):** the agent populates a **`rerank_instruction`**
per-query, so the *ranking criteria* can differ from the *search query*. "Tell me about
the legal brief" → the FTS query is generic, but the rerank instruction is *"prioritize
pages explicitly titled or described as a legal brief."* A fixed boost (`in_degree`,
`groundedness`, §2) can't do this — the instruction is *query-conditional*. This is the
difference from a non-steerable cross-encoder.

**The floor (no decision model, the default):** the rerank stage is **skipped** and
`kb_query` returns the fused FTS+semantic ordering with **`reranked: false`** in the
response (the deterministic RRF floor holds; the threshold-drop still applies to the fused
scores). Never a silent "re-rank with nothing" — the flag is explicit (INV-3). The rerank
**re-ranks *within* the fused candidate set** (the legs compose; the decision model does
not re-scan the vault — `010-decision` DE-O2). The `rerank_instruction` + the decision-model
call sites are owned by `010-decision`; `005-query` only defines that `kb_query` *accepts*
the optional `rerank_instruction` parameter and surfaces the `reranked` flag.

## 3. Stage 2 — `kb_read` (full page)

`kb_read(rel_path, section?)` returns the **full** page:

- **Whole page:** the frontmatter + the full body (untruncated).
- **A section:** `kb_read(rel_path, section="Why")` returns just that `##` section — the
  token-efficient drill-in for large pages.
- **With backlinks:** `kb_read(rel_path, include_backlinks=true)` appends the on-the-fly
  backlink list (which pages link here) — the "who depends on this" view.
- **With claims/anchors state:** for grounded pages, `kb_read` appends the pin-check
  result (`fresh` / `stale` / `unresolved`) so the agent sees whether the page is still valid *before*
  relying on it.

This is the only full-content path (like `mem_get_observation` in the memory change).

## 4. `kb_search` (the raw FTS, power path)

`kb_search` is the **raw FTS** over `kb_pages_fts` without the stage-1 projection
constraints — for the agent (or a human via CLI) that wants the full ranked list with
more rows, a `rel_path` filter, or a specific match mode. It returns the same BM25
ranking as `kb_query` but projects more (a longer snippet, all matched sections). It is
the "search the wiki" tool, distinct from `kb_query`'s "rank the links" tool. In practice
the agent uses `kb_query` (stage 1) → `kb_read` (stage 2); `kb_search` is the escape
hatch.

## 5. Backlinks (computed on the fly)

Backlinks are **never stored** (KB-INV-1 / `004-index` §6). `kb_query` and `kb_read`
compute them when requested by scanning the vault's markdown for `[[…]]` references to
the target page. This is a bounded grep (the vault is small — pages, not a corpus), so
on-the-fly is cheap and always consistent with the files. The in-degree boost in
`kb_query`'s ranking uses a cached count (rebuilt by `kb_index`) to keep the ranking
fast, but the *listed* backlinks in `kb_read` are always recomputed (accuracy over
speed for the view).

## 6. The wiki-first contract + the memory-fallback handoff (OPEN-12)

The **routing rule** (OPEN-12) is enforced by the agent protocol (the management skill,
`009-skill`), not the engine:

```
"what do we know about X"  →  kb_query        (this spec, the deep retrieval)
      └─ if kb_query is thin/empty on X  →  mem_lookup / mem_search   (memory, the fallback)
"give me that exact fact / the session where we decided Z"  →  mem_lookup  (raw, precise)
"explain the design of Y"  →  kb_query → kb_read   (the synthesis, with citations)
```

**The KB's obligation:** `kb_query` must be **good enough that the fallback is rare.**
The KB's *quality* (coverage, grounding) determines how often the agent falls back to raw
memory. The KB cannot fix a thin memory, but it must make the *wiki-first* path the
default, useful path.

**The handoff is explicit:** when `kb_query` returns `thin: true` (fewer than a
threshold of relevant pages on the topic), the result carries a `fallback: mem_lookup`
hint so the agent knows to go to raw memory. The KB does not *call* `mem_lookup` — the
agent does (the KB does not import the memory package; the agent is the router).

## 7. Retrieval trails (INV-5)

Every `kb_query` / `kb_search` / `kb_read` records a trail: the query, the filters, the
rank route (fts / fts+backlink-boost), what was returned (`rel_path`s), and the
`degraded` status (`stale_index` if the index lagged, `thin` if the result was
fallback-suggesting). `mnemonic trail recent|show` inspects them (the shared trail
store, `00-foundation/005-transport-cli`). This is the audit record for "why did the
agent recall this page."

## 8. Retrieval eval (the measurement leg — `kb_eval`)

Every retrieval system needs a way to answer: *"did that change make recall better or
worse?"* The ranking (§2), the semantic-leg fallback (§2.1), and the `thin` fallback
handoff (§6) are all **tunable** — and none of them is currently *measured*. `kb_eval`
is the frozen-query regression harness over the **memory/KB corpus** (the "what do we
know about X" recall), not the code corpus. It is the RAG-tutorial lesson — *"add evals:
a set of test questions so you can measure whether a change actually makes the answers
worse or better"* — applied to this spec's retrieval model.

**Why the mem/KB corpus, not code, is the one that needs it first:** the mem/KB corpus is
where *paraphrased* recall matters (a decision saved as "commit trailers are banned" must
surface for "how do we tag AI commits"). It is also the corpus §2.1's `degraded:
keyword` flag *warns about* — BM25-only recall can miss differently-phrased pages — but
nothing today *measures* when a query becomes fragile to an embedder outage. The code
corpus already has a leak-free git-history eval (the v1 `skillgrid eval` precedent,
commit-subject → changed-files); the mem/KB corpus is the unmeasured half.

### 8.1 The case

A **case** is a frozen `(query → expected page set)` pair, plus two knobs:

```
query:         "how do we tag AI commits"
expect:        ["decisions/ai-commit-tagging"]     # rel_paths that MUST surface
min_rank:      3                                     # …and must land in the top-N
degraded_ok:   true                                  # pass under keyword-only, or hybrid-only?
source:        manual | trail:<id>                   # provenance (hand-authored or promoted)
```

`min_rank` keeps the eval honest: a page that surfaces "somewhere in 50" is not a recall
win; it must land in the top window (default 3). `degraded_ok` is the **fragile-case
signal**: a case that passes under `hybrid` but **fails** under `keyword` is a
*fragile-to-embedder-outage* query — exactly the risk §2.1's explicit-degrade flag names
but the code corpus can't express (a code path's FTS leg is rarely the only way to find
it; a paraphrased decision often is).

### 8.2 The two sources (starter set + trail bootstrap)

- **Starter set (committed):** a small hand-authored set of cases under
  `.skillgrid/knowledge/_eval/cases.yaml` covering the recall shapes that matter —
  same-phrasing, paraphrase (the `degraded_ok=false`-critical ones), and a
  cross-page multi-hop ("explain the design of Y" that should surface ≥2 pages). Committed
  so the eval is reproducible and reviewable, not regenerated.
- **Trail bootstrap:** §7's `retrieval_trails` already log every `kb_query`/`kb_search`
  (the query, the rank route, the returned `rel_path`s, the `degraded` status). A
  **promote** path turns a good, real trail into a case: *"that query returned the right
  pages — pin it so a future ranking change can't silently regress it."* The set grows
  organically from real traffic instead of only from hand-authoring. Promotion is a
  deliberate act (a human or the agent confirms the trail's result was correct before
  freezing it) — a bad trail promoted becomes a bad test.

### 8.3 The replay (shares the v1 ablation-harness machinery)

`kb_eval` does **not** invent a new scoring rig. It **reuses** the retrieval-eval ablation
harness the v1 `skillgrid eval` already ships (seeded paired bootstrap 95% CI +
permutation p-value + the shipped-config gate — `mnemonic_back/internal/eval`), projecting
the **mem/KB corpus** into it:

- The **corpus** = the KB page set (`rel_path` → page), the mem/KB analogue of the code
  eval's file-path corpus.
- The **ranker** = the *live* `kb_query` fused search (FTS + semantic RRF, §2), the
  mem/KB analogue of the code eval's `shippedRank` (a lexical projection of
  `hybrid.RerankTable`). Baseline = the FTS-only ordering; shipped = the fused ordering.
  So the harness measures the **ranking delta** (the semantic leg's value), exactly as the
  code eval measures it — and `degraded_ok` adds the **per-case fragility** dimension the
  code corpus lacks.
- **Recall@k over `rel_path`s** (the code eval's recall@k over file paths), plus the
  fragility flag, pooled over the starter set + promoted trail cases.

**The live-store boundary:** the harness itself is pure-Go (the ranker is a pure function
of `(query, corpus)`), but the **mem/KB ranker reads the live store** (the FTS index + the
embedder seam). That boundary is tested at the CLI round-trip, the way the code eval's
`--corpus self` is — `kb_eval` against a real `.skillgrid/knowledge/` + shared store, not
an in-RAM fixture.

### 8.4 The report + the gate

`kb_eval --cases .skillgrid/knowledge/_eval/cases.yaml` prints, per case: pass /
degraded-pass (matched but `degraded: keyword`) / **fragile** (pass under hybrid, fail
under keyword) / fail, plus the pooled recall@k and the bootstrap CI + permutation
p-value of the shipped-vs-baseline delta (does the semantic leg *actually* help recall?).
It exits non-zero on a **new** failure so it can gate the QA/review step for any change
that touches `005-query`'s ranking, the `thin` threshold, or the semantic-leg fallback.

### 8.5 Non-goals (what `kb_eval` is NOT)

- **Not answer-quality grading.** It measures **retrieval recall** (did the right pages
  rank high), deterministically. It does *not* LLM-judge the prose `kb_read`/`kb_query`
  returns — that is unmeasurable by a frozen set and is a separate concern (the
  `kb_validate` pin-check covers *correctness of the page*, not *recall of the page*).
- **Not a new vector store or ranker.** It rides the §2.1 shared retrieval contract
  (the memory change's embedder seam, the single-blend-place service) — it *measures*
  that contract, it does not replace it.
- **Not the code eval.** The code corpus keeps its git-history leak-free eval. `kb_eval`
  is the mem/KB half of the same measurement idea, sharing the harness, not the corpus.

## 9. Invariants

INV-3 (explicit `degraded: stale_index`), INV-1 (cited sections — every stage-2 result
carries the page's `sources:`), INV-5 (trails), INV-4 (local truth — the query reads the
local index, never a remote authority).

**KB-INV-5 (added):** **The retrieval is measured, not just tuned.** Any change to
`kb_query`'s ranking, the `thin` threshold, or the semantic-leg fallback (§2.1) is
gated by `kb_eval` — a frozen case set whose recall@k and fragile-case signal must not
regress (a new failure fails the gate). A retrieval change that can't be shown
non-regressive by the eval is a spec violation.

## 10. Open items

| # | Question | Default |
|---|---|---|
| KQ-O1 | Where the eval cases live: `.skillgrid/knowledge/_eval/cases.yaml` (in the vault, git-tracked) vs a sidecar `kb_eval_cases` table in the shared store? | **In the vault, git-tracked** (KB-INV-1 — the vault is truth; a case is a durable test fixture, and git-tracking makes promotion + review + rebase auditable). The shared store holds only the *runs*, not the *cases*. |
| KQ-O2 | Trail-promote: a named `kb_eval promote <trail-id>` engine verb, or an agent-driven convention (the agent writes the case YAML from a trail)? | **Engine verb** (`kb_eval promote`), so the promote is atomic (freezes query + expect + `degraded_ok` from the trail's recorded route) and auditable in the run state. |
| KQ-O3 | Does `kb_eval` gate the **QA step** (non-zero exit on new failure), or is it advisory-only like Trivy? | **Gating** for changes that touch `005-query`'s ranking / `thin` / semantic fallback; advisory otherwise. (Trivy is advisory because it's a *scanner*; `kb_eval` is a *regression gate* on a change to the thing it measures.) |
| KQ-O4 | The `thin` threshold's value (how few pages before the §6 fallback hint fires) — and whether `kb_eval` should *tune* it by sweeping the threshold over the case set? | **Sweep it** — `kb_eval` reports recall@k across a small `thin`-threshold grid, so the threshold is *chosen* by measured recall, not asserted. The threshold itself is defaulted in `008-transport` config. |

The two-stage pattern, the ranking, the fallback handoff, and the eval are all fixed. The
`thin` threshold's value is the one *tunable* — and KQ-O4 makes it a *measured* tunable.
