# aiskillgrid CLI — Design

Date: 2026-08-14

Independent greenfield project. Not based on any previous Skillgrid repo.

## Goal

Cross-platform `aiskillgrid` command that syncs this GitHub repo into a managed home and installs skills + MCP into selected AI clients. Optional tools are detected and wired per [04-tools.md](../../04-tools.md).

## Decisions

| Topic | Decision |
|-------|----------|
| Install target | Skills + MCP wiring |
| v1 clients | Kilo / Kilo Code, OpenCode, Cursor, VS Code (Copilot) |
| Planned clients | Claude Code, pi, Gemini CLI, Antigravity, Codex |
| Memory | Engram — managed binary + MCP |
| Code map | GitNexus — managed npm + MCP (CocoIndex later) |
| Plans / specs / tasks | **Superpowers only** (`docs/superpowers/`, `.superpowers/`) |
| OpenSpec | **Deferred** — not in default v1 stack |
| Backlog.md | **Deferred** — not in default v1 stack |
| Library docs | Context7 — wire MCP via managed `npx` |
| Repo docs Q&A | DeepWiki — wire MCP via HTTP |
| Browser / E2E | Playwright — wire MCP via managed `npx`; ensure browsers |
| Planned skills | Superpowers + mattpocock + curated **Engram** + curated **gentle-ai** via **[qntx/skill](https://github.com/qntx/skill)** (`packs/skills/sources.yaml`); never `gentle-ai install` |
| Node / npx | Managed under `~/.aiskillgrid/npm/` for MCP servers and other npm tools; `aiskillgrid install` ensures `npx` there |
| CLI install methods | Release scripts (done); **Homebrew** planned; **Nix flake** planned (`flake.nix`) |
| Project agent files | On project install: generate/refresh `AGENT.md`, `CLAUDE.md`, `GEMINI.md` (merge Skillgrid sections) |
| CLI | Go static binary |
| Scope | Ask at install; default global; allow project |
| Home | `~/.aiskillgrid/` (`tools/`, `dependencies/bin/`, `npm/`, …) |
| Transport | Copy files (no default symlinks) |
| MCP | Prefix `aiskillgrid-`; merge only; one-time `.bak` |
| Tool binaries | Engram + `skills` binary into managed home; GitNexus via managed npm |

## Home

```text
~/.aiskillgrid/
  config.yaml
  tools/
  dependencies/
    bin/          # skills (qntx/skill), engram, …
  npm/            # isolated npm prefix/cache + npx for MCP tools
  state.json
  logs/
  sessions/
  memories/
```

## CLI

- `aiskillgrid version`
- `aiskillgrid sync`
- `aiskillgrid install` (`--scope`, `--agents`, `--yes`, `--skip-sync`, `--repo`)
- `aiskillgrid status`

## Out of scope (v1)

- OpenSpec / Backlog.md as default tools or skill sources
- Auto-install of Engram / GitNexus into `dependencies/` beyond current managed install
- CocoIndex as a peer indexer
- Using user-global `npx skills` (use qntx/skill binary instead)
- Uninstall / doctor
- Clients outside the v1 set
- Personas, remote ticketing SaaS, custom web UI

## Docs

- [00-start-here.md](../../00-start-here.md)
- [03-clients.md](../../03-clients.md)
- [04-tools.md](../../04-tools.md)
- [05-skills.md](../../05-skills.md)
- [06-agent-files.md](../../06-agent-files.md)
- [2026-08-14-aiskillgrid-skills-composition-design.md](./2026-08-14-aiskillgrid-skills-composition-design.md)
- [TODO.md](../../TODO.md)
