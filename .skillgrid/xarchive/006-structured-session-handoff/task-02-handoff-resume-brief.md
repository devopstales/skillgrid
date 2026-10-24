## 02-handoff-resume

### Goal

Cleave FS + MCP `session_handoff` / `session_resume` write the three files and SQL row, return a next-session prompt, and fail closed without orphan rows; soft-optional L0; `.cleave/` gitignored.

### Out of scope / Non-Goals

- Status/compact tools, CLI, watchdog
- Fact Memory

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-relay-schema

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/relay/relay.go`
- Create: `skillgrid-cli/internal/mnemonic/relay/cleave.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Modify: `.gitignore`
- Test: `skillgrid-cli/internal/mnemonic/mcp/server_test.go`, `skillgrid-cli/internal/mnemonic/relay/*_test.go`

**Interfaces:**
- Consumes: `session_handoffs` / `session_archives` from 01
- Produces: `Relay.Handoff` / `Relay.Resume`; MCP `session_handoff` → `{handoff_id, paths}`; `session_resume` → `{prompt, handoff_id, archive_id?}`; registrar hook for step 03

### Tasks

- [ ] 02.1 `[RED]` Mnemonic tool surface: `session_handoff` and `session_resume` registered **and** `mem_save` still registered/dispatches — Scenario: Fail closed and mem tools remain
  - [ ] 02.1.a Write failing test in `server_test.go`
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionHandoff|SessionTools|MemSave'` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — `tools_session_handoff.go` + registrar hook from `server.go` without dropping `mem_*`
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionHandoff|SessionTools|MemSave'` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): register session_handoff and session_resume`
- [ ] 02.2 `[AFK]` Create `cleave.go` — WriteBundle/ReadBundle for `.skillgrid/.cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md`; soft-optional L0 under `.skillgrid/workspace/sessions/{id}/`
- [ ] 02.3 `[AFK]` Create `relay.go` — `Handoff` / `Resume` writing SQL rows + cleave files; fail closed; clear errors for missing `.cleave/` / unknown id
- [ ] 02.4 `[AFK]` Modify `.gitignore` to ignore `.skillgrid/.cleave/` by default
- [ ] 02.5 `[AFK]` Cover WHAT happy path — Scenario: Handoff writes cleave bundle and row — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Handoff|Resume'` — Expected: PASS
- [ ] 02.6 `[AFK]` Cover WHAT edge — Scenario: Missing cleave or unknown handoff id — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Resume|Missing|Unknown'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Handoff\|Resume\|Session'` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/mcp/` | PASS | | |
| Rollback boundary | remove session tools; `mem_*` still work | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): session handoff and resume`

---

