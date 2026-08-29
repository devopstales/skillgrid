# MCP

Mnemonic exposes two transports:

- **MCP stdio server** — `skillgrid mcp` — consumed directly by AI agents as tools (`mem_*`, `code_*`, `web_*`).
- **HTTP REST API** — `skillgrid serve` — consumed by agent plugins for session lifecycle, compaction recovery, and index health nudges.

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
```

## MCP configuration

MCP servers are configured from `config.d/mcp.yaml` in the skillgrid repo. During install, skillgrid:

1. Installs MCP packages listed in `config.d/tools.yaml` (`mcp:` section) via `npm install -g`
2. Merges all servers from `config.d/mcp.yaml` into each selected agent's config file

### Server types

| Type | Entry format |
|---|---|
| `remote` | `{"type": "remote", "url": "https://..."}` |
| `local` | `{"type": "local", "command": ["bin", "arg1"], "enabled": true}` |

### Agent config mapping

| Agent | Config file | MCP key |
|---|---|---|
| OpenCode | `~/.config/opencode/opencode.jsonc` | `mcp.<server-name>` |
| Kilo | `~/.config/kilo/kilo.jsonc` | `mcp.<server-name>` |
| Cursor | `~/.cursor/mcp.json` | `mcpServers.<server-name>` |

### Backups

Before editing any agent config file, skillgrid creates a timestamped backup under `~/.skillgrid/backup/<agent-name>/<config>-YYYY-MM-DD-HH:MM.back`.

## MCP tools

### Memory tools (8)

| Tool | What It Does |
|---|---|
| `mem_save` | Save an observation to persistent memory. Deduplicates by content hash within 24h or upserts by `topic_key`. |
| `mem_search` | FTS5 full-text search over saved observations (match_mode: `any`/`all`, default 20 results). |
| `mem_context` | Recent session summaries (default 5) — run this first for fast recall before a full search. |
| `mem_get_observation` | Fetch full untruncated observation content by ID. |
| `mem_session_start` | Create a workspace session for the current directory; returns `session_id`. |
| `mem_session_end` | End a session, optionally with a summary. |
| `mem_session_summary` | Persist a structured end-of-session summary. |
| `mem_suggest_topic_key` | Suggest a stable `topic_key` for upserting an evolving topic. |

### Code tools (4)

| Tool | What It Does |
|---|---|
| `code_status` | Check index health — returns file/chunk counts and `stale`. |
| `code_index` | Run incremental code indexing for the cwd git root. |
| `code_search` | BM25 full-text search over indexed code chunks. |
| `code_read` | Fetch indexed source for a path, with optional `start_line` / `end_line`. |

### Web cache tools (5)

| Tool | What It Does |
|---|---|
| `web_cache_lookup` | Check the cache before a remote call. Returns `hit`/`miss`/`stale`. |
| `web_cache_save` | Persist a snapshot — call immediately after a remote MCP or web fetch returns. |
| `web_cache_search` | FTS5 search over cached snapshots (optional source filter, freshness filter). |
| `web_cache_get` | Fetch a full untruncated snapshot by ID. |
| `web_cache_status` | Cache health stats. |

> **Supported sources:** `context7`, `exa`, `deepwiki`, `fetch`, `manual`.

## HTTP API

`skillgrid serve` starts the HTTP API on `127.0.0.1:7438` (override `--port` / `SKILLGRID_MNEMONIC_PORT`, `--bind`). Write routes require a bearer token when `SKILLGRID_HTTP_TOKEN` is set; read routes are always open.

Browse the interactive API at the root `/` (data viewer) or `/swagger-ui`.

## Backups

Before editing any agent config file, skillgrid creates a timestamped backup under `~/.skillgrid/backup/<agent-name>/<config>-YYYY-MM-DD-HH:MM.back`.
