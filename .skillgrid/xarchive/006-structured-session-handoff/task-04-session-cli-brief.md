## 04-session-cli

### Goal

`skillgrid session handoff|resume|status` mirrors MCP outcomes on the same project store; bad input fails closed.

### Out of scope / Non-Goals

- Watchdog
- New MCP tools
- Compact CLI subcommand (MCP `knowledge_compact` only in this change unless already covered via status path)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-handoff-resume, 03-status-compact

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/session.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/cmd/skillgrid/*_test.go` or integration harness

**Interfaces:**
- Consumes: `Relay` Module / project store from 02–03
- Produces: CLI `skillgrid session handoff|resume|status` exit codes and stdout/stderr

### Tasks

- [ ] 04.1 `[RED]` CLI mirrors MCP on same store — Scenario: CLI mirrors MCP on the same store
  - [ ] 04.1.a Write failing test
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session'` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation — `session.go` + `main.go` dispatch calling same `relay` Module
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session'` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(cli): add skillgrid session subcommands`
- [ ] 04.2 `[AFK]` Bad flags / missing id — Scenario: Bad flags or missing id — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session.*(Bad|Missing|Flag)'` — Expected: PASS
- [ ] 04.3 `[AFK]` No usable store fails closed — Scenario: CLI fails closed without a store — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session.*(Store|Fail)'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session'` | PASS | | |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/cmd/skillgrid/` | PASS | | |
| Rollback boundary | remove `session` dispatch; MCP path unchanged | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(cli): session handoff resume status`

---

