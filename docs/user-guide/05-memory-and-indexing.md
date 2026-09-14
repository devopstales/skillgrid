# Memory and indexing

The **`mnemonic`** skill operates Skillgrid's local-first persistent memory engine (SQLite + FTS5, single `skillgrid` binary). Three things a bare agent cannot do:

| Area | Purpose |
|------|---------|
| **Persistent memory** | Decisions, bugfixes, and discoveries in `~/.skillgrid/mnemonic/<project>.sqlite` survive across sessions and compactions. Recall weeks later without re-reading everything. |
| **Code index** | Per-project chunk/symbol/edge graph with freshness tracking. Orient without dumping whole trees. |
| **Web cache** | Context7 / Exa / DeepWiki / WebFetch snapshots with TTLs, so online research is reused instead of re-fetched. |

## Quick path

```bash
skillgrid mcp              # agent tools (stdio)
skillgrid serve            # HTTP API + web UI
skillgrid index --dir .    # one-shot code index
```

Data dir: `~/.skillgrid/mnemonic/<project>.sqlite` (override `SKILLGRID_MNEMONIC_DATA_DIR`). Read `.skillgrid/config.yaml` before starting — the skill reads its config from there.

## When to save (agents)

Call `mem_save` after bug fixes, architecture decisions, non-obvious discoveries, config/env setup, patterns, and user preferences. Use a stable `topic_key` for evolving topics. End sessions with a session summary.

Recall ladder: `mem_context` → `mem_search` → `mem_timeline` → `mem_get_observation`.

## Code index ladder

```
code_status → code_index (if stale/empty) → code_search → code_read
```

Prefer `code_search` for unfamiliar large repos; use ripgrep for exact identifiers.

## Web cache

Before Context7 / Exa / DeepWiki / fetch: `web_cache_lookup` → remote call on miss → `web_cache_save` (cap 256 KB). Default TTLs: context7 30d, exa/fetch 7d, deepwiki 14d, manual never.

## Project identity

Each git repo binds to a **clone-private** identity under `.git/` so memory survives rename, re-clone, and linked worktrees. Parent of many repos → ambiguous; pick a project or set `SKILLGRID_MNEMONIC_PROJECT`.

## Environment

| Variable | Role |
|----------|------|
| `SKILLGRID_MNEMONIC_DATA_DIR` | Data directory |
| `SKILLGRID_MNEMONIC_PORT` | Serve port |
| `SKILLGRID_HTTP_TOKEN` | Bearer for write routes |
| `SKILLGRID_MNEMONIC_PROJECT` | Pin project for `index` / tools |

## Protocol reference

Full tool tables live in the skill references:

- `.agents/skills/mnemonic/references/memory.md`
- `.agents/skills/mnemonic/references/code-indexing.md`

## Next step

[Multi-agent work](06-multi-agent-work.md)
