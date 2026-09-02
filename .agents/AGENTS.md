<!-- skillgrid:global -->

# AGENTS.md

Global rules for all AI agents (Kilo, OpenCode, Cursor, Claude, Codex, Gemini). Applies to every project; project-specific rules in local `AGENTS.md`/`CLAUDE.md` take precedence when they conflict.

**Tradeoff:** These rules bias toward caution over speed. For trivial tasks, use judgment.

## If You Are an AI Agent

Be a tool that protects your human partner, not one that embarrasses them. Quality over volume:

- One problem per change. Never bundle unrelated edits.
- Solve a real problem your human experienced — not a theoretical one. If "it could cause issues" is your only justification, stop.
- Before contributing to an external repo: search existing (open and closed) PRs, read the PR template, identify yourself, show your human the full diff before submitting.
- Never fabricate: no invented claims, no "my review agent flagged this", no filler sentences.

## 1. Think Before Coding

Don't assume. Don't hide confusion. Surface tradeoffs.

- State assumptions explicitly; if uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.
- **Before writing any code, check for applicable skills and invoke them first.** No exceptions for "simple" tasks. If a skill exists for the work, use it.

## 2. Simplicity First

Minimum code that solves the problem. Nothing speculative.

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

Touch only what you must. Clean up only your own mess.

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- Unrelated dead code: mention it, don't delete it.
- Remove imports/variables/functions that YOUR changes made unused; don't remove pre-existing dead code unless asked.

The test: every changed line should trace directly to the human's request.

## 4. Goal-Driven Execution

Define success criteria. Loop until verified.

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- Multi-step work: state a brief plan with a verification step per item:
  ```
  1. [step] -> verify: [check]
  2. [step] -> verify: [check]
  ```

## Codebase Rules

- Concise, direct responses. No preamble, no filler.
- Prefer editing existing files over creating new ones.
- Follow existing code style and conventions before introducing new patterns.
- Never commit secrets or keys.

## Verification

Verify work before claiming it is done — evidence before assertions:

- Run the relevant tests/build/lint and confirm the output passes.
- For CLI tools, run the command the user would run and check exit code + output.
- Prefer no-op confirmation: a change is done only when its success criterion (tests, command output, validation) has been observed.

## Environment

- Mnemonic persistent memory is available (skillgrid local-first store: `mem_*`, `code_*`, `web_*` over the `mnemonic` MCP): update it on decisions, bug fixes, and non-obvious discoveries.

<!-- /skillgrid:global -->

<!-- skillgrid:mnemonic-memory -->

## Mnemonic Persistent Memory — Protocol

You have access to **Mnemonic**, skillgrid's local-first persistent memory system (SQLite + FTS5, single `skillgrid` binary). It survives across sessions and compactions. Tool families: `mem_*` (memory), `code_*` (code index), `web_*` (web-research cache).

### WHEN TO SAVE (mandatory — not optional)

Call mem_save IMMEDIATELY after any of these:
- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned

Format for mem_save:
- **title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList", "Chose Zustand over Redux")
- **type**: bugfix | decision | architecture | discovery | pattern | config | preference (Mnemonic also accepts: learning, convention, lesson, session_log)
- **scope**: project (default) | user | global
- **session_id**: REQUIRED — active session ID from mem_session_start
- **topic_key** (optional, recommended for evolving decisions): stable key like architecture/auth-model
- **content**:
  **What**: One sentence — what was done
  **Why**: What motivated it (user request, bug, performance, etc.)
  **Where**: Files or paths affected
  **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic update rules (mandatory)

- Different topics must not overwrite each other (e.g. architecture vs bugfix)
- Reuse the same topic_key to update an evolving topic instead of creating new observations (Mnemonic upserts by topic_key via mem_save — there is no `mem_update`)
- If unsure about the key, call mem_suggest_topic_key first and then reuse it
- Use mem_get_observation(id) to read the current untruncated content before rewriting an evolved topic

### WHEN TO SEARCH MEMORY

When the user asks to recall something — any variation of "remember", "recall", "what did we do",
"how did we solve", "recordar", "acordate", "qué hicimos", or references to past work:
1. First call mem_context — checks recent session history (fast, cheap)
2. If not found, call mem_search with relevant keywords (FTS5 full-text search)
3. If you find a match, use mem_get_observation for full untruncated content

Also search memory PROACTIVELY when:
- Starting work on something that might have been done before
- The user mentions a topic you have no context on — check if past sessions covered it

### SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", you MUST:
1. Call mem_session_summary with this structure:

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

### PASSIVE CAPTURE — automatic learning extraction

When completing a task or subtask, include a "## Key Learnings:" section at the end of your response
with numbered items. Mnemonic will automatically extract and save these as observations.

Example:
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions

This is a safety net — it captures knowledge even if you forget to call mem_save explicitly.

### AFTER COMPACTION

If you see a message about compaction or context reset, or if you see "FIRST ACTION REQUIRED" in your context:
1. IMMEDIATELY call mem_session_summary with the compacted summary content — this persists what was done before compaction
2. Then call mem_context to recover any additional context from previous sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost from memory.
<!-- /skillgrid:mnemonic-memory -->

<!-- skillgrid:mnemonic-code-index -->

# Mnemonic Code Index

Mnemonic's code index is a per-project BM25 full-text search over
line-based chunks of source code. It lives in the same SQLite store as
memory observations and the web cache (`~/.skillgrid/mnemonic/<project>.sqlite`),
and is accessed via four MCP tools: `code_status`, `code_index`,
`code_search`, `code_read`.

## Storage

| Table | Purpose |
|---|---|
| `files` | Indexed file paths, mtime_ns, size, content_hash, indexed_at |
| `chunks` | Per-file line-range chunks with text and content_hash |
| `chunks_fts` | FTS5 virtual table (trigram tokenizer) over `chunks.text` |

FTS5 is kept in sync via triggers on `chunks` INSERT/UPDATE/DELETE. The
project scope is the same as Mnemonic's memory layer: one SQLite file per
resolved project ID.

## Configuration

Indexing is configured via `config.d/indexing.yaml`, walked up from the
indexed directory:

```yaml
mnemonic:
  include:
    - "**/*.go"
    - "**/*.ts"
    - "**/*.tsx"
    - "**/*.md"
  exclude:
    - "**/node_modules/**"
    - "**/.git/**"
    - "**/dist/**"
    - "**/.skillgrid/**"
  chunk_lines: 80
  chunk_overlap: 10
```

Defaults (from `internal/mnemonic/config/load.go`):

| Field | Default |
|---|---|
| `include` | `**/*.go`, `**/*.ts`, `**/*.tsx`, `**/*.md` |
| `exclude` | `**/node_modules/**`, `**/.git/**`, `**/dist/**`, `**/.skillgrid/**` |
| `chunk_lines` | `80` |
| `chunk_overlap` | `10` |
| `max_file_size_kb` | `512` (hardcoded) |

An empty `include` list means "include everything that is not excluded".
User values in `indexing.yaml` override defaults (merged, not replaced).

## The Ladder

```
1. code_status   → check health and staleness
2. code_index    → run incremental index if stale or empty
3. code_search   → BM25 search over chunks
4. code_read     → read the matching slice
```

Prefer `code_search` over `rg`/`grep` when exploring an unfamiliar large
repository. Use `rg`/`grep` for exact-identifier lookups — those are not
Mnemonic's job.

## Tools

### code_status

No parameters.

Returns:

```json
{
  "file_count": 123,
  "chunk_count": 456,
  "last_indexed": "2026-08-29T08:00:00Z",
  "stale": false
}
```

**Staleness rule**: `stale` is `true` when `file_count == 0` or
`last_indexed` is empty. There is no time-based staleness — the index is
either populated or it is not. If you suspect the index lags behind the
working tree, run `code_index` anyway; it is cheap and idempotent.

### code_index

No parameters. Automatically resolves the index root:

1. Git root (`git rev-parse --show-toplevel`) if inside a git repository.
2. Otherwise the current working directory.

Returns:

```json
{
  "files_indexed": 3,
  "files_skipped": 120,
  "files_deleted": 1,
  "chunks_added": 45
}
```

**Incremental logic** (from `internal/mnemonic/codeindex/indexer.go`):

- A file is **skipped** when `(mtime_ns, size, content_hash)` match the
  stored row exactly.
- A file is **re-indexed** when any of those three changed: its old chunks
  are deleted and new ones are inserted.
- A file **deleted from the working tree** is removed from `files` and its
  chunks are removed.
- Files larger than **512 KB** are silently skipped.
- The entire run is wrapped in a single transaction.

### code_search

| Parameter | Required | Default | Description |
|---|---|---|---|
| `query` | yes | — | Search terms (FTS5 phrase query) |
| `limit` | no | `20` | Maximum hits to return |

Returns:

```json
{
  "hits": [
    {
      "path": "internal/auth/login.go",
      "start_line": 42,
      "end_line": 60,
      "snippet": "...",
      "score": -3.42
    }
  ]
}
```

**Search behavior** (from `internal/mnemonic/search/fts.go`):

- The query is wrapped as an FTS5 **phrase query**: `"<query>"`.
- Double-quotes inside the query are escaped (`"` → `""`).
- Results are ordered by BM25 rank (lower is better); the skill returns
  `score: -rank` so a higher score means a better match.
- The default `limit` is 20. Raise it if the first pass misses the symbol
  you want.
- `code_search` searches **chunks**, not files. A hit covers the chunk's
  `start_line`–`end_line` range, not the whole file.

### code_read

| Parameter | Required | Default | Description |
|---|---|---|---|
| `path` | yes | — | Repo-relative file path (from `code_search`) |
| `start_line` | no | `0` (all) | 1-based start line |
| `end_line` | no | `start_line` | 1-based end line |

Returns:

```json
{
  "path": "internal/auth/login.go",
  "start_line": 42,
  "end_line": 60,
  "text": "...joined chunk text..."
}
```

- When `start_line` is omitted or `0`, all indexed chunks for the path are
  returned.
- When `start_line` is provided, chunks that overlap the `[start_line,
  end_line]` range are returned. Because chunks are 80-line overlapping
  windows, the returned text may include a few extra lines at the
  boundaries.
- Returns `"file not indexed"` if the path is not in the `files` table.
- Returns `"no indexed chunks"` if the path exists but has no chunks.

## Chunking Model

From `internal/mnemonic/codeindex/indexer.go`:

- Files are split into **overlapping line windows** of `chunk_lines` lines
  (default 80).
- Consecutive windows overlap by `chunk_overlap` lines (default 10), so
  the step between window starts is `chunk_lines - chunk_overlap = 70` lines.
- Empty chunks (all whitespace) are skipped, except the last chunk.
- Each chunk gets a SHA-256 `content_hash` for incremental change detection.
- Line numbers are **1-based**: the first line of the file is line 1.

Example with `chunk_lines=4, chunk_overlap=2`:

```
Lines  1-4  → chunk 1 (start_line: 1,  end_line: 4)
Lines  3-6  → chunk 2 (start_line: 3,  end_line: 6)
Lines  5-8  → chunk 3 (start_line: 5,  end_line: 8)
Lines  7-10 → chunk 4 (start_line: 7,  end_line: 10)
```

In production (`chunk_lines=80, chunk_overlap=10`) the overlap is 10 lines,
so context bleeds across chunk boundaries without much duplication.

## Incremental Indexing

`code_index` is **incremental by design** — it never rebuilds from scratch
unless you delete the SQLite file. The three-way comparison per file is:

```
unchanged → skip (mtime_ns, size, content_hash all match)
changed   → upsert file + delete old chunks + insert new chunks
deleted   → remove file + chunks from the store
```

The `FilesSkipped` count from a healthy repo after the first full index is
typically high (90%+). That is expected and correct.

## Integration with Mnemonic

The code index is one of three Mnemonic tool families (`mem_*`, `code_*`,
`web_*`). They share the same project scope and session lifecycle:

- **Before searching** — call `code_status`. If `stale: true` or
  `file_count == 0`, run `code_index` first. Running `code_index` against
  an up-to-date index is cheap (it skips most files).
- **Before a fresh clone** — run `code_index` immediately; `code_search`
  against an empty index returns no hits.
- **After a large refactor** — run `code_index` again; the incremental
  run will only re-index changed files.
- **Record findings** — after indexing or searching, `mem_save` what you
  learned (architecture notes, surprising file locations, gotchas). Use the
  `mnemonic-memory` skill for the save/search protocol.

## CLI

The same indexer is exposed via the `skillgrid` CLI:

```bash
skillgrid index [--dir <path>] [--project <id>]
```

`skillgrid index` is equivalent to `code_index` over MCP. Use it when the
agent cannot call MCP (scripts, CI). The MCP `code_index` auto-detects the
git root; the CLI variant requires `--dir` or runs from cwd.

## Gotchas

- `code_index` walks the **git root** (if inside a git repo), not the cwd.
  If your sources live in a subdirectory, run from the repo root or pass
  `--dir`.
- **Max file size is 512 KB hardcoded** — large generated or bundled files
  are silently skipped.
- `code_search` is a **phrase query**, not a boolean query. The whole query
  string is quoted as one phrase. For "foo AND bar" style queries, split
  them into separate `code_search` calls.
- `code_read` returns **joined chunk text** with `\n` separators. Because
  chunks overlap, the returned text may contain a few duplicated lines at
  chunk boundaries.
- The `score` field is the **negated BM25 rank** (negative number). Lower
  = better match. Sort by score descending when ranking hits yourself.
- The FTS5 tokenizer is **trigram** (not Porter stemming). Substring and
  partial-word matches work; morphological variants (e.g. `validate` vs
  `validation`) do not stem to the same token.
- `code_status` staleness is **binary** (indexed or not). There is no
  "stale after N hours" — run `code_index` if in doubt.
- The index is **per-project** (one SQLite file). There is no cross-project
  search. Multi-repo workspaces require indexing each repo separately.

<!-- /skillgrid:mnemonic-code-index -->

<!-- skillgrid:mnemonic-search-cache -->

## Mnemonic Web Research Cache

Before calling a remote research MCP (Context7, Exa, DeepWiki, WebFetch):

1. `web_cache_lookup(source, ...)` — check the cache first. Sources: `context7`, `exa`, `deepwiki`, `fetch`, `manual`.
2. If miss: call the remote MCP.
3. `web_cache_save(source, content, ...)` — persist immediately after the call returns. Cap: 256 KB per snapshot — summarize before saving.
4. "What did we find about X online?" → `web_cache_search(query, fresh_only)`, then `web_cache_get(id)` for the untruncated body.

**TTL defaults** (overridable in `config.d/indexing.yaml` → `mnemonic.web_cache.ttl`):

| source   | TTL           |
|----------|---------------|
| context7 | 30 days       |
| exa      | 7 days        |
| deepwiki | 14 days       |
| fetch    | 7 days        |
| manual   | none (no expiry) |

See `.agents/skills/mnemonic-memory/SKILL.md` for the full protocol.

<!-- /skillgrid:mnemonic-search-cache -->
