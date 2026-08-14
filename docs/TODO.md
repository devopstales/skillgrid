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
- [x] OpenSpec: managed npm `@fission-ai/openspec` (skills + project `openspec/` scaffold — later)
- [x] Backlog.md: managed npm `backlog.md` + wire MCP (skills + project `backlog` scaffold — later)
- [x] Pack MCP fragments in `packs/mcp/servers.json` + placeholder resolve / `requires`
- [x] Skill composition profile: [`packs/skills/sources.yaml`](../packs/skills/sources.yaml) + [05-skills.md](05-skills.md) (Superpowers, mattpocock, OpenSpec, Engram, gentle-ai; hybrid SDD)
- [x] Install [qntx/skill](https://github.com/qntx/skill) binary → `~/.aiskillgrid/dependencies/bin/skills`
- [ ] Orchestrate sources from `packs/skills/sources.yaml` via that `skills` binary (not `npx skills`)
- [ ] Enforce composition policy: hybrid SDD when Engram+OpenSpec present; conflict map; never invoke `gentle-ai install`
- [ ] Map aiskillgrid client ids → skills agent targets; prefer non-interactive flags
- [ ] After mattpocock: document `/setup-matt-pocock-skills` once per repo; prefer Backlog.md tracker when enabled
- [x] Create `~/.aiskillgrid/npm/` and ensure Skillgrid-managed npm prefix/cache on install
- [x] Wire MCP invocations to managed npm prefix/cache and absolute bin paths
- [x] Context7: pack + merge MCP (`aiskillgrid-context7`) via managed npx
- [x] DeepWiki: pack + merge MCP (`aiskillgrid-deepwiki`) via HTTP remote URL
- [x] Playwright: pack + merge MCP (`aiskillgrid-playwright`) via managed npx; warn browsers may need install
- [x] `aiskillgrid status`: show managed npm + binary/npm tool presence
- [ ] Generate/refresh project `AGENT.md`, `CLAUDE.md`, `GEMINI.md` (merge Skillgrid blocks; include hybrid SDD + conflict map; see [06-agent-files.md](06-agent-files.md))
- [ ] Pack instruction templates under `packs/instructions/` (stub: `skillgrid-block.md` done; wire on project install)
- [ ] (Later) CocoIndex optional pack
