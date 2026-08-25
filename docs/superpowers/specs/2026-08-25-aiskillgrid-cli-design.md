# 2026-08-25 aiskillgrid-cli Design

> **STATUS: COMPLETE (2026-08-25)** — all steps of this spec are implemented; `go build` + all tests pass. See the plans doc for the full task history.

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

Implemented flags (per-subcommand parsing; flags work before or after the subcommand name):

- `--skip-clone` — skip the git clone step during install
- `--sync-repo <path>` — sync a local checkout into `~/.aiskillgrid/repos/aiskillgrid` instead of cloning
- `--dry-run` — print planned changes without writing (no npm installs, no MCP/rules writes, no backups)
- `--verbose` — print detailed changes (full MCP entries)
- `--yes` — skip interactive prompts; default agent selection

`AISKILLGRID_REPO_URL` env var overrides the clone target.

## Install Flow (Step-by-Step)

Implemented in `cmd/install.go` + `cmd/steps.go`, matching `docs/00-aiskillgrid-cli.md`:

0. **Agent selector** — `selectAgents` (interactive; `--yes`/`--dry-run` skip it and default to all agents) — runs **first**, before any setup

1. **Repo setup** — `internal/repo/repo.go`
   - `--sync-repo <path>`: copy the local checkout into `~/.aiskillgrid/repos/aiskillgrid`
   - otherwise clone `https://github.com/devopstales/aiskillgrid.git` (override: `AISKILLGRID_REPO_URL`)
   - `--skip-clone` + `--sync-repo` together: sync wins; both skipped: use existing `~/.aiskillgrid` state
   - Copy `config.d/` into `~/.aiskillgrid/config.d`

2. **Node validation** — `ensureNode`
   - `node` on PATH → pass; otherwise run `~/.aiskillgrid/repos/aiskillgrid/scripts/install_node.sh` (warn, do not abort)

3. **Engram binary installation** — `internal/engram/install.go`
   - Detect OS/ARCH; query GitHub Releases API for latest `engram` version
   - Download matching prebuilt tarball (`darwin_arm64`, `darwin_amd64`, `linux_arm64`, `linux_amd64`)
   - Extract `engram` binary into `~/.aiskillgrid/bin/`, chmod `+x`
   - Do not use Homebrew or `go install`

4. **Agent and tool installation** — `cmd/npm.go`
   - Read `~/.aiskillgrid/config.d/tools.yaml`
   - `npm install <agents...> <tools...> --prefix "$HOME/.aiskillgrid"` (binaries land in `~/.aiskillgrid/node_modules/.bin`)

5. **Plugin installation** — `installPlugins`
   - Per selected agent:
     ```bash
     npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/<agent>"
     ```
   - Register under the config's `plugin` key: `["~/.config/<agent>/node_modules/superpowers"]` (idempotent append)
   - If `opencode` selected: run `~/.aiskillgrid/bin/engram setup opencode`
   - Copy `~/.config/opencode/plugins/engram.ts` → `~/.config/kilo/plugins/engram.ts` if missing

6. **Skill installation** — `installSkills`
   - Read `~/.aiskillgrid/config.d/skills.yaml` (`repo`, `skill`, optional `agent` — default `amp`)
   - Per entry: `~/.aiskillgrid/node_modules/.bin/skills add <repo> --agent <agent> -g -s <skill> -y` (warn and continue on failure)

7. **MCP installation** — `internal/config/merger.go` + `internal/mcp/registry.go`
   - Read `~/.aiskillgrid/config.d/mcp.yaml`
   - Merge entries into each selected agent's config under the `mcp` key (gjson/sjson, JSONC-aware)
   - Preserves existing keys; overwrite only managed entries; **backup to `~/.aiskillgrid/backups/` before every edit** (keep last 10 per file)

8. **Rules** — `copyRules`
   - Copy `~/.aiskillgrid/config.d/AGENTS.md` → `~/.agents/AGENTS.md`
   - Append `~/.agents/AGENTS.md` to each selected agent config's `instructions` array (JSON-aware, idempotent)

9. **PATH output**
    - Print after the `install finished` line, separated by a blank line:
      ```bash
      export PATH="$HOME/.aiskillgrid/bin:$PATH"
      export PATH="$HOME/.aiskillgrid/node_modules/.bin:$PATH"
      ```

## MCP Tool Registry

Tools are defined in `config.d/mcp.yaml`, not hardcoded. Default set:

| ID | Type | Source |
|-----|------|--------|
| `context7` | remote | `https://mcp.context7.com/mcp` |
| `deepwiki` | remote | `https://mcp.deepwiki.com/mcp` |
| `exa` | remote | `https://mcp.exa.ai/mcp` |
| `engram` | local | `engram mcp` |
| `ccc` | local | `ccc mcp` |
| `gitnexus` | local | `npx -y gitnexus@1.3.11 mcp` |
| `trivy` | local | `trivy mcp` |

Local tools: dependency check warns but does not fail if the binary is missing.

## Skills Registry

Skills are defined in `config.d/skills.yaml`, not hardcoded. Default set:

| Repo | Skill | Agent (default `amp`) |
|------|-------|------|
| `obra/superpowers` | `*` | — |
| `gentleman-programming/engram` | `engram-memory` | — |
| `gentleman-programming/engram` | `engram-memory-protocol` | — |
| `gentleman-programming/engram` | `engram-testing-coverage` | — |

## Interactive Selection UI

Implemented as a prompt-based agent selector (`selectAgents`, `cmd/steps.go`):
1. Select which agents to configure (comma-separated; empty input = all) — `--yes` / `--dry-run` skip it and default to all agents

The bubbletea TUI for per-agent MCP tool selection (step 2) is **not yet implemented**; all MCP servers from `mcp.yaml` are merged into each selected agent.

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
│   ├── go.mod / go.sum
│   ├── cmd/
│   │   ├── main.go            # subcommand + flag parsing, usage, help
│   │   ├── main_test.go
│   │   ├── install.go         # runInstall flow, backups, rules, sync-repo
│   │   ├── npm.go             # npm install of tools.yaml packages
│   │   └── steps.go           # node check, agent selector, plugins, skills
│   ├── internal/
│   │   ├── config/
│   │   │   ├── types.go       # ToolsConfig, SkillsConfig, MCPConfig + YAML loaders
│   │   │   ├── merger.go      # gjson/sjson JSONC-aware MCP merge
│   │   │   ├── path.go        # PATH instruction writer
│   │   │   ├── testdata/      # tools.yaml, mcp.yaml fixtures
│   │   │   ├── *_test.go
│   │   ├── mcp/
│   │   │   └── registry.go    # mcp.yaml loader + dependency precheck
│   │   ├── engram/
│   │   │   └── install.go     # prebuilt binary installer
│   │   ├── repo/
│   │   │   └── repo.go        # Sync/Clone into ~/.aiskillgrid
│   │   ├── logging/
│   │   │   └── log.go         # file-based validation logger
│   │   └── smoke/
│   │       └── smoke_test.go  # integration smoke test
├── docs/
│   ├── 00-aiskillgrid-cli.md
│   └── superpowers/{plans,specs}/
├── config.d/          ← synced from repo for local dev
│   ├── AGENTS.md
│   ├── mcp.yaml
│   ├── skills.yaml
│   └── tools.yaml
├── scripts/
│   └── install_node.sh
├── Taskfile.yml
└── README.md
```

## Testing

- Unit tests for config merge logic (merge, comment preservation, dry-run)
- Unit tests for PATH output, YAML loaders (tools/mcp/skills), and dry-run semantics
- Integration smoke test: run `install --dry-run` against temp home dir
- Full-suite: `cd aiskillgrid-cli && go test ./...` (all packages green)
