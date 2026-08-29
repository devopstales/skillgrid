---
name: mnemonic-code-index
description: "Use when the user asks to index a repo, check index health, search code, read indexed source, or configure code indexing in Mnemonic (the code_* MCP tools: code_status, code_index, code_search, code_read)."
---

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
