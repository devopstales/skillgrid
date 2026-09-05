# Plugins

`skillgrid setup <agent>` installs Mnemonic integration for one agent and registers `skillgrid mcp`.

```bash
skillgrid setup opencode
skillgrid setup kilocode    # kilo is accepted as an alias
skillgrid setup cursor
```

Flags: `--agent`, `--repo-root`, `--dry-run`.

Sources live in the hub under `plugins/<agent>/` (synced to `~/.skillgrid/repos/skillgrid/plugins/`).

## OpenCode / Kilo

| Action | Detail |
|--------|--------|
| Copy plugin | `plugins/opencode/mnemonic.ts` → agent plugins dir |
| MCP upsert | `skillgrid-mnemonic` → `skillgrid mcp` in `opencode.jsonc` / `kilo.jsonc` |
| Kilo AGENTS | Memory Protocol between managed markers in `~/.config/kilo/AGENTS.md` |
| HTTP | Reads `SKILLGRID_MNEMONIC_HTTP_URL` (default `http://127.0.0.1:7438`) and optional `SKILLGRID_MNEMONIC_HTTP_TOKEN` |

Behaviour:

- Auto-start `skillgrid serve` if `GET /health` fails
- Create session on `session.created`
- Inject Memory Protocol each chat turn
- Recover context on compaction
- Nudge when the code index is stale

## Cursor

| Action | Detail |
|--------|--------|
| MCP | `~/.cursor/mcp.json` → `mcpServers.skillgrid-mnemonic` |
| Rule | `~/.cursor/rules/mnemonic.mdc` from `plugins/cursor/mnemonic.mdc` with Memory Protocol injected |

Cursor is app-side (no npm CLI install). Restart Cursor after MCP changes if tools do not appear.

## Related hub pieces

| Piece | Role |
|-------|------|
| `~/.agents/` | Skills + AGENTS overrides from hub `.agents/` |
| Superpowers / other harness plugins | Optional; install per that project’s docs — Skillgrid does not require Superpowers as a binary dependency |
| `config.d/mcp.yaml` | Broader MCP merge at `skillgrid install` time |

## Checklist

- [ ] `skillgrid setup <agent>` completed
- [ ] Agent lists `skillgrid-mnemonic` MCP tools
- [ ] `skillgrid serve` healthy (or plugin can start it)
- [ ] Memory Protocol present in rules / AGENTS

## Next step

[Web UI](10-webui.md)
