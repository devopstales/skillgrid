---
type: Entity
title: "pi-worktree plugin"
---

# pi-worktree plugin

The `@devopstales/pi-worktree` Pi plugin — the tier-1 harness
implementation of the skillgrid worktree standard.

- **Build location:** `plugins/pi/pi-worktree` inside this repo
  (programmer override, 2026-11-22; the briefing originally said a new
  sibling npm repo — superseded).
- **Change spec:** `.skillgrid/change-specs/2026-11-22-pi-worktree-plugin/`
- **Contract:** `.agents/skills/_shared/work-unit-commits/README.md`
  (scratch checkpoints §, tooling tiers, worktree removal §, lifecycle §)
- **Upstreams studied:** `mavam/pi-worktrunk` v0.13.2 (wt runner,
  directive protocol, linked sessions, markers),
  `prateekmedia/checkpoint-pi` v1.0.5 (checkpoint shape, safe restore),
  `arpagon/pi-rewind` (temp-index capture, dedup, prune, two-layer split)
- **Requirements:** R1–R13 in the briefing (scratch capture, dedup,
  restore with atomic pair, bounded refs, wt runs with session movement,
  no-replay recovery, standard-root verification, allow-list at creation,
  merge defaults, approval passthrough, markers, Pi-free testable core,
  contract-drift guard)
- **Floor:** Pi ≥ 0.99.1; `wt` on PATH for the worktrunk half; git; zero
  other runtime deps.
