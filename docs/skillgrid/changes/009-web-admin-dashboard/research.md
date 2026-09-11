# Research: UI Pattern Survey for 009 Dashboard

**Date:** 2026-09-08
**Sources:** Graphify-Labs/graphify, colbymchenry/codegraph, abhigyanpatwari/GitNexus

## Constraint reminder

009 is a **vanilla JS SPA embedded in a Go binary** (`embed.FS`, no build step, no npm toolchain, no external CDN). Any pattern adopted must work within that envelope or be deferred.

## Per-repo findings

### Graphify-Labs/graphify

- **graph.html**: vis.js (single-file minified, ~400KB, embeddable) interactive graph. Search box → dropdown results → click focuses node + shows info panel (type, community, source file, degree). Node click → neighbor list; clicking a neighbor re-focuses. Edges show relation + confidence (EXTRACTED/INFERRED/AMBIGUOUS) on hover. Community legend with per-community show/hide checkboxes + "Select All". Node size = degree (god nodes pop visually). Aggregated community-level meta-graph when node count exceeds a threshold.
- **GRAPH_REPORT.md**: corpus check (file/word counts + verdict), summary (node/edge/community counts + confidence percentages), **Graph Freshness** (commit hash + "run X to update" instruction), God Nodes section, Communities section, **Suggested Questions** section.
- **Query-first policy** (issue #580): all agent install surfaces point at `graphify query` first, not the full report. The report is a fallback for broad architecture review, not the primary surface.
- **Stack:** Python, NetworkX, vis.js, tree-sitter, Leiden via graspologic. Not directly portable — the *patterns* are, the code is not.

### colbymchenry/codegraph

- **Telemetry Dashboard** (stats.getcodegraph.com): plain static HTML + ES modules + Chart.js, no framework, no build step. Same vanilla envelope as 009.
- **Show-numbers table twin:** every chart has a toggle to render the raw `rows[]` data as a table. The table is the chart's source of truth; the chart is a convenience view.
- **Panel independence:** each panel fetches, draws, and reports independently — a failed query kills one panel, not the page.
- **Completeness honesty:** activation funnel marks incomplete cohort days with `complete: false` + `incomplete_from`; date-range clamp surfaced as `range.clamped` in the response.
- **Global date-range filter:** `?from=YYYY-MM-DD&to=YYYY-MM-DD` applied across all panels; ranges >366 days clamped.
- **Breakdown dimensions:** `/api/breakdown?dim=<field>` segments any metric by os/arch/version/language/etc.
- **Duration buckets:** `<10s / 10-60s / 1-5m / 5m+` for index event durations.
- **Stack:** static files, ES modules, Chart.js, HMAC-signed cookie auth. No framework.

### abhigyanpatwari/GitNexus

- **3-panel layout:** file tree (left) / graph canvas (center) / context panel (right, tabbed: AI chat + Processes).
- **Graph canvas:** Sigma.js + Graphology (WebGL), force/tree/circles layouts. Nodes color-coded by community. Click node → selection info bar. "Blast Radius" + AI-citation highlighting. Zoom in/out.
- **Code references panel:** syntax-highlighted source, dynamic context fetch (selected symbol ± buffer lines), collapse/expand/resize.
- **AI chat:** entity grounding — LLM responses parsed for file/node references; clicking a reference updates global selection. Auto-scroll unless user scrolls up.
- **Processes tab:** detected processes grouped by cross-community / intra-community; individual flowcharts + combined map.
- **Global search:** header search bar → dropdown (node name, type, color indicator).
- **Stack:** React 18 + TypeScript + Vite + Tailwind v4 + Sigma.js + Web Workers + Comlink + tree-sitter WASM + LadybugDB WASM + transformers.js (WebGPU embeddings). **Heavy** — opposite of 009's constraints.

## Patterns adopted into 009 (steal now)

| # | Pattern | Source | 009 step | Implementation |
|---|---------|--------|----------|----------------|
| 1 | Show-numbers table twin | codegraph | 01 (helper), applied 02–06 | Every data widget has a "show numbers" toggle → raw JSON/table. Baseline for all menu entries. |
| 2 | Per-widget error isolation | codegraph | 01 (shell), promoted in 02 | Each widget owns its fetch + error render; a 5xx kills the widget, not the entry/page. |
| 3 | Freshness/completeness banner | graphify + codegraph | 05 (Code), 02 (Tracker) | Code entry: last-indexed + stale as top banner with re-index action. Tracker entry: provider + version banner; unknown Backlog.md schema → warning. |
| 4 | Query-first + suggested prompts | graphify | 04 (Memory) | Empty state shows recent session prompts (from `GET /context`) as clickable suggestions, not blank "no results". |
| 5 | Relation drill-down with confidence badges | graphify | 04 (Memory) | Observation detail lists relations via `GET /relations/{id}`; confidence badges (EXTRACTED/INFERRED/AMBIGUOUS); click → navigate to related observation. |
| 6 | Forward-compat graph placeholder | GitNexus + graphify | 05 (Code) | Collapsed "Code graph (coming in 010)" panel; file-list fallback; mount point for graph canvas when 005/008 edge data lands. |

## Patterns deferred to follow-up (010 or later)

| Pattern | Source | Why deferred | Dependency |
|---------|--------|-------------|------------|
| Graph canvas (vis.js) | graphify + GitNexus | Code index has no edges yet; 005 (symbols/edges) + 008 (communities/god nodes) must land first | 005, 008 |
| Community legend with toggle filters | graphify | Needs Leiden communities from 008 | 008 |
| Degree-based node sizing / god-node visual pop | graphify | Needs edges from 005 | 005 |
| 3-panel code layout (tree/canvas/context) | GitNexus | Right IA but requires graph canvas first | 010 |

## Patterns explicitly NOT adopted

| Pattern | Source | Reason |
|---------|--------|--------|
| AI chat panel | GitNexus | Dashboard is operator-facing, not an agent client; the agent lives in the terminal |
| React/TS/Vite/Tailwind stack | GitNexus | Opposite of 009's vanilla-embedded constraint |
| WebGPU in-browser embeddings | GitNexus | No LLM/embedding inference in the dashboard |
| Password login UI | codegraph | 127.0.0.1 tool; existing bearer-token gate is sufficient; login UI is friction without threat |
| Telemetry funnels / retention cohorts | codegraph | For product-adoption analytics, not machine-state inspection; no telemetry pipeline in 009 |
| LLM semantic extraction in UI | graphify | Pipeline concern, not UI |

## Open follow-up: 010-web-graph-canvas

When 005 (symbol/edge extraction) and 008 (Leiden communities + god nodes) are applied, a follow-up change should add:
- vis.js (single-file, ~400KB, embeddable via `embed.FS`) to the Code tab
- Community legend with per-community show/hide checkboxes
- Degree-based node sizing (god nodes pop)
- Node click → info panel (type, community, source file, degree) + neighbor list
- Edge hover → relation + confidence label
- Aggregated community-level meta-graph when node count exceeds a threshold

This is a separate change because it depends on data 009 does not produce.
