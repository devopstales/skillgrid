# Tasks: 003-mnemonic-self-evolving-context-database — Step 03-semantic-retrieval

> Goal: Pure Go embeddings + search/load tools + trail logging.
> Depends on: 02-tiered-storage

> Change-level Review Workload Forecast: see `../01-schema-extensions/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 03.1 RED — Mnemonic tool surface: assert `semantic_search` response body is L1-only (overview + abstract; no L2 markdown body).
- [ ] 03.2 RED — Mnemonic tool surface: assert L2 content is reachable only via `load_full_details{path}` (not via search results).
- [ ] 03.3 Create `skillgrid-cli/internal/mnemonic/embedder/embedder.go` (`Embedder` + local/remote adapters; `MNEMONIC_EMBED` flag; pick local library or hash stub).
- [ ] 03.4 Modify `skillgrid-cli/internal/mnemonic/memory/embedding.go` to share Vector/blob/cosine helpers with the embedder.
- [ ] 03.5 Modify `skillgrid-cli/internal/mnemonic/service/service.go` for ranked L1 search, `load_full_details`, and retrieval-trail persistence.
- [ ] 03.6 Create `skillgrid-cli/internal/mnemonic/mcp/tools_retrieval.go` with `semantic_search` and `load_full_details` handlers (JSON-only); make 03.1–03.2 pass.
- [ ] 03.7 Cover WHAT: `semantic_search` returns ranked L1 (with abstracts), not L2.
- [ ] 03.8 Cover WHAT: given a result path, `load_full_details` returns L2 markdown.
- [ ] 03.9 Cover WHAT edge: embeddings off/empty → title/L0 fallback; trail still recorded.
