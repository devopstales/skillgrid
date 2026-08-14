# Clients

## v1 (implemented)

| Display name | `--agents` id | Skills (global / project) | Rules (global / project) | MCP (global / project) |
|--------------|---------------|---------------------------|--------------------------|------------------------|
| Kilo / Kilo Code | `kilo` | `~/.kilo/skills/` / `.kilo/skills/` | `~/.kilo/rules/` / `.kilo/rules/` | `~/.config/kilo/kilo.jsonc` / `.kilo/kilo.jsonc` (`mcp`) |
| OpenCode | `opencode` | `~/.config/opencode/skills/` / `.opencode/skills/` | `~/.config/opencode/rules/` / `.opencode/rules/` | `~/.config/opencode/opencode.json` / `opencode.json` (`mcp`) |
| Cursor | `cursor` | `~/.cursor/skills/` / `.cursor/skills/` | `~/.cursor/rules/` / `.cursor/rules/` | `~/.cursor/mcp.json` / `.cursor/mcp.json` (`mcpServers`) |
| VS Code (Copilot) | `copilot` | `~/.copilot/skills/` / `.github/skills/` | `~/.copilot/instructions/` / `.github/instructions/` (`.instructions.md`) | `…/Code/User/mcp.json` / `.vscode/mcp.json` (`servers`) |

Kilo and Kilo Code share one adapter in v1.

`aiskillgrid install` copies `packs/rules/*.mdc` into each selected agent’s rules directory (Copilot: renamed to `*.instructions.md`).

### Superpowers plugin (all v1 agents)

In addition to Skillgrid packs, install wires [obra/superpowers](https://github.com/obra/superpowers) as a **plugin** (not only skills):

| Agent | Plugin target |
|-------|----------------|
| `opencode` | `plugin` array in `opencode.json` |
| `kilo` | `kilo plugin install` or `plugin` array in `kilo.jsonc` |
| `cursor` | `~/.cursor/plugins/local/superpowers` |
| `copilot` | Copilot CLI marketplace plugin when `copilot` is on PATH |

See [05-skills.md](05-skills.md).

### Karpathy Guidelines (all v1 agents)

Install also copies [Karpathy Guidelines](https://github.com/multica-ai/andrej-karpathy-skills) skill + rules into each selected agent’s skills/rules directories (managed checkout under `~/.aiskillgrid/dependencies/andrej-karpathy-skills`).

## Planned later

- Claude Code
- pi
- Gemini CLI
- Antigravity
- Codex

## MCP merge rules

- Only overwrite keys prefixed `aiskillgrid-`
- Do not wipe unrelated servers
- One-time `*.bak` before first Skillgrid write to a file
