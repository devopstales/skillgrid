# Tasks: 001-hybrid-teams-architecture — Step 05-tests

> Goal: Unit tests for service + integration tests for MCP tools.
> Depends on: 03-mcp-tools, 04-http-routes

## Review Workload Forecast
See change-level forecast in `01-schema/tasks.md` (ask-on-risk; High; Chained PRs Yes; Decision needed Yes; Chain strategy pending).

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 05.1 RED (threat: Mnemonic tool surface): In `skillgrid-cli/internal/mnemonic/mcp/tools_teams_test.go`, assert registry/dispatch includes the six team tool names and bad spawn returns tool error (not panic) — fail until handlers cover the case.
- [ ] 05.2 GREEN: Implement dispatch + FS/SQL parity coverage in `tools_teams_test.go` so 05.1 passes.
- [ ] 05.3 Create `skillgrid-cli/internal/mnemonic/service/teams_test.go` — atomicity (FS ok + SQL fail → file rolled back), pull priority, status transitions (`pending`→`review_spec`→`done`).
- [ ] 05.4 Create `skillgrid-cli/internal/mnemonic/http/teams_test.go` — write routes 401 without bearer; open GET reads succeed.
- [ ] 05.5 Verify WHAT: `go test ./…` covers facade atomicity, MCP registration/dispatch, HTTP write auth; six new tool names in registry.
