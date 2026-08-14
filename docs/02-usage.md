# Usage

## Commands

| Command | Purpose |
|---------|---------|
| `aiskillgrid version` | Print version |
| `aiskillgrid sync` | Clone/pull hub into `~/.aiskillgrid/tools/` |
| `aiskillgrid install` | Wire skills + rules + MCP into selected clients |
| `aiskillgrid status` | Show home, sync rev, last install |

## Sync

```bash
aiskillgrid sync
aiskillgrid sync --repo https://github.com/OWNER/REPO.git
```

Aborts if `tools/` has local changes (no force wipe in v1).

## Install

Interactive:

```bash
aiskillgrid install
```

Asks for:

1. Scope — `global` (default) or `project`
2. Clients — multi-select (`kilo`, `opencode`, `cursor`, `copilot`)

Non-interactive:

```bash
aiskillgrid install --scope global --agents kilo,opencode,cursor,copilot --yes
aiskillgrid install --scope project --agents cursor --yes --skip-sync
```

| Flag | Meaning |
|------|---------|
| `--scope` | `global` or `project` |
| `--agents` | Comma-separated client ids |
| `--yes` | Non-interactive |
| `--skip-sync` | Skip git sync |
| `--repo` | Override hub URL for this run |

Install copies `packs/skills/` and `packs/rules/` into each selected agent, merges `packs/mcp/` (keys prefixed `aiskillgrid-`, including Exa), installs **Superpowers** as a native plugin and **Karpathy Guidelines** (skill + rules), installs a **git `commit-msg` hook** that strips AI co-authors (when cwd is a git repo; preserves any existing hook as `commit-msg.aiskillgrid-prev`), and writes `state.json` / logs. Project `AGENT.md` / `CLAUDE.md` / `GEMINI.md` generation is still backlog — see [06-agent-files.md](06-agent-files.md).

## Status

```bash
aiskillgrid status
```
