# 005 — Lookup (retrieval ladder + the wiki-first fallback)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `001-storage` (§3 FTS, §4 vector), `002-schema`
**Reference:** Engram 3-layer progressive disclosure (`docs/ARCHITECTURE.md` §72-80),
`mem_context` byte budget; the wiki-first/memory-fallback decision (OPEN-12).

---

## 1. Capability

This is the **lookup half** of the retrieval model. Per the memory↔wiki model, the
agent's *understanding* retrieval goes to the wiki (wiki-first). This spec is the
**raw-fact lookup** the agent uses to (a) drill into a specific fact and (b) **fall
back** when the wiki is thin/empty on a topic.

The retrieval is **progressive** (Engram's 3-layer pattern) — token-efficient, drill in:

```
1. mem_context                 → recent sessions/prompts/observations (byte budget)
2. mem_search "query"          → compact rows (~100 tokens each, ids + titles + previews)
3. mem_get_observation id=N    → full untruncated content (+ version history)
```

Plus the two v2 additions: `mem_lookup` (the narrow fallback) and `VectorSearch`
(armed when the embedder is present).

## 2. `mem_context` (Layer 0 — orientation)

Recent context with a **byte budget** (default 16 KiB, up to 64 KiB):
- ≤20 pinned observations (pinned first),
- recent sessions + prompts + observations,
- `older_history=true` adds a capped calendar overview of out-of-window observations.

This is the cheap "what have I been doing" recall — fired at session start (by the
plugin hook or the agent). It is **not** semantic; it is recency + pinning.

## 3. `mem_search` (Layer 1 — recall)

FTS (+ vector when armed) search:

- **Filters:** `type`, `project`, `scope`, `limit`, `match_mode` (`all`/`any`),
  `all_projects`.
- **Rank:** BM25 (title 5.0, content 1.0, topic_key 3.0) × composite boosts
  (Engram): `pinned +0.10`, `recency 0.06·1/(1+days/30)` on `last_seen_at||
  updated_at||created_at`, `stability 0.04·(rev+dups)/(rev+dups+4)`.
- **Vector blend:** when the embedder is armed, `VectorSearch` runs and results are
  RRF-fused with the FTS results. `degraded: true` when the vector leg is absent
  (INV-3) — never a silent fusion.
- **Projection:** compact rows only — id, title, type, a 300-char content preview +
  `truncated` flag, relation annotations. **Never hydrates full bodies** at this layer.
- **Excludes** soft-deleted observations.

**Provenance (INV-1):** mem results carry `observations`; code-family results (when
fused in a hybrid search) are never silently mixed with mem observations unlabeled.

## 4. `mem_get_observation` (Layer 2 — full detail)

Full untruncated content by id (or sync_id). `include_history` / `history_cursor`
page the `observation_versions` (50/page). This is the **only** full-content path —
previews are truncated by design.

## 5. `mem_lookup` — the wiki-first fallback (v2 addition)

The narrow, high-precision lookup the agent uses **when the wiki is thin/empty** on a
topic (OPEN-12):

- **By exact `topic_key`:** `mem_lookup(topic_key="architecture/auth-model", project,
  scope)` → the single upserted observation (or none). This is the highest-precision
  path — the topic_key is the identity.
- **By id / sync_id:** the full row.
- **By distinctive phrase:** a bounded FTS (trigram) with a hard recall cap (≤5), no
  vector, no boosts — a *lookup*, not a *search*.

`mem_lookup` is deliberately **dumb**: it does not rank, blend, or expand. It answers
"give me *that* fact," not "what do I know about X." The latter is the wiki's job.
If `mem_lookup` returns nothing, the agent treats it as *this specific fact is absent*
— not "search more broadly" (that's `mem_search` or the wiki).

## 6. The wiki-first contract (from `000-architecture` §3)

The **routing rule** (OPEN-12) is enforced by the agent protocol, not the engine:

```
"what do we know about X"  →  wiki_query        (the deep retrieval, llm-wiki)
     └─ if wiki is thin/empty on X  →  mem_lookup  (this spec, the fallback)
"find the exact fact / the session where we decided Z"  →  mem_lookup / mem_search
```

The memory function's obligation: `mem_lookup` and `mem_search` must be **good enough
that the fallback is rare and precise.** The wiki's quality determines how often
fallback happens; the memory function cannot fix a thin wiki, but it must make the
fallback *useful* when it happens.

## 7. Retrieval trails (INV-5)

Every `mem_search` / `mem_lookup` / `VectorSearch` records a trail: query, route
(`fts`/`vector`/`rrf`/`lookup`), the filters, what was returned (ids), `degraded`
status. `mnemonic trail recent|show` inspects them. This is the audit record for
"why did the agent recall this."

## 8. Invariants

INV-3 (explicit `degraded` on the vector leg), INV-1 (provenance: no silent fusion),
INV-5 (trails), INV-6 (ids are sync_id-visible).

## 9. Open items

None.
