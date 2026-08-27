# MemIndex Memory Protocol

Persistent, local-first memory backed by `skillgrid mcp` / `skillgrid serve`
over one SQLite store. Survives across sessions and compactions.

## Memory tool surface

| Tool | Purpose |
|------|---------|
| `mem_save` | Persist a curated observation (What/Why/Where/Learned). |
| `mem_search` | FTS5 full-text search over observations. |
| `mem_context` | Recent session summaries (fast recall before a full search). |
| `mem_get_observation` | Fetch full untruncated observation by id. |
| `mem_session_start` | Open a workspace session (plugin flow). |
| `mem_session_end` | Close a session, optional final summary. |
| `mem_session_summary` | Persist structured end-of-session summary. |
| `mem_suggest_topic_key` | Derive a stable `topic_key` from type + title. |

## Code tool surface

| Tool | Purpose |
|------|---------|
| `code_status` | Index health (counts, last-index, staleness hint). |
| `code_index` | Incremental indexing (mtime + content hash). |
| `code_search` | BM25 full-text search over chunks. |
| `code_read` | Fetch a path slice by path + line range. |

Ladder: `code_status → code_search → code_read`. Prefer over grep in unknown
territory; use grep for exact identifiers.

## Web cache surface

| Tool | Purpose |
|------|---------|
| `web_cache_lookup` | Check the local cache BEFORE a remote MCP/fetch. |
| `web_cache_save` | Persist a research snapshot AFTER a remote MCP/fetch returns. |
| `web_cache_search` | FTS5 search over cached research (fresh/stale aware). |

Sources: `context7`, `exa`, `deepwiki`, `fetch`, `manual`.
Defaults TTL: context7 30d, exa 7d, deepwiki 14d, fetch 7d, manual none.
Entries cap at 256KB; summarize before caching when larger.

## WHEN TO SAVE (mandatory — not optional)

Call `mem_save` IMMEDIATELY after any of:
- Bug fix completed
- Architecture / design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned

`mem_save` fields:
- `type`: standing | preference | convention | decision | bugfix | lesson |
  correction | discovery | session_log
  (aliases accepted: architecture→decision, pattern→convention, config→discovery)
- `scope`: project (default) | user | global
- `topic_key`: stable key for upserts (e.g. `architecture/auth-model`) —
  call `mem_suggest_topic_key` if unsure. Reuse the same key to update an
  evolving topic instead of creating duplicates.
- `title`: verb + what (short, searchable)
- `content`: `**What** / **Why** / **Where** / **Learned**` sections

## Topic update rules (mandatory)

- Different topics must not overwrite each other (`architecture` vs `bugfix`).
- Reuse `topic_key` to update an evolving topic, not to create new rows.
- Call `mem_suggest_topic_key` before the first save on an unfamiliar topic.

## WHEN TO SEARCH MEMORY

Any variation of "remember / recall / what did we do / how did we solve":
1. `mem_context` — recent session history (fast, cheap).
2. `mem_search` with keywords (FTS5) if not found.
3. `mem_get_observation` for the full untruncated content.

Also search PROACTIVELY when:
- Starting work that might have been done before.
- The user mentions a topic with no current context.

## Code index rules

1. `code_status` — confirm freshness before broad search (stale → `skillgrid index`).
2. `code_search(query, limit)` — ranked chunks with path + line range.
3. `code_read(path, start_line, end_line)` — full source for chosen hits.

## Web research rules

1. `web_cache_lookup(source, cache_key)` — exact prior fetch.
2. If miss: call the remote MCP (Context7 / Exa / DeepWiki / web fetch).
3. `web_cache_save(source, cache_key, content, ...)` IMMEDIATELY after.
4. "What did we find about X online?" → `web_cache_search(query)`.

## SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it":
1. `mem_session_summary` with:
   `Goal / Instructions / Discoveries / Accomplished / Next Steps / Relevant Files`

Not optional — the next session starts blind without it.

## After compaction

On a compaction / context reset message:
1. IMMEDIATELY persist what was done before compaction via `mem_session_summary`.
2. Then `mem_context` to recover additional prior-context.
3. Only THEN continue.
