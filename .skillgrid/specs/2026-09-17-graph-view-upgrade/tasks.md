# Graph View Upgrade — Tasks

Spec: `.skillgrid/specs/2026-09-17-graph-view-upgrade/`
Branch: `release/2`
UI: `skillgrid-ui/src/features/mnemonic/graph/` · Backend: `skillgrid-cli/internal/mnemonic/http/`

Legend: `[RED]` = TDD (test first) · `[AFK]` = implementation, verify by build/browser.

---

## 1-progressive-layout

### Goal

The force layout stops blocking the main thread. It animates in real time
("Layout optimizing…" → "Ready") and a "Stop Layout" button interrupts it
mid-settle, keeping the current (already-valid) positions.

### Out of scope
- Animated `tree` / `circles` (they're instant; leave synchronous).
- A custom WebGL renderer (stay on Sigma).

### Interfaces
- Consumes: `forceAtlas2` (already imported), `sigma` instance + camera (via `useSigma`/`useCamera`), `sigma/utils` `animateNodes` (returns a stop fn) OR a hand-rolled rAF ForceAtlas2 step loop.
- Produces: an animated force layout with an interrupt handle; a `layoutStatus` (`'idle' | 'optimizing' | 'ready'`) on the graph hook; a `stopLayout()` action.

### Design notes
- `graphology-layout-forceatlas2`'s `.assign()` runs to completion (blocking). For
  progressive motion we drive it ourselves: seed x/y, then on a `requestAnimationFrame`
  loop call the ForceAtlas2 **single-step** (or a small batch of iterations) and write
  positions back onto the graph each frame; Sigma re-renders automatically.
- Two viable approaches — pick one in design:
  - (a) **Hand-rolled rAF loop**: each frame run `forceAtlas2` for `k` iterations (e.g. 3),
    `g.setNodeAttribute(n,'x'|'y',...)` for all nodes, continue until `maxIterations` or
    the user calls stop. Stop = cancel the rAF + set status `ready`.
  - (b) `sigma/utils` `animateNodes(graph, targets, {duration})` → returns a stop fn;
    but `animateNodes` tweens toward fixed target positions, not a live simulation.
    It's better for "animate to computed layout" than "live simulation". Prefer (a) for a
    true settling simulation; (b) is the fallback if (a) is fiddly.
- `tree`/`circles` keep calling `applyLayout` synchronously (no animation, no stop).
- The hook's load effect currently does `applyLayout(graph, layout)` synchronously.
  For `force`, instead: build the graph, seed positions, then kick off the rAF loop and
  return its stop handle. Expose `layoutStatus` + `stopLayout` on the hook.
- Auto-stop when the layout "settles": track per-iteration displacement; if max
  displacement < epsilon for N frames, treat as settled → status `ready`.
- After the layout finishes (settled or stopped), trigger fit-to-screen (Phase 2).

### Tasks

- [x] 1.1 `[RED]` `forceLayoutAnimated(g, opts)` runs a rAF ForceAtlas2 simulation and returns `{ stop, done }`
  - [ ] 1.1.a Write failing test: a small graph (say 30 nodes / 50 edges) → `forceLayoutAnimated` returns a `done` promise; before `done` resolves, node positions change across frames; `stop()` resolves `done` early with finite (non-NaN) x/y on every node.
  - [ ] 1.1.b Run to confirm fail — `Run: cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/layouts.test.ts` (add vitest if absent — see 1.5) — Expected: FAIL
  - [ ] 1.1.c Minimal implementation: rAF loop stepping ForceAtlas2 a few iterations/frame, writing x/y back; `stop()` cancels rAF + resolves `done`; auto-settle on displacement epsilon.
  - [ ] 1.1.d Run to confirm pass — Expected: PASS
  - [ ] 1.1.e Commit — `feat(graph): animated force layout with stop handle`
- [x] 1.2 `[AFK]` Wire the hook: force uses the animated loop; expose `layoutStatus` + `stopLayout()`
  - `useMnemonicGraph`: for `force`, build graph + seed, start `forceLayoutAnimated`, store the stop handle in a ref; expose `layoutStatus` (`'optimizing'` while running, `'ready'` when done) and `stopLayout()`. `tree`/`circles` keep the synchronous `applyLayout` and set status `ready` immediately.
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [x] 1.3 `[AFK]` "Stop Layout" control + status readout in VectorGraph
  - A button in the top-right cluster: visible only while `layoutStatus === 'optimizing'` and `layout === 'force'`; label "Stop Layout"; calls `stopLayout()`.
  - Status text (bottom-right, next to the node/edge count): "Layout optimizing…" while optimizing, "Ready" when done.
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [x] 1.4 `[AFK]` Browser smoke: force layout animates + stop works
  - Serve the fixture, open `/mnemonic/graph`, confirm the graph visibly settles over ~1–3 s (positions change), "Layout optimizing…" shows then "Ready"; click "Stop Layout" mid-settle and confirm the graph freezes in place (no NaN, still interactive).
  - `Run: skillgrid serve -dir ~/.skillgrid/mnemonic -port 8131` + browser — Expected: PASS
- [x] 1.5 `[AFK]` (only if vitest absent) add a minimal vitest setup for graph unit tests
  - Add `vitest` dev dep + a `test` script; keep it light (only what 1.1 needs). If vitest is already present, skip.
  - `Run: cd skillgrid-ui && npx vitest run` — Expected: PASS

### Verification

Verdict: `PASS`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Unit (animated layout + stop + seed) | `cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/layouts.test.ts` | PASS | PASS | 7 tests: stop resolves done early w/ finite pos, motion across frames, self-settle, idempotent stop, empty fast-path, seed non-degenerate + deterministic |
| Type-check | `cd skillgrid-ui && npx tsc -b --noEmit` | PASS | PASS | 0 errors |
| Build | `cd skillgrid-ui && npm run build` | PASS | PASS | clean, initial ~814 kB |
| Browser smoke (animate + stop) | `skillgrid serve -dir ~/.skillgrid/mnemonic -port 8145` + browser | PASS | PASS | default: "Layout optimizing…" + extent growing 654, not settled, 4379 nodes; `maxiter=500000`: optimizing seen, Stop present, click froze frame+positions |

Key fix: `forceAtlas2.assign` produces ZERO movement on a graph seeded at the
origin (symmetric forces cancel) — added `seedPositions` (deterministic golden-angle
ring) before the sim. Settle heuristic is relative (`settleRel=1e-3` of graph
extent), not absolute, so it fires on large graphs.

---

## 2-fit-zoom-controls

### Goal

Explicit Fit-to-Screen + Zoom In / Zoom Out buttons, wired to the Sigma camera.
Fit-to-Screen also auto-runs after a layout completes and after a search-focus recenter.

### Out of scope
- Pan/rotation controls (Sigma default wheel/drag stays).

### Interfaces
- Consumes: `useSigma` (→ `sigma.getBBox()`, `sigma.getDimensions()`, `sigma.getCamera()`), `useCamera` (→ `zoomIn`, `zoomOut`, `goto`).
- Produces: a `ViewControls` component (Zoom In / Zoom Out / Fit to Screen); a `fitToScreen()` helper; auto-fit on layout-ready + recenter.

### Design notes
- `fitToScreen(padding = 0.1)`:
  ```
  const box = sigma.getBBox()            // { x:[min,max], y:[min,max] }
  const w = (box.x[1]-box.x[0]) || 1, h = (box.y[1]-box.y[0]) || 1
  const dims = sigma.getDimensions()     // { width, height }
  const ratio = Math.max(dims.width*(1-2*padding)/w, dims.height*(1-2*padding)/h)
  camera.animate({ x:(box.x[0]+box.x[1])/2, y:(box.y[0]+box.y[1])/2, angle:0, ratio }, { duration:400, easing:'quadraticInOut' })
  ```
  (Sigma v3 has no `getBoundaries`/`zoomTo` — those are v2. Use `getBBox` + `camera.animate`.)
- The control cluster must render **inside** `SigmaContainer` (it needs the Sigma context) —
  e.g. a new child component `<ViewControls/>` next to `<Loader/>`, OR lift the camera via
  `useSigma` in a child and render the buttons there.
- Zoom In/Out: `zoomIn()` / `zoomOut()` from `useCamera` (animated by default).
- Auto-fit triggers: (a) when `layoutStatus` transitions `optimizing` → `ready`
  (Phase 1), (b) when `focus` (search recenter) changes. Implement as a `useEffect` on
  those signals calling `fitToScreen()`. Guard so it doesn't refit on every hover.
- The existing "Export HTML" button + LayoutSwitcher + DepthSlider stay; add the new
  cluster without breaking the current layout.

### Tasks

- [ ] 2.1 `[RED]` `fitCameraToBBox(box, dims, padding)` returns the target `CameraState`
  - [ ] 2.1.a Write failing test: given a bbox + viewport dims + padding → returns `{x,y,angle,ratio}` where `ratio` is `max(vw*(1-2p)/w, vh*(1-2p)/h)` and x/y are the bbox center. Edge: zero-area bbox → ratio uses `|| 1`.
  - [ ] 2.1.b Run to confirm fail — `Run: cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/camera.test.ts` — Expected: FAIL
  - [ ] 2.1.c Minimal implementation: pure function (no Sigma import) computing the camera state.
  - [ ] 2.1.d Run to confirm pass — Expected: PASS
  - [ ] 2.1.e Commit — `feat(graph): fit-camera math`
- [ ] 2.2 `[AFK]` `ViewControls` component (Zoom In / Zoom Out / Fit to Screen) inside SigmaContainer
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 2.3 `[AFK]` Auto-fit on layout-ready + search recenter
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 2.4 `[AFK]` Browser smoke: fit + zoom work, auto-fit after layout
  - `Run: skillgrid serve` + browser — Expected: PASS (whole graph framed after load; zoom buttons zoom; recenter refits)

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Unit (fit-camera math) | `cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/camera.test.ts` | PASS | — | |
| Type-check | `cd skillgrid-ui && npx tsc -b --noEmit` | PASS | — | |
| Build | `cd skillgrid-ui && npm run build` | PASS | — | |
| Browser smoke (fit/zoom/auto-fit) | `skillgrid serve` + browser | PASS | — | |

---

## 3-theme-violet *(optional)*

### Goal

Match GitNexus: near-black bg `rgb(6,6,10)`, violet accent `#7c3aed`. Scoped so the
graph feature and the app shell both read as the same dark/violet theme.

### Out of scope
- Changing the `Outfit` font (keep Inter — font swap is a separate decision).
- Per-view theming (one theme app-wide).

### Interfaces
- Consumes: `index.css` `@theme` tokens; the graph files' hardcoded `indigo-*`/`slate-*` classes.
- Produces: updated `--color-accent` (violet), `--color-bg` (near-black), `--sigma-background-color` (near-black), and the graph files recolored to violet.

### Design notes
- `--color-accent: #7c3aed` (violet-600) in `index.css` — this drives `bg-accent`,
  `text-accent`, `border-accent` app-wide (intentional: violet everywhere).
- `--color-bg: #06060a` (GitNexus body). `--sigma-background-color: #06060a` in both
  `index.css` (the `.mnemonic-sigma.react-sigma` rule) and `VectorGraph.tsx` inline style.
- Recolor the graph feature's hardcoded `indigo-*` → violet (`violet-*`) and the
  `slate-*` surfaces → the near-black/zinc palette:
  - `VectorGraph.tsx` (spinner `border-t-indigo-400`→`violet-400`, bar `bg-indigo-400`→`violet-400`,
    `bg-slate-950`→`bg-bg` or `#06060a`, control borders `border-slate-700`→`border-edge`).
  - `controls/LayoutSwitcher.tsx` (`bg-indigo-500`→`bg-accent`), `SearchPanel.tsx`
    (`focus:border-indigo-500`→`focus:border-accent`), `DepthSlider.tsx`
    (`accent-indigo-500`→`accent-violet-500`, `text-indigo-400`→`text-violet-400`).
  - `communities.ts` first palette entry `#6366f1`→`#7c3aed`.
  - `exportHtml` default node color `#6366f1`→`#7c3aed`, bg `#0f172a`→`#06060a`.
  - `DependencyGraph.tsx` `#6366f1`→`#7c3aed` (keep consistent).
- Verify the app shell (AppLayout, ProjectSelector, nav) still looks right with the
  new accent — the token change is app-wide by design.

### Tasks

- [ ] 3.1 `[AFK]` Update `index.css` tokens (accent violet, bg near-black) + sigma bg
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 3.2 `[AFK]` Recolor the 5 graph files + communities.ts + DependencyGraph.tsx
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 3.3 `[AFK]` Browser smoke: theme matches GitNexus (near-black + violet), no contrast regressions
  - `Run: skillgrid serve` + browser — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Type-check | `cd skillgrid-ui && npx tsc -b --noEmit` | PASS | — | |
| Build | `cd skillgrid-ui && npm run build` | PASS | — | |
| Browser smoke (theme) | `skillgrid serve` + browser | PASS | — | |

---

## 4-file-explorer-agents *(optional)*

### Goal

A left "Explorer" panel with a repo file tree + "Search files…" box; clicking a file
highlights its node(s). Plus per-agent filter chips (cursor / kilo / opencode).

### Out of scope
- A backend per-agent dimension (no `agent` column exists on symbols/edges/files today).
  We use a **client-side path-prefix heuristic** for the agent chips (see design).
- Replacing the existing `SearchPanel` (node search) — the explorer is a separate,
  file-oriented panel.

### Interfaces
- Consumes: graph node `path` fields (already in `raw.nodes`), optionally the
  `/mnemonic/files/tree` endpoint (but that's the OpenViking memory tree, NOT code files —
  build the code tree client-side from `raw.nodes[].path`).
- Produces: an `ExplorerPanel` component (file tree + search), an `agentFilter` state
  (`Set<string>` of active agents), and node-highlight wiring (click file → highlight
  matching nodes via the existing `searchCenter`/`SearchHighlight` mechanism).

### Design notes
- **Code file tree (client-side):** build a tree from the distinct `path` values in
  `raw.nodes` (split on `/`). A pure helper `buildPathTree(paths: string[]): TreeNode` —
  unit-testable. Cap depth (e.g. 6) and collapse by default (expand on click).
- **File → node highlight:** on file click, find nodes whose `path` equals (or is under)
  the clicked path; if exactly one, set it as `searchCenter` (reuses `SearchHighlight`).
  If many, set a "path filter" that dims non-matching nodes (extend the reducer).
- **Agent chips (client-side heuristic):** a file is "cursor" if its path matches
  `(^|/)\\.cursor/` or ends with `.cursorrules`; "kilo" if `(^|/)kilo/` or under
  `plugins/kilo`; "opencode" if `(^|/)opencode/` or under `plugins/opencode`; plus
  `AGENTS.md`, `config.d/`, `mcp.yaml`, `indexing.yaml`, `tools.yaml`, `skills-lock.json`
  as "all agents" config files. A pure predicate `agentForPath(p): string | null` —
  unit-testable. Toggling a chip filters which nodes are visible (dim/hide the rest).
- **Data caveat (findings):** there's NO per-agent data on the backend. The heuristic is
  an approximation. If exact per-agent filtering is later needed, that's a backend task
  (e.g. an `agent` column on observations/files) — out of scope here.
- The explorer panel is collapsible (a "Collapse Panel" button) to give the graph full
  width. Render it as a left column inside the graph page (the page is currently
  `h-[calc(100vh-4rem)]` with just `<VectorGraph/>` — restructure to a flex row:
  explorer + graph).

### Tasks

- [ ] 4.1 `[RED]` `buildPathTree(paths)` builds a nested tree from file paths
  - [ ] 4.1.a Write failing test: paths `['a/b/c.ts','a/b/d.ts','e.ts']` → tree with `a` → `b` → [c.ts, d.ts] and `e.ts` as a leaf; dedupes; caps depth.
  - [ ] 4.1.b Run to confirm fail — `Run: cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/explorer.test.ts` — Expected: FAIL
  - [ ] 4.1.c Minimal implementation.
  - [ ] 4.1.d Run to confirm pass — Expected: PASS
  - [ ] 4.1.e Commit — `feat(graph): client-side code file tree`
- [ ] 4.2 `[RED]` `agentForPath(p)` returns the agent a config file belongs to (or null)
  - [ ] 4.2.a Write failing test: `.cursor/foo`→`cursor`, `plugins/kilo/mnemonic.ts`→`kilo`, `.opencode/agents/x`→`opencode`, `AGENTS.md`→`all`, `src/main.go`→`null`.
  - [ ] 4.2.b Run to confirm fail — Expected: FAIL
  - [ ] 4.2.c Minimal implementation (regex/path checks).
  - [ ] 4.2.d Run to confirm pass — Expected: PASS
  - [ ] 4.2.e Commit — `feat(graph): per-agent path heuristic`
- [ ] 4.3 `[AFK]` `ExplorerPanel` component (file tree + "Search files…" + agent chips + collapse)
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 4.4 `[AFK]` Wire file-click → node highlight + agent-chip filtering into VectorGraph
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 4.5 `[AFK]` Restructure `GraphPage` to a flex row (explorer + graph), collapsible
  - `Run: cd skillgrid-ui && npx tsc -b --noEmit` — Expected: PASS
- [ ] 4.6 `[AFK]` Browser smoke: explorer highlights nodes, agent chips filter, collapse works
  - `Run: skillgrid serve` + browser — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Unit (file tree + agent predicate) | `cd skillgrid-ui && npx vitest run src/features/mnemonic/graph/explorer.test.ts` | PASS | — | |
| Type-check | `cd skillgrid-ui && npx tsc -b --noEmit` | PASS | — | |
| Build | `cd skillgrid-ui && npm run build` | PASS | — | |
| Browser smoke (explorer + agents) | `skillgrid serve` + browser | PASS | — | |

---

## Global Constraints
- No new top-level deps (Phase 1.5 may add `vitest` as a dev dep only).
- `verbatimModuleSyntax` ON — `type X` imports.
- BDD zone rule: spec-ledger edits committed separately from code.
- Keep the graph interactive during the animated layout (no modal block).

## Commit plan (per phase, code + artifacts together; spec-ledger separate)
1. `feat(graph): progressive force layout + stop` (Phase 1)
2. `feat(graph): fit-to-screen + zoom controls` (Phase 2)
3. `style(graph): near-black + violet theme` (Phase 3, optional)
4. `feat(graph): file explorer + per-agent filters` (Phase 4, optional)
