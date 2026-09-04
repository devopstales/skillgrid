# Tasks: 004-hermes-memory — Step 02-fact-tools

> Goal: MCP fact_* + trail logging.
> Depends on: 01-facts-schema

> Change-level Review Workload Forecast: see `../01-facts-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 02.1 RED (Mnemonic tool surface): in `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (or `tools_facts_test.go`), assert `fact_add`/`fact_search`/`fact_forget`/`fact_decay` are registered **and** `mem_save` remains registered and dispatches successfully — expect fail until tools land.
- [ ] 02.2 RED (Mnemonic tool surface): soft-deleted fact absent from default `fact_search` — fixture forget then search returns no hit for that id; `mem_*` shape unchanged.
- [ ] 02.3 Make 02.1–02.2 pass — create `skillgrid-cli/internal/mnemonic/mcp/tools_facts.go` with `fact_add`/`fact_search`/`fact_forget`/`fact_decay`, **Retrieval Trail** (mode + fact ids), and `RegisterFactTools` (JSON-only; additive).
- [ ] 02.4 Cover WHAT: agent can add/search/forget/decay facts; soft-deleted facts out of default search.
- [ ] 02.5 Cover WHAT: `fact_search` writes a **Retrieval Trail** recording mode and fact ids.
- [ ] 02.6 Cover WHAT edge: decay lowers importance and logs `forgetting_events`; purge only below threshold.
