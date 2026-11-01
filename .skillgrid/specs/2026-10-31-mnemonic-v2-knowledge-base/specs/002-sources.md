# 002 — Sources (the four adapters, all read live)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (§5.4 the `sources:` citation format), the memory change
(`004-transport-mcp` §5 the `SaveEvent`; `002-schema` the memory store), the
`2026-10-31-mnemonic-v2-code-index` change (the code graph).
**Reference:** openwiki grounded claims, pi-llm-wiki source discipline, the reborn
web-cache (mnemonic v1 `web_*`).

---

## 1. Capability

The KB is synthesized **from four sources**. This spec defines each adapter and the one
rule that binds them: **all four are read live, none are stored as packets** (KB-8).
The KB does not copy source material into the vault. It **reads** memory, the code
graph, and git history from where they already live, and **cites** external sources by
URL+version in the page frontmatter.

The contrast with v1 is deliberate: v1 had a persistent `web_cache` (a stored packet
store). v2 drops it (OPEN-3 in the memory change). The KB's "external source" is not a
cached body — it is a **citation** the agent can re-fetch on demand.

## 2. The four adapters

| Adapter | Read from | Stored in the vault? | Citation form |
|---|---|---|---|
| **memory** | the memory store (live), triggered by the `SaveEvent` | ❌ (the fact stays in memory) | `memory:<sync_id>` |
| **code** | the code-index graph + working tree (live) | ❌ (the code stays in the repo) | `code:<path>#L<a>-<b>` |
| **git history** | `git log` / `git blame` (live) | ❌ | `code:<path>#L<a>-<b>` + a commit ref in the page body |
| **external** | re-fetched on demand (URL), section-granular | ❌ (cited only) | `external:<name>` + `version` + `source_url` + `source_commit` + `build_fingerprint` |

### 2.1 memory (the eager trigger)

The memory function emits a `SaveEvent` after every committed `mem_save`
(`004-transport-mcp` §5). The KB **subscribes** to the event bus (it does not import the
memory package — the event bus is the boundary, dependency inversion). On an event, the
KB **reads the fact live** by `sync_id` (and the related judged relations, conflicts,
supersedes) from the memory store. The event is the *trigger*; the read is *live*.

**The graduate gate:** not every fact graduates to a page. The graduate-rule
(`003-generate`) reads the fact's `importance` (memory `002-schema` §2.2.3) + `type` to
decide whether it is durable enough to synthesize. A low-importance `session_summary` is
not a page; a high-importance `decision` is.

### 2.2 code (the graph + working tree)

The KB reads the **code-index graph** (symbols, edges, files — the `02-code-index`
change, cross-referenced) to (a) resolve `code:` citations to real spans, (b) build
`architecture`/`workflow` pages from the **package map** (`code_packages`,
`02-code-index/004` §4) and the **call graph** (`code_get_callers`/`code_get_callees`),
and (c) resolve `symbol:` anchors during the pin-check (`code_resolve`,
`02-code-index/004` §3, `006-validate` §2). For token-level anchors (`all_of`, `line`)
the KB reads the **working tree** directly (ripgrep) — no graph needed. **The graph is a
first-class but optional dependency** (`02-code-index` CI-7): without it, the `symbol:`
check degrades to a ripgrep occurrence check (`degraded: no_graph`) and the coverage-gap
lint falls back to a `go list` package scan — both explicit, never a silent floor.

The code adapter is **live**: it reflects the current working tree, not a snapshot. If
the code moves, the page's anchors/claims go stale and `kb_validate` catches it.

### 2.3 git history (the why-provenance)

Git history is read live (`git log`, `git blame`, `git show`) to (a) attribute a decision
to the commit/session that made it, and (b) surface the *why* behind a change (the commit
message, the linked session). This feeds `decision` pages: the "why" is often in the
commit history, not in a memory fact.

**Not stored:** the KB does not snapshot commits. It cites the commit ref in the page
body and re-derives from git on demand.

### 2.4 external (the reborn web-cache, cited not stored)

When a page relies on an external source (a library's docs, a spec, an API reference),
the KB records a **citation** in `sources:` using the **neuledge/context identity form**
(a versioned, searchable, re-fetchable reference — not a stored packet):

```yaml
- external:somelib            # the library name (identity, not the URL)
  version: v1.4.2             # the pinned version
  source_url: https://pkg.go.dev/somelib#Capture   # where the docs live
  source_commit: 9f2c1ab      # the upstream revision this version was built from
  build_fingerprint: sha256:ab…  # the staleness token (see below)
```

The KB does **not** store the fetched body. The agent (or the LLM during generation)
re-fetches the body **on demand**, at **section granularity** (a `##` heading, not a line
and not a whole page) when it needs it. This is the v2 inversion of neuledge's
download-once-local model: neuledge downloads a pre-built `.db` once and never re-fetches
(freshness is a publish-side nightly job); the KB **cites + re-fetches on demand**, and the
staleness check neuledge runs at *publish* time is moved to the KB's **re-fetch + lint**
time.

- **`kb_cite_source`** records a citation: given a library name (+ version + source_url),
  it computes the `build_fingerprint` (below) and appends the `external:` citation to a
  page's `sources:` (or a page under construction). It does not fetch or store the body.
- **Re-fetch (on demand):** when the agent needs the body, it re-fetches the cited
  version's docs and searches them **section-granularly** — BM25 over the sections, with
  the neuledge weights (section-title 10.0, doc-title 5.0, content 1.0) and a token
  budget (default 2000 tokens) + a 50% relevance drop, so the result is bounded and the
  right section surfaces. The citation the agent records back into a page is
  `name@version` + `doc_path` + section title (the in-repo path, not a line range).
- **Staleness = the fingerprint gate, run at re-fetch/lint time (neuledge's
  `publicationSkipReason`, moved client-side).** `build_fingerprint` is a SHA-256 over
  `(name, version, source_url, source_commit, effective source settings, ingestion
  revision)`. On re-fetch or `kb_lint`:
  - if the cited `source_commit` / `build_fingerprint` still matches what's upstream →
    the citation is **fresh** (no work);
  - if the fingerprint changed (the docs moved at the same version) → **stale** (the
    agent should re-cite at the new revision);
  - if the version/URL no longer resolves → **unresolved** (the source is gone or renamed).
  This is the TTL replacement (KB-O4, moot → fingerprint-gated freshness in re-fetch +
  lint), not a clock expiry.
- **No auto-discovery in v2** (KB-O6): external sources are cited explicitly. Auto-
  finding "you now use lib X, cite its docs" is a v2.1 follow-up.

## 3. The citation format (the contract)

The `sources:` frontmatter is the **only** place a source is recorded. Its three forms
(the citation contract defined in `001-structure` §5.4) are:

```yaml
sources:
  - memory:obs-1f3a…                        # memory fact (live)
  - code:internal/memory/service.go#L480-L500   # code span (live)
  - external:somelib                        # external (cited, re-fetchable)
    version: v1.4.2
    source_url: https://pkg.go.dev/somelib#Capture
    source_commit: 9f2c1ab
    build_fingerprint: sha256:ab…
```

**Rules:**

- A `memory:` citation must reference a **live** `sync_id` (or a soft-deleted one, which
  lint flags as stale). It is the provenance link back to the raw fact.
- A `code:` citation must resolve to a path + line span that **exists** in the working
  tree at validate time (a deleted/renamed file is a drift signal).
- An `external:` citation is a reference, not a body. It carries the **neuledge identity
  form** (`external:<name>` + `version` + `source_url` + `source_commit` +
  `build_fingerprint`). The `build_fingerprint` is what lint checks for drift (a
  changed fingerprint = `stale`; an unresolvable version/URL = `unresolved`). `version`
  and `source_url` are required for a versioned lib; `source_commit` /
  `build_fingerprint` are computed by `kb_cite_source`.
- **A grounded page with no `sources:` is a lint finding** (KB-INV-2). A contextual page
  may have fewer, but still should cite what it synthesizes from.

## 4. What is NOT a source

- **The vault itself.** A page is not a source for another page in the provenance sense —
  wikilinks are *cross-references*, `sources:` are *provenance*. You can link to
  `[[decisions/auth-model]]` (cross-reference) while citing `memory:obs-…` (provenance)
  independently.
- **Trajectories / working memory.** pi's agent-trajectory / working-memory tools are
  out of scope for v2 (not adopted). The KB synthesizes from durable sources, not from a
  session's scratch state.

## 5. Invariants

INV-1 (every page cites its sources), INV-3 (LLM-first, explicit `degraded` when a
source read fails), KB-INV-2 (grounded pages are grounded), KB-8-as-invariant (no
packets — sources are read live or cited).

## 6. Open items

KB-O6 (auto-discovery, deferred to v2.1). KB-O4 is moot (no TTLs — no packets). No other
open items; the four adapters and the citation contract are fixed.
