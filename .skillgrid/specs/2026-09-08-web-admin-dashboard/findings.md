# Research: Web Dashboard Architecture (Vite + React SPA, Sigma.js, Mermaid)

**Date:** 2026-09-16
**Supersedes:** the 2026-09-08 vanilla-embedded research (Graphify / codegraph / GitNexus UI-pattern survey). That survey's *interaction* patterns (show-numbers table twin, per-widget error isolation, freshness banner, query-first, relation drill-down) are still adopted — see "Interaction patterns carried forward." This document covers the *architecture* decisions for the new Vite + React SPA.

## Constraint reminder

The dashboard is a **single binary**: the Vite + React build must compile into `ui/dist/` and be embedded with `//go:embed all:ui/dist`, served by `skillgrid serve` at `/` with no separate frontend process and no external CDN at runtime. Any library or pattern adopted must survive that envelope (or be explicitly CDN-only inside an exported artifact).

## Per-repo / per-tool findings

### abhigyanpatwari/GitNexus (knowledge-graph reference)

- **Main knowledge graph is Sigma.js + graphology (WebGL)**, not D3. Force/tree/circles layouts; nodes color-coded by community (Louvain/Leiden); click → selection info bar; "Blast Radius" + AI-citation highlighting; zoom in/out.
- **Converter pattern:** `knowledgeGraphToGraphology(nodes, edges)` maps the backend graph payload into a graphology `Graph` with node/edge attributes (label, type, size-by-degree, color, x/y seed). We mirror this as `mnemonicGraphToGraphology`.
- **Layouts + communities:** `graphology-layout-forceatlas2` (force), `graphology-layout` (tree/circles presets), `graphology-communities-louvain` (community coloring), `graphology-metrics` (degree/centrality for node sizing).
- **Depth filtering:** `filterGraphByDepth(graph, root, n)` for the N-hop neighborhood; a visible-labels filter for decluttering.
- **Mermaid / process diagrams:** a `ProcessFlowModal` renders `mermaid.render(chart)` client-side, sanitized, with the rendered SVG. Dark theme with an indigo accent. This is the reference for our `MermaidBlock`.
- **3-panel code layout:** file tree (left) / graph canvas (center) / context panel (right). We adopt the file-tree + graph split for the Mnemonic suite (OpenViking tree + vector graph).
- **Stack:** React 18 + TS + Vite + Tailwind v4 + Sigma.js + Web Workers + Comlink + tree-sitter WASM + LadybugDB WASM + transformers.js (WebGPU embeddings). **Heavy** — we take the graph + mermaid + layout ideas but *not* the in-browser WebGPU embeddings (embeddings stay in the Go index) or the WASM symbol extraction (the Go `codeindex` already does it).

### Vite + `go:embed` (build-pipeline reference)

- Vite's `build.outDir` can point at any directory (including a sibling Go package's `ui/dist/`); `emptyOutDir: true` wipes stale output; content-hash `entryFileNames`/`chunkFileNames`/`assetFileNames` give deterministic, cacheable asset names that `go:embed` picks up.
- `//go:embed all:ui/dist` embeds the whole build (including dotfiles); `fs.Sub(uiFS, "ui/dist")` + `http.FileServer` serves it. An SPA fallback handler returns `index.html` for any non-API, non-asset route so client-side routing (TanStack Router) works on hard refresh.
- **Gotcha:** the SPA fallback must *not* shadow API routes — prefix-based routing (API prefixes → 404 JSON, known assets → file server, everything else → `index.html`) is required, otherwise a missing API route returns HTML and the client JSON parser breaks. This is a RED-test target (Phase 1).
- **Gotcha:** `go build` must work offline without Node — `ui/dist/` is a generated, gitignored artifact produced by `ui:build`; the committed `embed.go` references a path that must exist at build time, so CI always runs `ui:build` before `go build` (Taskfile `build:all` ordering).

### react-markdown + mermaid + DOMPurify (docs-rendering reference)

- **Pipeline:** `react-markdown` with `remark-frontmatter` + `remark-gfm` (tables, task lists), `rehype-slug` (heading anchors) + `rehype-autolink-headings` (TOC links) + `rehype-highlight` (syntax) + `rehype-sanitize` (XSS). A custom `code` renderer intercepts `language-mermaid` blocks → `MermaidBlock`.
- **MermaidBlock:** `mermaid.initialize({ startOnLoad: false, theme: 'dark', securityLevel: 'strict', themeVariables: { primaryColor: '#6366f1', background: '#0a0a0b' } })`; `mermaid.render(id, chart)` → `DOMPurify.sanitize(svg)` → `dangerouslySetInnerHTML`. Rendered SVGs cached by content hash (re-renders on identical content are free).
- **XSS surface:** untrusted markdown can carry `<script>` or a mermaid block with `click` handlers. `securityLevel: 'strict'` + `rehype-sanitize` + DOMPurify on the SVG is the defense — verified by a RED test (a `<script>` in the body must not appear in the rendered output).
- **Frontmatter:** `remark-frontmatter` + `gray-matter` parse YAML frontmatter into chips (status, author, updated-at) above the rendered body.

### OpenViking-style file tree (memfs visualization reference)

- The `internal/mnemonic/memfs` package exposes a virtual filesystem over `mnemonic://` URIs. The tree view mirrors it: each node shows an icon, name, a memory-count badge, and a last-indexed timestamp; the content panel shows **L0 (abstract) / L1 (overview) / L2 (details)** tiers (the layering that 013 governs).
- Breadcrumbs are `mnemonic://` URIs; scoped search (`find`) is constrained to the selected directory; inline memory annotations link to the memory content view.
- The Go side is read-only: `GET /mnemonic/files/tree?path=/` + `GET /mnemonic/files/content?uri=mnemonic://...`. No file mutation from the browser.

## Patterns adopted into the dashboard (steal now)

| # | Pattern | Source | Phase | Implementation |
|---|---------|--------|-------|----------------|
| 1 | **Sigma.js + graphology vector graph** | GitNexus | 4 | `VectorGraph` + `mnemonicGraphToGraphology`; force/tree/circles; Louvain coloring; depth filter; search highlight; HTML export |
| 2 | **Client-side Mermaid + DOMPurify** | GitNexus `ProcessFlowModal` | 3 | `MermaidBlock` (strict + sanitize + cache-by-hash); dark theme + indigo accent |
| 3 | **react-markdown GFM + anchors + highlighting** | GitNexus / standard | 3 | MarkdownView with remark/rehype plugins + frontmatter chips + on-this-page TOC |
| 4 | **OpenViking file tree (L0/L1/L2)** | GitNexus 3-panel | 5 | memfs tree + content tiers + `mnemonic://` breadcrumbs + scoped search |
| 5 | **Vite → `go:embed` SPA fallback** | Vite + Go embed | 1 | `build.outDir` → `ui/dist/`; `//go:embed all:ui/dist`; prefix-based SPA fallback preserving API + asset routes |
| 6 | **SSE for one-way real-time** | standard | 2 / 6 | `/tracker/stream` (fsnotify) + `/activity/stream` (event log); native `EventSource`; leak-free cleanup |
| 7 | **`TicketProvider` registry + `UnifiedTask`** | this change's design | 2 | one `/tracker/tasks` endpoint; provider adapters (backlog file-parse + gh/glab/jira shell-out); canonical `board` column |

## Interaction patterns carried forward (from the 2026-09-08 survey)

| # | Pattern | Source | Where it applies |
|---|---------|--------|------------------|
| A | **Per-widget error isolation** | codegraph | Every feature view owns its fetch + error render; a failed endpoint kills that widget, not the view/page (Phase 1 shell rule, enforced in 2–7) |
| B | **Freshness / completeness banner** | graphify + codegraph | Code/memory views: last-indexed + stale + re-index; tracker: provider + schemaVersion warning |
| C | **Query-first with suggested prompts** | graphify | Memory empty state shows recent session prompts as clickable suggestions, not a blank "no results" |
| D | **Relation drill-down with confidence badges** | graphify | Memory detail lists relations (EXTRACTED/INFERRED/AMBIGUOUS); graph edges carry relation type + weight |
| E | **Show-numbers table twin** | codegraph | Data widgets expose a raw JSON/table toggle (the table is the source of truth) — applied to memory grid, activity feed, git list |

## Patterns explicitly NOT adopted

| Pattern | Source | Reason |
|---------|--------|--------|
| In-browser WebGPU embeddings | GitNexus (transformers.js) | Embeddings are computed by the Go index (`resolveEmbedder`); no LLM/embedding inference in the browser |
| WASM symbol extraction (tree-sitter / LadybugDB) | GitNexus | The Go `codeindex` already does structural extraction; the graph renders its output |
| AI chat panel | GitNexus | Dashboard is operator-facing, not an agent client; the agent lives in the terminal |
| D3.js for the main graph | common alternative | Sigma.js is WebGL + has the layout/community libs; D3 is a hand-rolled simulation (slower at scale). D3 may appear only for static exports |
| Separate frontend process + CORS | common alternative | Breaks the single-binary property; the API has no CORS handling; `go:embed` removes the need |
| Password login UI | codegraph | 127.0.0.1 tool; the existing `SKILLGRID_HTTP_TOKEN` bearer gate is sufficient; login UI is friction without threat |
| WebSockets | common alternative | All real-time flows are one-way (server → client); SSE is simpler, proxy-friendly, auto-reconnect |
| Multi-tenant teams / role layers / LLM proxy | TencentDB | Product-scale layer; 009 renders single-operator governance (013), not the multi-tenant machinery |

## Soft-dependency degradation (forward-compat)

| Dependency | What it provides | Degraded behavior when absent |
|------------|------------------|-------------------------------|
| **005 / 008** (code-index edges + communities) | the graph's semantic/structural edge set + Louvain communities | `GET /mnemonic/graph` returns nodes + empty edges + `degraded: true`; UI renders node-only + file-list fallback + labeled placeholder |
| **013** (layered-memory governance) | owner/version/status/usage/visibility + L0→L3 layering + `mem_share` | governance widgets render labeled collapsed placeholders + the flat pre-013 view; the Memories view stays fully interactive (per-widget isolation) |

## Open follow-ups

- **010-web-graph-canvas** is absorbed: the vector graph (Sigma.js) now ships *in* this change (Phase 4), so the old "defer the graph to 010" follow-up is no longer a separate change. If 005/008 edge data lands later, the graph simply gains its edge set — no new change needed.
- **Bundle-size governance** (Phase 7): if the Sigma.js + graphology + mermaid bundle grows past budget, consider lazy-loading mermaid only in Docs and code-splitting the graph feature further.
