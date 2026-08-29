---
name: mnemonic-search-cache
description: "Use when the user needs to cache a remote research result, check whether a cached lookup is still fresh, search prior cached research, or configure the web-research cache in Mnemonic (web_cache_lookup, web_cache_save, web_cache_search, web_cache_get, web_cache_status)."
---

# Mnemonic Search Cache

Mnemonic's web-research cache stores snapshots of remote research (Context7,
Exa, DeepWiki, raw fetch) in the same per-project SQLite store as memory
observations and the code index. It is exposed via five MCP tools:
`web_cache_lookup`, `web_cache_save`, `web_cache_search`, `web_cache_get`,
`web_cache_status`.

## Storage

| Table | Purpose |
|---|---|
| `web_cache` | Cached snapshots (project, source, cache_key, content, expires_at, …) |
| `web_cache_fts` | FTS5 virtual table over `web_cache.content` |

Upsert is by `(project, source, cache_key)`. The same source + key
combination is silently overwritten on save — there is no version history.

## The Five Tools

### web_cache_lookup

Check the cache **before** calling a remote MCP.

| Parameter | Required | Source it applies to |
|---|---|---|
| `source` | yes | all |
| `url` | conditional | `fetch` |
| `library_id` | conditional | `context7` |
| `version_tag` | conditional | `context7` |
| `query` | conditional | `context7`, `exa` |
| `repo_name` | conditional | `deepwiki` |
| `question` | conditional | `deepwiki` (falls back to `query`) |
| `sort_params` | conditional | `exa` |
| `title` | conditional | `manual` |
| `content_hash` | conditional | `manual` |

Returns:

```json
{"status": "hit", "fresh": true, "id": 42, "fetched_at": "...", "expires_at": "..."}
```

Status values:
- `"hit"` — entry found and not expired
- `"stale"` — entry found but past its TTL (`fresh: false`)
- `"miss"` — no entry for this `(project, source, cache_key)`

### web_cache_save

Persist a snapshot **immediately** after a remote MCP or fetch returns.

| Parameter | Required | Notes |
|---|---|---|
| `source` | yes | one of the 5 valid sources |
| `content` | yes | snapshot body, max 256 KB (configurable) |
| `url` | no | source URL |
| `title` | no | human-readable title |
| `query` | no | original query text |
| `library_id` | no | Context7 library id |
| `version_tag` | no | Context7 version tag |
| `repo_name` | no | DeepWiki repo name |
| `question` | no | DeepWiki question |
| `sort_params` | no | Exa sort params |
| `session_id` | no | active Mnemonic session id |
| `metadata` | no | JSON string, optional |

Returns `{"id": 42}` — the upserted entry id. Re-saving the same `(project,
source, cache_key)` updates all fields and refreshes `fetched_at` /
`expires_at`.

### web_cache_search

FTS5 search over cached snapshot content.

| Parameter | Required | Default | Notes |
|---|---|---|---|
| `query` | yes | — | search keywords |
| `source` | no | all | filter by source |
| `fresh_only` | no | `true` | exclude expired entries |
| `limit` | no | `20` | max results |

Returns:

```json
{"entries": [{"id": 42, "source": "context7", "title": "...", "query": "...", "url": "...", "library_id": "...", "fetched_at": "...", "expires_at": "..."}]}
```

Search is a **multi-term OR query** — each whitespace-delimited term is
quoted and joined with `OR`. `"foo bar"` searches for `"foo" OR "bar"`.
Results are ordered by BM25 rank (best match first).

### web_cache_get

Fetch the full untruncated snapshot by id.

| Parameter | Required | Notes |
|---|---|---|
| `id` | yes | entry id from `lookup` or `search` |

Returns the full `WebEntry` including `content`, `content_hash`,
`metadata`, `cache_key`, etc.

### web_cache_status

No parameters. Returns aggregate stats:

```json
{
  "total_entries": 42,
  "expired_entries": 3,
  "by_source": {"context7": 10, "exa": 8, "deepwiki": 5, "fetch": 12, "manual": 7},
  "oldest_fetch": "2026-08-01T00:00:00Z",
  "newest_fetch": "2026-08-29T08:00:00Z"
}
```

## Cache Key Derivation

The cache key is a SHA-256 hex digest of source-specific material. This is
what `lookup` matches against — if you get the key wrong, you get a miss even
when an entry exists.

| Source | Required fields | Key material |
|---|---|---|
| `fetch` | `url` | normalized URL (lowercased host, no trailing slash, no fragment) |
| `exa` | `query` | `query\|sort_params` |
| `context7` | `library_id`, `query` | `library_id\|version_tag\|query` |
| `deepwiki` | `repo_name`, `question` (or `query`) | `repo_name\|question` |
| `manual` | `title`, `content_hash` | `title\|content_hash` |

When calling `web_cache_save`, you do not pass the cache key — it is derived
automatically from the fields above. When calling `web_cache_lookup`, pass
the same fields you would pass to save; the service derives the same key and
matches it.

## Staleness and TTL

TTL defaults (from `internal/mnemonic/config/load.go:DefaultWebCache`):

| Source | TTL |
|---|---|
| `context7` | 30 days |
| `exa` | 7 days |
| `deepwiki` | 14 days |
| `fetch` | 7 days |
| `manual` | none (no expiry) |

TTLs are overridable per-project in `config.d/indexing.yaml`:

```yaml
mnemonic:
  web_cache:
    ttl:
      context7: 720h
      exa: 168h
      deepwiki: 336h
      fetch: 168h
      manual: 0
    max_entry_bytes: 262144
```

`web_cache_lookup` returns `"status": "stale"` when the entry's `expires_at`
is in the past. `"stale"` is NOT the same as `"miss"` — the entry exists,
it is just expired. A stale entry can still be fetched with `web_cache_get`
if you need the old content for comparison.

`web_cache_search` with `fresh_only: true` (the default) filters out expired
entries. Pass `fresh_only: false` to include stale hits in results.

## Search Behavior

From `internal/mnemonic/webcache/service.go:buildFTSQuery`:

- The query is split on whitespace into individual terms.
- Each term is double-quote-escaped (`"` → `""`) and wrapped in `"..."`.
- Terms are joined with ` OR `.
- Empty or whitespace-only queries return no hits (not an error).

Example: `"Context7 React hooks"` → FTS query: `"Context7" OR "React" OR "hooks"`.

This is different from `code_search` (which is a single phrase query). The
web cache uses an OR query by design — you want any matching snapshot, not
only snapshots that contain the exact phrase.

## The Workflow

```
1. web_cache_lookup(source, ...)     → check before remote call
2. [if miss or stale] remote MCP      → fetch fresh content
3. web_cache_save(source, content)    → persist immediately after
4. [later] web_cache_search(query)    → find what was cached
5. web_cache_get(id)                  → read full snapshot
```

Rule of thumb: **always** `web_cache_lookup` before Context7 / Exa /
DeepWiki / fetch. **Always** `web_cache_save` immediately after. The cache
is only useful if it is populated.

## Integration with Mnemonic

The web cache is one of three Mnemonic tool families (`mem_*`, `code_*`,
`web_*`):

- **Session scope** — cache entries are scoped to the same project as memory
  observations. `mem_save(scope: project)` and `web_cache_save` write to
  the same SQLite file.
- **Before a remote call** — `web_cache_lookup` is the cheapest possible
  action. A hit is sub-millisecond; a remote call is seconds.
- **After a remote call** — `web_cache_save` within the same turn. Do not
  defer it to a later session — the session id / project context is tighter
  now.
- **When the user asks "what did we find about X"** — `web_cache_search`
  with `fresh_only: true` (default). If the answer is stale, re-fetch and
  re-save in the same turn.
- **Memory tie-in** — after a particularly useful cached session, `mem_save`
  a `discovery` observation with the `cache_key` and the source so you can
  find it again via `mem_search` even if the web cache entry expires.

## CLI

The same cache is accessible via HTTP when `skillgrid serve` is running
(default `127.0.0.1:7438`):

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/web/lookup` | Check cache before remote call |
| `POST` | `/web/cache` | Persist a snapshot |
| `GET` | `/web/search` | FTS5 search over cached snapshots |
| `GET` | `/web/entry/{id}` | Full snapshot by id |
| `GET` | `/web/status` | Cache health stats |

Write routes require `SKILLGRID_HTTP_TOKEN` bearer auth when configured.

## Gotchas

- **256 KB cap** (default, configurable via `max_entry_bytes`). Snapshot
  content larger than this is **rejected** by `web_cache_save` with an
  explicit error — not silently truncated. Summarize before saving.
- **`manual` source never expires** — `expires_at` is empty. `fresh_only`
  does not filter manual entries.
- **Cache key collisions are silent** — two different lookups with the same
  derived key overwrite each other. This is by design (dedup), but it means
  you cannot store two Context7 results for the same `library_id + query`
  pair. Use `query` specificity to differentiate.
- **`web_cache_search` uses OR, not AND** — do not expect exact-phrase
  behavior. Use `web_cache_get(id)` to confirm a hit is the right one.
- **No cache invalidation tool** — expired entries accumulate until
  `web_cache_status` reports them. There is no `web_cache_clean` command
  in v1. Use `web_cache_search(fresh_only: false)` if you need to inspect
  stale entries, or delete the SQLite file to start fresh.
- **Per-project scope** — entries are not shared across projects. Multi-repo
  workspaces index each repo into its own SQLite file; the web cache follows
  the same rule.
- **Content is stored as-is** — no normalization, no markdown stripping.
  Large HTML pages from `fetch` are stored verbatim. Pre-process before
  saving if size is a concern.
- **Metadata is JSON-string** — the `metadata` field on `web_cache_save` is
  a JSON string, not an object. MCP marshals it; pass a JSON-string value.
