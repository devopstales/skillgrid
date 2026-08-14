# Start Here

`aiskillgrid` is a cross-platform Go CLI that syncs **this** GitHub repo into a managed home directory and installs **skills + MCP** wiring into selected AI clients.

This project is independent. It is not a continuation of any earlier Skillgrid attempt.

## What it does

1. Installs as the `aiskillgrid` command (macOS, Linux, Windows).
2. Syncs the hub repo into `~/.aiskillgrid/tools/`.
3. Interactively installs skills and MCP config into the clients you choose.
4. Wires agreed tools (Engram, GitNexus, Context7, DeepWiki, Exa, Playwright) — [04-tools.md](04-tools.md).
5. Orchestrates upstream skill packs (Superpowers plugin + Karpathy rules/skill now; others via **qntx/skill** soon) — [05-skills.md](05-skills.md).
6. On project install, will generate/refresh `AGENT.md`, `CLAUDE.md`, `GEMINI.md` (backlog) — [06-agent-files.md](06-agent-files.md).

**Plans / specs / tasks:** Superpowers only (`docs/superpowers/`, `.superpowers/`). OpenSpec and Backlog.md are not in the default stack.

## v1 clients

- Kilo / Kilo Code (`kilo`)
- OpenCode (`opencode`)
- Cursor (`cursor`)
- VS Code (Copilot) (`copilot`)

Planned later: Claude Code, pi, Gemini CLI, Antigravity, Codex.

## v1 tools (agreed)

| Pillar | Tool | Integration |
|--------|------|-------------|
| Memory | Engram | Managed binary + MCP |
| Code map | GitNexus | Managed npm + MCP |
| Library docs | Context7 | Wire MCP (managed npx) |
| Repo docs Q&A | DeepWiki | Wire MCP (HTTP) |
| Web search | Exa | Wire MCP (HTTP) |
| Browser / E2E | Playwright | Wire MCP (managed npx) + browsers |

## Planned skills (agreed delivery)

| Pack | Delivery |
|------|----------|
| Superpowers | **Native plugin** per agent; owns process spine + plans/specs/tasks |
| Karpathy Guidelines | Skill + always-on rules for all selected agents |
| mattpocock/skills | Engineering (grill, triage, architecture); curated via `skills add` (slice B) |
| Gentleman-Programming/engram | Curated; `memory-protocol` when Engram wired |
| Gentleman-Programming/gentle-ai | Curated PR/RDD helpers only |

Compose via [`packs/skills/sources.yaml`](../packs/skills/sources.yaml); do not fork packs; do not run `gentle-ai install`.

Managed Node for MCP/other: `~/.aiskillgrid/npm/` + Skillgrid `npx` created on install.

Details: [04-tools.md](04-tools.md), [05-skills.md](05-skills.md).

## Reading order

1. [01-installation.md](01-installation.md) — binary + managed home
2. [02-usage.md](02-usage.md) — sync / install / status
3. [03-clients.md](03-clients.md) — client ids and paths
4. [04-tools.md](04-tools.md) — Engram, GitNexus, Context7, DeepWiki, Exa, Playwright
5. [05-skills.md](05-skills.md) — skill composition (Superpowers owns plans/specs/tasks)
6. [06-agent-files.md](06-agent-files.md) — AGENT.md / CLAUDE.md / GEMINI.md
7. [TODO.md](TODO.md) — backlog
8. [superpowers/specs/2026-08-14-aiskillgrid-cli-design.md](superpowers/specs/2026-08-14-aiskillgrid-cli-design.md) — design lock
9. [superpowers/specs/2026-08-14-aiskillgrid-skills-composition-design.md](superpowers/specs/2026-08-14-aiskillgrid-skills-composition-design.md) — skills composition
