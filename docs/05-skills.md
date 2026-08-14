# Skills

Skill packs that `aiskillgrid install` should get onto selected clients.

**Strategy:** curate + compose upstream packs; do **not** fork/vendor them into Skillgrid by default. Own only thin glue (router policy, tool wiring) and the install profile.

**Plans / specs / tasks:** **Superpowers only** (`docs/superpowers/`, `.superpowers/`). Do **not** install OpenSpec or Backlog.md in the default stack — they compete with Superpowers and confuse agents.

Machine-readable profile: [`packs/skills/sources.yaml`](../packs/skills/sources.yaml).

## Stub (present now)

- `packs/skills/aiskillgrid-stub/` — confirms wiring works (copied by aiskillgrid itself)

## Rules packs (installed for all agents)

Copied from `packs/rules/` on every `aiskillgrid install` into each selected client’s rules/instructions directory (see [03-clients.md](03-clients.md)):

- `no-ai-commit-coauthors.mdc` — never add Cursor or other AI agents as `Co-authored-by`

## Skillgrid-owned packs / templates

Copied or merged from this repo when relevant tools are wired (see [04-tools.md](04-tools.md)):

- Optional short usage skills for Engram / GitNexus (thin glue only)
- Project instruction block: [`packs/instructions/skillgrid-block.md`](../packs/instructions/skillgrid-block.md) — see [06-agent-files.md](06-agent-files.md)

## Upstream packs (default profile)

| Source | Role | Install |
|--------|------|---------|
| **[Superpowers](https://github.com/obra/superpowers)** | Process spine **and** plans/specs/tasks on disk | **Native plugin** for every selected agent (not `skills add`) |
| **[Karpathy Guidelines](https://github.com/multica-ai/andrej-karpathy-skills)** | Coding discipline (think / simplify / surgical / goal-driven) | **Skill + rules** copied into every selected agent |
| **[mattpocock/skills](https://github.com/mattpocock/skills)** | Engineering: grill, domain language, architecture survey, triage | `skills add` (curated) |
| **[Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram/tree/main/skills)** | Memory discipline + selected habits | `skills add` (curated) |
| **[Gentleman-Programming/gentle-ai](https://github.com/Gentleman-Programming/gentle-ai/tree/main/skills)** | PR / issue / RDD helpers only | `skills add` (curated) |

**Not in default profile:** OpenSpec skills, Backlog.md skills, Engram `sdd-flow` (would dual-track Superpowers).

### Superpowers = plugin (all agents)

`aiskillgrid install` installs [obra/superpowers](https://github.com/obra/superpowers) as a **harness plugin**, not only as skills:

| Agent | How Skillgrid installs the plugin |
|-------|-----------------------------------|
| **OpenCode** | Merge `"plugin": ["superpowers@git+https://github.com/obra/superpowers.git"]` into `opencode.json` |
| **Kilo** | `kilo plugin install …` when CLI is on PATH; else same plugin entry in `kilo.jsonc` |
| **Cursor** | Copy checkout into `~/.cursor/plugins/local/superpowers` (reload window); marketplace `/add-plugin superpowers` remains valid |
| **Copilot** | `copilot plugin marketplace add` + `copilot plugin install` when Copilot CLI is on PATH; otherwise warn with the manual commands |

Managed checkout: `~/.aiskillgrid/dependencies/superpowers`. Do **not** also run `skills add` for Superpowers when the plugin is installed (avoids dual registration).

### Karpathy Guidelines (all agents)

`aiskillgrid install` clones [multica-ai/andrej-karpathy-skills](https://github.com/multica-ai/andrej-karpathy-skills) into `~/.aiskillgrid/dependencies/andrej-karpathy-skills` and copies:

- Skill: `skills/karpathy-guidelines/` → each agent’s skills directory
- Rule: `.cursor/rules/karpathy-guidelines.mdc` → each agent’s rules/instructions directory (Copilot: `*.instructions.md`)

### Agreed skill manager: [qntx/skill](https://github.com/qntx/skill)

Use the **Rust single-binary** drop-in for the Vercel/`skills.sh` CLI — **not** `npx skills`.

On `aiskillgrid install`:

1. Ensure `skills` binary under `~/.aiskillgrid/dependencies/bin/skills` (download from qntx/skill releases if missing).
2. Prefer that binary on `PATH` for Skillgrid-driven installs (ahead of accidental `npx skills`).
3. For each source in `packs/skills/sources.yaml` with `install: skills` (or default), run `skills add <source> …` targeting selected clients. **Skip** `install: plugin` (Superpowers) and `install: rules_and_skill` (Karpathy — handled by the CLI).
4. Prefer **one** install path per pack (plugin **or** CLI rules/skill copy **or** `skills add` — not both).
5. After mattpocock skills: user should run `/setup-matt-pocock-skills` once per repo; prefer **local files / GitHub** as tracker (Superpowers docs), not Backlog.md.
6. Do **not** shell out to `gentle-ai install` — Skillgrid owns agent wiring; take gentle-ai **skills only**.

Do **not** vendor Superpowers, mattpocock, Engram, or gentle-ai skill trees into `packs/skills/` by default.

## Persistence (memory + Superpowers files)

| Backend | Role |
|---------|------|
| **Engram** | Cross-session memory, compaction survival (`memory-protocol`) |
| **Superpowers files** | Git-visible plans, specs, tasks (`docs/superpowers/`, `.superpowers/`) |

This replaces the earlier “hybrid = Engram + OpenSpec” idea. One planning system on disk; Engram for memory only.

## Composition / conflict map

One winner per intent in the **default** profile. Losers are not installed (or installed with model-invocation disabled) unless the user opts in.

| Intent | Winner | Do not dual-install as auto-invokers |
|--------|--------|--------------------------------------|
| Process spine (brainstorm → plan → implement → verify) | Superpowers | Competing routers without Skillgrid policy |
| Plans / specs / tasks on disk | Superpowers | OpenSpec, Backlog.md, Engram `sdd-flow` |
| Coding discipline (think / simplify / surgical / goal-driven) | Karpathy Guidelines | — |
| Domain grilling / CONTEXT.md / ADRs | mattpocock `grill-me` / `grill-with-docs` | Superpowers brainstorming for the same “grill the domain” job |
| Memory discipline | Engram `memory-protocol` (required when Engram MCP is wired) | — |
| TDD | Superpowers `test-driven-development` | Engram `testing-coverage` + mattpocock `/tdd` as simultaneous auto-invokers |
| Bug diagnosis | Superpowers `systematic-debugging` | mattpocock `diagnosing-bugs` as second default |
| Issue triage | mattpocock triage | Engram `backlog-triage` |
| Branch / PR | Engram `branch-pr` | gentle-ai `branch-pr` (upstream-repo-specific) |
| Engram product-UI skills | Opt-in only | Not in Skillgrid default profile |

Skillgrid may ship one thin **router** skill or AGENT.md block that restates this map.

## Curated allowlists (default)

Exact lists live in [`packs/skills/sources.yaml`](../packs/skills/sources.yaml). Intent:

- **Superpowers** — full process spine; owns plans/specs/tasks paths.
- **Karpathy** — always-on coding discipline via skill + rules (not optional).
- **mattpocock** — include `setup-matt-pocock-skills`, `grill-me`, `grill-with-docs`; prefer grill / architecture / triage over duplicate TDD/debug.
- **Engram** — always `memory-protocol` when Engram is enabled; exclude `sdd-flow` and product-UI skills from default.
- **gentle-ai** — curated PR/issue/RDD helpers only; no OpenSpec/hybrid SDD role; no `gentle-ai install`.

## Managed Node / npm (separate from skills)

Skill installs do **not** require Node. For **MCP servers and other npm-based tools**, aiskillgrid maintains an isolated prefix:

```text
~/.aiskillgrid/npm/          # npm prefix + cache (do not use user global npm)
  bin/                       # npx, and packages installed for Skillgrid
  lib/
  …
~/.aiskillgrid/dependencies/bin/   # native binaries (skills, engram, …)
```

On `aiskillgrid install`:

1. Ensure `~/.aiskillgrid/npm/` exists.
2. Ensure a usable **`npx`** for Skillgrid (via a managed Node toolchain or by wiring npm prefix so `npx` resolves under `~/.aiskillgrid/npm/bin`).
3. MCP `npx -y …` server configs should use that isolated prefix/cache when launched through Skillgrid-managed wiring.

Never require polluting the user’s global npm for Skillgrid MCP tools.

## Deferred (not default)

- OpenSpec (CLI, skills, `openspec/` scaffold)
- Backlog.md (skills, MCP, project scaffold)

Revisit only if Superpowers-on-disk proves insufficient for a concrete need — do not re-add both as peers of Superpowers without a new conflict map.

## Notes

- aiskillgrid still **copies** its own packs (no default symlinks).
- Updates for upstream skill packs: via the `skills` binary (`skills update` / equivalent), not via npx.
