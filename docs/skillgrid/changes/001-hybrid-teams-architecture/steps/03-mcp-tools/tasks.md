# Tasks: 001-hybrid-teams-architecture — Step 03-mcp-tools

> Goal: Register 6 new MCP tools in tools_teams.go with handlers.
> Depends on: 02-service-facade

## Review Workload Forecast
See change-level forecast in `01-schema/tasks.md` (ask-on-risk; High; Chained PRs Yes; Decision needed Yes; Chain strategy pending).

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 03.1 RED (threat: Mnemonic tool surface): Update `skillgrid-cli/internal/mnemonic/mcp/server_test.go` `TestAllToolsRegistered` want list to include `team_spawn_task`, `agent_pull_next_task`, `agent_read_task`, `agent_submit_output`, `agent_submit_review`, `agent_mark_done` — expect fail until tools exist.
- [ ] 03.2 RED (threat: Mnemonic tool surface): Add failing test in `server_test.go` or `tools_teams_test.go` — bad/unknown spawn args → tool error result, not panic.
- [ ] 03.3 Create `skillgrid-cli/internal/mnemonic/mcp/tools_teams.go` — six `s.AddTool` handlers calling service teams methods only (no inbox MCP).
- [ ] 03.4 Modify `skillgrid-cli/internal/mnemonic/mcp/server.go` — call `registerTeamsTools(s)` beside `registerMemoryTools` in Start/NewServer.
- [ ] 03.5 GREEN: Make 03.1 and 03.2 pass; mem/code/web tool names unchanged.
- [ ] 03.6 Verify WHAT: spawn→pull→read→submit round-trip keeps SQL/FS consistent; unknown id / no pending → clear tool error.
