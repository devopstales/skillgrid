# Tasks: 006-structured-session-handoff

> **STATUS:** `in-progress` (2026-09-04) — 0/5 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Ship Cleave-style Session Relay so agents and operators can hand off and resume work across sessions without re-deriving state.

**Architecture:** Deep `relay` module owns SQL + `.cleave/` FS; MCP/CLI are thin callers. Soft-after 003 L0 only. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), SQLite, MCP (`mcp-go`), `.skillgrid/.cleave/` FS; optional env/flag-gated watchdog.

**Spec:** `docs/skillgrid/changes/006-structured-session-handoff/change.md`

**Acceptance:** `docs/skillgrid/changes/006-structured-session-handoff/acceptance.feature` (`@step-NN`)

---

## Goal

Agents and operators get a Session Relay: explicit handoff writes a cleave bundle and SQL row; resume returns a next-session prompt; status and thin knowledge compact keep continuity without Fact Memory; CLI and optional watchdog drive the same path.

## Out of scope / Non-Goals

- Fact Memory / Agent Skills (**004**) — thin `knowledge_compact` must not depend on facts
- **003** Tiered Storage core, `semantic_search`, `mnemonic_commit`, trail CLI
- **002** identity / **005** code intelligence
- OpenCode plugin; full plan-06 rewrite
- Stretching Long-term Memory / replacing Engram `mem_session_*`
- Always-on watchdog (must stay off by default)

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- Session Relay only — not Fact Memory / Agent Skills (**004**), not **003** Tiered Storage / `semantic_search` / `mnemonic_commit` / trail CLI, not **002**/**005**, not OpenCode plugin
- Do not stretch Long-term Memory or replace Engram `mem_session_*`
- Additive SQL only via `012_*`; leave `009_*`/`010_*`/`011_*` for 001/003/004
- Soft-after **003** L0 paths only; no hard SQL dependency on `010_*`
- Thin `knowledge_compact` refreshes `.cleave/KNOWLEDGE.md` only — no Fact Memory dependency
- Watchdog off by default; never always-on in this change
- `.skillgrid/.cleave/` gitignored by default
- Store open / migrate fails → `abort` with clear migrate error; do not rewrite sessions/observations
- Handoff cannot write cleave files → `abort`; fail closed — no orphan `session_handoffs` row without files
- Resume missing `.cleave/` or unknown handoff id → `abort` with clear error; do not invent prompt content
- Status with no handoffs yet → `warn+continue`; zero counts; not a crash
- Compact with empty/missing knowledge inputs → `warn+continue`; empty or minimal `KNOWLEDGE.md`
- CLI bad flags / missing id / no usable store → `abort`; non-zero exit; stderr; no partial cleave bundle
- Watchdog disabled or below threshold → no-op (never auto-handoff)
- Invalid watchdog threshold / flag → `abort`; fail closed — no auto-handoff; clear config error

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-relay-schema
status: in_progress  # in_progress | blocked | done
updated: 2026-09-04T22:00:00+02:00
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `relay-schema` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `handoff-resume` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `status-compact` | `@step-03` | 02 | Feature tagged `@step-03` |
| 04 | `session-cli` | `@step-04` | 02, 03 | Feature tagged `@step-04` |
| 05 | `handoff-watchdog` | `@step-05` | 02 | Feature tagged `@step-05` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~900–1400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

Honest notes: five vertical steps across migrations, new `relay` package, MCP tools (×2 files + registrar), CLI, and optional watchdog. Single PR would likely blow the 400-line review budget; prefer chained PRs per step (or 01+02, then 03, then 04, then 05) after human risk gate.

---

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

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
