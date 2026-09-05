# MCP servers

`skillgrid install` merges MCP server definitions from the hub into each selected agent’s config.

## Quick path

1. Servers are declared in hub `config.d/mcp.yaml`.
2. Install merges them into OpenCode / Kilo / Cursor config files.
3. Agents call tools over MCP; Mnemonic is `skillgrid mcp` (stdio).

## Config source

```yaml
# config.d/mcp.yaml (hub)
servers:
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
  skillgrid-mnemonic:
    type: local
    command:
      - skillgrid
      - mcp
  # … more servers
```

| Type | Fields |
|------|--------|
| `remote` | `url` |
| `local` | `command` (first entry on `PATH` or absolute after install) |

Related: `config.d/tools.yaml` lists npm packages install may provision for local MCP binaries.

## Shipped servers (defaults)

| Key | Kind | Role |
|-----|------|------|
| `skillgrid-mnemonic` | local | Memory, code index, web cache (`mem_*`, `code_*`, `web_*`) |
| `context7` | remote | Library docs |
| `deepwiki` | remote | GitHub repo docs Q&A |
| `exa` | remote | Web search / fetch |
| `playwright` | local | Browser automation |
| `agent-browser` | local | Browser MCP |
| `trivy` | local | Security scanning |
| `engram` | local | Optional Engram MCP if installed |

Exact set follows the hub’s `config.d/mcp.yaml` on your machine after sync.

## Per-agent merge

| Agent | Config file | Object key |
|-------|-------------|------------|
| OpenCode | `~/.config/opencode/opencode.jsonc` | `mcp` |
| Kilo | `~/.config/kilo/kilo.jsonc` | `mcp` |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` |

Parent directories are created as needed. Install backs up existing config before merge when the installer supports it.

## Mnemonic MCP

```bash
skillgrid mcp          # stdio server for agents
skillgrid mcp --debug
```

Primary tool families:

- `mem_*` — session memory
- `code_*` — code index ladder
- `web_*` — research cache

Details: [Memory and indexing](07-memory-and-indexing.md).

## Context cost

Every enabled MCP injects tool schemas into the agent’s context on each turn. Disable unused heavy servers (e.g. Playwright) for phases that do not need them — keeps the smart zone under budget ([Start here](00-start-here.md#main-logics)).

## Next step

[Multi-agent work](06-multi-agent-work.md)
