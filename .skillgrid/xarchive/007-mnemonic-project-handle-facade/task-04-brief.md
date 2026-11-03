## 04-collapse-wrappers

### Goal

The shallow open-delegate-close surface is gone or cannot double-open; locality restored for single-project ops.

### Out of scope / Non-Goals

- store.DB unexport
- MigrateProjects rename
- New features

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-mcp-single-open, 03-http-single-open

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/internal/mnemonic/integration/integration_test.go`
- Test: `skillgrid-cli/internal/mnemonic/service/*_test.go`

**Interfaces:**
- Consumes: MCP + HTTP already on handle path (02, 03)
- Produces: no production path double-opens; dead aliases removed or redirected without second open

### Tasks

- [ ] 04.1 `[RED]` No production path opens the same project twice for one logical op (threat: double-open regression)
  - [ ] 04.1.a Write failing test — Scenario: Facade path cannot double-open
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation (collapse wrappers; open counter / shared path)
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1` — Expected: PASS
  - [ ] 04.1.e Commit — `refactor(mnemonic): collapse double-open Service wrappers`
- [ ] 04.2 `[AFK]` Dead aliases removed or redirected; integration MCP+HTTP smoke — Scenarios: Dead aliases neutralized; Integration smoke still passes — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/integration/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/internal/mnemonic/integration/ -count=1` | PASS | | |
| Acceptance `@step-04` / `@p0` | map scenarios to tests | PASS | | |
| Full suite | `go test ./...` (from `skillgrid-cli`) | PASS | | |
| Rollback boundary | revert branch; no migrations | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `refactor(mnemonic): collapse shallow Project Handle wrappers`

---

