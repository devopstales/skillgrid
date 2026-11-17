# Skillgrid host plugins — auto-load the using-skillgrid router

> **STATUS:** `draft` (2026-11-17)

## Goal

Make every supported agent host **deterministically auto-load the skillgrid router**
(`.agents/skills/using-skillgrid/SKILL.md`) at session start, the way
[obra/superpowers](https://github.com/obra/superpowers) auto-loads `using-superpowers`
(session-start injection of a router bootstrap, per-host adapters over one shared
skill tree). Today the router is only *reachable*: the AGENTS.md entry line (added
2026-11-17) tells the agent to read it, and instruction-following carries the weight.

## Why

- `using-skillgrid` is `disable-model-invocation: true` — hidden from every host's
  auto skill list (omp, pi, opencode, cursor all honor it; kilo likely ignores it).
  It runs only if the agent notices the AGENTS.md line — a soft trigger.
- The product already ships per-host plugin infrastructure
  (`plugins/opencode/`, `plugins/pi/mnemonic/`, `plugins/kilo/`, `plugins/cursor/`,
  `.cursor-plugin/`, `hooks/`, the `skillgrid install --agents …` Go installer) —
  it wires Mnemonic memory, not the SDD router. This change adds the router to the
  same pipeline.
- Superpowers proof: full-content bootstrap injection at session start makes the
  meta-skill load reliably on 10+ hosts; the pattern transfers directly.

## Requirements

- **R1** One shared bootstrap asset (single source of truth), compact (~15 lines),
  read live at session start by every host injector — updates propagate without
  reinstall.
- **R2** Per-host session-start injection for all five hosts: **omp, pi, opencode,
  Kilo Code, Cursor** — each using its native strongest mechanism (see blueprint §2).
- **R3** Injection is **project-conditional**: active only where
  `.skillgrid/config.yaml` exists (or a change-spec is in flight); silent elsewhere.
  Main agent only — subagents skip (mirrors `<SUBAGENT-STOP>`).
- **R4** Fail-open: injector failure never blocks session start; static fallback
  string when the bootstrap asset is missing.
- **R5** Distribution through the existing installer: `skillgrid install --agents
  omp,skill-pi,opencode,kilo,cursor` (extend the existing agent list) and
  `sync-repo` refreshes the bootstrap. No new distribution mechanism.
- **R6** Per-host tool-mapping hint in the bootstrap where a mapping file already
  exists (`using-skillgrid/references/{pi,opencode,…}-tools.md`).

## Out of scope

- Claude Code / Codex / Gemini adapters (not requested; superpowers-style hooks for
  those exist only as future work).
- Shipping the *full* router body per session (skill is 14KB; bootstrap is a pointer
  + core rules, full body loads on demand via each host's skill system).
- Fixing the stale `~/.agents/skills` mirror (existing installer sync path).
- Marketplace publishing (Cursor marketplace, Kilo marketplace, omp marketplace) —
  the local/installer path is the v1 distribution.

## Acceptance (one observable per host)

In a skillgrid project, a fresh session of each host shows the agent read the router
before first action (observable: "Using [skill] to [purpose]" announcement or the
router's start gates in the transcript); in a non-skillgit directory, no injection.

## Research (2026-11-17, four parallel subagents)

- `/tmp/skillgrid-plugin-research/superpowers.md` — superpowers v7.0.2 clone: layout,
  per-host injection mechanisms, install/update, adapters.
- `/tmp/skillgrid-plugin-research/omp-local.md` — omp v18.16.0: skill discovery,
  `disable-model-invocation` honored, extension system, live AGENTS.md proof.
- `/tmp/skillgrid-plugin-research/pi-opencode.md` — pi + opencode V2: events,
  injection APIs, local plugin precedent.
- `/tmp/skillgrid-plugin-research/ide-rules.md` — Cursor + Kilo: rules, plugins,
  native skills; local `~/.config/kilo/plugin/superpowers.js` as proven pattern.
