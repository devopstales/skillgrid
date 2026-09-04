# Tasks: 004-hermes-memory — Step 05-commit-hooks-cli

> Goal: Commit fact extraction + auto-skill; memory/skill CLI.
> Depends on: 02-fact-tools, 03-skills-registry, 04-skill-execute-hybrid

> Change-level Review Workload Forecast: see `../01-facts-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 05.1 Modify `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go` — after **003** `mnemonic_commit`, extract facts ± optional auto `write_skill`; preserve existing commit behaviour; no tier/trail CLI redo.
- [ ] 05.2 Create `skillgrid-cli/cmd/skillgrid/memory.go` — `skillgrid memory` fact|forget|decay dispatching to Fact Memory **Module** (parity with MCP).
- [ ] 05.3 Create `skillgrid-cli/cmd/skillgrid/skill.go` — `skillgrid skill` list|search|execute dispatching to SkillRegistry / Executor (parity with MCP).
- [ ] 05.4 Modify `skillgrid-cli/cmd/skillgrid/main.go` — register `memory` and `skill` subcommands.
- [ ] 05.5 Cover WHAT: `mnemonic_commit` keeps **003** behaviour and adds fact extract ± auto-skill.
- [ ] 05.6 Cover WHAT: `skillgrid memory` / `skillgrid skill` (incl. execute) match MCP outcomes.
- [ ] 05.7 Cover WHAT edge: skip auto-skill when no reusable pattern; trail CLI unchanged.
