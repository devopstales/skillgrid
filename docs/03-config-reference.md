# Config Reference

Everything the installer consumes lives in `config.d/`. It is the single source of truth — package names, MCP servers, and skills are not hardcoded in the Go code.

Four files matter (the CLI reads three of them):

| File | Consumed by CLI | Purpose |
|------|----------------|---------|
| `tools.yaml` | yes (step 4) | npm packages to install (agent CLIs + tools) |
| `mcp.yaml` | yes (step 7) | MCP servers to merge into each agent config |
| `skills.yaml` | yes (step 6) | Skills to add via the local `skills` CLI |
| `AGENTS.md` | yes (step 8) | Rules file copied to `~/.agents/` and registered in each agent |
| `indexing.yaml` | yes (mnemonic) | Code index + web cache profile when `profile: mnemonic` |

## tools.yaml

The list of npm packages installed with `--prefix "$HOME/.skillgrid/npm" --cache "$HOME/.skillgrid/npm/cache"`.

```yaml
agents:
  - "@kilocode/cli"
  - "opencode-ai"

tools:
  - "vercel-labs/skills"
  - "@playwright/cli@latest"
  - "@playwright/mcp@latest"
  - "agent-browser"
```

Semantics:

- `agents` — agent CLIs. Not used to drive selector behavior (selector is a fixed list of Kilo/OpenCode in the code), just installed alongside.
- `tools` — general CLI tools the agent will use (skills, playwright, agent-browser).
- Any valid `npm install` specifier works: scoped packages, version pins, or git URLs.

## mcp.yaml

The MCP server registry merged into every selected agent's config under the top-level `mcp` key.

```yaml
servers:
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
  engram:
    type: local
    command:
      - engram
      - mcp
  gitnexus:
    type: local
    command:
      - npx
      - -y
      - gitnexus@1.3.11
      - mcp
```

Fields:

- `type: remote` — expects `url`. Merged as `{ type, url, enabled }`.
- `type: local` — expects `command` array. Merged as `{ type, command, enabled }`.
- `enabled` is always set to `true` by the CLI; you can't disable via this file (use each agent's own config to turn off an MCP server).

See [04-mcp-servers](04-mcp-servers.md) for merge and backup details.

## skills.yaml

Skills to install via the local `skills` CLI.

```yaml
skills:
  - repo: obra/superpowers
    skill: "*"
  - repo: gentleman-programming/engram
    skill: engram-memory
  - repo: gentleman-programming/engram
    skill: engram-memory-protocol
  - repo: gentleman-programming/engram
    skill: engram-testing-coverage
```

Per-entry fields:

- `repo` (required) — GitHub repo the skill is installed from
- `skill` (defaults to `"*"`) — the skill name (or `*` for all)
- `agent` (defaults to `amp`) — target agent identifier for the `skills` CLI

The CLI invokes `<skills> add <repo> --agent <agent> -g -s <skill> -y` for each entry, using `~/.skillgrid/npm/node_modules/.bin/skills` if present.

## indexing.yaml

Mnemonic profile for built-in SQLite memory, code FTS index, and web research cache. Read by `skillgrid index`, `skillgrid mcp`, and `skillgrid serve` — they walk up from the working directory to find `config.d/indexing.yaml` (same pattern as project resolution).

```yaml
profile: mnemonic
mnemonic:
  include: ["**/*.go", "**/*.ts", "**/*.tsx", "**/*.md"]
  exclude: ["**/node_modules/**", "**/.git/**", "**/dist/**"]
  chunk_lines: 80
  chunk_overlap: 10
  embeddings: false
  web_cache:
    enabled: true
    max_entry_bytes: 262144
    ttl:
      context7: 720h
      exa: 168h
      deepwiki: 336h
      fetch: 168h
      manual: 0
    sources: [context7, exa, deepwiki, fetch, manual]
  http:
    enabled: true
    host: 127.0.0.1
    port: 7438
    auto_start: plugin
gitnexus:
  enabled: false
```

Fields consumed in v1:

| Field | Purpose |
|-------|---------|
| `profile` | `mnemonic` selects the built-in indexer; other profiles fall back to defaults |
| `mnemonic.include` / `exclude` | Glob patterns for `skillgrid index` |
| `mnemonic.chunk_lines` / `chunk_overlap` | Line-based chunking for FTS |
| `mnemonic.web_cache.*` | TTL and size limits for cached Context7/Exa/DeepWiki/fetch snapshots |
| `gitnexus.enabled` | Opt-in flag for GitNexus impact graphs (not consumed until hybrid profile lands) |

v1.1+ keys (`watch`, `branch_aware`, `search.fusion_strategy`, `plugin.*`, `embeddings`) are documented in the shipped file for forward compatibility; the Go loader ignores them until their tasks land.

When `profile: mnemonic` is active, uncomment `skillgrid-mnemonic` in `mcp.yaml` to replace external `engram mcp` after cutover. See [04-mcp-servers](04-mcp-servers.md).

## AGENTS.md

The rules file. See [06-rules](06-rules.md) for the pipeline that moves it into every agent.
