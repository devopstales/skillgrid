# Tasks: 006-structured-session-handoff

> **STATUS:** `complete` (2026-09-10) — 5/5 steps PASS · verify PASS (16/16 scenarios, -race) · QA waived · archived `a04ca0b`
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

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
phase: archive          # spec | apply | verify | archive
current_step: 05-handoff-watchdog
status: done  # in_progress | blocked | done
updated: 2026-09-10T22:00:00+02:00
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

- [x] 01.1 `[AFK]` Create `012_session_relay.sql` with additive `session_handoffs` and `session_archives` (leave `009_*`/`010_*`/`011_*` for 001/003/004)
- [x] 01.2 `[RED]` Store open creates handoff/archive tables without rewriting sessions/observations — Scenario: Store open creates handoff tables
  - [x] 01.2.a Write failing test
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: FAIL
  - [x] 01.2.c Minimal implementation (ensure migration applies on open)
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): add 012 session relay schema`
- [x] 01.3 `[AFK]` Re-open idempotent; prior rows survive — Scenarios: Re-open is idempotent; Prior rows survive migration — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'Handoff|SessionRelay|012'` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` | PASS | PASS | TestSessionRelayMigration: open-creates-tables + prior-rows-survive + re-open-idempotent; RED real (relay tables absent pre-migration) |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | Store open creates handoff tables; Re-open idempotent; Prior rows survive migration |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/store/` | PASS | PASS | store suite `ok` |
| Rollback boundary | drop/skip `019_*` leaves prior tables | PASS | PASS | 019 is pure `CREATE TABLE/INDEX IF NOT EXISTS`; seeded 001-era sessions+observations rows survive (count + content intact); no ALTER/DROP/rewrite |
| Global Constraints | — | held | held | additive SQL only; store open/migrate additive; no rewrite of sessions/observations; schema-only (no FS/tools/CLI/watchdog); migration 019 (brief's 012 was taken by 008) |

Review: task reviewer `approved` (clean). The only nit (unclosed raw `*sql.DB` handle) was a false positive — the handle IS closed after seeding (`db.Close()` at session_relay_schema_test.go:44). The app-enforced archive→handoff link (scoped `UNIQUE(project, handoff_id)`, no hard FK) is correct given SQLite's FK-target restriction (handoff_id uniqueness is scoped, not a PK/UNIQUE a hard FK could target) and the "no hard SQL dependency" constraint.

Commits (step 01): 53cc95b (019_session_relay.sql — `session_handoffs` + `session_archives`, additive; + store test). Migration slot 019 (brief's 012 was taken by `012_community_knowledge_graph.sql` from 008).

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

- [x] 02.1 `[RED]` Mnemonic tool surface: `session_handoff` and `session_resume` registered **and** `mem_save` still registered/dispatches — Scenario: Fail closed and mem tools remain
  - [x] 02.1.a Write failing test in `server_test.go`
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionHandoff|SessionTools|MemSave'` — Expected: FAIL
  - [x] 02.1.c Minimal implementation — `tools_session_handoff.go` + registrar hook from `server.go` without dropping `mem_*`
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionHandoff|SessionTools|MemSave'` — Expected: PASS
  - [x] 02.1.e Commit — `feat(mnemonic): register session_handoff and session_resume`
- [x] 02.2 `[AFK]` Create `cleave.go` — WriteBundle/ReadBundle for `.skillgrid/.cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md`; soft-optional L0 under `.skillgrid/workspace/sessions/{id}/`
- [x] 02.3 `[AFK]` Create `relay.go` — `Handoff` / `Resume` writing SQL rows + cleave files; fail closed; clear errors for missing `.cleave/` / unknown id
- [x] 02.4 `[AFK]` Modify `.gitignore` to ignore `.skillgrid/.cleave/` by default
- [x] 02.5 `[AFK]` Cover WHAT happy path — Scenario: Handoff writes cleave bundle and row — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Handoff|Resume'` — Expected: PASS
- [x] 02.6 `[AFK]` Cover WHAT edge — Scenario: Missing cleave or unknown handoff id — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Resume|Missing|Unknown'` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | PASS | relay (Handoff/Resume/fail-closed/soft-L0/missing/unknown/archive) + mcp (session tools registered + mem_save dispatches + bad args) |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | Handoff writes cleave bundle and row; Missing cleave or unknown handoff id; Fail closed and mem tools remain |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/mcp/` | PASS | PASS | mcp suite `ok` |
| Rollback boundary | remove session tools; `mem_*` still work | PASS | PASS | registerSessionTools is additive; expectedMemToolSurface re-asserted (78→80, all mem_* names+required params pinned); TestMemSaveDispatch drives a real dispatch |
| Global Constraints | — | held | held | fail-closed no-orphan (file-before-row, EACCES-forced test); resume aborts on missing .cleave/unknown id (no invented prompt); .cleave/ gitignored; soft-optional L0 degrades; session-relay-only (no 002/003/004/005 deps) |

Review: task reviewer `approved with fixes`. Fix commit ef53f61 (the Resume archive path swallowed the follow-up `UPDATE session_handoffs SET status='archived'` with `_, _ =` — now surfaces a clear error "relay: archived but failed to mark handoff archived" + new `TestResumeArchiveStatusFlipFails`; happy path `TestResume` asserts the status flips to `archived` + archive row exists). Other nits (redundant `.skillgrid/.cleave/` gitignore line — `.skillgrid/` already ignored; root-account fragility of the EACCES test) are harmless/non-blocking.

Commits (step 02): 29feb6e (relay cleave bundle + handoff/resume fail-closed), b72ecb8 (register session_handoff/session_resume, 78→80 surface), 174eeb4 (gitignore .skillgrid/.cleave/), ef53f61 (surface failed archive status flip).

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

- [x] 03.1 `[RED]` Mnemonic tool surface: `session_status` and `knowledge_compact` registered; `mem_save` still works; compact succeeds with **no** Fact Memory — Scenario: New tools leave mem_save intact
  - [x] 03.1.a Write failing test in `server_test.go`
  - [x] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionStatus|KnowledgeCompact|MemSave'` — Expected: FAIL
  - [x] 03.1.c Minimal implementation — `tools_session_status.go` via step-02 registrar hook **without** re-editing `server.go`
  - [x] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'SessionStatus|KnowledgeCompact|MemSave'` — Expected: PASS
  - [x] 03.1.e Commit — `feat(mnemonic): register session_status and knowledge_compact`
- [x] 03.2 `[AFK]` Create `status.go` — Status aggregation (handoff count + last known cost/context when caller supplies stats)
- [x] 03.3 `[AFK]` Create `compact.go` — thin `CompactKnowledge` refreshes `.cleave/KNOWLEDGE.md` from handoff inputs/session notes only
- [x] 03.4 `[AFK]` Cover WHAT happy — Scenario: Status and compact without facts — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -run 'Status|Compact'` — Expected: PASS
- [x] 03.5 `[AFK]` Cover WHAT edge — Scenario: No handoffs yet — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Status|Compact|Empty'` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1` | PASS | PASS | relay (Status zero-count/optional-stats, Compact no-fact-memory/empty→minimal, ctx threaded) + mcp (session_status/knowledge_compact registered + mem_save dispatch + bad args) |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | Status and compact without facts; No handoffs yet (zero counts, no crash); New tools leave mem_save intact |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/mcp/` | PASS | PASS | mcp suite `ok` |
| Rollback boundary | remove status/compact tools; handoff/resume remain | PASS | PASS | tools_session_status.go is additive (appended to registerSessionTools slice); 80→82 surface lock; handoff/resume tools unchanged |
| Global Constraints | — | held | held | thin compact (no Fact Memory, only session_handoffs.context_summary + bundle); status zero-counts on empty (no crash); empty inputs → minimal KNOWLEDGE.md (warn+continue); registered via step-02 registrar hook (server.go NOT re-edited); 82-tool surface additive; session-relay-only |

Review: task reviewer `approved` (clean). Fix commit 95a280b (compact.go `contextNotes` used `context.Background()` and dropped the caller's `_ = ctx` — now threads the caller's `ctx` through `CompactKnowledge`→`gatherKnowledge`→`contextNotes`→`QueryContext`). The `QueryContext` addition to the `relay.Store` interface is purely additive (`*sql.DB`-compatible, no breaking implementer). Bad-args on the two new tools is covered by the handler arg-validation path (no dedicated case, non-blocking).

Commits (step 03): 3cad7e7 (session_status + thin knowledge_compact, registrar hook, 80→82), 95a280b (thread caller ctx through compact query).

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

- [x] 04.1 `[RED]` CLI mirrors MCP on same store — Scenario: CLI mirrors MCP on the same store
  - [x] 04.1.a Write failing test
  - [x] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session'` — Expected: FAIL
  - [x] 04.1.c Minimal implementation — `session.go` + `main.go` dispatch calling same `relay` Module
  - [x] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session'` — Expected: PASS
  - [x] 04.1.e Commit — `feat(cli): add skillgrid session subcommands`
- [x] 04.2 `[AFK]` Bad flags / missing id — Scenario: Bad flags or missing id — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session.*(Bad|Missing|Flag)'` — Expected: PASS
- [x] 04.3 `[AFK]` No usable store fails closed — Scenario: CLI fails closed without a store — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'Session.*(Store|Fail)'` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/cmd/skillgrid/ -count=1` | PASS | PASS | Session mirrors-MCP (subprocess CLI + in-process MCP on one shared store), bad-flag/missing-id (exit 2 + stderr), no-store (exit 1 + stderr + no partial bundle) |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | CLI mirrors MCP on the same store; Bad flags or missing id; CLI fails closed without a store |
| Runtime harness | `go test ./skillgrid-cli/cmd/skillgrid/` | PASS | PASS | cmd/skillgrid suite `ok` |
| Rollback boundary | remove `session` dispatch; MCP path unchanged | PASS | PASS | session.go is a new caller of the relay Module; main.go only registers the dispatch; the only mcp/ touch is two pure test aliases (no behavior change, no new tool) |
| Global Constraints | — | held | held | mirrors MCP (identical relay.Handoff/Resume/Status + identical store resolution, MNEMONIC_PROJECT override); fail-closed (non-zero exit + stderr + no partial cleave); no watchdog; no compact CLI subcommand (MCP-only); session-relay-only (no 002/003/004/005) |

Review: task reviewer `approved` (clean). The mirrors-MCP property is proven non-vacuously: a real subprocess CLI drives the shared store, then the in-process MCP handlers read back the CLI-written NEXT_PROMPT / row / status count — CLI and MCP agree in both directions (handoff→MCP resume, resume, status). Fail-closed is asserted as exit-code + message + no-partial-bundle. Two minor (non-blocking) notes: session_test.go uses CombinedOutput so the stderr split isn't independently asserted (correct in the CLI itself, verified by code); reorderSessionArgs is a naive flag/positional splitter (value starting with `-` edge case, consistent with the mem/trail convention).

Commits (step 04): 0a86df3 (session.go + main.go dispatch + session_test.go; two pure MCP test aliases).

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

- [x] 05.1 `[AFK]` Decide usage signal (client `%` vs token estimate); document choice in `watchdog.go` comment; Interface takes a fraction either way
- [x] 05.2 `[RED]` Enabled watchdog past threshold triggers same Handoff path — Scenario: Enabled watchdog past threshold
  - [x] 05.2.a Write failing test
  - [x] 05.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog'` — Expected: FAIL
  - [x] 05.2.c Minimal implementation — flag/env-gated (`SKILLGRID_HANDOFF_WATCHDOG` + threshold); off by default; Check → same `Handoff`
  - [x] 05.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog'` — Expected: PASS
  - [x] 05.2.e Commit — `feat(mnemonic): optional session handoff watchdog`
- [x] 05.3 `[AFK]` Disabled/default or below threshold is no-op — Scenario: Disabled or below threshold is no-op — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog.*(Disabled|Below|Default)'` — Expected: PASS
- [x] 05.4 `[AFK]` Invalid config fails closed — Scenario: Invalid watchdog config fails closed — `Run: go test ./skillgrid-cli/internal/mnemonic/relay/ -run 'Watchdog.*(Invalid|Config)'` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/relay/ -count=1` | PASS | PASS | Watchdog (enabled+past-threshold → 1 handoff row + 3 cleave files via the SAME Relay.Handoff), Disabled/Default/Below (no row, no bundle), Invalid config (clear error, no auto-handoff) |
| Acceptance `@step-05` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | Enabled watchdog past threshold; Disabled or below threshold is no-op; Invalid watchdog config fails closed |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/relay/` | PASS | PASS | relay suite `ok` |
| Rollback boundary | unset env/flag → never auto-handoff | PASS | PASS | `isOff` is the FIRST statement in Check — env unset → no-op before any threshold/config/Handoff code; library-only (no live always-on call site); even if wired later the env gate holds |
| Global Constraints | — | held | held | off by default (never always-on); disabled/default/below-threshold → no-op; invalid threshold (non-numeric, <0, >1) → fail closed (error before any handoff, no fall-through); same Relay.Handoff (not a re-impl); usage = caller-supplied fraction (dependency-free, documented); no separate MCP tool; no Fact Memory; session-relay-only |

Review: task reviewer `approved` (clean). The threshold comparison is `usage < threshold → no-op` (i.e. at/exactly-threshold triggers, consistent with the "at/past" wording + test). Fix commit 9409c34 (the enable-gate doc comment understated the falsy-set check — now documents `""/0/off/false/no` case-insensitive/whitespace-trimmed as the disable set). One cosmetic note (non-blocking): the empty-threshold error prints the env name rather than the operator's value; the message still names the required `threshold in [0,1]`.

Commits (step 05): d529769 (watchdog.go + watchdog_test.go), 9409c34 (doc comment).

### Commit

When step DoD is met: `feat(mnemonic): optional handoff watchdog`

---

## Change-level Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

**All 16 `@step-NN` scenarios COMPLIANT at runtime** (step-01: 3, step-02: 4, step-03: 3, step-04: 3, step-05: 3). No CRITICAL findings. All Global Constraints held. No unchecked tasks.

| Step | Verdict | Scenarios (COMPLIANT at runtime) | Runtime proof |
|------|---------|----------------------------------|---------------|
| 01 relay-schema | PASS | Store open creates handoff tables; Re-open idempotent; Prior rows survive migration | `store` `TestSessionRelayMigration` — ok (-race) |
| 02 handoff-resume | PASS | Handoff writes cleave bundle and row; Resume with optional archive; Missing cleave or unknown handoff id; Fail closed and mem tools remain | `relay` (Handoff/Resume/FailClosedNoOrphan/SoftOptionalL0/ResumeMissing/ResumeUnknown/ResumeArchive) + `mcp` (session tools + mem_save) — ok (-race) |
| 03 status-compact | PASS | Status and compact without facts; No handoffs yet; New tools leave mem_save intact | `relay` (Status/StatusEmpty/Compact/CompactEmpty) + `mcp` (session_status/knowledge_compact + mem_save) — ok (-race) |
| 04 session-cli | PASS | CLI mirrors MCP on the same store; Bad flags or missing id; CLI fails closed without a store | `cmd/skillgrid` (subprocess CLI + in-process MCP on one shared store; exit codes + stderr + no partial bundle) — ok (-race) |
| 05 handoff-watchdog | PASS | Enabled watchdog past threshold; Disabled or below threshold is no-op; Invalid watchdog config fails closed | `relay` (Watchdog / Disabled+Default+Below / Invalid) — ok (-race) |

Runtime suite (change-affected packages, `-race`):

| Package | Command | Result |
|---------|---------|--------|
| store | `go test -race ./internal/mnemonic/store/ -count=1` | ok (24.5s) |
| relay | `go test -race ./internal/mnemonic/relay/ -count=1` | ok (21.3s) |
| mcp | `go test -race ./internal/mnemonic/mcp/ -count=1` | ok (204.2s) |
| cmd/skillgrid | `go test -race ./cmd/skillgrid/ -count=1` | ok (89.2s) |

Additive regression (005/008/010/011/013 baselines, `-race`): memory, route, affected, community, codeindex `ok`. `pdg` had ONE pre-existing flaky failure — `TestResolveMemberCallsHonorsBoundedCtx` (an 011 wall-clock 40ms LSP-timeout test) timed out under `-race` in the batch run but PASSES cleanly when re-run alone (0.30s / 0.04s) and in the no-race full pdg suite (`ok`). It is an 011 artifact, NOT touched by 006 (006 only ADDs migration 019 + the `relay` package + session MCP tools + `session` CLI). Additive migrations 017/018/019 and the 78→82 tool surface do not regress any baseline.

Global Constraints held across all 5 steps (verified per-step + at runtime): session-relay-only (no 002/003/004/005/Fact-Memory deps); additive SQL (019 `CREATE TABLE IF NOT EXISTS` only, prior rows survive); fail-closed (no orphan handoff row without files; bad CLI input / no store → non-zero exit + stderr + no partial bundle; resume missing `.cleave/`/unknown id → clear error, no invented prompt); `.cleave/` gitignored; soft-optional L0 degrades; thin `knowledge_compact` (no Fact Memory); status zero-counts on empty; watchdog off-by-default (never always-on, fail-closed on invalid config); additive 82-tool surface (all `mem_*` names + required params pinned).

Review: per-step task reviewers (01 approved, 02 approved-with-fixes→fixed, 03 approved→fixed, 04 approved, 05 approved→doc-fix). All findings closed.

## Archive gate checklist

- [x] Change-level **Definition of Done** fully checked
- [x] No unchecked `- [ ]` under any `### Tasks`
- [x] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] No Global Constraint violated
- [x] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [x] STATUS banner updated to `complete`
