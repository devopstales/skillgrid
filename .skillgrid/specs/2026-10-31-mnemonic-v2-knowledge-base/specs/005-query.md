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

## 8. Invariants

INV-3 (explicit `degraded: stale_index`), INV-1 (cited sections — every stage-2 result
carries the page's `sources:`), INV-5 (trails), INV-4 (local truth — the query reads the
local index, never a remote authority).

## 9. Open items

None. The two-stage pattern, the ranking, and the fallback handoff are fixed. The
`thin` threshold (how few pages before the fallback hint fires) is the one tunable —
defaulted in `008-transport` config.
