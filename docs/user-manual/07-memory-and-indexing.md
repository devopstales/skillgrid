# Memory and indexing

**Mnemonic** is Skillgrid’s local-first persistent memory engine (SQLite + FTS5) embedded in the `skillgrid` CLI.

Three capabilities:

| Area | Purpose |
|------|---------|
| **Session memory** | Observations, sessions, relations (`mem_*`) |
| **Code search** | Incremental BM25 index (`code_*`) |
| **Web research cache** | TTL-cached remote research (`web_*`) |

## Quick path

```bash
skillgrid mcp              # agent tools (stdio)
skillgrid serve            # HTTP API + Web UI (default 127.0.0.1:7438)
skillgrid index --dir .    # one-shot code index
```

Data dir: `~/.skillgrid/mnemonic/<project>.sqlite` (override `SKILLGRID_MNEMONIC_DATA_DIR`).

## When to save (agents)

Call `mem_save` after bug fixes, architecture decisions, non-obvious discoveries, config/env setup, patterns, and user preferences. Use a stable `topic_key` for evolving topics. End sessions with `mem_session_summary`.

Recall ladder: `mem_context` → `mem_search` → `mem_timeline` → `mem_get_observation`.

## Code index ladder

```
code_status → code_index (if stale/empty) → code_search → code_read
```

Prefer `code_search` for unfamiliar large repos; use ripgrep for exact identifiers.

Indexing config: hub `config.d/indexing.yaml` (include/exclude, `chunk_lines` default 80, overlap 10). Files over 512 KB are skipped.

## Project identity

Each git repo binds to a **clone-private** identity under `.git/` so memory survives rename, re-clone, and linked worktrees. Parent of many repos → ambiguous; pick a project or set `MNEMONIC_PROJECT` / `SKILLGRID_MNEMONIC_PROJECT`.

## Web cache

Before Context7 / Exa / DeepWiki / fetch: `web_cache_lookup` → remote call on miss → `web_cache_save` (cap 256 KB). Default TTLs: context7 30d, exa/fetch 7d, deepwiki 14d, manual never.

## HTTP API (plugins)

`skillgrid serve` — plugins auto-start it if `/health` fails. Write routes may require `SKILLGRID_HTTP_TOKEN`. Browse the data viewer at `/` — see [Web UI](10-webui.md).

Env:

| Variable | Role |
|----------|------|
| `SKILLGRID_MNEMONIC_DATA_DIR` | Data directory |
| `SKILLGRID_MNEMONIC_PORT` | Serve port (default 7438) |
| `SKILLGRID_HTTP_TOKEN` | Bearer for write routes |
| `SKILLGRID_MNEMONIC_PROJECT` | Pin project for `index` / tools |
| `SKILLGRID_MNEMONIC_HTTP_URL` | Plugin → serve URL |

## Agent wiring

```bash
skillgrid setup opencode|kilocode|cursor
```

Installs plugin/rule + registers `skillgrid-mnemonic` MCP. Details: [Plugins](09-plugins.md).

## Protocol reference

Full tool tables and HTTP routes live in the agent conventions:

- `.agents/skills/_shared/conventions/mnemonic-memory.md`
- `.agents/skills/_shared/conventions/mnemonic-code-indexing.md`
- Skill: `mnemonic-memory`

## Next step

[Ticketing](08-ticketing-integrations.md) · [Web UI](10-webui.md)
