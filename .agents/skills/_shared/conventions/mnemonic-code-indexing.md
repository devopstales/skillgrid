# Mnemonic Code-Indexing Convention (shared)

NOTE: This is the shared convention for the Mnemonic **code index** (`code_*` MCP tools: `code_status`, `code_index`, `code_search`, `code_read`). It is supplementary — the full reference, including exact return schemas, the chunking model, and per-tool parameters, lives in the **`mnemonic-code-index` skill** (`.agents/skills/mnemonic-code-index/SKILL.md`). Sub-agents do NOT need to read that whole file to use the ladder; read it only when a tool returns an unexpected shape or you need to tune indexing.

Applies to every Skillgrid SDD skill that explores, reads, or changes source code. The code index lives in the same per-project SQLite store as the memory layer and web cache (`~/.skillgrid/mnemonic/<project>.sqlite`) — same project scope, same session lifecycle as `mem_*`.

## The Ladder (canonical order)

Run in this order; do not skip to reading before you know the index is populated.

```
1. code_status   → health + staleness. If stale:true or file_count==0, go to 2.
2. code_index    → incremental index (cheap, idempotent). Safe to run in doubt.
3. code_search   → BM25 search over chunk ranges → path + line range + snippet.
4. code_read     → read the exact slice for the path + range that code_search gave.
```

Rules:
- `code_read` is **only** for a path + line range that `code_search` already narrowed. Never read a whole file speculatively.
- Search first, then read. A hit covers a chunk's `start_line`–`end_line`, not the whole file.
- Prefer `code_search` over `rg`/`grep` when exploring an **unfamiliar** large repo. Use `rg`/`grep` for exact-identifier lookups and when the index is a poor fit (generated code, single-token exact match).

## When to `code_index`

- **Fresh clone** — run before the first `code_search`; an empty index returns zero hits.
- **`code_status` reports `stale: true` or `file_count == 0`.**
- **After a large refactor / branch switch** — the incremental run re-indexes only changed files.
- **In doubt** — it is cheap (90%+ of files skip as unchanged); run it anyway.

## Tool quick-reference

| tool | required params | optional | returns (shape) |
|---|---|---|---|
| `code_status` | — | — | `file_count`, `chunk_count`, `last_indexed`, `stale` |
| `code_index` | — | — | `files_indexed`, `files_skipped`, `files_deleted`, `chunks_added` |
| `code_search` | `query` | `limit` (default 20) | `hits[]`: `path`, `start_line`, `end_line`, `snippet`, `score` |
| `code_read` | `path` | `start_line`, `end_line` | `path`, `start_line`, `end_line`, `text` |

Index root: `code_index` walks the **git root** (`git rev-parse --show-toplevel`) when inside a repo, else cwd. Raise `limit` if the first `code_search` pass misses the symbol.

## Configuration

Config file `config.d/indexing.yaml`, searched up from the indexed dir; repo-local and `~/.skillgrid/config.d/indexing.yaml` override defaults (first found wins per key).

| field | default |
|---|---|
| `chunk_lines` | `80` |
| `chunk_overlap` | `10` (windows overlap by 10 lines) |
| `max_file_size_kb` | `512` (hard cap; larger files silently skipped) |
| `exclude` | `.git`, `node_modules`, `vendor`, `dist`, `build`, `target`, `__pycache__`, `.next`, `.cache`, `.venv`, `venv`, `coverage`, `.idea`, `.vscode`, `go.sum`, `.terraform`, … |

An empty `include` means "everything not excluded". A skill should never hardcode a different chunk window — it inherits these.

## Recording findings

After indexing or searching, `mem_save` what you learned (architecture notes, surprising file locations, gotchas) following the memory rules in [mnemonic-memory.md](mnemonic-memory.md) (`scope: project`, active `session_id`, `title == topic_key`).

## CLI fallback

The same indexer is exposed via the `skillgrid` CLI for contexts that cannot call MCP (scripts, CI):

```bash
skillgrid index [--dir <path>] [--project <id>]
```

Equivalent to `code_index` over MCP; the CLI requires `--dir` or runs from cwd (MCP auto-detects the git root).

## Gotchas

- `code_index` indexes the **git root**, not the cwd. If your sources live in a sub-repo, run from the repo root or pass `--dir`.
- **512 KB file cap is hardcoded** — large generated/bundled files are silently skipped (counted in `files_skipped`).
- `code_search` is an **FTS5 phrase query** — the whole string is one phrase. For "A AND B", issue separate `code_search` calls.
- `score` is the **negated BM25 rank** (a negative number; lower = better). Sort score descending when ranking hits yourself.
- FTS5 tokenizer is **trigram** (no Porter stemming): substring / partial-word matches work; morphological variants (`validate` vs `validation`) do not collapse.
- `code_read` returns **joined chunk text** with a few duplicated lines at boundaries (chunks overlap) — don't treat boundary lines as authoritative.
- `code_status` staleness is **binary** (populated or not) — there is no "stale after N hours". Run `code_index` if unsure.
- The index is **per-project** (one SQLite file, one project scope). No cross-project search; index each repo in a multi-repo workspace separately.

## Why This Convention

- Single canonical ladder → every skill orders its code-exploration calls identically; no skill skips `code_status`.
- Search-then-read → context stays bounded (a slice, not a file), matching the memory layer's "preview then retrieve" recovery pattern.
- Shared config → chunking/caps live in `config.d/indexing.yaml`, so skills and the CLI stay consistent instead of each hardcoding a window.
- Pointers, not copies → the heavy detail (schemas, chunking, params) stays in the `mnemonic-code-index` skill; this file only keeps the contract every skill must honor.
