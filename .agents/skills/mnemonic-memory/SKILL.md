---
name: mnemonic-memory
description: "ALWAYS ACTIVE — persistent memory protocol for skillgrid Mnemonic (mem_*, code_*, web_*, `skillgrid-mnemonic` MCP). You MUST save decisions, conventions, bugs, and discoveries proactively. Do NOT wait for the user to ask."
---

# Mnemonic Persistent Memory — Protocol

You have access to **Mnemonic**, skillgrid's local-first persistent memory
subsystem (SQLite + FTS5 in a single `skillgrid` binary). It survives across
sessions and compactions. MCP server name: `skillgrid-mnemonic`, launched via
`skillgrid mcp`. It exposes the `mem_*`, `code_*`, and `web_*` tool families
over MCP and a REST API.

This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on
demand.

## AVAILABLE TOOLS

Core `mem_*` tools are loaded automatically at session start by the plugin
hook. They are available immediately — no manual ToolSearch needed:

- `mem_save`, `mem_search`, `mem_context`, `mem_get_observation`
- `mem_session_start`, `mem_session_end`, `mem_session_summary`
- `mem_suggest_topic_key`

Code-index family (use for searching large unfamiliar repos):
- `code_status`, `code_index`, `code_search`, `code_read`

Web-research cache (check before remote MCPs):
- `web_cache_lookup`, `web_cache_save`, `web_cache_search`,
  `web_cache_get`, `web_cache_status`

**Fallback**: If tools are unexpectedly unavailable, run
`skillgrid setup <opencode|kilocode|cursor>` again and restart the agent.
Setup repairs the durable MCP config and the agent rule for the
`skillgrid-mnemonic` server.

## PROACTIVE SAVE TRIGGERS (mandatory — do NOT wait for user to ask)

Call `mem_save` IMMEDIATELY and WITHOUT BEING ASKED after any of these:

### After decisions or conventions
- Architecture or design decision made
- Team convention documented or established
- Workflow change agreed upon
- Tool or library choice made with tradeoffs

### After completing work
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- External artifact created or updated with significant content
  (GitHub issue, PR, Notion, etc.)
- Configuration change or environment setup completed

### After discoveries
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User correction to your approach

### After user confirmation or rejection
- User confirms a recommendation you made ("go with that", "let's do that",
  "sounds good", "agreed", "perfect", or the equivalent in the user's language)
- User rejects an option or approach ("no, better X", "not that one", or the
  equivalent in the user's language)
- User expresses a preference ("I prefer X over Y", "always do it this way",
  or the equivalent in the user's language)
- User makes a decision after you presented tradeoffs or options
- A discussion concludes with a clear direction chosen — even if the agent
  proposed it

### Self-check — ask yourself after EVERY task:
> "Did I or the user just make a decision, confirm a recommendation, express a
> preference, fix a bug, learn something non-obvious, or establish a
> convention? If yes, call `mem_save` NOW."

Format for `mem_save`:
- **title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList", "Chose Zustand over Redux")
- **type**: one of the 13 allowed values (`internal/mnemonic/memory/service.go:360`):
  `standing | preference | convention | decision | architecture | bugfix |
  pattern | config | correction | discovery | learning | lesson | session_log`
- **scope**: `project` (default) | `user` | `global`
- **session_id**: your active session ID (required — from `mem_session_start`)
- **topic_key** (optional but recommended for evolving topics): stable key
  like `architecture/auth-model`
- **content**:
  **What**: One sentence — what was done
  **Why**: What motivated it (user request, bug, performance, etc.)
  **Where**: Files or paths affected
  **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic update rules (mandatory)

- Different topics MUST NOT overwrite each other (example: architecture
  decision vs bugfix)
- To evolve an existing topic, re-call `mem_save` with the same `topic_key`
  plus the updated content — Mnemonic upserts instead of creating a new row.
  There is no `mem_update` tool in Mnemonic — upsert-via-`mem_save` is the
  mechanism.
- If unsure about the key, call `mem_suggest_topic_key` first, then reuse
  that key consistently for the same topic
- Use `mem_get_observation(id)` to read the current untruncated content
  before rewriting it

## WHEN TO SEARCH MEMORY

When the user asks to recall something — any variation of "remember",
"recall", "what did we do", "how did we solve", or the equivalent in the
user's language, or references to past work:
1. First call `mem_context` — recent session summaries (fast, cheap)
2. If not found, call `mem_search` with relevant keywords (FTS5 full-text,
   `match_mode: any|all`)
3. If you find a match, use `mem_get_observation(id)` for the full untruncated content

Also search memory PROACTIVELY when:
- Starting work on something that might have been done before
- The user mentions a topic you have no context on — check if past sessions
  covered it
- The user's FIRST message references the project, a feature, or a problem —
  call `mem_search` with keywords from their message to check for prior work
  before responding

## SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", you MUST:
1. Call `mem_session_summary(session_id, summary)` with this structure:

## Goal
[What we were working on this session]

## Instructions
[User preferences or constraints discovered — skip if none]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains to be done — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]

2. Call `mem_session_end(session_id)` (the plugin hook also does this for
   you, but calling it explicitly is safe).

This is NOT optional. If you skip this, the next session starts blind.

## AFTER COMPACTION

If you see a message about compaction or context reset, or "FIRST ACTION
REQUIRED" in your context:
1. IMMEDIATELY call `mem_session_summary(session_id, summary)` with the
   compacted summary content — this persists what was done before compaction
2. Then call `mem_context` to recover any additional context from previous
   sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost
from memory.

## CODE INDEX LADDER

Prefer the code-index tools over a repo-wide grep when exploring an
unfamiliar large repository:

| Situation | Tool |
|-----------|------|
| Index may be stale / never built | `code_status` → `code_index` (or `skillgrid index`) |
| Find relevant code by concept | `code_search(query, limit)` → `code_read` |
| Exact identifier / exhaustive text | `rg` / `grep` (not Mnemonic) |

1. `code_status` — check index health; do NOT search until it is fresh
2. `code_index` — run the incremental indexer if stale
   (mtime + content hash)
3. `code_search(query, limit)` — BM25 over indexed chunks; ranked by
   path + line + score
4. `code_read(path, start_line, end_line)` — read only the slice the search
   narrowed to

## WEB RESEARCH CACHE

Before calling a remote research MCP (Context7, Exa, DeepWiki, WebFetch):

1. `web_cache_lookup(source, ...)` — exact prior fetch. Sources:
   `context7`, `exa`, `deepwiki`, `fetch`, `manual`.
2. If miss: call the remote MCP.
3. IMMEDIATELY `web_cache_save(source, content, ...)` after it returns.
   **Cap: 256 KB per snapshot** — summarize before saving.
4. When the user asks "what did we find about X online?", use
   `web_cache_search(query, fresh_only)`, then `web_cache_get(id)` for the
   untruncated body.
5. Lookup returns `stale: true` → re-fetch and re-save.

Check cache health any time with `web_cache_status`.

**TTL defaults** (`internal/mnemonic/config/load.go:DefaultWebCache`,
overridable in `config.d/indexing.yaml` → `mnemonic.web_cache.ttl`):

| source   | TTL           |
|----------|---------------|
| context7 | 30 days       |
| exa      | 7 days        |
| deepwiki | 14 days       |
| fetch    | 7 days        |
| manual   | none (no expiry) |

## PASSIVE CAPTURE

When completing a task or subtask, include a "## Key Learnings:" section at
the end of your response with numbered items. Mnemonic auto-extracts these
and saves them as observations even if you forget to call `mem_save`
explicitly.

Example:
```
## Key Learnings:
1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions
```

## DATA LAYOUT

- Per-project SQLite: `~/.skillgrid/mnemonic/<projectID>.sqlite` (WAL mode)
- Project ID from `.skillgrid/config.json` → git remote origin →
  `{base}-{hash}` fallback
- Config: `config.d/indexing.yaml` (mnemonic section) — include/exclude globs,
  chunk size, web cache TTL

## REST API

The `skillgrid serve` command exposes the same functionality over HTTP
(`SKILLGRID_HTTP_TOKEN` bearer auth required on write routes):

- `GET /health`
- `POST /sessions`, `POST /sessions/{id}/end`
- `GET /context`
- `GET /observations`, `GET /observations/recent`, `POST /observations`
- `GET /search`
- `GET /code/status`, `POST /code/index`, `GET /code/files`, `GET /code/search`, `GET /code/read`
- `GET /web/lookup`, `POST /web/cache`, `GET /web/search`, `GET /web/entry/{id}`, `GET /web/status`
- `GET /projects`

The plugin hook uses this API for session lifecycle
(`POST /sessions` etc.).

## CLI

- `skillgrid mcp` — start MCP stdio server
- `skillgrid serve` — start HTTP API (default :7438)
- `skillgrid index` — run code indexing
- `skillgrid setup <opencode|kilocode|cursor>` — install agent plugins
  (also `--agent`)

## ENV

- `SKILLGRID_MNEMONIC_DATA_DIR` — data directory (default `~/.skillgrid/mnemonic`)
- `SKILLGRID_MNEMONIC_PORT` — HTTP port (default 7438)
- `SKILLGRID_HTTP_TOKEN` — bearer token for write routes

## Session lifecycle in one line

`mem_session_start` (hook) → `mem_save` throughout → `mem_session_summary`
→ `mem_session_end` (hook). `mem_save` dedups by normalized content hash
within 24 h and upserts by `topic_key`.

`mem_session_start` accepts an optional `title` — a short name describing the
session's goal. It is stored on the sessions table and shown in the web
dashboard session list. Example:
`mem_session_start(directory: "/path/to/repo", title: "Skillgrid CLI dashboard status card updates")`.
