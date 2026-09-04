# Tasks: 004-hermes-memory — Step 03-skills-registry

> Goal: Skills FS + table + FTS; write/search/list.
> Depends on: 01-facts-schema

> Change-level Review Workload Forecast: see `../01-facts-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 03.1 RED (Mnemonic tool surface): in `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (or `tools_skills_registry_test.go`), assert `write_skill`/`search_skills`/`list_skills` are registered **and** `mem_save` remains registered and dispatches successfully — expect fail until tools land.
- [ ] 03.2 Create `skillgrid-cli/internal/mnemonic/skills/skills.go` — SkillRegistry Write/List/Search (lexical); FS under `.skillgrid/files/skills/{name}.{ext}` + SQL metadata; soft-delete omits from default list/search.
- [ ] 03.3 Make 03.1 pass — create `skillgrid-cli/internal/mnemonic/mcp/tools_skills_registry.go` with write/search/list tools + registrar (JSON-only; additive; no `use_skill` yet).
- [ ] 03.4 Cover WHAT: write/list/search **Agent Skills** (lexical) with FS + SQL metadata.
- [ ] 03.5 Cover WHAT: soft-deleted skills omitted from default list/search.
- [ ] 03.6 Cover WHAT edge: `overwrite=false` rejects name collision with a clear error.
