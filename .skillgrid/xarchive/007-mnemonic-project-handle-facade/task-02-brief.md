## 02-mcp-single-open

### Goal

Each MCP tool call opens SQLite at most once for its target project and calls domain ops through the Project Handle.

### Out of scope / Non-Goals

- HTTP rewire (step 03)
- Deleting unused Service wrappers (step 04)
- Changing tool names, params, or DTO shapes

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-export-project-handle

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_code.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_web.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server_test.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/*_test.go`

**Interfaces:**
- Consumes: exported `ProjectHandle` + `Service.Open*` from 01
- Produces: MCP handlers that open once and use handle; `SetService` / inject still works

### Tasks

- [x] 02.1 `[RED]` mem_save happy path opens the store once (threat: Mnemonic tool surface + double-open)
  - [x] 02.1.a Write failing test — Scenario: mem_save opens store once
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|SingleOpen|OpenOnce' -count=1` — Expected: FAIL
  - [x] 02.1.c Minimal implementation (open handle once; call Memory via handle)
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|SingleOpen|OpenOnce' -count=1` — Expected: PASS
  - [x] 02.1.e Commit — `feat(mnemonic): MCP tools single-open via Project Handle` (02.1/02.2/02.3 in one, per repo convention)
- [x] 02.2 `[RED]` mem_search (and contract shape) unchanged after handle path (threat: Mnemonic tool surface)
  - [x] 02.2.a Write failing test — Scenario: mem_search result shape unchanged
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSearch|Contract|Shape' -count=1` — Expected: FAIL
  - [x] 02.2.c Minimal implementation
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSearch|Contract|Shape' -count=1` — Expected: PASS
  - [x] 02.2.e Commit — `feat(mnemonic): MCP tools single-open via Project Handle` (see 02.1.e)
- [x] 02.3 `[AFK]` code_* and web_cache_* handlers use handle; SetService inject still works — Scenarios: code and web tools use handle; Injected service still works — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | map scenarios to MCP tests | PASS | | |
| Runtime harness | optional e2e if already present | PASS | | |
| Rollback boundary | revert; no migrations | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): MCP tools single-open via Project Handle`

---

