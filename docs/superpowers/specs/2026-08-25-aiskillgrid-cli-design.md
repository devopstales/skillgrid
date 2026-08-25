# 2026-08-25 aiskillgrid-cli Design

## Goal

A single Go binary that installs and configures AI agent tooling from a checked-out `aiskillgrid` repo using config-driven steps for repo setup, Node validation, npm installs, plugins, skills, MCPs, and PATH output.

## Supported Agents

| Agent ID | Config paths |
|----------|-------------|
| `kilo` | `~/.config/kilo/kilo.jsonc` |
| `opencode` | `~/.config/opencode/opencode.jsonc` |
| `cursor` | *(config path TBD)* |
| `claude` | *(config path TBD)* |
| `codex` | *(config path TBD)* |
| `gemini-cli` | *(config path TBD)* |
| `antigravity` | *(config path TBD)* |

> Config paths for `cursor`, `claude`, `codex`, `gemini-cli`, and `antigravity` are not yet defined in `docs/00-aiskillgrid-cli.md`; listed here from `docs/NOTE.md` for completeness.

## Base Directory

All install paths are rooted at `~/.aiskillgrid/`.

## Config-Driven Data

`config.d/tools.yaml` defines the canonical install inputs:

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

The CLI reads this file after cloning/syncing the repo; it does not hardcode these package names.

## Subcommands

| Command | Alias | Purpose |
|---------|-------|---------|
| `install` | `in` | Clone repo, validate Node, install plugins/skills/MCPs, print PATH additions |
| `sync-repo` | — | Sync external repo contents into `~/.aiskillgrid/repos/aiskillgrid` without running full install |

## Flags

- `--skip-clone` — skip the git clone step during install
- `--sync-repo` — sync extra paths into `~/.aiskillgrid/repos/aiskillgrid` during install
- `--dry-run` / `-d` — print planned changes without writing

## Install Flow (Step-by-Step)

1. **Repo setup**
   - Create `~/.aiskillgrid/` with `repos/` subdir
   - Clone `https://github.com/devopstales/aiskillgrid.git` into `repos/` on `release/2` branch
   - Copy `repos/aiskillgrid/config.d` into `~/.aiskillgrid/config.d`

2. **Node validation**
   - Detect or install Node.js per `scripts/install_node.sh`
   - Fail with clear message if Node is missing and auto-install is unavailable

3. **Engram binary installation**
   - Detect OS/ARCH
   - Query GitHub Releases API for latest `engram` version
   - Download matching prebuilt tarball:
     - `darwin_arm64`, `darwin_amd64`, `linux_arm64`, `linux_amd64`
   - Extract `engram` binary into `~/.aiskillgrid/bin/`
   - Make binary executable
   - Do not use Homebrew or `go install`

4. **Agent and tool installation**
   - Read `~/.aiskillgrid/config.d/tools.yaml`
   - Install each package into `~/.aiskillgrid/` via npm:
     ```bash
     npm install <package> --prefix "$HOME/.aiskillgrid"
     ```
   - Packages are exactly those listed under `agents:` and `tools:` in `tools.yaml`

5. **Plugin installation**
   - Run:
     ```bash
     npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"
     npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/opencode"
     ```
   - Update each agent's JSON config with:
     ```json
     { "plugin": ["<resolved-plugin-path>"] }
     ```

6. **Skill installation**
   - Read `~/.aiskillgrid/config.d/skills.yaml`
   - For each entry, run:
     ```bash
     npx skills add <repo> --agent <agent> -g -s '*' -y
     ```

6. **MCP installation**
   - Read `~/.aiskillgrid/config.d/mcp.yaml`
   - Merge MCP server entries into each selected agent's config under the `mcp` key
   - Preserve existing keys; overwrite only new/updated entries

7. **PATH output**
   - Print shell export statements the user should add to their rc file:
     ```bash
     export PATH="$HOME/.aiskillgrid/bin:$PATH"
     export PATH="$HOME/.aiskillgrid/npm/.bin:$PATH"
     ```

## MCP Tool Registry

Tools are defined in config files, not hardcoded. Default set:

| ID | Type | Source |
|-----|------|--------|
| `context7-http` | remote | `https://mcp.context7.com/mcp` |
| `deepwiki-http` | remote | `https://mcp.deepwiki.com/mcp` |
| `exa-http` | remote | `https://mcp.exa.ai/mcp` |
| `engram` | local | `engram mcp` |
| `ccc` | local | `ccc mcp` |
| `gitnexus` | local | `gitnexus mcp` |
| `trivy` | local | `trivy mcp` |

Local tools: dependency check warns but does not fail if the binary is missing.

## Interactive Selection UI

Two-step selection using bubbletea TUI:
1. Select which agents to configure
2. For each selected agent, select which MCP tools to enable

## Config Merge Semantics

- Read existing JSON/JSONC config as `map[string]interface{}`
- Merge MCP entries under the `mcp` key
- Do not strip comments or unrelated keys (JSONC-aware parsing required; round-trip must preserve comments)
- Write back with 2-space indentation

## Error Handling

- Repo clone failure → abort install, print message
- Node validation failure → abort install, print instructions
- Plugin install failure → warn, continue
- Skill install failure → warn, continue
- MCP config write failure → warn, continue to next agent
- Local tool missing on PATH → warn, continue

## Non-Goals

- No agent runtime management (starting/stopping agents)
- No auto-update of MCP tools post-install
- No Windows support in v1
- No interactive wizard beyond agent/tool selection

## File Layout

```
aiskillgrid-v2/
├── aiskillgrid-cli/
│   ├── go.mod
│   ├── cmd/
│   │   ├── main.go
│   │   └── install/
│   │       └── install.go
│   ├── internal/
│   │   ├── config/
│   │   │   ├── writer.go
│   │   │   └── merger.go
│   │   ├── mcp/
│   │   │   └── registry.go
│   │   └── tui/
│   │       └── select.go
│   └── scripts/
│       └── install_node.sh
├── docs/
│   └── 00-aiskillgrid-cli.md
├── config.d/          ← synced from repo for local dev
│   └── tools.yaml
├── Taskfile.yml
└── README.md
```

## Testing

- Unit tests for config merge logic
- Unit tests for `--dry-run` output
- Integration smoke test: run `install --dry-run` against temp home dir
