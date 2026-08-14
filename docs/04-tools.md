# Tools

Agreed optional tool integrations for aiskillgrid. Default pattern on `aiskillgrid install`: **install into managed home (binary or npm) → resolve MCP placeholders → merge into selected agents**. If system `node`/`npm` is missing, npm-only tools are skipped with a warning; binary installs and agent skill copy continue. **No Homebrew or Nix** for tool provisioning.

## v1 tool set

| Pillar | Tool | Role | Integration |
|--------|------|------|-------------|
| Persistent memory | [Engram](https://github.com/Gentleman-Programming/engram) | Cross-session agent memory (SQLite + MCP) | **Managed binary** → `dependencies/bin/engram` + MCP |
| Code map / indexing | [GitNexus](https://github.com/abhigyanpatwari/GitNexus) | Graph / impact analysis (“what depends on X”) | **Managed npm** → `gitnexus` + MCP |
| Spec-driven development | [OpenSpec](https://github.com/Fission-AI/OpenSpec) | Change specs, propose/apply/verify workflow | **Managed npm** `@fission-ai/openspec` + skills + scaffold (scaffold later) |
| Issue / task tracking | [Backlog.md](https://github.com/MrLesk/Backlog.md) | Markdown tasks, milestones, AI review checkpoints | **Managed npm** `backlog.md` + skills + MCP + project scaffold (scaffold later) |
| Library docs | [Context7](https://context7.com) MCP | Up-to-date library/framework API docs in-agent | **Managed npm** `@upstash/context7-mcp` via managed `npx` |
| Repo / wiki docs | [DeepWiki](https://deepwiki.com) MCP | Ask questions about GitHub repos / codebase docs | **HTTP remote MCP** (always available; no local binary) |
| Browser / E2E | [Playwright](https://playwright.dev) MCP | Drive a real browser for UI checks and flows | **Managed npm** `@playwright/mcp` via managed `npx`; browsers may need separate install |

### Engram

- Real store is Engram’s own DB (typically under `~/.engram/`), not a second DB in `~/.aiskillgrid/memories/`.
- `aiskillgrid install` downloads the GitHub release binary into `~/.aiskillgrid/dependencies/bin/engram`.
- Merge Skillgrid-owned MCP entry (prefix `aiskillgrid-`) for selected clients when the binary is present; warn on download failure.
- When Engram is wired, install curated [engram/skills](https://github.com/Gentleman-Programming/engram/tree/main/skills) (always include `memory-protocol`) per [05-skills.md](05-skills.md) — orchestration is slice B.

### GitNexus

- Primary codebase indexer for v1 (structural / impact).
- Installed via managed npm (`npm install --prefix ~/.aiskillgrid/npm gitnexus`); MCP uses absolute path to managed `gitnexus` bin.
- **CocoIndex (`ccc`) is deferred** — optional later; not a v1 peer (avoids two overlapping search tools).

### OpenSpec

- Not primarily an MCP server: CLI + repo artifacts + agent skills.
- Installed via managed npm (`@fission-ai/openspec`); `openspec` CLI available under `~/.aiskillgrid/npm/bin/`.
- Install official skills from [Fission-AI/OpenSpec/skills](https://github.com/Fission-AI/OpenSpec/tree/main/skills) via the managed `skills` binary (`skills add Fission-AI/OpenSpec`) — do not vendor copies into `packs/skills/` (slice B).
- On **project** scope install: initialize OpenSpec in the project if not already present (`openspec/` layout/config) — **not yet implemented** (slice C).
- When Engram is also available, Skillgrid defaults SDD artifact store to **hybrid** (Engram + OpenSpec files) — see [05-skills.md](05-skills.md).

### Backlog.md

- Local-first AI issue tracker: tasks as markdown in the repo, optional board/web UI, MCP for agents.
- Complements OpenSpec: **Backlog.md = tasks & execution tracking**; **OpenSpec = change specs / SDD artifacts**.
- Installed via managed npm (`backlog.md`); MCP uses absolute path to managed `backlog` bin.
- On install (future slices): install skills, wire MCP, scaffold project (`backlog init`) — skills/scaffold not in tooling spine v1.
- **No brew/nix/global npm** for Backlog; only managed npm or a future GitHub release binary into `dependencies/bin/`.

### Context7

- MCP for current library and framework documentation (reduces hallucinated APIs).
- On install: `npm install --prefix ~/.aiskillgrid/npm @upstash/context7-mcp`; merge Skillgrid-owned MCP entry (e.g. `aiskillgrid-context7`) via managed `npx`.
- Requires system `node` + `npm` on PATH for the install phase.
- Optional API key: document env var if Context7 requires one; do not store secrets in the hub repo.

### DeepWiki

- MCP for Q&A over GitHub repositories / generated wiki-style docs ([DeepWiki](https://deepwiki.com)).
- On install: merge Skillgrid-owned MCP entry (e.g. `aiskillgrid-deepwiki`) using the **HTTP remote URL** pinned in `packs/mcp/servers.json` — no local npm package required.
- Always considered available for MCP resolution (`http:deepwiki`); complements GitNexus: **GitNexus = structural code map**; **DeepWiki = natural-language docs/repo understanding**.

### Playwright

- MCP for browser automation and UI verification ([Playwright MCP](https://github.com/microsoft/playwright-mcp)).
- On install: `npm install --prefix ~/.aiskillgrid/npm @playwright/mcp`; merge Skillgrid-owned MCP entry via managed `npx`.
- Playwright browsers (`npx playwright install`) are **not** run automatically in v1; install warns that browsers may be required on first use.

## Explicitly later

- CocoIndex Code (semantic search)
- Web research / other browser stacks (agent-browser, etc.) as alternatives
- Security scanners
- OpenSpec / Backlog project scaffolds and upstream skill orchestration (see [05-skills.md](05-skills.md))

## Runtime layout for tools

| Path | Purpose |
|------|---------|
| `~/.aiskillgrid/dependencies/bin/` | Native binaries (`engram`, `skills` from qntx/skill, …) |
| `~/.aiskillgrid/npm/` | Isolated npm prefix/cache + bins (`gitnexus`, `openspec`, `backlog`, `npx`, MCP CLIs) |

## Status

`aiskillgrid status` prints managed npm path, system node availability, and which managed binaries are present:

```text
Managed npm: ~/.aiskillgrid/npm (node: yes|no)
Binaries: engram=yes|no skills=yes|no
NPM bins: gitnexus=yes|no backlog=yes|no openspec=yes|no
```

Tooling spine (managed install + MCP resolve/merge) is **implemented** in the CLI. Skill orchestration, project scaffolds, and agent-file generation remain in [TODO.md](TODO.md).
