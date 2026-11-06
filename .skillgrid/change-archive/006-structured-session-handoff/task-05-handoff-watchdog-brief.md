## 05-handoff-watchdog

### Goal

Optional flag/env-gated context-limit watchdog triggers the same Handoff path when enabled and past threshold; default off; invalid config fails closed.

### Out of scope / Non-Goals

- Always-on watchdog
- Separate MCP tool for watchdog
- Fact Memory

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-handoff-resume

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/relay/watchdog.go`
- Test: `skillgrid-cli/internal/mnemonic/relay/watchdog_test.go`

**Interfaces:**
- Consumes: `Relay.Handoff` from 02
- Produces: Watchdog Check → Handoff when `SKILLGRID_HANDOFF_WATCHDOG` + threshold satisfied; documented usage-signal choice (client `%` vs token estimate)

### Tasks

- [ ] 05.1 `[AFK]` Decide usage signal (client `%` vs token estimate); document choice in `watchdog.go` comment; Interface takes a fraction either way
- [ ] 05.2 `[RED]` Enabled watchdog past threshold triggers same Handoff path — Scenario: Enabled watchdog past threshold
  - [ ] 05.2.a Write failing test
  - [ ] 05.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog'` — Expected: FAIL
  - [ ] 05.2.c Minimal implementation — flag/env-gated (`SKILLGRID_HANDOFF_WATCHDOG` + threshold); off by default; Check → same `Handoff`
  - [ ] 05.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog'` — Expected: PASS
  - [ ] 05.2.e Commit — `feat(mnemonic): optional session handoff watchdog`
- [ ] 05.3 `[AFK]` Disabled/default or below threshold is no-op — Scenario: Disabled or below threshold is no-op — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog.*(Disabled|Below|Default)'` — Expected: PASS
- [ ] 05.4 `[AFK]` Invalid config fails closed — Scenario: Invalid watchdog config fails closed — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog.*(Invalid|Config)'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog'` | PASS | | |
| Acceptance `@step-05` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/relay/` | PASS | | |
| Rollback boundary | unset env/flag → never auto-handoff | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): optional handoff watchdog`

---

