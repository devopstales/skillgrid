# TODO

## Clients

- [x] Kilo / Kilo Code (`kilo`)
- [x] OpenCode
- [x] Cursor
- [x] VS Code (Copilot)
- [ ] Claude Code
- [ ] pi
- [ ] Gemini CLI
- [ ] Antigravity
- [ ] Codex

## CLI

- [x] sync / install / status / version
- [x] Bootstrap scripts + Task release
- [ ] Homebrew formula/tap for `aiskillgrid`
- [ ] Nix flake (`flake.nix`: `nix run` / `nix profile install` package `aiskillgrid`)
- [ ] Uninstall / doctor
- [ ] Split Kilo vs Kilo Code if paths diverge

## Tools (agreed — see [04-tools.md](04-tools.md))

- [x] Engram: managed GitHub release binary → `dependencies/bin/engram` + merge MCP (warn on failure)
- [x] GitNexus: managed npm install + merge MCP
- [x] Pack MCP fragments in `packs/mcp/servers.json` + placeholder resolve / `requires` (no Backlog/OpenSpec entries)
- [x] Pack rules in `packs/rules/` (e.g. no-ai-commit-coauthors) and copy on install to all agents
- [x] Install git `commit-msg` hook from `packs/git-hooks/commit-msg` (strip AI co-authors; chain previous hook)
- [x] Skill composition profile: [`packs/skills/sources.yaml`](../packs/skills/sources.yaml) + [05-skills.md](05-skills.md) — Superpowers owns plans/specs/tasks; OpenSpec/Backlog deferred
- [x] Install Superpowers as **native plugin** for all selected agents (OpenCode/Kilo config, Cursor local plugin, Copilot CLI) — not only via `skills add`
- [x] Install [Karpathy Guidelines](https://github.com/multica-ai/andrej-karpathy-skills) skill + rules for all selected agents
- [x] Install [qntx/skill](https://github.com/qntx/skill) binary → `~/.aiskillgrid/dependencies/bin/skills`
- [ ] Orchestrate non-plugin sources from `packs/skills/sources.yaml` via that `skills` binary (not `npx skills`); skip `install: plugin` (Superpowers)
- [ ] Enforce composition policy: Superpowers for plans/specs/tasks; conflict map; never invoke `gentle-ai install`
- [ ] Map aiskillgrid client ids → skills agent targets; prefer non-interactive flags
- [ ] After mattpocock: document `/setup-matt-pocock-skills` once per repo; tracker = local files / GitHub
- [x] Create `~/.aiskillgrid/npm/` and ensure Skillgrid-managed npm prefix/cache on install
- [x] Wire MCP invocations to managed npm prefix/cache and absolute bin paths
- [x] Context7: pack + merge MCP (`aiskillgrid-context7`) via managed npx
- [x] DeepWiki: pack + merge MCP (`aiskillgrid-deepwiki`) via HTTP remote URL
- [x] Exa: pack + merge MCP (`aiskillgrid-exa`) via HTTP remote URL (`https://mcp.exa.ai/mcp`)
- [x] Playwright: pack + merge MCP (`aiskillgrid-playwright`) via managed npx; warn browsers may need install
- [x] `aiskillgrid status`: show managed npm + binary/npm tool presence
- [ ] Generate/refresh project `AGENT.md`, `CLAUDE.md`, `GEMINI.md` (merge Skillgrid blocks; Superpowers paths + conflict map; see [06-agent-files.md](06-agent-files.md))
- [ ] Pack instruction templates under `packs/instructions/` (stub: `skillgrid-block.md` done; wire on project install)
- [ ] (Later) CocoIndex optional pack
- [ ] (Deferred) OpenSpec / Backlog.md — only if Superpowers-on-disk proves insufficient
