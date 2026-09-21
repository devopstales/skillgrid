# Graph View Upgrade — Briefing

Inspired by GitNexus (a code-graph explorer running the same `aiskillgrid` fixture).
Goal: make the `/mnemonic/graph` view feel fast, fluid, and navigable at ~5k nodes.

## Scope (4 items, items 3–4 optional)

1. **Progressive / animated layout + "Stop Layout" button**
   - Replace the synchronous `applyLayout` block (which freezes the tab for seconds)
     with an animated force layout that settles in real time.
   - A "Stop Layout" control lets the user interrupt mid-settle (the layout keeps the
     current positions — it's already a valid layout, just under-optimized).
   - Status: "Layout optimizing…" → "Ready" (mirrors GitNexus).
   - `tree` and `circles` layouts are instant (no animation) — only `force` animates.

2. **Fit-to-Screen + explicit Zoom In / Zoom Out buttons**
   - Add a small control cluster (top-right or bottom-right) with:
     - Zoom In, Zoom Out, Fit to Screen.
   - Fit to Screen computes the camera state that fits `sigma.getBBox()` into the
     viewport with padding, then `camera.animate(...)` to it.
   - These also auto-run after a layout completes / after a search-focus recenter.

3. **Darker background + violet accent theme** *(optional)*
   - Match GitNexus: bg `rgb(6,6,10)` (near-black), accent violet `#7c3aed`.
   - Update `--color-accent` in `index.css`, the `--sigma-background-color` in both
     `index.css` and `VectorGraph.tsx`, and the hardcoded `indigo-*` / `slate-*`
     usages in the 5 graph files + `communities.ts` + `DependencyGraph.tsx`.
   - Keep it scoped so the app shell (which uses the same `--color-accent` token)
     doesn't clash — the token change is intentional (violet everywhere).

4. **File-tree-as-explorer left panel + per-agent filters** *(optional)*
   - Left panel: a repo file tree (derived from the `files` table / graph node paths)
     with a "Search files…" box. Clicking a file highlights its node(s) in the graph.
   - Per-agent filters: toggle chips for `cursor` / `kilo` / `opencode` (which AI
     agents' config files to show). **Data gap**: no per-agent dimension exists on
     symbols/edges/files today — this needs either a backend field or a client-side
     path-prefix heuristic (files under `plugins/cursor`, `.cursor/`, `AGENTS.md`, etc.).
     See findings for the data-availability detail.

## Out of scope
- Swapping Sigma for a custom WebGL renderer (GitNexus's fluidity comes from a custom
  renderer; we keep Sigma — the animated layout + fit-to-screen get us most of the win).
- The "Nexus AI" / natural-language-query-over-graph feature (GitNexus's moat).
- Sequential / Radial layout algorithms (we keep force / tree / circles).

## Constraints
- No new top-level deps. `graphology-layout-forceatlas2`, `sigma`, `@react-sigma/core`
  are already present. `sigma/utils` exports `animateNodes` (returns a stop fn) — use it.
- Must stay TDD where it makes sense (layout animation stop behavior, fit-camera math,
  file-tree build, agent-filter predicate). Visual/animated parts verified by browser smoke.
- `verbatimModuleSyntax` is ON — type imports must use `type X` syntax.
- BDD zone rule: don't mix `.skillgrid/specs/**` with code in one commit.

## Acceptance (top-level)
- Force layout animates visibly over ~1–3 s and can be stopped mid-flight; graph stays
  interactive while settling.
- Fit-to-Screen frames the whole graph; zoom buttons work.
- (If done) Theme matches GitNexus (near-black bg, violet accent).
- (If done) File tree highlights nodes on click; agent chips filter the view.
