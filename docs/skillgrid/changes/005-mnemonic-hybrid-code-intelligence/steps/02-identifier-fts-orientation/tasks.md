# Tasks: 005-mnemonic-hybrid-code-intelligence — Step 02-identifier-fts-orientation

> Goal: Identifier-Aware FTS + Tier-1 orientation MCP/CLI; keep existing `code_*` stable.
> Depends on: 01-schema-extractors
> Ticket: TASK-005

> Change-level forecast: see `../01-schema-extractors/tasks.md`
> Decision needed before apply: No · Chained PRs recommended: Yes · Chain strategy: stacked-to-main · 400-line budget risk: High

## Execution

- [ ] 02.1 RED (Mnemonic tool surface): in `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (or sibling), assert `code_search` tool name + required `query` param schema unchanged; expect fail only if production regresses schema — lock baseline before new tools.
- [ ] 02.2 RED (Mnemonic tool surface): assert orientation tools `code_map`, `code_search_symbols`, `code_get_symbol`, `code_get_signature`, `code_symbols_in_file`, `code_list_projects`, `code_index_status` will be registered; bad/missing args rejected with clear error — expect fail until tools land.
- [ ] 02.3 Create `skillgrid-cli/internal/mnemonic/search/symbol_fts.go` — camelCase/snake_case pre-split into FTS5 `unicode61` symbol search.
- [ ] 02.4 Create `skillgrid-cli/internal/mnemonic/mcp/tools_code_orient.go` — Tier-1 orientation handlers (JSON-only); make 02.2 pass; wire registrar without dropping `code_status`/`code_index`/`code_search`/`code_read`.
- [ ] 02.5 Modify `skillgrid-cli/internal/mnemonic/service/service.go` — orient facade methods used by MCP/CLI.
- [ ] 02.6 Create `skillgrid-cli/cmd/skillgrid/code_intel.go` — CLI parity for orientation commands.
- [ ] 02.7 Cover WHAT: Identifier-Aware FTS finds camelCase/snake_case symbols that chunk `code_search` misses; signature / file TOC / map / list / symbol metadata work.
- [ ] 02.8 Cover WHAT edge: unknown symbol → empty/not-found; `code_search` schema and behavior unchanged (02.1 stays green).
