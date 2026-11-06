# 000 — Memory: Overview

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `00-foundation/000-architecture` (the memory↔wiki model)

---

## 1. What this function is

The memory function is the **raw, sync-ready source-of-truth fact store** for an AI
coding agent. Per the memory↔wiki model (`000-architecture` §3):

- It holds **atomic durable facts** — decisions, bugfixes + root-cause, discoveries,
  preferences, session records, and the relations/provenance between them.
- It is **not a brain.** No distillation, no personas, no deep retrieval-for-
  understanding. Those live in the llm-wiki function, which reads *this* store.
- Its retrieval is **lookup** — find the exact raw fact. The "what do we know about X"
  question goes to the wiki (wiki-first, memory-fallback).

## 2. The lifecycle

```
   agent decides something is worth remembering   (INV-7: agent saves, engine stores)
        │
        ▼
   mem_save  ────────────────────────────────────────────────────────────────┐
        │                                                                     │
        ├─(1) redact <private> + truncate                                     │
        ├─(2) conflict-candidate pre-filter (lexical FTS + relevance gate)     │
        ├─(3) LLM Classify (dedup: add/update/delete/noop)   [degraded: hash]  │
        ├─(4) upsert on topic_key (version snapshot) / insert                 │
        ├─(5) enqueue sync mutation (if sync enabled)                          │
        ├─(6) commit (one local transaction)                                   │
        ├─(7) return result + candidates[] + judgment_required                 │
        └─(8) emit SaveEvent ──────────────────────────────► llm-wiki          │
                                                                   (eager,    │
                                                                    incremental)
   next session:
   mem_context → mem_search → mem_get_observation        (the lookup ladder)
   OR  wiki-query (the deep retrieval) → mem_lookup fallback  (wiki-first)

   session close:
   mem_session_summary (six-section) → mem_session_finalize (atomic)
```

## 3. The six specs

| Spec | What it defines |
|---|---|
| `001-observations` | The core fact row, the save pipeline (redact→dedup→upsert→sync→event), the write queue, the save event. |
| `002-sessions` | Session lifecycle, identity, terminal semantics, resume, finalize, the six-section summary. |
| `003-facts-relations` | The relation model: vocabulary, conflict-candidate detection, judgment, consolidation. |
| `004-dedup` | The LLM 4-way classification + hash floor, pre-filter, the identity rules. |
| `005-lookup` | The retrieval ladder (context/search/get) + the wiki-first `mem_lookup` fallback. |
| `006-sync` | The full replication protocol: mutation journal, tombstones, deferred-apply, autosync, conflict model, export/import. |

## 4. The invariants (re-stated for this function)

INV-1 (provenance), INV-2 (dual-mode), INV-3 (LLM-first, explicit degradation),
INV-4 (local truth), INV-5 (retrieval trails), INV-6 (sync_id), INV-7 (agent saves).

## 5. What this function is NOT

- **Not the wiki.** It emits the SaveEvent; it does not render pages, resolve entities
  into docs, or do the graduate-rule. That is `03-llm-wiki`.
- **Not a transcript store.** It is selective memory (INV-7) — the agent curates.
- **Not a second authority.** Postgres is replication; local SQLite is truth (INV-4).

## 6. Design lineage

- **Engram** (`/data/git/AI/engram`): the flat `observations` model, `sync_id` identity,
  `topic_key` upserts, `memory_relations` + conflict vocabulary, decay/review,
  consolidation (preview/confirm + fingerprint), replay-safe saves, the 27-tool MCP
  surface, the 3-layer progressive-disclosure retrieval, the full sync/autosync
  protocol, last-write-wins. **This is the foundation.**
- **mnemonic v1** (`mnemonic/internal/memory`): the LLM seams (`DedupLLM.Classify`),
  the importance score, the two-phase async commit. **Adopted selectively.**
- **Dropped from v1:** the dream/L0→L3 distillation and personas (→ wiki), the deep
  hybrid retrieval as the primary path (→ wiki-query), the web-cache (dropped entirely).

## 7. Open items

None — the function's scope is fixed by the memory↔wiki model.
