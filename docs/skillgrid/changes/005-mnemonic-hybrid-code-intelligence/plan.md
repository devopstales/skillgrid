# Plan: 005 — Mnemonic Hybrid Code Intelligence (Foundation Slice)

## Technical Approach
Add foundation **Hybrid Search**/graph atop the existing SQLite code index: additive schema, Go+TS/TSX **Extractor**s, **Identifier-Aware FTS**, Tier-1/2 `code_*` tools, offline RRF (FTS+signals) with optional embeddings. Preserve `code_status`/`code_index`/`code_search`/`code_read`. Do not contradict **002**, **003** `semantic_search`, or **004**. Source: `docs/plan/07-nmemonic-hybid-search.md` (later tiers = later **Change**s).

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Graph write **Seam** | Hook extract/prune into `Indexer.Run` same tx | Separate graph-index cmd | Same hash+mtime guard; one `code_index` |
| **Extractor** **Module** | `Extract→FileGraph`; Go=`go/parser`; TS=pure-Go; regex fallback | CGo tree-sitter now | CGo-free (modernc); richer **Adapter**s later |
| **Identifier-Aware FTS** | Pre-split camel/snake into FTS5 `unicode61` | Custom C tokenizer | Works on modernc; meets UAT |
| Tool names | All new tools `code_*` | Plan-07 bare names | No clash with **003** `semantic_search` |
| Hybrid | RRF: FTS + MinHash/LSH + proximity + TF-IDF + type/API; embed off by default (Null **Adapter**) | Require ONNX to ship | Approved offline ship criterion |
| Migration | `011_hybrid_code_intel.sql` | Take `009` | Leave `009`/`010` for 001/003 |

Depth: small **Interface**s (**Extractor**, ranker, embedder **Seam**) hide large implementation.

## Data Flow

```
Indexer.Run → chunks → Extractor → symbols/edges/LSH/symbol_fts → prune deletes
code_* search → FTS[+signals][+embed] → RRF → hits+provenance+freshness
code_get_* → resolve Symbol → walk Edge (Confidence Label)
```

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/011_hybrid_code_intel.sql` | Create | 01 | symbols, edges, symbol_fts, embeddings, embed_meta, lsh_buckets, index_freshness |
| `skillgrid-cli/internal/mnemonic/extract/extract.go` | Create | 01 | **Extractor** **Interface** + registry |
| `skillgrid-cli/internal/mnemonic/extract/go.go` | Create | 01 | Go **Adapter** |
| `skillgrid-cli/internal/mnemonic/extract/tsx.go` | Create | 01 | TS/TSX **Adapter** |
| `skillgrid-cli/internal/mnemonic/extract/fallback.go` | Create | 01 | Regex fallback |
| `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` | Modify | 01 | Graph extract/prune hook |
| `skillgrid-cli/internal/mnemonic/search/symbol_fts.go` | Create | 02 | Identifier symbol search |
| `skillgrid-cli/internal/mnemonic/mcp/tools_code_orient.go` | Create | 02 | Tier-1 orientation tools |
| `skillgrid-cli/internal/mnemonic/service/service.go` | Modify | 02 | Orient facade (+03–04) |
| `skillgrid-cli/cmd/skillgrid/code_intel.go` | Create | 02 | CLI orient (+03–04) |
| `skillgrid-cli/internal/mnemonic/graph/resolve.go` | Create | 03 | Edge resolve + **Confidence Label** |
| `skillgrid-cli/internal/mnemonic/mcp/tools_code_graph.go` | Create | 03 | Tier-2 graph tools |
| `skillgrid-cli/internal/mnemonic/hybrid/rank.go` | Create | 04 | Signals + RRF + provenance |
| `skillgrid-cli/internal/mnemonic/embedder/` | Create | 04 | Extend **003** embedder for code units |
| `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` | Create | 04 | hybrid/semantic/embedding_status |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 04 | Register new tool sets |
| `skillgrid-cli/cmd/skillgrid/main.go` | Modify | 04 | CLI dispatch |

## Step WHAT

### Step 01-schema-extractors — What it delivers
- Store open creates graph/FTS/embed/freshness tables without rewriting `files`/`chunks`.
- After index, Go/TS/TSX yield queryable **Symbol**s and **Edge**s.
- Edge: one malformed file → fallback + continue.

### Step 02-identifier-fts-orientation — What it delivers
- **Identifier-Aware FTS** finds camelCase/snake_case symbols chunk search misses.
- Orientation tools: signature, file TOC, map/list, symbol metadata.
- Edge: unknown symbol → empty/not-found; `code_search` unchanged.

### Step 03-call-graph-traversal — What it delivers
- Callers, callees, dependents (transitive), implementors, hierarchy, tests-for.
- Every **Edge** carries a **Confidence Label**.
- Edge: ambiguity → `AMBIGUOUS`, not silent drop.

### Step 04-hybrid-search-core — What it delivers
- `code_hybrid_search` with per-signal provenance; works embeddings-off.
- `code_semantic_search` / `code_embedding_status`; offline FTS+signals remain.
- Edge: embedder down → degrade to FTS+signals.

## Interfaces / Contracts

```go
type Extractor interface {
  Language() string
  Patterns() []string
  Extract(path string, content []byte) (*FileGraph, error)
}
// New: code_map, code_search_symbols, code_get_symbol, code_get_signature,
// code_symbols_in_file, code_list_projects, code_get_callers, code_get_callees,
// code_get_dependents, code_get_implementors, code_get_tests_for, code_get_type_hierarchy,
// code_hybrid_search, code_semantic_search, code_embedding_status, code_index_status
// Unchanged: code_status, code_index, code_search, code_read
```

## Mnemonic Integration
New `code_*` only; memory `semantic_search` untouched. Topic: `sdd/005-mnemonic-hybrid-code-intelligence/plan`. Convention updates are apply-phase.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests | Owner step |
|---|---|---|---|---|
| Doc-like / git / commit / push / PR | N/A: no exec classification, gitRoot change, or VCS/PR automation | — | — | — |
| **Mnemonic tool surface** | Applicable — new `code_*`; four existing unchanged | Additive; ≠ memory `semantic_search` | `code_search` schema stable; `code_hybrid_search` registered distinct; bad args rejected | 02, 04 |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` in design | — | — | — |

## Migration / Rollout
Additive `011_*`. Embeddings optional. No watcher/communities/docs/`graph.sqlite.zst`. Rollback drops new tools/packages/migration; chunk index stays.

## Open Questions
- [ ] Local embedder: hash-stub vs ONNX at apply; Null **Adapter** unblocks UAT.
- [ ] Default RRF weights — tune in 04; provenance always required.
