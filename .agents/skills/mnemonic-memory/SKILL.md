---
name: mnemonic-memory
description: "ALWAYS ACTIVE — Persistent memory protocol. You MUST save decisions, conventions, bugs, and discoveries to Mnemonic proactively. Do NOT wait for the user to ask."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
   notes: "Documents the Mnemonic MCP protocol used by skillgrid."
---

# Mnemonic Persistent Memory — Protocol

You have access to **Mnemonic**, skillgrid's local-first persistent memory system
(SQLite + FTS5, single `skillgrid` binary). It survives across sessions and compactions.
This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on demand.

## AVAILABLE TOOLS

Core tools are loaded automatically at session start by the UserPromptSubmit hook
(or by the `mnemonic` MCP). They are available immediately — no manual ToolSearch needed.

- `mem_save`, `mem_search`, `mem_context`, `mem_session_summary`
- `mem_get_observation`, `mem_suggest_topic_key`
- `mem_session_start`, `mem_session_end`, `mem_save_prompt`

**Fallback**: If tools are unexpectedly unavailable, run `skillgrid setup` again
and restart the agent. Setup repairs the durable MCP config and permissions
allowlist for both the current server id (`mcp__mnemonic__...`) and older
plugin-scoped variants (`mcp__plugin_mnemonic_mnemonic__...`).

Admin tools (deferred — use ToolSearch only if needed):
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
- **type**: bugfix | decision | architecture | discovery | pattern | config | preference (Mnemonic also accepts: learning, convention, lesson, session_log)
- **scope**: `project` (default) | `user` | `global`
- **session_id**: REQUIRED — active session ID from `mem_session_start`
- **topic_key** (optional but recommended for evolving topics): stable key like `architecture/auth-model`
- **content**:
  **What**: One sentence — what was done
  **Why**: What motivated it (user request, bug, performance, etc.)
  **Where**: Files or paths affected
  **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic update rules (mandatory)

- Different topics MUST NOT overwrite each other (example: architecture decision vs bugfix)
- Reuse the same `topic_key` to update an evolving topic — Mnemonic **upserts by `topic_key` via `mem_save`**. There is no `mem_update` tool; updating a known observation is the same `mem_save` call with the existing `topic_key`.
- If unsure about the key, call `mem_suggest_topic_key` first, then reuse that key consistently
- Before rewriting an evolved topic, call `mem_get_observation(id)` to read the current untruncated content

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

## PASSIVE CAPTURE — automatic learning extraction

When completing a task or subtask, include a `## Key Learnings:` section at the end of your response
with numbered items. Mnemonic will automatically extract and save these as observations.

Example:
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions

This is a safety net — it captures knowledge even if you forget to call `mem_save` explicitly.

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
All core tools are loaded automatically by the hook at session start. If they are unexpectedly missing, rerun `skillgrid setup` and restart the agent.
