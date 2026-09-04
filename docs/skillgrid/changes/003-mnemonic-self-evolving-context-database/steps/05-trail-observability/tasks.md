# Tasks: 003-mnemonic-self-evolving-context-database — Step 05-trail-observability

> Goal: `skillgrid trail show|recent` CLI.
> Depends on: 04-session-compaction

> Change-level Review Workload Forecast: see `../01-schema-extensions/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 05.1 Create `skillgrid-cli/cmd/skillgrid/trail.go` with `trail recent` and `trail show <id>` reading `retrieval_trails` (query, directories, files, result path).
- [ ] 05.2 Modify `skillgrid-cli/cmd/skillgrid/main.go` to dispatch `migrate` and `trail` subcommands.
- [ ] 05.3 Cover WHAT: `trail recent` / `trail show <id>` show query, directories, files, result path.
- [ ] 05.4 Cover WHAT: empty store → empty list, not error.
- [ ] 05.5 Cover WHAT edge: unknown id → not-found.
