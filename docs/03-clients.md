# Clients

Shipped targets: **OpenCode**, **Kilo**, and **Cursor**, **global** config only.

| | OpenCode | Kilo | Cursor |
|--|----------|------|--------|
| Agent id (`--agents`) | `opencode` | `kilo` | `cursor` |
| CLI (npm, if selected) | `opencode-ai` | `@kilocode/cli` | — (app-side only) |
| Config file | `~/.config/opencode/opencode.jsonc` | `~/.config/kilo/kilo.jsonc` | `~/.cursor/mcp.json` |
| MCP object key | `mcp` | `mcp` | `mcpServers` |
| Rules dir | `~/.config/opencode/rules/` | `~/.config/kilo/AGENTS.md` | `~/.cursor/rules/` |
| Plugin file | `~/.config/opencode/plugins/mnemonic.ts` | `~/.config/kilo/plugins/mnemonic.ts` | `~/.cursor/rules/mnemonic.mdc` |

Parent directories are created as needed.

| Topic | Doc |
|-------|-----|
| Skills via managed `skills` CLI | [Skills](05-skills.md) |
| Curated rule files from the hub | [Rules](06-rules.md) |
| MCP merge, shapes, backups | [MCP](08-mcp.md) |
| Superpowers plugin for OpenCode and Kilo | [Plugins](09-plugins.md) |

## PATH vs MCP

MCP `command` values use absolute paths. You do not need `~/.skillgrid/bin` on `PATH` for agents to start those servers.
