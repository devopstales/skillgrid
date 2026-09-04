# Tasks: 005-mnemonic-hybrid-code-intelligence — Step 01-schema-extractors

> Goal: Additive `011_*` schema + Extractor Module + Go/TS/TSX + indexer graph hook.
> Depends on: none
> Ticket: TASK-005

## Review Workload Forecast (change-level)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | ~450–650 |
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk (change) | High |
| Chained PRs recommended (change) | Yes |
| Suggested split | PR1 schema+extract → PR2 FTS/orient → PR3 graph → PR4 hybrid |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Schema + extractors + indexer hook | PR 1 | `go test ./internal/mnemonic/extract/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/store/...` | `skillgrid index` on fixture repo; query symbols/edges via SQL | Drop `011_*` + `extract/`; revert indexer hook |
| 2 | Identifier FTS + Tier-1 orientation | PR 2 | `go test ./internal/mnemonic/search/... ./internal/mnemonic/mcp/...` | MCP orient tools on indexed fixture | Drop orient tools + `symbol_fts.go` |
| 3 | Call-graph Tier-2 tools | PR 3 | `go test ./internal/mnemonic/graph/... ./internal/mnemonic/mcp/...` | callers/callees on known symbol | Drop `graph/` + graph tools |
| 4 | Hybrid/RRF + embed adapters | PR 4 | `go test ./internal/mnemonic/hybrid/... ./internal/mnemonic/mcp/...` | `code_hybrid_search` embeddings-off | Drop `hybrid/` + hybrid tools |

## Execution

- [ ] 01.1 Create `skillgrid-cli/internal/mnemonic/store/migrations/011_hybrid_code_intel.sql` — symbols, edges, symbol_fts, embeddings, embed_meta, lsh_buckets, index_freshness (leave `009_*`/`010_*` for 001/003).
- [ ] 01.2 Create `skillgrid-cli/internal/mnemonic/extract/extract.go` — `Extractor` Interface, `FileGraph`, registry by Patterns().
- [ ] 01.3 Create `skillgrid-cli/internal/mnemonic/extract/go.go` — Go Adapter via `go/parser` → symbols/edges.
- [ ] 01.4 Create `skillgrid-cli/internal/mnemonic/extract/tsx.go` — pure-Go TS/TSX Adapter → symbols/edges.
- [ ] 01.5 Create `skillgrid-cli/internal/mnemonic/extract/fallback.go` — regex fallback when primary extract fails.
- [ ] 01.6 Modify `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — same-tx extract/prune into graph tables; keep chunks path.
- [ ] 01.7 Cover WHAT: after `code_index`, Go/TS/TSX files yield queryable Symbols and Edges; store open creates new tables without rewriting `files`/`chunks`.
- [ ] 01.8 Cover WHAT edge: one malformed file → fallback + continue; index run does not abort.
