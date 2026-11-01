# 003 — Intelligence (the LLM-first layer)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture` (INV-3)
**Reference:** Engram's "agent is the intelligence" stance (semantic judgment is the
agent's LLM, not the store) + mnemonic v1's injectable LLM seams (`ExtractionLLM`,
`DedupLLM`, `DreamLLM`).

---

## 1. Capability

The intelligence layer is the **first-class LLM + embedder seam**. In v2 the LLM is not
optional garnish (v1's "deterministic floor is the default") — it is the primary
mechanism for semantic work, and the deterministic path is the *degraded* path. But
"LLM-first" is **structured**: every LLM call has a deterministic fallback, and when the
LLM is absent or failing the system degrades *explicitly* (`degraded: true`), never
silently (INV-3).

## 2. The seams

Two injectable interfaces. A nil/disabled seam means "degraded" — the deterministic
floor runs and the result carries `degraded: true`.

### 2.1 `LLM`

```go
type LLM interface {
    // Classify: 4-way pre-write dedup classification of a new observation
    // against pre-filtered candidates. Returns the verdict + target id + reason.
    Classify(ctx, newContent string, candidates []string) (DedupDecision, error)

    // Judge: semantic judgment on a pending memory_relation (conflict? supersedes?
    // compatible?). Returns relation + confidence + evidence.
    Judge(ctx, source, target Observation, relation Rel) (Judgment, error)

    // Compare: proactive semantic comparison of two observations → a judged relation.
    Compare(ctx, a, b Observation) (Judgment, error)

    // Extract: pull discrete facts from free text (passive capture, session summary
    // structuring). Returns candidate observations.
    Extract(ctx, text string) ([]CandidateObservation, error)
}
```

`DedupDecision` (from v1 `types.go`): `Verdict` ∈ `add`/`update`/`delete`/`noop`,
`CandidateID int64`, `Reason` ∈ `"llm"`/`"hash"`/`""`.

### 2.2 `Embedder`

```go
type Embedder interface {
    Embed(ctx, texts []string) ([][]float32, error)
    Model() string        // recorded on the observation's embedding_model
    Dimensions() int      // must match the vector column
}
```

## 3. The deterministic floors (the degraded path)

Every LLM call has a floor that runs when the seam is nil/failing:

| LLM call | Deterministic floor | `degraded` when |
|---|---|---|
| `Classify` | **hash dedup** — `normalized_hash + project + scope + type + title` exact match (Engram's rolling-window dedupe). No LLM verdict; the write proceeds or exact-dupes. | no `LLM` seam |
| `Judge` | relation stays `pending` (not `judged`); the agent is told `judgment_required` and the semantic verdict is left to the *agent's* own LLM (Engram: the agent is the intelligence). | no `LLM` seam |
| `Compare` | no relation created; the agent does it. | no `LLM` seam |
| `Extract` | `## Key Learnings:` line-parsing (v1 passive capture) — heuristic, no LLM. | no `LLM` seam |
| `Embed` | no vector written; `VectorSearch` returns empty. | no `Embedder` seam |

**The floor is always available.** This is the difference between v2's "LLM-first,
structured" and v1's "deterministic floor is the default": the *default* is the LLM; the
floor is the *degraded* mode, and it is *explicit*.

## 4. Degradation policy

`intelligence/degrade` owns the policy:

- A seam that **errors** (not just nil) is treated as degraded for that call, with a
  bounded retry (≤1) — not an unbounded loop.
- Every degraded result sets `degraded: true` on its return value *and* logs a
  degradation line. Callers (MCP tools) surface `degraded` in the response envelope.
- **No silent fusion.** A result that mixed LLM and deterministic provenance labels each
  part (`Reason: "llm"` vs `"hash"`). (INV-1, provenance.)
- Degradation is **per-call**, not global: a failing `Embedder` does not degrade
  `Classify`.

## 5. Provider selection (resolves OPEN-6)

The provider is selected by config (`006-config`): an **OpenAI-compatible** endpoint
(base URL + model + key) is the default seam, and a **local** backend (Ollama-compatible)
is a one-config-change alternative. Both implement the same `LLM`/`Embedder` interfaces.
The memory function never names a provider — it calls the seams. The seam is
**process-lifetime** (one `LLM` + one `Embedder` per process), constructed at startup.

## 6. Cost discipline

- **Pre-filter before the LLM.** `Classify`/`Judge` receive a small, SQL-pre-filtered
  candidate set (≤20 for classify, ≤3 per source for relations) — the LLM input is
  bounded. (Engram's `dedupPreFilterLimit`.)
- **Batch where possible.** `mem_judge_many` (1–32 judgments) is a single queued write,
  not N round-trips.
- **Eager wiki handoff is the big cost.** The per-save LLM call for wiki regeneration
  is the dominant cost (see `01-memory/001-observations.md` §save-event). It is
  **incremental + entity-targeted** by contract — never a whole-wiki rebuild.

## 7. Invariants

INV-3 (explicit degradation). INV-1 (provenance: `marked_by_model` on judged relations;
`embedding_model` on embeddings).

## 8. Open items

None — OPEN-6 (provider) resolved above.
