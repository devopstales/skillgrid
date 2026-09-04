# Tasks: 003-mnemonic-self-evolving-context-database — Step 04-session-compaction

> Goal: `mnemonic_commit` → long-term memories with tiers.
> Depends on: 03-semantic-retrieval

> Change-level Review Workload Forecast: see `../01-schema-extensions/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 04.1 RED — Mnemonic tool surface: after registering new tools, assert existing `mem_save` remains registered and callable (additive-only; no mem_* contract break).
- [ ] 04.2 Create `skillgrid-cli/internal/mnemonic/service/compaction.go` for explicit commit → `long_term_memories` with L0/L1/L2 + optional source link.
- [ ] 04.3 Create `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go` with `mnemonic_commit` (`task_id?`, `lessons_learned?`, `title?` → `{memory_id,paths}`).
- [ ] 04.4 Modify `skillgrid-cli/internal/mnemonic/mcp/server.go` to register retrieval + compaction tools; make 04.1 pass (no auto-commit on session end).
- [ ] 04.5 Cover WHAT: `mnemonic_commit` persists Long-term Memory with L0/L1/L2 and optional source link.
- [ ] 04.6 Cover WHAT: session end alone does not auto-commit.
- [ ] 04.7 Cover WHAT edge: missing sources → clear error; no partial write.
