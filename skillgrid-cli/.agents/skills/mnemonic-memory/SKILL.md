---
name: mnemonic-memory
description: Use when the user asks to save, search, or recall curated observations, session history, code index state, or cached web research in skillgrid's local-first Mnemonic subsystem (SQLite + FTS5).
---

# Mnemonic Memory Skill

Mnemonic is skillgrid's local-first persistent memory. It replaces the older Engram/Mnemonic MCP tools with a unified SQLite+FTS5 store, exposed via `mem_*`, `code_*`, and `web_*` MCP tools and a REST API.

## Data Layout

- Per-project SQLite: `~/.skillgrid/mnemonic/<projectID>.sqlite` (WAL mode)
- Project ID from `.skillgrid/config.json` → git remote origin → `{base}-{hash}` fallback
- Config: `config.d/indexing.yaml` (mnemonic section) — include/exclude globs, chunk size, web TTL

## Tools

### Memory
- `mem_session_start` — open a workspace session (required before mem_save)
- `mem_save` — save observation (dedup by hash 24h, upsert by topic_key)
- `mem_search` — FTS5 over observations (match_mode: any|all)
- `mem_context` — recent session summaries
- `mem_get_observation` — full content by ID
- `mem_session_summary` / `mem_session_end` — close session
- `mem_suggest_topic_key` — derive stable key from type+title

### Code
- `code_status` — index health (file/chunk count, stale flag)
- `code_index` — incremental index (mtime + content hash)
- `code_search` — BM25 FTS over chunks
- `code_read` — source slice for a path

### Web Cache
- `web_cache_lookup` — check cache before remote MCPs
- `web_cache_save` — persist after Context7/Exa/DeepWiki/WebFetch
- `web_cache_search` — FTS5 over cached research (fresh_only filter)
- `web_cache_get` / `web_cache_status`

## Storage

All `mem_save` observations must include:
- **title**: verb + what
- **type**: decision | architecture | bugfix | pattern | config | discovery | learning | preference | convention
- **content**: structured What/Why/Where/Learned
- **scope**: project (default) | user | global

Reuse `topic_key` to upsert evolving topics. Use `mem_suggest_topic_key` when unsure.

## Sessions

1. Start with `mem_session_start`
2. Save observations with `mem_save`
3. Close with `mem_session_summary` + `mem_session_end`

## REST API

The `skillgrid serve` command exposes the same functionality over HTTP (`SKILLGRID_HTTP_TOKEN` for write auth):
- `GET /health`
- `POST /sessions`, `POST /sessions/{id}/end`
- `GET /context`, `POST /observations`, `GET /search`
- `GET /code/status`, `GET /code/search`
- `GET /web/lookup`, `POST /web/cache`, `GET /web/search`, `GET /web/status`

## CLI

- `skillgrid mcp` — start MCP stdio server
- `skillgrid serve` — start HTTP API (:7438)
- `skillgrid index` — run code indexing
- `skillgrid setup --agent opencode|kilocode|cursor` — install plugins

## Env

- `SKILLGRID_MNEMONIC_DATA_DIR` — data directory (default `~/.skillgrid/mnemonic`)
- `SKILLGRID_MNEMONIC_PORT` — HTTP port (default 7438)
- `SKILLGRID_HTTP_TOKEN` — bearer token for write routes
