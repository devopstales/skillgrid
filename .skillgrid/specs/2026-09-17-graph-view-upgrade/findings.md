# Graph View Upgrade — Findings

Research notes backing the spec (see `tasks.md`).

## GitNexus (the reference, at localhost:4747)
- Custom **WebGL2** renderer (7 interleaved canvases, 3× webgl2 + 2d label layers). Not Sigma.
- Body bg `rgb(6,6,10)`, text `rgb(228,228,237)`, font **Outfit**.
- Accent violet `rgb(124,58,237)` = `#7c3aed` on active tabs/buttons.
- Layouts as **tabs**: Force Graph / Sequential Layout / Radial Layout.
- **"Layout optimizing…" → "Ready"** with a **"Stop Layout"** button = progressive,
  interruptible layout (the key UX we want).
- **Zoom In / Out / Fit to Screen** buttons + "Turn off all highlights".
- Left **Explorer** panel: repo file tree + "Search files…" + agent filter chips
  (agents/cursor/kilo/opencode).
- **Nexus AI** + **Query** = NL questions over the graph (out of scope for us).
- Rendered 5477 nodes / 15124 edges on the same `aiskillgrid` fixture.

## Current skillgrid graph (before change)
- `VectorGraph.tsx` (318 lines): SigmaContainer + Loader + HoverEffects + SearchHighlight
  + HoverTooltip + exportHtml + LoadingState. Controls: SearchPanel (top-left),
  LayoutSwitcher + DepthSlider + Export HTML (top-right), Legend (bottom-left), status (bottom-right).
- `layouts.ts`: `forceLayout` calls `forceAtlas2.assign(g, {iterations:300, settings:{...}})`
  **synchronously** (blocks main thread). `tree`/`circles` are instant.
- `useMnemonicGraph`: load effect does `applyLayout(graph, layout)` synchronously after a
  one-frame `requestAnimationFrame` yield. `recenter` refetches a depth-limited neighborhood.
- Sigma v3 **has no** `getBoundaries`/`zoomTo`/`applyState` (those are v2). Use
  `sigma.getBBox()` + `camera.animate(...)`. `useCamera` exposes `zoomIn`/`zoomOut`/`reset`/`goto`/`gotoNode`.
- `sigma/utils` exports `animateNodes(graph, targets, opts)` → returns a **stop fn** (candidate
  for the animated layout, though it tweens to fixed targets rather than a live simulation).

## Backend data availability
- `GET /mnemonic/graph/data` returns `{ nodes, edges, truncated, degraded, files?, project }`.
  **`files` is only populated when `degraded` (0 edges)** — a flat `[]string` of distinct paths,
  no tree, no agent info. So for the explorer we build the code tree **client-side** from
  `raw.nodes[].path` (always present), not from `files`.
- `GET /mnemonic/files/tree` is the **OpenViking memory tree** (from `observations.topic_key`),
  NOT code files — don't reuse it for the code explorer.
- **No per-agent dimension** exists on symbols/edges/files. Agents (`cursor`/`kilo`/`opencode`)
  exist only in the CLI install/setup layer. The `observations.source` column is free-text
  provenance, not an agent tag. → Phase 4 uses a **client-side path-prefix heuristic**
  (`agentForPath`); exact per-agent filtering would be a separate backend task.

## Theme
- `index.css` `@theme`: `--color-bg:#0a0a0b`, `--color-card:#141416`, `--color-edge:#27272a`,
  `--color-accent:#6366f1` (indigo-500).
- Graph feature hardcodes `slate-*` surfaces + `indigo-*` accents (5 files) — these do NOT
  follow the `--color-accent` token, so a violet re-theme must touch them explicitly.
- `communities.ts` palette[0] = `#6366f1`; `DependencyGraph.tsx` uses `#6366f1` too.

## Tooling
- No vitest (no `test` script, not in node_modules). Phase 1.5 adds it (dev dep only) for the
  layout/camera/explorer unit tests.
