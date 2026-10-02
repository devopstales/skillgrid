# Prototype 001: Mnemonic Web UI Mockup

**Date:** 2026-10-01
**Hypothesis:** A single HTML file (React 18 + Tailwind CDN + D3 v7 + Babel standalone) can serve as a functional admin console for the Mnemonic enterprise API with a nested navigation structure (6 top-level groups, 14 leaf sections) using real data from the live Go API.

## Verdict: PARTIAL

The single-file approach is viable for a working prototype with the nested nav structure. All 14 sections render without JS errors. Remaining gaps are API-side, not client-side.

### Section Test Results (v2 — nested nav)

| Section | Status | Evidence |
|---------|--------|----------|
| Overview | PASS | 36 obs, 18 sessions, 751 files, 20,263 chunks, type distribution, 12 recent events |
| Project → Kanban | PASS (empty) | 7 columns rendered, 0 tasks (backlogmd provider confirmed empty) |
| Project → Changes | PASS (empty) | `/specs` returns empty file list — doc roots not configured |
| Project → Decisions (ADR) | PASS (empty) | `/mnemonic/decisions` returns `[]` — bridge has no entries. Help text points to ASSUMPTIONS.md |
| Project → Prototypes | PASS (empty) | No `/prototypes` JSON endpoint. Help text lists 3 prototypes on disk |
| Memory → Sessions | PASS | 21 sessions with status badges, memory counts, summary indicators |
| Memory → Code Graph | PASS | D3 force-directed graph: 500 nodes + 2,110 edges, type legend, index status sidebar |
| Memory → MemFS | PASS | Topic hierarchy tree (root → 10 branches → 35+ leaves), expandable, content viewer |
| Observe → Telemetry | PASS | 21 sessions + 36 events in stream, type badges, timestamps |
| Observe → Compaction | PASS | Compaction blocks + context panel with session/project/generated_at |
| Observe → Web Cache | PASS (empty) | 0 entries, source filter dropdown, per-source KPI cards |
| Docs | PASS (empty) | Tree viewer + content panel — `/docs/tree` returns empty nodes |
| System → Settings | PASS (empty) | Config + state viewer — doc roots not configured for `.skillgrid/` |
| System → Swagger | PASS | iframe pointing to `/swagger.html` on the Go API |

### API Gaps Discovered

1. **CORS missing** (FIXED in this prototype) — `corsMiddleware` added to `server.go:259-277`
2. **`/mnemonic/search` returns empty** for all queries — FTS5 gap blocks search features
3. **`/code/status`, `/activity/stats`, `/web/status`, `/mnemonic/files/tree`, `/context`, `/context/compaction`, `/mnemonic/decisions`** all require `?project=` param (returned 400 without it)
4. **`/docs/tree` and `/docs/content`** have empty doc roots — `.skillgrid/` files not served
5. **`/prototypes` and `/decisions`** (bare) return HTML (SPA fallback), not JSON
6. **`/mnemonic/decisions`** returns `[]` — no decisions in the bridge

### What Changed from v1

- Nav restructured from 8 flat sections to 6 top-level groups with 14 leaf sections
- Added Project group: Kanban, Changes, Decisions (ADR), Prototypes
- Added Observe group: Telemetry, Compaction, Web Cache
- Added System group: Settings, Swagger Docs
- Added `project` param to all `apiGet` calls (was missing, caused 400s)
- Removed Memory list + Second Brain chat (replaced by nested structure)
- Fixed `/mnemonic/graph/data` to include `project` param
- Fixed `/code/status` to include `project` param

### Edge cases handled

- Empty states for all sections with helpful messages pointing to API gaps
- Error states with retry
- Large graph: limited to 500 nodes via API param
- Console: zero JS errors (only CDN warnings for Tailwind + Babel)

### Edge cases NOT handled

- No pagination
- No deep-linking (URL doesn't reflect active panel)
- No auth
- No mobile responsive testing

## Files Modified

- `skillgrid-cli/internal/mnemonic/http/server.go`: Added `corsMiddleware` function + wrapped `Handler()` return (lines 259-277)

## Deliverables

- `index.html` — single-file mockup with nested nav (React 18 + Tailwind CDN + D3 v7 + Babel standalone)
- `prototype.md` — this file
- `findings.md` — CORS fix, FTS gap, graph volume learnings

## Key Learnings

1. **CORS middleware is a prerequisite** for any browser-based UI against the Go API.
2. **`project` param is required** on most endpoints — the `apiGet` helper now injects it automatically.
3. **FTS5 search needs investigation** — `/mnemonic/search` returns empty regardless of query.
4. **Doc roots are not configured** for `.skillgrid/` — Changes, Prototypes, Decisions, Docs, Settings all show empty states.
5. **Single-file approach works for prototyping** — 14 sections, D3 graphs, tree hierarchies, all without a build step.
6. **The nested nav structure works well** for the admin console use case — clear information architecture with 6 groups.
