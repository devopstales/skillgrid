---
name: mnemonic-memory-protocol
description: >
  Persistent memory discipline for skillgrid Mnemonic contributors (mem_*,
  code_*, web_* on the `skillgrid-mnemonic` MCP server).
  Trigger: Decisions, bugfixes, discoveries, preferences, or session closure.
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.1"
---

## When to Use

Use this skill when:
- Making architecture or implementation decisions
- Fixing bugs with non-obvious root causes
- Discovering patterns, gotchas, or user preferences
- Searching for prior work before starting a similar task
- Using the Mnemonic code-index or web-research-cache tool families
- Closing a session or after compaction

---

## Save Rules

Call `mem_save` IMMEDIATELY after:
- decision (architecture, design, tool choice)
- bugfix (include root cause)
- pattern / discovery / gotcha
- config or environment change
- preference or user correction

`mem_save` fields (Mnemonic — 8 tools on the `skillgrid-mnemonic` server):
- `title` — verb + what, short and searchable
- `type` — bugfix | decision | architecture | discovery | pattern | config |
  preference (also: learning, convention, lesson, session_log)
- `scope` — `project` (default) | `user` | `global`
- `session_id` — REQUIRED. Your active session ID from `mem_session_start`.
- `topic_key` — stable key like `architecture/auth-model` (call
  `mem_suggest_topic_key` first if unsure)
- `content` — four sections:
  - **What** — one sentence, what was done
  - **Why** — what motivated it
  - **Where** — files or paths affected
  - **Learned** — gotchas, edge cases (omit if none)

Topic rules:
- Reuse the same `topic_key` to evolve an existing topic — Mnemonic
  **upserts via `mem_save` with the same `topic_key`** (there is no
  `mem_update`; this is the mechanism).
- Different topics must not overwrite each other (`architecture` vs
  `bugfix`).
- Use `mem_get_observation(id)` to read untruncated current content before
  rewriting it.

---

## Search Rules

- On recall requests ("remember / recall / what did we do / how did we
  solve"): `mem_context` first (recent session summaries, fast), then
  `mem_search` (FTS5 over observations), then `mem_get_observation` for the
  full untruncated body.
- Before starting similar work: run a proactive `mem_search` with the
  relevant keywords.
- On the user's FIRST message: if it references the project, a feature, or a
  problem, call `mem_search` with their keywords before responding.

---

## Code Index (large unfamiliar repos)

Ladder — do NOT grep a large unknown tree until indexed:
1. `code_status` — check index health (fresh? stale?)
2. `code_index` — run the incremental index if stale
3. `code_search(query, limit)` — BM25 over chunks; ranked path + line
4. `code_read(path, start_line?, end_line?)` — the narrowed slice

For exact identifiers, regex, or exhaustive text matches, use `rg` /
`grep` instead — those are not Mnemonic's job.

---

## Web Research Cache

Before calling a remote research MCP (Context7, Exa, DeepWiki, WebFetch):
1. `web_cache_lookup(source, ...)` — exact prior fetch. Sources: `context7`,
   `exa`, `deepwiki`, `fetch`, `manual`.
2. If miss: call the remote MCP.
3. IMMEDIATELY `web_cache_save(source, content, ...)` (256 KB cap).
4. "What did we find about X online?" → `web_cache_search(query,
   fresh_only)`, then `web_cache_get(id)` for untruncated body.

`web_cache_status` reports counts, expiries, and the oldest/newest fetch.

---

## Session Lifecycle

At session start: the plugin hook calls `mem_session_start(directory?)` and
creates a workspace session. Keep the returned `session_id` — every
`mem_save` and `mem_session_summary` call needs it.

Before ending: `mem_session_summary(session_id, summary)` →
`mem_session_end(session_id)`.

---

## Session Close Rules

Before saying done / listo / that's it:
1. Call `mem_session_summary(session_id, summary)` with the structure:
   - `## Goal` — what we were working on
   - `## Instructions` — user preferences / constraints (skip if none)
   - `## Discoveries` — technical findings, gotchas, non-obvious learnings
   - `## Accomplished` — completed items with key details
   - `## Next Steps` — what remains for the next session
   - `## Relevant Files` — paths with one-line descriptions
2. Call `mem_session_end(session_id)` (optional — the plugin hook does this
   too).

Not optional. The next session starts blind without this.

---

## After Compaction

When a compaction or context-reset message appears (or "FIRST ACTION
REQUIRED" in context):
1. IMMEDIATELY `mem_session_summary(session_id, compacted-summary)` —
   persists what was done before compaction.
2. `mem_context` to recover additional prior context.
3. Only then continue work.

Do not skip step 1.

---

## Fallback

If Mnemonic tools are unexpectedly unavailable, run
`skillgrid setup kilocode` (or `opencode` / `cursor`) and restart your
agent. Setup repairs the durable MCP config for the `skillgrid-mnemonic`
server and the always-apply rule.
