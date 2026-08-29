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

1. `mem_session_start` at session open (creates workspace session). Pass `title` — a short human-readable name for what the session is about, e.g. `"Skillgrid CLI dashboard status card updates"`. It is shown in the web dashboard session list (`mem-sessions`).
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

## Privacy

Wrap anything sensitive (tokens, secrets, PII, internal paths) in `<private>…</private>`.
The value is stripped from stored content, replaced by `[REDACTED]`.

## Passive Capture (automatic)

Mnemonic captures memory beyond explicit `mem_save` — no agent work required:
- **User prompts** — each non-trivial chat message is stored (for "what were we
  working on" recall).
- **Task output** — when a `Task`/subagent finishes, the server extracts
  `## Key Learnings:` bullets and numbered items into passive observations
  (type inferred, e.g. `bugfix` / `decision` / `discovery`).
- **Save nudges** — if 15+ min pass without a `mem_save`, the system prompt
  reminds you.

To make Task output captureable, include a short `## Key Learnings:` section
(numbered or bulleted) at the end of long task output.

## After Compaction

If you see `FIRST ACTION REQUIRED` or a compaction notice as your first
prompt, call `mem_session_summary` with the compacted summary before doing
anything else — without this, everything done before the compaction is lost
from the persistent store.
