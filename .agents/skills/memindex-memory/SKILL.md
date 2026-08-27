---
name: memindex-memory
description: "Use when work must persist across sessions or compactions — recall past work, save decisions/bugfixes/discoveries, drive the code-indexing and web-research-cache tool families (mem_*, code_*, web_*), or recover context after a compaction. Backs the skillgrid-memindex local store."
---

# MemIndex Persistent Memory — Protocol

You have access to **MemIndex**, a local-first persistent memory system
(SQLite + FTS5) inside `skillgrid`. It survives across sessions and
compactions. Tool families: `mem_*` (memory), `code_*` (code index),
`web_*` (research cache).

## Memory Taxonomy

`mem_save` requires `type` + `scope`:

| `type` | Purpose |
|--------|---------|
| `standing` | Hard rules — always apply |
| `preference` | User workflow preferences |
| `convention` | Project norms (naming, structure) |
| `decision` | Architecture choices (aliases: `architecture`) |
| `bugfix` / `lesson` | Failures and fixes |
| `correction` | User corrections |
| `discovery` | Tool/env quirks (aliases: `config`, `learning`) |
| `session_log` | Dated session notes |

`scope`: `project` (default) | `user` | `global`.

## When to Save (mandatory — not optional)

Call `mem_save` IMMEDIATELY after any of:
- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User correction to your approach

`mem_save` fields:
- `title`: verb + what (short, searchable, e.g. "Fixed N+1 query in UserList")
- `type`: see taxonomy
- `scope`: project (default) | user | global
- `topic_key` (optional, recommended for evolving decisions): stable key like
  `architecture/auth-model` — call `mem_suggest_topic_key` first if unsure
- `content`: **What** (one sentence) / **Why** / **Where** (paths) / **Learned** (gotchas)

Topic rules:
- Different topics must not overwrite each other (`architecture` vs `bugfix`).
- Reuse the same `topic_key` to update an evolving topic instead of adding rows.
- Use `mem_get_observation` for full untruncated content by id.

## When to Search Memory

"remember / recall / what did we do / how did we solve" (or any language):
1. `mem_context` — recent session history (fast, cheap).
2. `mem_search` with keywords (FTS5 full-text) if not found.
3. `mem_get_observation` for full untruncated content.

Also search PROACTIVELY when:
- Starting work that may have been done before.
- The user mentions a topic you have no context on.

## Code Search Ladder

| Situation | Tool |
|-----------|------|
| Index may be stale / never built | `code_status` → run `code_index` (or `skillgrid index`) |
| Find relevant code by concept | `code_search(query, limit)` → `code_read` |
| Exact identifier / exhaustive text | grep / ripgrep (not MemIndex) |

Workflow: `code_status` (freshness) → `code_search` (ranked path+line) →
`code_read(path, start_line, end_line)` (full source).

## Web Research Cache (NEW vs engram)

Before calling remote research MCPs (Context7, Exa, DeepWiki, WebFetch):

1. `web_cache_lookup(source, cache_key)` — exact prior fetch. Sources:
   `context7`, `exa`, `deepwiki`, `fetch`, `manual`.
2. If miss: call the remote MCP.
3. `web_cache_save(source, cache_key, content, ...)` — IMMEDIATELY after a
   remote call returns. 256KB cap (summarize first); TTLs: context7 30d,
   exa 7d, deepwiki 14d, fetch 7d, manual none.
4. "What did we find about X online?" → `web_cache_search(query, fresh_only)`.
   `stale: true` → re-fetch and re-save.

## Session Lifecycle

- At session start: call `mem_session_start(directory?)` (plugin also does
  this via `POST /sessions`).
- Before ending: call `mem_session_summary(session_id, summary)` per the
  structure in the **Session Close Protocol** below. Then `mem_session_end`.

## SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it":
1. `mem_session_summary(session_id, summary)` with:

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

Not optional — the next session starts blind without it.

## After Compaction (FIRST ACTION REQUIRED)

1. IMMEDIATELY `mem_session_summary` with the compacted summary — persists
   what was done before compaction.
2. `mem_context` to recover additional prior context.
3. Only then continue.
