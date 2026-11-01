# 004 — Dedup (LLM 4-way classification + hash floor)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `003-intelligence` (§2.1 `Classify`), `001-observations` (step 5)
**Reference:** mnemonic v1 `internal/memory/types.go` (`DedupLLM.Classify`,
`DedupDecision`, the four verdicts); Engram's rolling-window hash dedupe.
**Pipeline position:** step 5 of the save pipeline (`001-observations` §2).

---

## 1. Capability

Dedup decides what a new `mem_save` *does* to existing memory: store it as new, update
an existing one, supersede one, or do nothing. In v2 the **LLM is the primary
classifier** (LLM-first); the **deterministic hash is the degraded floor**. This is the
opposite of v1, where hash was the default and LLM was opt-in.

## 2. The four verdicts

`Classify(newContent, candidates) → DedupDecision`:

| Verdict | Meaning | Action |
|---|---|---|
| `add` | genuinely new | INSERT a new observation |
| `update` | updates an existing one (`CandidateID` names the target) | UPDATE the target in place (revision++, version snapshot) |
| `delete` | supersedes / deletes an existing one (`CandidateID`) | mark target superseded, store the new |
| `noop` | matches an existing one and changes nothing | no write |

`DedupDecision{ Verdict, CandidateID int64, Reason }` where `Reason` ∈ `"llm"` /
`"hash"` / `""`. A zero decision (empty verdict, id 0) means "no dedup decision" — the
caller proceeds with a normal save and the hash fallback does the work.

## 3. The pre-filter (bound the LLM input)

Before the LLM, a **cheap SQL pre-filter** narrows candidates (Engram's
`dedupPreFilterLimit`): the project's non-deleted observations, **most-recent first,
capped (≤20)** for classify. This stands in for vector pre-filtering (no embeddings
required) and bounds the LLM call cost.

> **Not the same pre-filter as conflict-candidate detection.** Dedup uses this
> *recent-window* filter (≤20, no title query). Conflict-candidate detection
> (`003-facts-relations` §3) uses a *different* filter — an OR-based FTS on the saved
> **title**, bounded per-source (default 3), then the relevance gate. Two distinct
> pre-filters, two distinct purposes.

## 4. The LLM path (primary)

```
candidates = preFilter(project, limit=20)
if LLM seam present:
    decision = LLM.Classify(ctx, newContent, candidates)
    if decision.CandidateID > 0 and not a live row in this project:
        decision.CandidateID = most-recent live candidate   (trust-but-verify)
    decision.Reason = "llm"
else:
    decision = hashFloor(newContent)      # see §5
    decision.Reason = "hash"
```

**Trust-but-verify the LLM's id:** if the LLM returns a positive `CandidateID` that is
not a live row in this project, fall back to the most-recent live candidate (the head of
the pre-filter). The LLM's *verdict* is trusted; its *pointer* is verified.

## 5. The hash floor (degraded)

When the LLM seam is nil/failing (degraded, INV-3):

```
hashFloor(newContent):
    h = normalized_hash(newContent)
    match = lookUp(normalized_hash=h, project, scope, type, title)   # idx_obs_dedupe
    if match:  → VerdictNoop (exact dupe: bump duplicate_count, last_seen_at, updated_at)
    else:      → VerdictAdd
```

This is Engram's rolling-window exact dedupe. It catches *exact* repeats; it does not
catch semantic near-duplicates (that's the LLM's job, and when degraded we accept the
gap and set `degraded: true`).

## 6. Identity rules (the discipline)

These are agent-protocol rules (Engram memory-protocol), restated as spec invariants:

- **A failed read is not evidence of absence.** Don't overwrite or fall back to a
  duplicate save on a failed lookup.
- **A suggested `topic_key` is a hint, not proof of identity.** `mem_suggest_topic_key`
  applies a family heuristic; the agent decides.
- **For the same bounded topic, update by id** (keep `topic_key`, preserve every
  still-valid fact, record superseded facts as history). For a genuinely new topic,
  save with a deliberate, distinct `topic_key`.
- **If already covered, write nothing.** (The `noop` verdict is a first-class outcome.)
- **Never merge across projects or scopes.** Dedup is scoped to `project + scope`
  (and `topic_key` when set).
- **Distinct decisions must not share a `topic_key`** or they overwrite each other.

## 7. Normalization

`normalized_hash` is computed over the **redacted, truncated** content (after step 3,
REDACT, of the save pipeline), so two saves that differ only in `<private>` or length
still hash the same. Title normalization for the *relevance gate* is separate (gate-only,
`003-facts-relations` §3) and does not affect the hash.

## 8. Invariants

INV-3 (LLM-first, hash is the explicit degraded floor; `Reason` labels which ran),
INV-1 (dedup provenance: `Reason` is stored on the result), INV-6 (CandidateID resolves
to a sync_id before merge).

## 9. Open items

None.
