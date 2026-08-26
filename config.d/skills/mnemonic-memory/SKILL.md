---
name: mnemonic-memory
description: "ALWAYS ACTIVE — Skillgrid Mnemonic persistent memory protocol. You MUST save decisions, conventions, bugs, and discoveries proactively. Do NOT wait for the user to ask."
---

# Skillgrid Mnemonic — Persistent Memory Protocol

You have access to **Mnemonic**, Skillgrid's local-first persistent memory system that survives across sessions and compactions.
This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on demand.

## AVAILABLE TOOLS

Core tools are loaded automatically at session start by the Mnemonic plugin or MCP server.
They are available immediately — no manual ToolSearch needed.

**Memory:** `mem_save`, `mem_search`, `mem_context`, `mem_get_observation`, `mem_suggest_topic_key`, `mem_session_start`, `mem_session_end`, `mem_session_summary`

**Code index:** `code_status`, `code_index`, `code_search`, `code_read`

**Web cache:** `web_cache_lookup`, `web_cache_save`, `web_cache_search`, `web_cache_get`, `web_cache_status`

**Fallback:** If tools are unexpectedly unavailable, run `skillgrid setup <agent>` for your agent
(opencode, kilocode, or cursor) and restart the session. Setup wires the `skillgrid-mnemonic` MCP
server and memory protocol. Data lives under `~/.skillgrid/mnemonic/`.

Admin tools (deferred — use only if needed):
- `mem_stats`, `mem_delete`, `mem_timeline`, `mem_capture_passive`

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
- Notion/Jira/GitHub artifact created or updated with significant content
- Configuration change or environment setup done

### After discoveries
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned

### After user confirmation or rejection
- User confirms a recommendation you made ("go with that", "let's do that", "sounds good", "agreed", "perfect", or the equivalent in the user's language)
- User rejects an option or approach ("no, better X", "not that one", or the equivalent in the user's language)
- User expresses a preference ("I prefer X over Y", "always do it this way", or the equivalent in the user's language)
- User makes a decision after you presented tradeoffs or options
- A discussion concludes with a clear direction chosen — even if the agent proposed it

### Self-check — ask yourself after EVERY task:
> "Did I or the user just make a decision, confirm a recommendation, express a preference, fix a bug, learn something non-obvious, or establish a convention? If yes, call mem_save NOW."

Format for `mem_save`:
- **title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList", "Chose Zustand over Redux")
- **type**: bugfix | decision | architecture | discovery | pattern | config | preference
- **scope**: `project` (default) | `personal`
- **topic_key** (optional but recommended for evolving topics): stable key like `architecture/auth-model`
- **content**:
  **What**: One sentence — what was done
  **Why**: What motivated it (user request, bug, performance, etc.)
  **Where**: Files or paths affected
  **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic update rules (mandatory)

- Different topics MUST NOT overwrite each other (example: architecture decision vs bugfix)
- If the same topic evolves, call `mem_save` with the same `topic_key` so memory is updated (upsert) instead of creating a new observation
- If unsure about the key, call `mem_suggest_topic_key` first, then reuse that key consistently

## WHEN TO SEARCH MEMORY

When the user asks to recall something — any variation of "remember", "recall", "what did we do",
"how did we solve", or the equivalent in the user's language, or references to past work:
1. First call `mem_context` — checks recent session history (fast, cheap)
2. If not found, call `mem_search` with relevant keywords (FTS5 full-text search)
3. If you find a match, use `mem_get_observation` for full untruncated content

Also search memory PROACTIVELY when:
- Starting work on something that might have been done before
- The user mentions a topic you have no context on — check if past sessions covered it
- The user's FIRST message references the project, a feature, or a problem — call `mem_search` with keywords from their message to check for prior work before responding

## CODE SEARCH

Use Mnemonic's code index for repository text search. Tool names unchanged from the MCP surface.

**OCBI ladder (v1):**

1. `code_status` — check index freshness when results seem stale or the repo changed recently
2. `code_search` — find likely files/symbols when you don't know where code lives
3. `code_read` — fetch full chunk or file slice after search narrows the location

**v1.1 additions:** `code_peek` (minimal tokens) and `code_context` (cheap routing) slot between status and search.

**When to use what:**

| Need | Tool |
|------|------|
| Decisions, bugs, session history | `mem_search` / `mem_context` |
| Route a repo question cheaply (v1.1) | `code_context` |
| Likely files/symbols, minimal tokens (v1.1) | `code_peek` |
| Full matching source in repo | `code_search` → `code_read` |
| Exact identifier / exhaustive text match | grep (not Mnemonic) |
| Callers, blast radius, rename | GitNexus (opt-in) |

**Rules:**
- Call `code_search` before grepping large unknown areas of the codebase
- Call `code_status` when the index may be stale (after large merges, branch switches, or missing hits)
- Use grep only when you already know the exact identifier or need exhaustive literal matches
- After `code_search` returns a hit, use `code_read` for the full context — don't guess from snippets alone

## WEB RESEARCH CACHE

Cache remote research so you don't re-fetch the same docs, search results, or wiki answers.

**Mandatory pattern:**

1. **Before** Context7 / Exa / DeepWiki / WebFetch — call `web_cache_lookup` with `query`, `url`, or `(source, library_id, version_tag)`
2. On **hit** — use `web_cache_get` for full content; skip the remote call
3. On **miss** — call the remote MCP tool, then **immediately** call `web_cache_save`
4. When the user asks "what did we find about X online?" — call `web_cache_search`

**After every remote research call**, persist the snapshot:

```
web_cache_save({
  source: "exa",           // context7 | exa | deepwiki | fetch | manual
  query: "search terms",
  url: "https://...",      // optional
  title: "Result title",   // optional
  content: "<paste MCP result>",
  metadata: { tool: "web_search_exa", ... }  // optional
})
```

Required fields: `source`, `content`. Optional: `url`, `query`, `title`, `library_id`, `version_tag`, `metadata`.

| Need | Tool |
|------|------|
| Cached research before remote MCP | `web_cache_lookup` → (on miss) remote call → `web_cache_save` |
| "What did we look up about X?" | `web_cache_search` |
| Full cached snapshot by id | `web_cache_get` |
| Cache health / counts | `web_cache_status` |

## SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "that's it", you MUST:
1. Call `mem_session_summary` with this structure:

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

This is NOT optional. If you skip this, the next session starts blind.

## AFTER COMPACTION

If you see a message about compaction or context reset:
1. IMMEDIATELY call `mem_session_summary` with the compacted summary content — this persists what was done before compaction
2. Then call `mem_context` to recover any additional context from previous sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost from memory.
