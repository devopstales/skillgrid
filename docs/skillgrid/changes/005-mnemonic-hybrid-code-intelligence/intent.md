# Intent: 005 — Mnemonic Hybrid Code Intelligence (Foundation Slice)

## Classification
**Architectural** — new code-intelligence subsystem (not chunk-FTS tweak).

## Business Problem
Mnemonic code index is "better grep": 80-line chunks + trigram FTS miss identifiers, cannot answer callers/callees, lack hybrid ranking. Agents burn tokens on grep/read loops. Store + sync + MCP exist; srclight / codegraph / codebase-memory-mcp show the gap.

## Target Users & Situations
- **Coding agent** — navigate / blast-radius before edits; high urgency.
- **Operator** — CLI parity.
- Owns **code** intelligence only (not memory **Changes** 002–004).

## Business Rules
- First product slice; later tiers = later **Changes**.
- Additive SQL; keep `code_search` / `code_index` / `code_read` name + signature.
- CGo-free (`modernc.org/sqlite`); per-project; content-hash (+ mtime) sync.
- No clash with memory `semantic_search` (**003**); distinct `code_*` names.
- Graph+FTS offline without embeddings; embeddings pluggable.
- Per-file extract failure → fallback + continue.
- Edge **Confidence Label**: `EXTRACTED | INFERRED | AMBIGUOUS`.

## Success Criteria (UAT-level)
- [ ] Go/TS/TSX yield queryable **Symbol**s and **Edge**s after incremental index.
- [ ] **Identifier-Aware FTS** finds camelCase/snake_case symbols chunk search misses.
- [ ] MCP/CLI: signature, file TOC, callers, callees, dependents for a known symbol.
- [ ] **Hybrid Search** ranks with per-signal provenance; works with embeddings off.
- [ ] Existing `code_search` unchanged; `go test ./...` passes for touched packages.
- [ ] One malformed file does not fail the index run.

## Scope

### In Scope
- Schema: symbols, edges, symbol FTS, embeddings/embed_meta, LSH, **Index Freshness**.
- **Extractor** **Interface** + Tier-1 Go/TS/TSX + fallback + incremental graph index.
- Tier-1 orientation + Tier-2 graph tools (callers/callees/dependents/implementors/hierarchy/tests).
- Hybrid core: identifier FTS + signal subset + pluggable embeddings + RRF/v1 + code hybrid/semantic + embedding status.

### Out of Scope
- Full 42 tools; communities/impact; git; build/config; documents.
- Languages beyond Go/TS/TSX; `graph.sqlite.zst`; fsnotify watcher.
- Memory rewrite (002–004); cloud sync; replacing `chunks_fts`.

## Step Blueprint
- `01-schema-extractors`: Migrations + **Extractor** + Go/TS + incremental graph index.
- `02-identifier-fts-orientation`: Identifier-aware FTS + Tier-1 orientation tools.
- `03-call-graph-traversal`: Edge resolution + confidence + Tier-2 graph tools.
- `04-hybrid-search-core`: Signals + embedding **Adapter**s + hybrid/semantic tools.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `…/mnemonic/store/migrations/` | New | Symbols, edges, FTS, embeddings, freshness |
| `…/mnemonic/codeindex/` | Modified | Hook extractors; keep chunks |
| `…/mnemonic/` (extract/graph/search) | New | Extractors, resolver, ranker |
| `…/mnemonic/mcp/` + CLI | Modified | Tools/commands beside `code_*` |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Scope → full 12-week platform | High | Hard Out of Scope |
| Clash with memory semantic search | Med | Distinct `code_*` names |
| False edges / extract quality | Med | Confidence; fallback |
| Embedding/ONNX size | Med | Flag; offline FTS+signals first |

## Rollback Plan
Remove new migrations/tools/packages; leave `files`/`chunks`/`chunks_fts` and existing `code_*` intact.

## Dependencies
- Prefer after **002**; orthogonal to **003** (shared embeddings OK).
- Plan: `docs/plan/07-nmemonic-hybid-search.md`.
