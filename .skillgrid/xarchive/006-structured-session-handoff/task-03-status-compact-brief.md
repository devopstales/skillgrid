## 03-status-compact

### Goal

`session_status` reports handoff count and last cost/context; thin `knowledge_compact` refreshes `KNOWLEDGE.md` without Fact Memory; tools register via step-02 hook without re-editing `server.go`.

### Out of scope / Non-Goals

- CLI, watchdog
- Fact Memory integration / full plan-06 compact

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-handoff-resume

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/relay/status.go`
- Create: `skillgrid-cli/internal/mnemonic/relay/compact.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_session_status.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server_test.go`
- Test: relay + mcp tests

**Interfaces:**
- Consumes: registrar hook + `Relay` from 02; handoff rows from 01
- Produces: `Relay.Status` / `Relay.CompactKnowledge`; MCP `session_status` → `{handoff_count, context_usage_percent?, cost_usd?}`; `knowledge_compact` → `{knowledge_path}`

### Tasks

- [ ] 03.1 `[RED]` Mnemonic tool surface: `session_status` and `knowledge_compact` registered; `mem_save` still works; compact succeeds with **no** Fact Memory — Scenario: New tools leave mem_save intact
  - [ ] 03.1.a Write failing test in `server_test.go`
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionStatus|KnowledgeCompact|MemSave'` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `tools_session_status.go` via step-02 registrar hook **without** re-editing `server.go`
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionStatus|KnowledgeCompact|MemSave'` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): register session_status and knowledge_compact`
- [ ] 03.2 `[AFK]` Create `status.go` — Status aggregation (handoff count + last known cost/context when caller supplies stats)
- [ ] 03.3 `[AFK]` Create `compact.go` — thin `CompactKnowledge` refreshes `.cleave/KNOWLEDGE.md` from handoff inputs/session notes only
- [ ] 03.4 `[AFK]` Cover WHAT happy — Scenario: Status and compact without facts — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Status|Compact'` — Expected: PASS
- [ ] 03.5 `[AFK]` Cover WHAT edge — Scenario: No handoffs yet — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Status|Compact|Empty'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Status\|Compact'` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/mcp/` | PASS | | |
| Rollback boundary | remove status/compact tools; handoff/resume remain | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): session status and thin knowledge compact`

---

