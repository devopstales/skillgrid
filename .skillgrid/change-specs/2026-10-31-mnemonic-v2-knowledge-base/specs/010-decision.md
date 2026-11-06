# 010 — Decision Model (the System-1 layer)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `00-foundation/003-intelligence` (the `Embedder`/`LLM` seam the decision
model is a third, sibling seam to), `005-query` (§2 `kb_query` rerank — a *consumer*),
`006-validate` (§2 pin-check — a *consumer*), `003-generate` (the no-op clean-run the
router can skip), `008-transport` (the `decision` config + the `kb_*` verbs that route
through it).
**Reference:** the "agent RAG / System-1 decision model" pattern (Typescale "Jev" as the
exemplar): a non-chat model that returns a **decision + confidence**, not prose.

---

## 1. Capability

A **decision model** is the opposite of a chat model. A chat model *writes* — it answers
one token at a time, as a stream of text you have to parse. A decision model **writes
nothing**: you hand it a *state* (a passage, a question, a conversation slice) plus a
*form* of structured questions, and it hands back **decisions + confidence scores** — a
few numbers. Because the output shape is defined by the *form* (the schema you write), it
**cannot hallucinate a new option** — it can make the *wrong* decision, but it never
invents a category you didn't declare. That is what makes it cheap (you pay for the input
only; the output is ~free) and fast (no token stream to stream out), and what makes it
the right tool for the **micro-decisions** inside the KB pipeline: the small, high-volume,
fuzzy judgments a chat LLM can also make but more slowly and more expensively.

This spec defines the **decision seam** — the third intelligence seam beside `Embedder`
and `LLM` (`00-foundation/003-intelligence`) — and the **three KB uses** that ride on it:
the **steerable reranker** (`005-query` §2), the **citation-confidence** check
(`006-validate` §2.2), and the **router** (skip the expensive synthesis on a clean run,
§4). The KB does **not** own a decision-model implementation — it owns the *contract* and
the *call sites*. The model itself is a config-driven seam (local Ollama, an OpenAI-
compatible endpoint, or a dedicated decision-model API), exactly like the `LLM` seam.

## 2. The contract (a form, not a prompt)

A decision-model call is a **form**: a `state` (the context) + a list of **questions**.
Three question kinds, all answerable in **one** JSON call (a "form" with several fields):

| Kind | Shape | Returns |
|---|---|---|
| **new** (yes/no) | a binary question with a plain-English "what a yes looks like" + "what a no looks like" | `bool` + `confidence ∈ [0,1]` |
| **choice** | a closed list of options the caller writes; the model picks one | the chosen option + a per-option score (a distribution) |
| **score** | a scale the caller writes (e.g. 1–5 with a label per level) | the position on the scale + `confidence` |

**The rules:**

- **The state is the context.** The passage(s), the query, the conversation slice — the
  same role `prompt` plays for a chat LLM. It is the *input*, not the *question*.
- **The questions are the form.** Each question is written in plain English as a
  definition of the decision ("a yes is when the passage directly answers the query; a
  no is when it is related but does not answer"). The definitions *are* the prompt,
  behind the scenes.
- **The output is guaranteed-shape JSON.** The caller declares the options/scale; the
  model returns *only within* that shape. **Zero hallucinated categories** — the
  model's error budget is *wrong decision within the declared set*, never *new option*.
- **Several questions per call.** A rerank (one `new` per chunk) or a citation check
  (one `choice` per claim) is a *single* call with N questions, not N calls. The
  "hundreds of yes/no calls" the reference pattern describes is one form, one round-trip.
- **Confidence is a first-class return.** Every decision carries a confidence the caller
  can bake into logic (a rerank threshold, a `--strict` gate, a router cutoff). A
  decision without a confidence is not a decision-model output.

**The seam (the `Decision` interface, the third `00-foundation/003-intelligence` seam):**

```go
// Decision is the System-1 judgment seam. One call = one form (state + N questions),
// one JSON reply. It is a sibling of LLM (prose) and Embedder (vectors): the KB owns
// the contract + call sites, not the implementation. A nil/absent decision model
// degrades every consumer to its deterministic floor (INV-3) — never a silent gap.
type Decision interface {
    Decide(ctx context.Context, state string, questions []Question) (Reply, error)
}
type Question struct {
    Kind      string   // "new" | "choice" | "score"
    Prompt    string   // the plain-English definition of the decision
    Options   []string // for "choice" / "score": the declared closed set
}
type Reply struct {
    Answers []Answer `json:"answers"` // one per question, in order
}
type Answer struct {
    Decision  string    // "new": "true"/"false"; "choice"/"score": the chosen option
    Confidence float64  // ∈ [0,1]; how sure the model is on THIS decision
    Scores    []float64 // "choice"/"score": the per-option distribution
}
```

> **One form, one round-trip, one JSON reply.** The contract is deliberately shaped so a
> 60-chunk rerank or a 10-claim citation check is *one* `Decide` call (the reference
> pattern's "hundreds of yes/no calls" is one form). That is what keeps it cheap — the
> alternative (a chat LLM per chunk, one prose reply each) is exactly the "slow and the
> costs added up" baseline the reference pattern replaces.

## 3. The three KB uses (consumers)

The decision seam is **inert until a KB call site consumes it**. Three call sites, each
additive to an existing spec, each with a **deterministic floor** when no decision model
is configured (INV-3 — explicit degrade, never a silent gap):

### 3.1 Steerable reranker (`005-query` §2)

**The use:** `kb_query` already fuses FTS + semantic (RRF). The reranker adds a second
stage: **search wide** (the 60-candidate pool) → each candidate gets a `new` ("does this
page directly answer the query?") → sort by confidence → **threshold-drop** (below the
config cutoff, e.g. 0.20, it never reaches the agent). Only the ~10 that clear the bar
go to `kb_read` / the answer.

**The *steerable* part (the KB-specific value):** the agent populates a **rerank
instruction** per-query, so the *ranking criteria* can differ from the *search query*.
"Tell me about the legal brief" → the FTS query is generic, but the rerank instruction is
*"prioritize pages explicitly titled or described as a legal brief."* A fixed boost
(`in_degree`, `groundedness`) can't do this — the instruction is *query-conditional*.
This is the difference from a non-steerable cross-encoder (Cohere's reranker does the
cross-encoder part; the *instruction* is what the reference pattern adds).

**The floor (no decision model):** the rerank stage is **skipped** and `kb_query` returns
the fused FTS+semantic ordering, with `reranked: false` in the response (the deterministic
RRF floor holds). The threshold-drop still applies to the fused scores. Never a silent
"re-rank with nothing" — the `reranked` flag is explicit.

### 3.2 Citation-confidence (`006-validate` §2.2)

**The use:** `kb_validate`'s pin-check is **code-grounded** (does the page's anchor/claim
still match *live code*?). The citation-checker is the **same mechanism at a different
target** — **passage-grounded**: does the *generated answer* match the *cited passage*?
Add an optional check kind, **`cite`**, that runs a `choice` over the cited chunk:
`fully supported | partially supported | says nothing | contradicts`, with the per-option
score. A claim "partially supported at 0.57" is a *different finding* than "contradicts
at 0.88" — the **degree** of support, not just a stale/fresh boolean, is the signal.

**The floor (no decision model):** the `cite` check is **skipped** (the deterministic
pin-check — token/span/symbol/claim vs live code — is unaffected and remains the
fail-closed gate). `kb_validate` reports `cite: skipped (no decision model)`, explicit
per INV-3. The pin-check never *depends* on the decision model; the citation-confidence
is an *additive* lens.

### 3.3 The router (skip the expensive synthesis on a clean run)

**The use:** the reference pattern routes *queries* between cheap/expensive chat models.
The KB is **local-first on one Ollama model** (`llama3.2:1b`) — there is no model *tier*
to route between yet. So the router is reframed to what it actually saves here: **the
*cost/latency* of the expensive LLM synthesis** (`kb_generate`, the LLM that writes pages),
not a model tier. A `choice` per incoming save-event / turn — *cache hit (no page change
needed) | minor edit (one page) | full regenerate (many pages)* — decides whether the
expensive LLM synthesis runs at all. This sharpens KB-INV-4 ("generation is a no-op when
clean") from a *deterministic* no-op (no new/changed input) to a *judged* no-op (a
clean-ish input the decision model scores as "cache hit" → skip the LLM).

**The floor (no decision model):** the router is **bypassed** — every save-event goes to
the deterministic KB-INV-4 clean-run check (no LLM when no new/changed input, as today).
The router is a *v2.1* cost/latency optimization, **not a v2 requirement**; it is the
weakest of the three fits because the KB's single-model architecture removes the
"route between tiers" half of the value.

## 4. The config + the degrade contract (INV-3)

`config.json.decision` (mirrors the `llm`/`embedder` config blocks):

```yaml
decision:
  enabled: false            # default OFF — every consumer uses its deterministic floor
  provider: ollama          # ollama | openai_compatible | (dedicated decision-model API)
  base_url: http://localhost:11434/v1
  model: <decision-model>   # a small, fast judgment model
  timeout: 3s
  rerank_threshold: 0.20    # §3.1 cutoff (a candidate below this is dropped)
  cite_strict: false        # §3.2: does a "contradicts" citation-confidence fail the turn?
```

**The degrade contract (every consumer, INV-3, never silent):**

- **`decision.enabled: false`** (the default) → the seam is nil → **every consumer uses
  its deterministic floor** (§3.1 `reranked: false`; §3.2 `cite: skipped`; §3.3 bypassed).
  The KB is fully functional with *no* decision model configured. The decision model is an
  *enhancement*, not a *dependency*.
- **A runtime error / timeout** → the *call site* fails open to its floor (the rerank
  returns the fused ordering; the `cite` check reports `skipped: <reason>`; the router
  falls back to the deterministic no-op check). The decision model's error never blocks a
  `kb_query` or `kb_validate`.
- **`degraded` is surfaced, not logged.** Every response that *could* have used the
  decision model but didn't carries the explicit flag (`reranked: false`,
  `cite: skipped`), so an agent or a `kb_eval` (§ of `005-query`) sees *why* a result is
  floor-quality. This is the same explicit-degrade discipline as the `LLM` seam (INV-3).

## 5. Why a decision model and not the chat LLM

The reference pattern's core claim: a RAG pipeline has **many micro-decisions**, and
"until now we've been relying on LLMs to make these calls." A decision model is the
cheaper, faster, *structured* alternative for exactly the calls that are:

- **high-volume** (a 60-chunk rerank, a 10-claim citation check, a per-save router call),
- **fuzzy** (relevance, degree-of-support, "is this a clean run?"),
- **bounded** (the answer is one of *declared* options — not free text).

The KB already has the deterministic floors (RRF fusion, the pin-check, the no-op
clean-run). The decision model is what makes those floors *query-conditional* (the
steerable rerank), *degree-aware* (the citation-confidence), and *cheap-to-judge* (the
router). None of the three is a v2 requirement — all three degrade to the floor.

## 6. Non-goals (what the decision model is NOT)

- **Not the chat LLM.** It does not synthesize pages (`kb_generate`), does not answer
  questions (`kb_query`'s answer is the agent's, from `kb_read`), does not write prose.
  It returns *decisions + scores*.
- **Not the embedder.** It is a judgment (a decision over a state), not a vector. The
  semantic *leg* of `kb_query` is the `Embedder`; the *rerank* is the decision model.
  Different seams, different roles.
- **Not a new dependency on a specific vendor.** The seam is config-driven (Ollama /
  OpenAI-compatible / dedicated). The reference pattern's "Jev"/Typescale is one
  implementation the seam can target; the KB does not bind to it.
- **Not a v2 requirement.** The default is `enabled: false` (every floor holds). The
  decision model is a v2 *option* (the rerank + citation-confidence) and a **v2.1**
  option (the router). The KB ships without it.

## 7. Invariants

INV-3 (explicit `degraded` — a consumer that could use the decision model but didn't
surfaces `reranked: false` / `cite: skipped`, never a silent floor), INV-4 (local truth —
the decision model judges *local* passages/chunks; it does not reach a remote authority),
the LLM-seam degrade contract (a decision-model error never blocks a `kb_query` /
`kb_validate`).

**KB-INV-6 (added):** **The decision model is an enhancement, not a dependency.** With
`decision.enabled: false` (the default) every consumer uses its deterministic floor and
the KB is fully functional. No KB read path (`kb_query`), gate (`kb_validate`), or write
path (`kb_generate`) *requires* the decision model.

## 8. Open items

| # | Question | Default |
|---|---|---|
| DE-O1 | The decision-model *implementation* for v2: reuse the Ollama chat model (`llama3.2:1b`) in a "form" call (it can do structured output), or a dedicated small judgment model? | **Ollama, structured-output mode** for v2 (one local model, no new pull); a dedicated decision-model API is a v2.1 config option. The seam is implementation-agnostic. |
| DE-O2 | Does the rerank (§3.1) **replace** the fused RRF ordering or **re-rank within** it (cross-encoder *within* the FTS candidate set, the reference pattern's composition)? | **Re-rank within** — the FTS/semantic leg narrows the candidate set; the decision model scores *within* that set. The candidate-scoping rule from `005-query` §2.1 (the legs compose, they don't each scan the vault). |
| DE-O3 | Does the citation-confidence's `contradicts` verdict **fail the turn** in `--strict`? | **`cite_strict: false` by default** (the pin-check's code-grounded gate is the strict blocker; a `contradicts` citation is a *finding*, surfaced, not a build-fail). `cite_strict: true` makes it a strict gate. |
| DE-O4 | Is the router (§3.3) a **v2** or **v2.1** deliverable? | **v2.1** — it is the weakest fit (the single-model architecture removes the "route between tiers" half) and is a cost/latency optimization, not a correctness concern. v2 ships the rerank + citation-confidence; the router is deferred. |
| DE-O5 | One `Decide` call per form (N questions), or a batched multi-state call (N states × M questions)? | **One form per call** (N questions, one state). Multi-state batching (a 60-chunk rerank as 60 states) is an implementation optimization behind the seam; the contract is one-state-per-form. |
