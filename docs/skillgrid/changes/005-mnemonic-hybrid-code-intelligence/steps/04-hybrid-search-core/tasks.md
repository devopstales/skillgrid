# Tasks: 005-mnemonic-hybrid-code-intelligence — Step 04-hybrid-search-core

> Goal: Offline RRF hybrid (FTS+signals) + pluggable embedders + hybrid/semantic/status tools.
> Depends on: 03-call-graph-traversal
> Ticket: TASK-005

> Change-level forecast: see `../01-schema-extractors/tasks.md`
> Decision needed before apply: No · Chained PRs recommended: Yes · Chain strategy: stacked-to-main · 400-line budget risk: High

## Execution

- [ ] 04.1 RED (Mnemonic tool surface): assert `code_hybrid_search` will register as distinct from memory `semantic_search`; `code_search` name/params still stable; bad args to hybrid/semantic rejected — expect fail until hybrid tools land.
- [ ] 04.2 Create `skillgrid-cli/internal/mnemonic/hybrid/rank.go` — signal subset (FTS, MinHash/LSH, proximity, TF-IDF, type/API) + RRF/v1 + per-signal provenance; embeddings off by default.
- [ ] 04.3 Create/extend `skillgrid-cli/internal/mnemonic/embedder/` — code-unit Adapter Seam (Null Adapter default; optional local/hash stub); do not require ONNX to ship.
- [ ] 04.4 Create `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` — `code_hybrid_search`, `code_semantic_search`, `code_embedding_status`; make 04.1 pass.
- [ ] 04.5 Modify `skillgrid-cli/internal/mnemonic/mcp/server.go` — register orient/graph/hybrid tool sets without dropping existing `code_*` or memory `semantic_search`.
- [ ] 04.6 Modify `skillgrid-cli/cmd/skillgrid/main.go` — dispatch `code_intel` CLI; extend facade/CLI for hybrid commands as needed.
- [ ] 04.7 Cover WHAT: `code_hybrid_search` returns ranked hits with per-signal provenance embeddings-off; `code_semantic_search` / `code_embedding_status` available; offline FTS+signals ship criterion met.
- [ ] 04.8 Cover WHAT edge: embedder down/unavailable → degrade to FTS+signals (not hard fail).
