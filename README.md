<p align="center">
  <img src="docs/assets/v9NDj7Jw.jpeg" alt="skillgrid" width="100%" />
</p>

<h1 align="center">skillgrid</h1>

<p align="center">
  <strong>One CLI. One hub. Every agent, wired.</strong>
</p>

<p align="center">
  Sync skills and MCP tools into Cursor, Kilo, OpenCode, and VS Code Copilot —<br />
  without hunting configs, copy-pasting JSON, or reinventing your stack every project.
</p>

<p align="center">
  <a href="docs/00-start-here.md">Start here</a> ·
  <a href="docs/01-installation.md">Install</a> ·
  <a href="docs/02-usage.md">Usage</a> ·
  <a href="docs/README.md">Docs</a>
</p>

---

## Why skillgrid?

AI coding agents are only as good as the skills and tools behind them. Today that means scattered repos, hand-rolled MCP configs, and a different setup for every client.

**skillgrid** (`aiskillgrid`) is the cross-platform Go CLI that ends that chaos:

1. **Sync** this hub into a managed home (`~/.aiskillgrid`)
2. **Install** skills + MCP wiring into the agents you actually use
3. **Ship** the same memory, code-map, specs, and docs stack everywhere

Independent project. Fresh start. Built to scale with your agent fleet.

---

## What you get

| | |
|---|---|
| **Clients (v1)** | Kilo / Kilo Code · OpenCode · Cursor · VS Code (Copilot) |
| **Coming next** | Claude Code · pi · Gemini CLI · Antigravity · Codex |
| **Tools** | Engram · GitNexus · Context7 · DeepWiki · Exa · Playwright |
| **Skills** | Superpowers (plugin) · Karpathy Guidelines · mattpocock · Engram · gentle-ai via [qntx/skill](https://github.com/qntx/skill) |
| **Runtime** | Managed `~/.aiskillgrid/npm/` + absolute managed bins for MCP — no global Node pollution |

Deep dives: [docs/04-tools.md](docs/04-tools.md) · [docs/05-skills.md](docs/05-skills.md)

---

## Install

### From a release (recommended)

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/aiskillgrid/aiskillgrid/main/scripts/install.sh | bash
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/aiskillgrid/aiskillgrid/main/scripts/install.ps1 | iex
```

Release assets: `aiskillgrid-<version>-<os>-<arch>`  
(e.g. `aiskillgrid-0.1.0-darwin-arm64`, `aiskillgrid-0.1.0-windows-amd64.exe`)

### From source

```bash
cd aiskillgrid-cli
go build -o ../bin/aiskillgrid .
```

Or with [Task](https://taskfile.dev):

```bash
task --taskfile .Taskfile.yml build
```

### Coming soon

- **Homebrew** — `brew install …`
- **Nix** — `nix run .#aiskillgrid`

Full guide: [docs/01-installation.md](docs/01-installation.md)

---

## Managed home

```
~/.aiskillgrid/          # override with AISKILLGRID_HOME
  config.yaml
  tools/                 # git checkout of this hub repo
  dependencies/          # upstream checkouts (superpowers, karpathy, …)
    bin/                 # native binaries (engram, skills, …)
  npm/                   # isolated npm prefix for MCP packages
  state.json
  logs/
  sessions/
  memories/
```

---

## Usage

```bash
aiskillgrid sync                 # clone/pull repo into ~/.aiskillgrid/tools
aiskillgrid install              # pick scope + agents, wire skills & MCP
aiskillgrid install --scope global --agents cursor,kilo --yes
aiskillgrid status
aiskillgrid version
```

Install defaults to **global** scope when prompted; project scope writes under the current working directory.

Install also wires Superpowers as a plugin, Karpathy Guidelines (skill + rules), pack rules, git `commit-msg` hook, and MCP entries (including Exa). Project `AGENT.md` / `CLAUDE.md` / `GEMINI.md` generation is still on the backlog — [docs/06-agent-files.md](docs/06-agent-files.md).

---

## Docs

| Doc | What |
|-----|------|
| [00 — Start here](docs/00-start-here.md) | Product overview |
| [01 — Installation](docs/01-installation.md) | Binary + managed home |
| [02 — Usage](docs/02-usage.md) | sync / install / status |
| [03 — Clients](docs/03-clients.md) | Client ids and paths |
| [04 — Tools](docs/04-tools.md) | Engram, GitNexus, Context7, DeepWiki, Exa, Playwright |
| [05 — Skills](docs/05-skills.md) | Superpowers plugin · Karpathy · composition profile |
| [06 — Agent files](docs/06-agent-files.md) | AGENT.md / CLAUDE.md / GEMINI.md |

---

## Development

```bash
task --taskfile .Taskfile.yml test
task --taskfile .Taskfile.yml release
```

---

<p align="center">
  <em>Stop configuring agents. Start compounding skills.</em>
</p>
