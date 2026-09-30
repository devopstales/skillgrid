# 08 — Second Brain Roadmap (mnemonic)

Framing: **mnemonic IS the second brain.** ADR-0012 locked the *engine* (SQLite as the second-brain store, in place of llm-wiki markdown). This artifact is the *capability roadmap* on top of that store: what makes it feel like a second brain rather than a search API.

Reference: `brobertsaz/claude-os` (https://github.com/brobertsaz/claude-os, 417★, "Give Your AI a Memory", 100% local) — the closest same-shape product. Also the 5-level "building your own AI second brain" taxonomy (video DTCyvo6cC54): L1 exact-match → L2 LLM-wiki (the layer ADR-0012 replaces) → L3 embeddings → L4 relationship graph → L5 always-on autonomous.

## Status: what the store already is (L1–L4)

| Level | Capability | Mnemonic | Status |
|---|---|---|---|
| L1 | Exact-match routing | FTS5 | built |
| L2 | Whole-topic retrieval (replaces markdown wiki) | SQLite | built — the ADR-0012 win |
| L3 | Paraphrase/semantic | `code_semantic_search` + sqlite-vec seam | partial — in-memory hot path, no durable store (ADR-0009) |
| L4 | Relationship chains | Triple-Store Cross-Linkage (`graph_ref`) | built |

## claude-os → mnemonic architecture map

claude-os fuses a second brain + an agent-OS scaffold (specs, kanban, skills). skillgrid already splits those: **skillgrid is the OS scaffold, mnemonic is the memory.** Cleaner. The gap is purely in the memory product.

| claude-os pillar | Mnemonic equivalent | Status |
|---|---|---|
| Real-Time Learning (Redis Pub/Sub captures insights) | tool-call capture hook → `session_events` | built, **hook off / Task-only** |
| Memory MCP (persistent + instant recall) | `mem_*` (SQLite + FTS5) | built |
| Docs RAG (knowledge_docs KB) | web_cache + observations | built |
| Code index (tree-sitter L1 + selective embed L2) | code index + graph + `code_semantic_search` | built (yours ahead: real graph, not just PageRank) |
| Skills Library (36+ community skills) | skillgrid skills | skillgrid's job, not mnemonic's |
| Kanban / spec tracking | skillgrid `.skillgrid/` + backlog | skillgrid's job |

## Capability gaps → change list (prioritized)

### P0 — foundation (autonomous capture)
- **A. Tool-call capture hook** — default-on, observe-mode; `mem_query_events` audit surface. *Already designed* (Monitoring plan, 07). Hook trigger currently off/Task-only.
- **B. Natural-language capture** — detect user intent ("remember this", "what did we decide?", "how did we fix that?") → `mem_save` with inferred `type`. *New — from claude-os "The Magic."* Thin layer over existing `mem_save`.

### P1 — synthesis + lifecycle (the "brain" feel)
- **C. `mem_ask`** — synthesis tool: gather L1+L2+L4 over the graph, return a *cited* answer to a question ("summarize what we decided about auth"). *New — maps to claude-os Chat-with-citations.* Turns retrieval into understanding.
- **D. Knowledge lifecycle** — consolidate (merge near-dupes), archive (retire stale), health report; inline staleness warning appended to search results. *New — from claude-os v2.4/2.5.* Dedup-by-hash already exists; consolidate/archive/health do not.
- **E. Session-inject** — distill → FTS5 index → hybrid-retrieve → inject at resume. *Already designed* (session-inject blueprint, 2026-09-24). Depends on A.

### P2 — human view
- ~~**F. Durable `sqlite-vec` store**~~ — **DONE.** Migration `042_vec0_tables.sql` creates real `vec0` vtabs (`vec_symbols`/`vec_chunks`, `float[768]`, `modernc.org/sqlite/vec` cgo-free), rowid-keyed to the BLOB source-of-truth tables, degrading to in-memory on absence. Confirmed by deep-dive (`09-claude-os-deep-dive.md`) — was misfiled as P2. This also unblocks C (`mem_ask`) on paraphrase queries.
- **G. Human browse view** — topic/PARA-grouped observations in `serve`; you can *see* your brain, not just query it. Partial (graph view exists); observations view not built.

### P3 — autonomous learning
- **H. Passive learning** — auto-extract learnings from completed/failed sessions at session end. *Planned* (session-inject refs: claude-mem).
- **I. Cross-project pattern surfacing** — `all_projects` retrieval exists; add *pattern synthesis* across projects at recall (claude-os v2.5 cross-KB search).

### North star (not now)
- **L5 — always-on autonomous brain** (video's Gbrain: constant sync/refresh). List as the destination, not a change.

## What claude-os does NOT change
- ADR-0012 (SQLite store) — claude-os uses SQLite too.
- Code index — mnemonic is ahead (real dependency graph vs. their PageRank + token-budget repo map).
- Architecture — mnemonic/skillgrid split is cleaner than claude-os's fused product.

## Minimum viable second brain
Store (done) + **B (NL capture)** + **C (mem_ask)** + **D (lifecycle)**. That's the "gets smarter every conversation" loop.

## Deep dive
Verified repo-level analysis + steal list (MCP response shape, lifecycle patterns, anti-hallucination extraction, installer/service patterns, what NOT to copy) is in `09-claude-os-deep-dive.md`. Headline: **we already beat claude-os on the two things they brag about — real vec0 (migration 042) and real FTS5; their "sqlite-vec" is brute-force Python and their "hybrid" has no keyword leg.**
