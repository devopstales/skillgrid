## 01-relay-schema

### Goal

Migrations create `session_handoffs` and `session_archives` on store open without rewriting existing sessions/observations.

### Out of scope / Non-Goals

- Relay FS, MCP tools, CLI, watchdog
- Any migration slot other than `012_*`

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/012_session_relay.sql`
- Modify: (none required beyond migrate registration if auto-discovered)
- Test: migrate / store open test under `skillgrid-cli/internal/mnemonic/store/` (e.g. `upgrade_test.go` or sibling)

**Interfaces:**
- Consumes: none
- Produces: `session_handoffs`, `session_archives` tables present after `store.Open`

### Tasks

- [ ] 01.1 `[AFK]` Create `012_session_relay.sql` with additive `session_handoffs` and `session_archives` (leave `009_*`/`010_*`/`011_*` for 001/003/004)
- [ ] 01.2 `[RED]` Store open creates handoff/archive tables without rewriting sessions/observations — Scenario: Store open creates handoff tables
  - [ ] 01.2.a Write failing test
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation (ensure migration applies on open)
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): add 012 session relay schema`
- [ ] 01.3 `[AFK]` Re-open idempotent; prior rows survive — Scenarios: Re-open is idempotent; Prior rows survive migration — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff\|SessionRelay\|012'` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/store/` | PASS | | |
| Rollback boundary | drop/skip `012_*` leaves prior tables | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): session relay schema migration`

---

