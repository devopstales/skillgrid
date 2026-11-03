## 03-http-single-open

### Goal

HTTP single-project routes match the MCP lifecycle: one Project Handle open per request.

### Out of scope / Non-Goals

- Changing auth, routes, or JSON shapes
- Cross-store migrate/merge routes may keep root Service calls
- Deleting Service wrappers (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-export-project-handle

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server_test.go`

**Interfaces:**
- Consumes: exported `ProjectHandle` from 01
- Produces: single-project handlers that open once; migrate/merge still on root Service

### Tasks

- [ ] 03.1 `[RED]` Observation create / recent path uses one handle open
  - [ ] 03.1.a Write failing test — Scenario: Observation routes open store once
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Observation|SingleOpen|Handle' -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Observation|SingleOpen|Handle' -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): HTTP observation routes use Project Handle`
- [ ] 03.2 `[AFK]` Session, search, code, web single-project handlers use handle; migrate/merge stay on root; bearer auth unchanged — Scenarios: Single-project HTTP uses handle; Migrate and merge stay on root — `Run: go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/ -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | map scenarios to HTTP tests | PASS | | |
| Runtime harness | N/A | PASS | | |
| Rollback boundary | revert; no migrations | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): HTTP single-project routes use Project Handle`

---

