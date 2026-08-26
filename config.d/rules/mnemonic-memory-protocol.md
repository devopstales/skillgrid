# Skillgrid Mnemonic — Memory Protocol

You have access to **Mnemonic** (`skillgrid-mnemonic` MCP), a persistent memory system with code FTS and web research cache. It survives across sessions and compactions.

## Session Start

When a Mnemonic plugin is active (OpenCode/Kilo), it calls `mem_session_start` at session open and injects standing rules plus recent context.

Without a plugin (Cursor: MCP + rule only):

1. Call `mem_context` to load recent session history for this project
2. Call `mem_search` when the task may overlap prior work
3. Call `code_status` if code search may be needed and the index might be stale

## When to Save (mandatory — not optional)

Call `mem_save` IMMEDIATELY after any of these:

- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User correction to your approach

Format for `mem_save`:

- **title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList", "Chose Zustand over Redux")
- **type**: See **Memory Taxonomy** below (Engram-compatible: `bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`)
- **scope**: `project` (default) | `user` | `global`
- **topic_key** (optional, recommended for evolving decisions): stable key like `architecture/auth-model`
- **content**:
  - **What**: One sentence — what was done
  - **Why**: What motivated it (user request, bug, performance, etc.)
  - **Where**: Files or paths affected
  - **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic Update Rules (mandatory)

- Different topics must not overwrite each other (e.g. architecture vs bugfix)
- Reuse the same `topic_key` to update an evolving topic instead of creating new observations
- If unsure about the key, call `mem_suggest_topic_key` first and then reuse it
- Use `mem_update` when you have an exact observation ID to correct

## When to Search Memory

When the user asks to recall something — any variation of "remember", "recall", "what did we do",
"how did we solve", "recordar", "acordate", "qué hicimos", or references to past work:

1. First call `mem_context` — checks recent session history (fast, cheap)
2. If not found, call `mem_search` with relevant keywords (FTS5 full-text search)
3. If you find a match, use `mem_get_observation` for full untruncated content

Also search memory PROACTIVELY when:

- Starting work on something that might have been done before
- The user mentions a topic you have no context on — check if past sessions covered it
- An error matches a prior failure — check `bugfix` / `lesson` observations

## Code Search

Use Mnemonic code tools before expensive repo-wide greps when exploring unknown areas.

| Situation | Tool |
|-----------|------|
| Index may be stale or never built | `code_status` first, then `skillgrid index` if needed |
| Find relevant code by concept or keyword | `code_search` → `code_read` for full source |
| Cheap routing to likely files (v1.1) | `code_context` |
| Minimal tokens — file/symbol hints only (v1.1) | `code_peek` |
| Exact identifier, regex, or exhaustive text match | grep / ripgrep (not Mnemonic) |
| Callers, blast radius, rename impact | GitNexus (opt-in, hybrid profile only) |

Workflow:

1. `code_status` — confirm index freshness before broad search
2. `code_search(query, limit)` — ranked chunks with path and line range
3. `code_read(path, start_line, end_line)` — full source for chosen hits

Prefer `code_search` over grepping large unknown trees. Prefer grep when you know the exact symbol or need every textual occurrence.

## Web Research Cache

Before calling remote research MCPs (Context7, Exa, DeepWiki, WebFetch), check the local cache.

Workflow:

1. `web_cache_lookup(source, key)` — exact prior fetch (Context7 doc id, Exa URL, etc.)
2. If miss: call the remote MCP (Context7 / Exa / DeepWiki / fetch)
3. `web_cache_save(source, key, title, content)` — **immediately** after every remote research call
4. When the user asks "what did we find about X online?" → `web_cache_search(query)`

Sources: `context7`, `exa`, `deepwiki`, `fetch`, `manual`. Cached entries expire per TTL in `config.d/indexing.yaml`.

## Tool Routing

| Need | Tool |
|------|------|
| Decisions, bugs, session history | `mem_search` / `mem_context` |
| Route a repo question cheaply (v1.1) | `code_context` |
| Likely files/symbols, minimal tokens (v1.1) | `code_peek` |
| Full matching source in repo | `code_search` → `code_read` |
| Exact identifier / exhaustive text | grep (not Mnemonic) |
| Cached research before remote MCP | `web_cache_lookup` → `web_cache_search` |
| Callers, blast radius, rename | GitNexus (opt-in) |

## Memory Taxonomy

Hermes-inspired `type` + `scope` conventions for `mem_save`:

| `type` | `scope` | Purpose | Retrieval |
|--------|---------|---------|-----------|
| `standing` | `user` or `project` | Hard rules — always injected | Plugin L0, not search-ranked |
| `preference` | `user` | User workflow prefs | `mem_search` |
| `convention` | `project` | Project norms | `mem_search` |
| `decision` | `project` | Architecture choices (Engram default) | `mem_search` |
| `bugfix` / `lesson` | `project` | Failures and fixes | `mem_search` + error prefetch |
| `correction` | `project` | User corrections | Instant save via plugin |
| `discovery` | `project` or `global` | Tool/env quirks | `mem_search` |
| `session_log` | `project` | Dated session notes | `mem_search` with decay (Task 24) |

Engram-compatible aliases still accepted: `architecture` → `decision`, `pattern` → `convention`, `config` → `discovery`.

## Session Close Protocol (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", you MUST:

1. Call `mem_session_summary` with this structure:

```
## Goal
[What we were working on this session]

## Instructions
[User preferences or constraints discovered — skip if none]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]
```

This is NOT optional. If you skip this, the next session starts blind.

## Passive Capture — Automatic Learning Extraction

When completing a task or subtask, include a `## Key Learnings:` section at the end of your response
with numbered items. Mnemonic will automatically extract and save these as observations.

Example:

```
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions
```

You can also call `mem_capture_passive(content)` directly with any text that contains a learning section.
This is a safety net — it captures knowledge even if you forget to call `mem_save` explicitly.

## After Compaction

If you see a message about compaction or context reset, or if you see "FIRST ACTION REQUIRED" in your context:

1. IMMEDIATELY call `mem_session_summary` with the compacted summary content — this persists what was done before compaction
2. Then call `mem_context` to recover any additional context from previous sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost from memory.
