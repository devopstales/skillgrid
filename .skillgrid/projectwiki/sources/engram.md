---
type: SourceSummary
title: "engram — predecessor study (product shape)"
status: stable
sources:
  - "local: ../engram (Go, single binary, MCP + daemon + cloud)"
---

# engram — predecessor study (product shape)

`../engram`: "persistent memory for AI coding agents" — single Go binary,
zero runtime dependencies, MCP server + daemon + cloud sync + TUI, setup
plugins for 14 agent harnesses, Obsidian export, doctor/selftest, per-project
stores. Studied 2026-11-23 for the OSS *product shape* (not the data model —
that comes from [[sources/skillgrid-mnemonic-v2]]).

## What it proves (inherited by the new design)

- **Single binary, zero deps, agent-agnostic** — one `engram` binary serves CLI
  and MCP; setup plugins wire any agent. The new `mnemonic` binary follows the
  same shape (per-tool: `mnemonic memory mcp` —
  [[decisions/0010-multi-tool-binary]]).
- **Doctor as a first-class command** — whole-system health in one report;
  the new design's `mnemonic doctor` (top-level, cross-tool) inherits this.
- **Per-project store files** — portable brain per project; confirmed by the
  new design's [[decisions/0003-per-project-sqlite-file]].
- **Local-or-cloud is a split, not a blend** — engram's cloud tier (sync,
  leases, dashboard) is exactly the weight the new design drops.

## What to cut

- Cloud sync (inbox/lease/autosync), daemon, TUI, 14-agent setup-plugin
  surface, identity file. v1 of [[entities/mnemonic]] is local-only, no
  daemon, two adapters (CLI + MCP).

## Related

- Supersedes: none (study, not a decision).
- Consumed by: [[syntheses/mnemonic-core]], [[entities/mnemonic]].
- Depends on: [[decisions/0003-per-project-sqlite-file]],
  [[decisions/0010-multi-tool-binary]].
