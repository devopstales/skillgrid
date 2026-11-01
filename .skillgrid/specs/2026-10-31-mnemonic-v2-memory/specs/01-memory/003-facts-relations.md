# 003 — Facts & Relations (the edge model)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `002-schema` (§2.5), `003-intelligence`
**Reference:** Engram `internal/store/relations.go` (vocabulary, candidates, scans),
`candidate_relevance.go` (the gate), `consolidate.go` + `OBSERVATION-CONSOLIDATION.md`
(consolidation contract).

---

## 1. Capability

Relations are the **edges between facts**: the conflict/supersession/compatibility
structure that lets the agent (and the wiki) see how memories relate. This is where
**semantic conflict surfacing** happens — the engine does the *lexical* pre-filter and
candidate detection; the **LLM does the semantic judgment** (INV-3, LLM-first).

## 2. The relation model

`memory_relations` (schema §2.5): directed `source_id → target_id` on **`sync_id`**
(INV-6), with the 7-value vocabulary:

`pending` (default, not a verdict) · `related` · `compatible` · `scoped` ·
`conflicts_with` · `supersedes` · `not_conflict`.

Plus: `reason`, `evidence`, `confidence`, `judgment_status`
(`pending`/`judged`/`orphaned`/`ignored`), provenance (`marked_by_actor/kind/model`),
`session_id`, `superseded_at`/`superseded_by_relation_id`.

**No UNIQUE on `(source_id, target_id)`** — multi-actor disagreement is allowed. Two
agents (or an agent + the LLM seam) may judge the same pair differently; both rows
coexist. **No `project` column** — project is derived by JOIN on the endpoints.

## 3. Conflict-candidate detection (the write path)

After a `mem_save`, before returning:

1. **Lexical pre-filter:** OR-based FTS on the saved title within the same project+scope,
   raw BM25 ordering.
2. **Relevance gate** (opt-in, deterministic SQLite function applying a Go rule):
   accept a candidate iff it shares a nonempty `topic_key`, **or** shares ≥2 distinct
   significant title terms, **or** both titles reduce to the same single significant
   term. Terms are case-insensitive letter/digit words; `+`/`#` preserved in
   identifiers (`C++` ≠ `C#`); common connectives + generic change verbs (`updated`,
   `fixed`) ignored. Content-only matches do **not** establish title relevance.
3. **Materialize** the surviving candidates (bounded per-source, default 3) as **pending
   `memory_relations`** rows, returned to the agent as `candidates[]` with
   `judgment_required: true`.

**This is a lexical heuristic, not semantic detection** (Engram, explicitly). Synonyms
or one-term overlaps can miss a real conflict; the ignored-word list is English-specific.
The **LLM is the semantic layer** — the agent (or `mem_compare`) performs the actual
judgment.

## 4. Judgment

| Tool | What it does |
|---|---|
| `mem_judge(judgment_id, relation, reason, evidence, confidence)` | Records a verdict on a pending candidate. The LLM (agent's or the seam's) decides `conflicts_with` / `supersedes` / `compatible` / `not_conflict` / `related` / `scoped`. Sets `judgment_status=judged` + provenance. |
| `mem_judge_many` | Batch 1–32 judgments in one queued write (ordered, independent verdicts). |
| `mem_compare(a, b)` | Proactive: the LLM compares two observations and **creates a judged relation directly** (system provenance, idempotent per pair). |

A `pending` relation is **not** a verdict — it is a *candidate for judgment*. The wiki
reads judged relations (conflicts, supersedes) as first-class content; pending ones are
surfaced as "contested by #id (pending)" annotations.

## 5. Consolidation (merging facts)

`Store.ConsolidateObservations` / `mem_consolidate` merges ≥2 live observations from one
resolved canonical project (Engram `consolidate.go`):

**Two-phase, fingerprint-gated:**
1. **Preview** (`confirm: false`): returns `plan.sources`, `plan.survivor`,
   `plan.project`, and an opaque `Fingerprint`. Nothing is written.
2. **Confirm** (`confirm: true` + the same selection + the fingerprint): executes. If
   source state or the resolved survivor changed → `ErrConsolidateFingerprintMismatch`
   + a fresh plan, no writes. Inspect and re-confirm the new fingerprint; **never
   auto-confirm a replacement**.

**Rules:**
- The survivor inherits the **newest source's** session, type, and scope (unless a
  supported field is overridden); `topic_key` is explicit-only.
- Survivor content includes the source `sync_id`s in a `consolidated_from` footer.
- Sources are soft-deleted with `superseded_by_sync_id` set (structured provenance).
- **One local transaction** does all of it; any failure rolls back.
- **Not distributed atomicity** (Engram's disclaimer): a receiver may defer the survivor
  upsert while its parent session is missing; LWW can resurrect a source after a later
  peer edit. Accepted limit.
- Refusals (before writes): missing/deleted sources, duplicate ids, cross-project
  selection, provenance cycles, unresolvable project.

**MCP contract:** required `ids_to_delete` (≥2 distinct positive ints), `new_title`,
`new_content`, `expected_project`. Optional `type`, `topic_key`, `fingerprint`;
`confirm` bool defaulting false. Destructive and non-idempotent even though preview is
the default. Error codes: `invalid_arguments`, `project_mismatch`, `confliction_refused`,
`fingerprint_mismatch`.

## 6. Rendering into results

Relations are rendered into `mem_search`/`mem_get_observation` output as annotation
lines (Engram): `supersedes: #id (title)`, `superseded_by:`, `conflicts:`,
`conflict: contested by #id (pending)`. The agent sees the structure inline, not as a
separate query.

## 7. Scan (re-surfacing)

`engram conflicts scan` / `ScanProject` does page-batched conflict re-scans with an
optional semantic runner hook. It re-runs the candidate detection over a project to
surface *new* pending relations (e.g. after a sync pull introduced new observations).
This is the periodic "are there conflicts I haven't judged?" pass.

## 8. Invariants

INV-1 (provenance on every judged relation), INV-6 (relations key on sync_id),
INV-3 (LLM is the semantic layer; the engine is lexical).

## 9. Open items

None.
