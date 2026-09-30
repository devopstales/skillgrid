# Spike: 002-dashboard-variants

**Type:** comparison (3 variants, same content, different visual system)
**Date:** 2026-09-30
**Status:** awaiting user selection

## Hypothesis

Given the skillgrid dashboard data surface (memory observations, sessions, code index,
pipeline/tracker state) and the ui-ux-pro-max dark design system, when three
radically different dark-mode visual systems render that same content, then the user
can select a direction (layout, color, typography, density) that the real build
(`skillgrid-ui`) will adopt.

Falsified if: none of the three is selectable, or the data surface cannot be
rendered believably in any of the three without contorting it.

## How to run

```bash
open .skillgrid/spikes/002-dashboard-variants/index.html
```

Single self-contained page, no build step, no network fonts (system font stacks),
mock data inlined. Variant switcher in the top bar: **A / B / C**.

## Variants

### A — Terminal Ops (dense mono console)
- Style: Dark Mode (OLED) from ui-ux-pro-max — near-black bg, slate surfaces,
  `#22C55E` run-green accent.
- Type: JetBrains Mono / IBM Plex Sans (system fallbacks: ui-monospace / system-ui).
- Layout: dense 12-col grid, 8px rhythm, sidebar + KPI strip + streaming-style
  sparklines + event feed.
- Chart guidance: bullet charts for KPI vs target, streaming area for index health.

### B — Glass Aurora (glassmorphic, ambient)
- Style: Glassmorphism from ui-ux-pro-max — frosted cards (backdrop-blur 16px,
  rgba white 0.06-0.12), aurora gradient field behind, violet/cyan/rose glows.
- Type: Inter-style system sans, larger scale, softer radii (16-20px).
- Layout: spacious card grid, floating top nav, hero stat band, glass panels.

### C — Linear Mono (minimal, spacious)
- Style: Minimal dark, single violet accent (matches current product accent
  #7c3aed), hairline borders, no gradients, no glass.
- Type: system sans + mono only for numerals/code.
- Layout: sticky left nav + single-column feed of "cards as rows", generous
  vertical rhythm, status dots instead of color blocks.

## Mock data (all variants, identical)

- Memory: 1,284 observations (612 project / 390 user / 282 global), 14 active
  sessions, 3 pinned.
- Code index: 3,905 symbols, 18,676 chunks, 768-dim nomic-embed-code, warm
  semantic leg ~28ms, last indexed 2m ago, 3 stale files.
- Pipeline: 17 completed changes, 1 in progress (2026-09-24-mnemonic-monitoring),
  0 blocked; QA gate PASS.
- Recent events feed: saves, search hits, index updates, web-cache fetches.
- 14-day observation trend (area/line).
- ADR count: 16, in force 13.

## Investigation trail

- [ ] Generated design system via ui-ux-pro-max (`--design-system --density 8`):
      pattern match was off-target (FAQ landing) but style = Dark Mode (OLED),
      colors, and typography (JetBrains Mono + IBM Plex Sans) are on-target and
      used for variant A.
- [ ] Style search `glassmorphism neon dark terminal` → 1 result (glassmorphism),
      used for variant B (backdrop-blur 10-20px, rgba white 10-30%, 1px light
      border, vibrant background).
- [ ] Chart search `dashboard status timeline trend` → line chart (trend),
      bullet (KPI vs target), streaming area (real-time). Applied per variant.
- [x] Built A, B, C with the same mock dataset; screenshot each at 1440x900.
- [x] Rendered all three in real Chrome (agent-browser, 1440x900); variant
      switcher verified programmatically (visibility + tag text flip per variant).
- [x] Pixel-sampled the three screenshots to prove distinct rendering (model has
      no image input — pixels are the evidence): A bg `(27,35,54)` = slate card
      on #0f172a; B bg `(25,60,76)` = cyan aurora bleed at (1300,500) and
      `(43,8,29)` rose bleed bottom-center — glassmorphism field confirmed;
      C bg `(13,13,18)` = neutral #0d0d12, no gradient.

## Verdict

_**PENDING** — set after user selects (or rejects all)._

| Variant | Screenshot | Notes |
|---------|------------|-------|
| A Terminal Ops | `shots/A-terminal-ops.png` | |
| B Glass Aurora | `shots/B-glass-aurora.png` | |
| C Linear Mono | `shots/C-linear-mono.png` | |

## Liftable Module

None expected (purely visual). If a winner's token table proves useful, the
`THEME` object in `index.html` is the only liftable artifact — a flat map of CSS
custom properties the real build would port into `skillgrid-ui/src/index.css`.

## Decision (filled after selection)

- Selected variant:
- Rejected variants + reason:
- Constraints carried into the build:
