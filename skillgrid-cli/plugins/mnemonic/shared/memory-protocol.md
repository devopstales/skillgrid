# Mnemonic Memory Protocol

Mnemonic is skillgrid's local-first persistent memory subsystem (SQLite + FTS5).
It powers `mem_*`, `code_*`, and `web_*` tools over MCP stdio and HTTP.

## When to Save

Call `mem_save` immediately after any of:
- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User correction to your approach

## Save Format

**title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList")
**type**: decision | architecture | bugfix | pattern | config | discovery | learning | preference | convention
**scope**: project (default) | user | global
**topic_key** (optional): stable key like `architecture/auth-model` for upserts

Content body with structured sections:
- **What**: One sentence — what was done
- **Why**: What motivated it
- **Where**: Files or paths affected
- **Learned**: Gotchas, edge cases (omit if none)

## Topic Update Rules

- Reuse the same `topic_key` to update an evolving topic
- Different topics must not overwrite each other
- Use `mem_suggest_topic_key` when unsure

## Session Protocol

1. `mem_session_start` at session open (creates workspace session)
2. `mem_save` during work (dedups by hash within 24h, upserts by topic_key)
3. `mem_session_summary` before closing
4. `mem_session_end` to close

## Search

- `mem_context` — recent session summaries (fast recall)
- `mem_search` — FTS5 over observations (match_mode: any|all)
- `mem_get_observation` — full content by ID

## Code Index Ladder

1. `code_status` — check index health
2. `code_index` — run incremental index if stale
3. `code_search` — BM25 FTS over chunks
4. `code_read` — source slice for a path

## Web Cache

- `web_cache_lookup` — check cache before remote MCPs
- `web_cache_save` — persist after Context7/Exa/DeepWiki/WebFetch
- `web_cache_search` — FTS5 over cached research
