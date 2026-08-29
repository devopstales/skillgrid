# Mnemonic

Mnemonic is the local-first persistent memory engine embedded in the `skillgrid`
CLI. It provides three integrated capabilities — **session memory** (Engram-aligned),
**code search** (OpenClaw-aligned), and **web research cache** (Neuledge-aligned) —
backed by a per-project SQLite database with FTS5 indexing.

Mnemonic exposes two transports:

- **MCP stdio server** — `skillgrid mcp` — consumed directly by AI agents as tools
  (`mem_*`, `code_*`, `web_*`).
- **HTTP REST API** — `skillgrid serve` — consumed by agent plugins for session
  lifecycle, compaction recovery, and index health nudges.

```
agent plugin (opencode/kilo/cursor)
   │  auto-starts  `skillgrid serve` on GET /health failure
   │  registers     `skillgrid mcp` as the MCP server
   ▼
skillgrid mcp          ◄── stdio MCP transport (tool calls)
skillgrid serve        ◄── HTTP API transport (127.0.0.1:7438)
skillgrid index        ◄── one-shot incremental indexer (CLI)
   │
   ▼
Service facade  ──► store (SQLite per project)
   ├─ memory service    (observations + sessions, FTS5)
   ├─ codeindex         (files + chunks, FTS5 trigram/porter)
   └── webcache service  (web snapshots, FTS5 + TTL expiry)
```

## Data model

Data is stored under `~/.skillgrid/mnemonic/` (override with
`SKILLGRID_MNEMONIC_DATA_DIR`). Each resolved **project** gets its own SQLite file
named `<project-id>.sqlite`, so observations, code index, and web cache are
scoped per project.

### Project resolution

The project ID is resolved per working directory (`internal/mnemonic/project/resolve.go`):

1. Nearest `.skillgrid/config.json` with a `project` field — highest priority.
2. Git remote `origin` URL — the repo basename (e.g. `my-org/my-repo` → `my-repo`).
3. Fallback — `{basename}-{sha256(cwd)[:8]}`.

IDs are normalized (lowercased, `[-_]+` collapsed to `-`, trimmed of leading/trailing
dashes) and validated against path traversal (`..` is rejected).

### SQLite schema

Embedded migrations (`store/migrations/001_initial.sql`) create:

| Table | Purpose |
|---|---|
| `sessions` | Workspace sessions (UUID, project, directory, status) |
| `observations` | Memory entries (FTS5 `observations_fts` with Porter stemming) |
| `files` / `chunks` | Indexed source files (FTS5 `chunks_fts` with trigram tokenizer) |
| `web_cache` | Cached web research snapshots (FTS5 `web_cache_fts` with Porter) |
| `index_meta` | Schema version tracking |

## Indexing configuration

Code indexing reads `config.d/indexing.yaml`, walking up from the indexed directory
to find the first match. Defaults (merged with the repo root `config.d/indexing.yaml`):

| Field | Default |
|---|---|
| `include` | `**/*.go`, `**/*.ts`, `**/*.tsx`, `**/*.md` |
| `exclude` | `node_modules`, `.git`, `dist`, `.skillgrid`, `vendor`, `.venv`, … |
| `chunk_lines` | 80 |
| `chunk_overlap` | 10 |
| `max_file_size_kb` | 512 |

Web cache TTLs (defaults): `context7` 30d, `exa` 7d, `deepwiki` 14d, `fetch` 7d,
`manual` never expires. Maximum entry size is 256 KB.

## Agent plugins

`skillgrid setup <agent>` installs Mnemonic integration for one of `opencode`,
`kilocode`, or `cursor` (`kilo` is accepted as an alias for `kilocode`). It copies
plugin files from `plugins/<agent>/` in the repo into the agent's config
directory and registers `skillgrid mcp` as the MCP server.

### OpenCode / Kilo

- Copies `plugins/opencode/mnemonic.ts` → `~/.config/opencode/plugins/mnemonic.ts`
- Copies `plugins/kilo/mnemonic.ts` → `~/.config/opencode/shared/http-client.ts`
- Upserts `mcp.skillgrid-mnemonic` in `~/.config/opencode/opencode.jsonc` (or `kilo.jsonc` for Kilo)
- For Kilo: writes the Memory Protocol (from `plugins/kilo/memory-protocol.md`)
  into `~/.config/kilo/AGENTS.md` between managed markers, and bridges shared files from
  OpenCode's config.
- The OpenCode plugin reads `SKILLGRID_MNEMONIC_HTTP_URL` (default `http://127.0.0.1:7438`)
  and `SKILLGRID_MNEMONIC_HTTP_TOKEN` for requests to `skillgrid serve`.
- Both plugins auto-start `skillgrid serve` if `GET /health` fails, create a session
  on `session.created`, inject the Memory Protocol on every chat turn, recover context
  on compaction, and nudge on a stale code index.

### Cursor

- Registers `skillgrid mcp` in `~/.cursor/mcp.json` under `mcpServers.skillgrid-mnemonic`
- Writes `~/.cursor/rules/mnemonic.mdc` from the template at
  `plugins/cursor/mnemonic.mdc`, injecting the Memory Protocol
  in place of `{{MEMORY_PROTOCOL}}`.

## CLI commands

| Command | Purpose |
|---|---|
| `skillgrid mcp` | Stdio MCP server (primary agent interface). |
| `skillgrid serve` | HTTP API server (primary plugin interface). |
| `skillgrid index` | One-shot incremental code indexer for a directory (`--dir`, `--project`). |
