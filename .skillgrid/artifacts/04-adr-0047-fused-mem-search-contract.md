# Fused mem_search Contract

---
status: "accepted"
supersedes: none
date: 2026-10-18
---

**Related:** `04-adr-0012-sqlite-as-second-brain.md` (the fused corpus is ADDITIVE
on that single store — no second source of truth), `04-adr-0046-knowledge-class-dimension.md`
(the knowledge_class filter keeps applying to the observations leg of the fused
search).

## Context and Problem Statement

`mem_search` queried `observations_fts` only. The project's `web_cache` —
snapshots of web research fetched via `web_cache_lookup` / `web_cache_save` —
was reachable solely through its own tools (`web_cache_search`,
`web_cache_get`). So a web finding an agent had cached weeks ago never
surfaced in default recall: an agent asking "what do we know about X" got the
observations corpus and nothing from the research corpus, even when a fresh,
unexpired web snapshot for X sat in the same store.

The two corpora already live in the same project-scoped SQLite file
(ADR-0012), and `web_cache_fts` (title, content, query, url, source,
library_id) already exists with triggers. The gap was purely on the read
path: `mem_search` never looked at the second corpus.

## Considered Options

- **Document the two tools (status quo):** zero code change, but recall stays
  split by habit — the agent must remember to call `web_cache_search`
  separately, and a `mem_search` result never tells it the web corpus exists.
- **Separate `mem_search` + `web_search` responses merged by the caller:**
  pushes the fusion (ordering, labeling, expiry) to every caller instead of
  doing it once at the source.
- **Fuse at the source (chosen):** every `mem_search` also runs an FTS
  `MATCH` over `web_cache_fts` (project-scoped, expired rows excluded) and
  merges the hits into the same ranked result set via the existing RRF
  (k=60). Callers get one result list; web hits are labeled so they can be
  distinguished and fetched.

## Decision Outcome

Every `mem_search` (the `SearchOwnerScopedBlend` path — the scoped,
unfiltered default) is now a **two-corpus** search:

1. **Observations leg — unchanged.** FTS over `observations_fts` + the
   optional vector leg, owner-scoped, exactly as before. The vector leg
   stays observations-only: web rows are not embedded and never participate
   in vector fusion. The `knowledge_class` filter (ADR-0046) still applies to
   this leg; web hits are not classed.
2. **Web leg — new.** One additional FTS `MATCH` over `web_cache_fts`,
   restricted to this project's rows and to rows not yet expired
   (`expires_at IS NULL` or in the future). No ownership/visibility gating —
   web_cache is project-shared research, not a per-owner asset.
3. **Fusion.** The two ranked lists are merged with the existing
   ReciprocalRankFusion (k=60). When there is no vector leg, web hits follow
   the observation hits in their own BM25 order; with a vector leg, all
   three ranked lists fuse.
4. **Labeling.** A web hit is labeled `corpus=web_cache` and carries the row's
   own provenance: `source`, `url`, `fetched_at`, `expires_at`, plus a
   retrieve pointer (`web_cache_get id=<id>`) so the caller can fetch the
   full untruncated body. Observation hits are labeled `corpus=observations`
   (their retrieve pointer is `mem_get_observation id=<id>`).

This is a **BEHAVIOR SHIFT for every `mem_search` caller**: a result that
previously contained only observations may now contain labeled web_cache
entries, interleaved by fused rank. The shift is recorded here so the
contract is explicit — callers that pinned exact result sets were
re-baselined in the change that introduced it (TICKET-06,
change `2026-10-06-second-brain-learnings`).

## Consequences

- **Additive for the observations leg.** Observation hits are ranked,
  labeled, and fetched exactly as before; the vector leg is untouched.
- **Web hits are new in the result set.** Any unexpired, project-scoped
  web_cache row matching the query terms can now surface in `mem_search`.
- **Expired rows never surface.** The web leg filters `expires_at` at query
  time (NULL or future), so stale snapshots do not enter default recall.
- **No schema change.** `web_cache_fts` already exists (migration 001); the
  leg is a read-side addition only.
- **The all-projects path is not fused.** `mem_search all_projects=true`
  spans the observations corpus across stores only; the web leg is
  project-scoped to the store the search runs under.
- **The class-filtered path is observations-only.** Web hits have no
  durability class, so a `knowledge_class`-scoped search cannot return web
  hits without inventing a class for them.
- **Budget truncation keeps fused rank.** The in-list budget cuts the fused
  (interleaved) list to the cap, so a high-rank web hit is kept, not
  dropped, when the cap is reached.
