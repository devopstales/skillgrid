# Tools

Agreed optional tool integrations for aiskillgrid. Default pattern on `aiskillgrid install`: **install into managed home (binary or npm) → resolve MCP placeholders → merge into selected agents**. If system `node`/`npm` is missing, npm-only tools are skipped with a warning; binary installs and agent skill copy continue. **No Homebrew or Nix** for tool provisioning.

**Plans / specs / tasks** are **not** tools here — they are owned by **Superpowers** (`docs/superpowers/`, `.superpowers/`). See [05-skills.md](05-skills.md). OpenSpec and Backlog.md are **out of the default v1 stack**.

## v1 tool set

| Pillar | Tool | Role | Integration |
|--------|------|------|-------------|
| Persistent memory | [Engram](https://github.com/Gentleman-Programming/engram) | Cross-session agent memory (SQLite + MCP) | **Managed binary** → `dependencies/bin/engram` + MCP |
| Code map / indexing | [GitNexus](https://github.com/abhigyanpatwari/GitNexus) | Graph / impact analysis (“what depends on X”) | **Managed npm** → `gitnexus` + MCP |
| Library docs | [Context7](https://context7.com) MCP | Up-to-date library/framework API docs in-agent | **Managed npm** `@upstash/context7-mcp` → absolute managed bin |
| Repo / wiki docs | [DeepWiki](https://deepwiki.com) MCP | Ask questions about GitHub repos / codebase docs | **HTTP remote MCP** (always available; no local binary) |
| Web search | [Exa](https://exa.ai) MCP | Web search / fetch / research | **HTTP remote MCP** `https://mcp.exa.ai/mcp` (always available) |
| Browser / E2E | [Playwright](https://playwright.dev) MCP | Drive a real browser for UI checks and flows | **Managed npm** `@playwright/mcp` → absolute managed bin; browsers may need separate install |

### Engram

- Real store is Engram’s own DB (typically under `~/.engram/`), not a second DB in `~/.aiskillgrid/memories/`.
- `aiskillgrid install` downloads the GitHub release binary into `~/.aiskillgrid/dependencies/bin/engram`.
- Merge Skillgrid-owned MCP entry (prefix `aiskillgrid-`) for selected clients when the binary is present; warn on download failure.
- When Engram is wired, install curated [engram/skills](https://github.com/Gentleman-Programming/engram/tree/main/skills) (always include `memory-protocol`; **exclude** `sdd-flow`) per [05-skills.md](05-skills.md) — orchestration is slice B.

### GitNexus

- Primary codebase indexer for v1 (structural / impact).
- Installed via managed npm (`npm install -g --prefix ~/.aiskillgrid/npm gitnexus`); MCP uses absolute path to managed `gitnexus` bin.
- **CocoIndex (`ccc`) is deferred** — optional later; not a v1 peer (avoids two overlapping search tools).

### Context7

- MCP for current library and framework documentation (reduces hallucinated APIs).
- On install: `npm install -g --prefix ~/.aiskillgrid/npm @upstash/context7-mcp`; merge Skillgrid-owned MCP entry (e.g. `aiskillgrid-context7`) pointing at the managed package bin.
- Requires system `node` + `npm` on PATH for the install phase.
- Optional API key: document env var if Context7 requires one; do not store secrets in the hub repo.

### DeepWiki

- MCP for Q&A over GitHub repositories / generated wiki-style docs ([DeepWiki](https://deepwiki.com)).
- On install: merge Skillgrid-owned MCP entry (e.g. `aiskillgrid-deepwiki`) using the **HTTP remote URL** pinned in `packs/mcp/servers.json` — no local npm package required.
- Always considered available for MCP resolution (`http:deepwiki`); complements GitNexus: **GitNexus = structural code map**; **DeepWiki = natural-language docs/repo understanding**.

### Exa

- MCP for web search and page fetch ([Exa MCP](https://docs.exa.ai/reference/exa-mcp)).
- On install: merge Skillgrid-owned MCP entry (`aiskillgrid-exa`) with hosted URL `https://mcp.exa.ai/mcp` — no local npm package required.
- Always considered available for MCP resolution (`http:exa`). Free tier works anonymously with rate limits; for higher limits add an [API key](https://dashboard.exa.ai/api-keys) via client headers (`x-api-key` / `Authorization: Bearer`) — do not commit secrets to the hub.

### Playwright

- MCP for browser automation and UI verification ([Playwright MCP](https://github.com/microsoft/playwright-mcp)).
- On install: `npm install -g --prefix ~/.aiskillgrid/npm @playwright/mcp`; merge Skillgrid-owned MCP entry pointing at the managed package bin.
- Playwright browsers are **not** run automatically in v1; install warns that browsers may be required on first use.

## Explicitly later / out of default

- **OpenSpec** — deferred (Superpowers owns plans/specs/tasks)
- **Backlog.md** — deferred (Superpowers owns task checklists)
- CocoIndex Code (semantic search)
- Other browser stacks (agent-browser, etc.) as alternatives
- Security scanners
- Upstream skill orchestration (see [05-skills.md](05-skills.md))

## Runtime layout for tools

| Path | Purpose |
|------|---------|
| `~/.aiskillgrid/dependencies/bin/` | Native binaries (`engram`, `skills` from qntx/skill, …) |
| `~/.aiskillgrid/npm/` | Isolated npm prefix/cache + bins (`gitnexus`, MCP CLIs under `bin/`) |

## Status

`aiskillgrid status` prints managed npm path, system node availability, and which managed binaries are present:

```text
Managed npm: ~/.aiskillgrid/npm (node: yes|no)
Binaries: engram=yes|no skills=yes|no
NPM bins: gitnexus=yes|no context7=yes|no playwright=yes|no
```

Tooling spine (managed install + MCP resolve/merge) is **implemented** in the CLI. Skill orchestration and agent-file generation remain in [TODO.md](TODO.md).
