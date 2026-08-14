# aiskillgrid — Skills Composition Design

Date: 2026-08-14  
Status: Approved (revised — Superpowers owns plans/specs/tasks)  
Extends: [2026-08-14-aiskillgrid-cli-design.md](./2026-08-14-aiskillgrid-cli-design.md)

## Goal

Compose upstream skill packs without forking them, with a single conflict map and **one** on-disk planning system (Superpowers) so agents are not confused by OpenSpec/Backlog peers.

## Decisions

| Topic | Decision |
|-------|----------|
| Strategy | Curate + compose; thin Skillgrid glue only |
| Manager | qntx/skill → `skills add` for non-plugin sources (not `npx skills`) |
| Superpowers | **Native plugin** for every selected agent (`install: plugin`); do not also `skills add` |
| Karpathy Guidelines | Skill + rules from [multica-ai/andrej-karpathy-skills](https://github.com/multica-ai/andrej-karpathy-skills) for every selected agent |
| Profile source of truth | `packs/skills/sources.yaml` + [05-skills.md](../../05-skills.md) |
| Packs | Superpowers, Karpathy Guidelines, mattpocock/skills, engram/skills (curated), gentle-ai/skills (curated) |
| Plans / specs / tasks | **Superpowers only** (`docs/superpowers/`, `.superpowers/`) |
| OpenSpec / Backlog.md | **Deferred** — not in default profile |
| Persistence | Engram = memory; Superpowers files = git-visible plans/specs/tasks |
| gentle-ai | Skills only (PR/RDD) — never `gentle-ai install`; no OpenSpec hybrid role |
| Process spine | Superpowers |
| Domain grilling | mattpocock (`grill-me`, `grill-with-docs`) |
| Memory | Engram `memory-protocol` when Engram wired |
| Branch/PR default | Engram `branch-pr` (exclude gentle-ai `branch-pr` from default) |
| Engram `sdd-flow` | Excluded from default (competes with Superpowers) |

## Non-goals

- Maintaining Skillgrid forks of upstream packs
- Running gentle-ai as a nested installer
- Shipping Engram product-UI skills in the default profile
- Dual planning systems (OpenSpec + Superpowers, or Backlog + Superpowers)

## Implementation

- Superpowers: `aiskillgrid-cli/plugins` clones `obra/superpowers` under `~/.aiskillgrid/dependencies/superpowers` and installs the harness plugin for each selected agent (OpenCode/Kilo `plugin` config, Cursor `~/.cursor/plugins/local/`, Copilot CLI marketplace).
- Karpathy: clones `multica-ai/andrej-karpathy-skills` and copies `karpathy-guidelines` skill + rule into each agent’s skills/rules dirs.
- Slice B (remaining): read `sources.yaml`, ensure `skills` binary, run curated `skills add` for non-plugin sources; skip `install: plugin`.
- Slice C: encode Superpowers paths + conflict map into project agent files.
