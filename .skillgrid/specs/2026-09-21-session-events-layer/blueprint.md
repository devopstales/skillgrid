# Session/Checkpoint Layer Consolidation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 (project default; briefing carries no `Tier:` line and config has no `rules.tiers.default`)

**Build shape:** Tracer thread (the session→events write→read path works end-to-end first — migration + session_start event + SessionChanges — then capture widens, then the old layer is removed)

**Goal:** Replace `checkpoint.json` + the Handoff Hub with a single session→events layer where a session's changes are its ordered events.

**Architecture:** Extend the existing `sessions` table (commit range + gryph counters), add an append-only `session_events` stream keyed `(session_id, sequence)`, capture via the existing `RunHook` seam plus a canonical ingest endpoint, derive "what changed" on demand (`SELECT … ORDER BY sequence` + `git diff from..to`), then delete the old layer. Git stays the durable record; the DB is the index.

**Tech Stack:** Go 1.26 (`skillgrid-cli`, `modernc.org/sqlite` via existing store), SQLite migrations (`go:embed migrations/*.sql`, applied in filename order), bash (hook shims), existing MCP/CLI/HTTP surfaces.

**Spec:** `.skillgrid/specs/2026-09-21-session-events-layer/briefing.md`

**Class:** `standard` (additive migration + deletions + contract removals; verification floor L2 — full suite + build. One-way steps carry explicit STOPs below.)

## Already complete (do NOT re-plan; context only)

- Repo hook layout: `hooks/` (9 impl) + `git-hooks/` (4 shims, `../hooks/`-relative) + `plugins/` (opencode/kilo `mnemonic.ts`, cursor `mnemonic.mdc`).
- `skillgrid install` / `SyncRepo` mirror `$REPO/*` recursively → `~/.skillgrid/` (`planMirror` + `syncFullTree`, exclusions: `.git`/`node_modules` any depth; top-level `.skillgrid`/`dist`/`out`; never touch `mnemonic`/`repos`/`backup`/`bin`/`tmp`/`logs`/`config.d`); global `core.hooksPath` → `~/.skillgrid/git-hooks`.
- Staged-first plugin reads (`install/agents.go` `copyAsset`, `mnemonic/setup` `stagedPluginPath`).
- Hook-path reference updates across skills + living docs.

## Hypothesis

**Claim:** A session→events stream replaces `checkpoint.json` + the Handoff Hub with no resume capability loss.
**Right condition:** `SessionChanges(sid)` returns every tool call of the session in order plus the net `git diff from_commit..to_commit`, and the resume skill recovers position from it alone.
**Wrong condition:** Events are missing, out of order, or resume needs data only the Hub held (named checkpoints, cleave bundles).
**Thinnest MVP:** Migration 040 + `session_start`/`session_end` events + `SessionChanges` (Tasks 0–2).
**Door check:** Task 2 (tracer thread green) — if session identity or ordering fails, STOP and revise.

## Must-Haves (goal-backward verification)

**Truths:**
- `SessionStart` inserts a `sessions` row + `session_start` event (sequence 0) with `from_commit` = `git HEAD`.
- Every tool call appends one `session_events` row with the next per-session `sequence` and bumps the matching `sessions` counter (**backstop** — needs the live hook path; held-out test: two `RunHook post_tool_use` calls → sequences 1, 2, counters incremented).
- `SessionEnd` writes `session_end` + `to_commit` + `ended_at`; `SessionChanges(sid)` returns the full ordered stream and the net diff.
- Paths matching the sensitive set flag `is_sensitive` and bump `sensitive_actions`; file content is stored as hash/preview only, never full content (**backstop** — needs a real write event through the matcher).
- Work-unit commits append a `commit` event whose payload holds the parsed `[skillgrid-context]` block.
- Old surfaces are absent: `skillgrid handoff` is an unknown subcommand; the MCP tool list contains no `handoff_*`, `session_handoff`, `session_resume`, `session_status`, `knowledge_compact`; `checkpoint-state.sh snapshot|restore` no longer exist; no `checkpoint.json` is written.
- `.skillgrid/sdd/<YYYY-MM-DD-topic>/` holds `progress.md` (line 1 names the plan file), `task-N-brief.md`, `task-N-report.md`, `review-<base7>..<head7>.diff`.

**Artifacts:**
- `skillgrid-cli/internal/mnemonic/store/migrations/040_session_events.sql` (+ schema test)
- `skillgrid-cli/internal/mnemonic/store/migrations/041_drop_handoff_tables.sql` (after all readers gone)
- `SessionChanges` read path + `skillgrid session <id> [--show-diff]` + MCP read tool
- `.skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature` (Task 0), `blueprint.md` (this file)
- Repointed skills (14 files) + `sdd-structure.md` + regenerated site partial

**Key links:**
- `commit` event payload carries the parsed `[skillgrid-context]` (Task/Decisions/Remaining/Tried) — the Hub's `context_json`, now session-scoped.
- `session_events.sequence` is assigned max+1 in the same transaction as the counter bump (parallel-subagent safety).
- Consumers read plugins/hooks from the `~/.skillgrid/` mirror (already wired).

**One-way-door decisions:**
- Migration 041 drops five tables (`change_snapshots`, `checkpoints`, `handoff_refs`, `session_handoffs`, `session_archives`) — reconstructable from git log, but the Hub UI history goes away.
- Removing MCP tools (`handoff_*`, `session_handoff`, `session_resume`, `session_status`, `knowledge_compact`) breaks existing callers (UI panes, skills, any external MCP client).
- Removing `checkpoint-state.sh snapshot|restore` breaks the current resume skill until it is repointed (same change, Task 9 ordering covers it).

## Global Constraints

- Fail-open: a crashed/timed-out capture hook never blocks the agent (try/finally or `|| true`, short timeout; Cursor `failClosed` stays false).
- Sensitive content stored as hash/preview only, never full content (gryph redaction rule).
- `sequence` max+1 and counter bump happen in one transaction.
- Conventional commits, no AI-attribution trailers (repo commit-msg guard enforces).
- Spec-zone before code-zone per commit (zone guard enforces); never mix spec + code in one commit.
- Observe-only: `preToolUse`/`tool.execute.before` adapters log but return no deny/permission shape.
- Go 1.26; migrations are `CREATE … IF NOT EXISTS` / additive-first per repo idiom.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`mem_*`, `handoff_*`, `session_*`) | Applicable: removing 9 tools, adding 1 read tool | Explicit contract delta in Task 7; UI panes + skill call-sites updated in the same wave | MCP tool-list test asserting absence + presence; one test per removed tool's former callers |
| Shared-convention drift | Applicable: `sdd-structure.md` rewrite rebinds execution/resume/ship skills | Impact line in Task 9; name every bound skill | Grep check: no `.agents/hooks`, `checkpoint.json`, `handoff verify/archive` refs remain |
| Git repository selection | Applicable: `from_commit`/`to_commit` via `git rev-parse`, hook `cwd` authority | `cwd` comes from the hook payload; best-effort empty outside a repo | Non-repo cwd yields empty commits, no error; worktree-root resolution test |
| Commit state | Applicable: commit events parse `[skillgrid-context]` from `git log -1` | Reuse the existing block parser semantics | Commit without a block still yields a row with NULL payload |
| Documentation-like paths | N/A: no executable docs added | — | — |
| Push state / PR commands | N/A: no push/PR automation touched | — | — |

## File Structure

- `skillgrid-cli/internal/mnemonic/store/migrations/040_session_events.sql` — sessions columns + `session_events` table + indexes (Task 1)
- `skillgrid-cli/internal/mnemonic/store/migrations/041_drop_handoff_tables.sql` — drops 039/019 tables (Task 11, last)
- `skillgrid-cli/internal/mnemonic/memory/service.go` — `SessionStart`/`SessionStartByClientID` capture `from_commit` + `session_start` event; `SessionEnd`/`SessionSummary` capture `to_commit` + `session_end` (Tasks 2–3)
- `skillgrid-cli/internal/mnemonic/memory/skills.go` — extend `HookPayload` + `RunHook` with `post_tool_use`; append events + bump counters (Task 4)
- `skillgrid-cli/internal/mnemonic/memory/sensitive.go` (new) — gryph path/credential matcher → `is_sensitive` (Task 4)
- `skillgrid-cli/internal/mnemonic/memory/changes.go` (new) — `SessionChanges(ctx, sessionID)` (Task 5)
- `skillgrid-cli/cmd/skillgrid/session.go` — replace handoff/resume/status with `session <id> [--show-diff]` (Tasks 5, 8)
- `skillgrid-cli/internal/mnemonic/mcp/tools_session_events.go` (new) — read tool; delete `tools_handoff.go`, `tools_session_handoff.go`, `tools_session_status.go` (Tasks 5, 7)
- `skillgrid-cli/cmd/skillgrid/handoff.go` + `internal/mnemonic/handoff/` + `internal/mnemonic/relay/` + `http/handoff.go` + `memory/envelope.go` HandoffRefs — deleted (Tasks 6, 8)
- `skillgrid-ui/.../sessions/HandoffsPane.tsx`, `ChangesPane.tsx`, `api.ts` — removed/repointed (Task 8)
- `hooks/checkpoint-state.sh` — drop `snapshot`/`restore` + hub mirror; keep guards (Task 8)
- `.agents/skills/*` (14 files) — repoint to events model (Task 9)
- `.skillgrid/sdd/` per-plan layout + `sdd-workspace`/`task-brief`/`review-package` repathed (Task 10)
- `.skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature` — scenarios (Task 0)

---

### Task 0: Acceptance scenarios for the events layer

**Files:**
- Create: `.skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature`
- Test: `npx cucumber-js --dry-run` lists the scenarios (BDD gate presence)

**Interfaces:**
- Consumes: briefing.md requirements (nothing else — first task)
- Produces: scenario names every later task's `SATISFIES` references: `session-lifecycle`, `tool-call-stream`, `sensitive-redaction`, `commit-events`, `resume-from-events`, `old-surfaces-removed`, `per-plan-workspace`
- Seam: none (in-process)
- Deletion test: pass-through, delete it (scenarios without implementation fail)
- Adapters: 1 (this file) — justified: it is the traceability oracle, not a seam

**SATISFIES:** (bootstrap — authoring the oracle itself)

- [ ] **Step 1: Write `acceptance.feature` with 7 scenarios**

```gherkin
Feature: Session events layer
  Background:
    Given a project with the mnemonic store migrated to 040

  Scenario: session-lifecycle
    When a session starts in a git repo at commit "abc1234"
    Then the sessions row has from_commit "abc1234" and status "active"
    And a session_start event with sequence 0 exists
    When the session ends at commit "def5678"
    Then the sessions row has to_commit "def5678" and ended_at set
    And a session_end event exists as the last sequence

  Scenario: tool-call-stream
    Given an active session
    When two tool calls complete (a file write then a shell command)
    Then session_events holds file_write at sequence 1 and command_exec at sequence 2
    And sessions.files_written is 1 and sessions.commands_exec is 1

  Scenario: sensitive-redaction
    When a file_write targets "config/.env"
    Then the event has is_sensitive 1 and sessions.sensitive_actions is 1
    And the payload holds a content hash and preview but not the full content

  Scenario: commit-events
    When a work-unit commit carries a [skillgrid-context] block
    Then a commit event holds the sha and the parsed Task/Decisions/Remaining/Tried

  Scenario: resume-from-events
    Given a session with a start event, tool events, and commits
    When SessionChanges is called for that session
    Then every event returns in sequence order with the net from..to diff

  Scenario: old-surfaces-removed
    Then `skillgrid handoff` reports an unknown subcommand
    And the MCP tool list has no handoff_snapshot, handoff_status, handoff_checkpoint, handoff_verify, handoff_rollup, session_handoff, session_resume, session_status, knowledge_compact
    And checkpoint-state.sh has no snapshot or restore subcommand
    And no checkpoint.json is written

  Scenario: per-plan-workspace
    Then .skillgrid/sdd/<topic>/ holds progress.md, task-N briefs/reports, review diffs
    And progress.md line 1 names the plan file
```

- [ ] **Step 2: Confirm the scenarios are RED**

Run: `npx cucumber-js .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature --dry-run`
Expected: 7 scenarios listed (implementation arrives in Tasks 1–10)

- [ ] **Step 3: Commit**

```bash
git add .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
git commit -m "test(sdd): acceptance scenarios for session events layer"
```

---

### Task 1: Migration 040 — sessions range + counters + session_events

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/040_session_events.sql`
- Test: `skillgrid-cli/internal/mnemonic/store/migrations_040_test.go` (follow the existing `*_schema_test.go` pattern: migrate a temp DB, assert tables/columns/indexes)

**Interfaces:**
- Consumes: 001 `sessions` table, 039/019 tables (coexist until Task 11)
- Produces: `sessions.from_commit/to_commit/agent_session_id` + 6 counters; `session_events` table + `UNIQUE(session_id, sequence)` + 2 indexes — every later task's storage contract
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 (schema file) — justified: DDL has no seam

**SATISFIES:** session-lifecycle (storage half)

- [ ] **Step 1: Write the failing schema test**

```go
func TestMigration040SessionEvents(t *testing.T) {
	db := migrateTempDB(t) // helper per existing schema tests: fresh sqlite + Migrate
	for _, col := range []string{"agent_session_id", "from_commit", "to_commit",
		"files_read", "files_written", "commands_exec", "errors",
		"sensitive_actions", "blocked_actions"} {
		if !columnExists(t, db, "sessions", col) {
			t.Errorf("sessions missing column %s", col)
		}
	}
	for _, tbl := range []string{"session_events"} {
		if !tableExists(t, db, tbl) {
			t.Errorf("missing table %s", tbl)
		}
	}
	// UNIQUE(session_id, sequence) enforced: second insert with same pair fails.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/store/ -run TestMigration040SessionEvents -v`
Expected: FAIL (no such table/columns)

- [ ] **Step 3: Write migration `040_session_events.sql`**

```sql
-- 040: session events layer (replaces checkpoint.json + handoff hub 039 + cleave relay 019).
-- Additive only. sessions gains the commit range + gryph counters; session_events
-- is the append-only per-tool-call stream ordered by (session_id, sequence).
ALTER TABLE sessions ADD COLUMN agent_session_id TEXT;
ALTER TABLE sessions ADD COLUMN from_commit TEXT;
ALTER TABLE sessions ADD COLUMN to_commit TEXT;
ALTER TABLE sessions ADD COLUMN files_read INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN files_written INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN commands_exec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN errors INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN sensitive_actions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN blocked_actions INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    result_status TEXT NOT NULL DEFAULT 'success',
    is_sensitive INTEGER NOT NULL DEFAULT 0,
    tool_name TEXT,
    path TEXT,
    command TEXT,
    commit TEXT,
    payload TEXT,
    timestamp TEXT NOT NULL,
    UNIQUE (session_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_events_session_seq ON session_events (session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_events_project_time ON session_events (project, timestamp DESC);
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/store/ -run TestMigration040SessionEvents -v`
Expected: PASS. Also run the full store suite to catch migration-order regressions.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/store/migrations/040_session_events.sql skillgrid-cli/internal/mnemonic/store/migrations_040_test.go
git commit -m "feat(mnemonic): migration 040 sessions range plus session_events"
```

---

### Task 2: Tracer thread — session_start/end events + SessionChanges (door check)

> Door check: if session identity or event ordering fails here, STOP — the blueprint is invalidated.

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (`SessionStart` ~:973, `SessionStartByClientID` ~:912, `SessionEnd` ~:1139, `SessionSummary` ~:1087)
- Create: `skillgrid-cli/internal/mnemonic/memory/changes.go` (`SessionChanges`)
- Test: `skillgrid-cli/internal/mnemonic/memory/changes_test.go`

**Interfaces:**
- Consumes: 040 schema (Task 1); existing `sessions` insert/select helpers
- Produces: `SessionChanges(ctx, sessionID) (events []Event, from, to string, err error)` — Task 5's CLI/MCP read; event-row writer helper used by Tasks 3–4
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 — justified: first writer+reader pair

**SATISFIES:** session-lifecycle, resume-from-events

- [ ] **Step 1: Write the failing test**

```go
func TestSessionStartEndEvents(t *testing.T) {
	svc := newTestService(t) // temp store + temp git repo at known HEAD
	sid, err := svc.SessionStart(ctx, repoDir, "tracer")
	// ...
	evts, from, to, err := svc.SessionChanges(ctx, sid)
	// expect: 1 event (session_start, sequence 0), from == HEAD, to == ""
	if err := svc.SessionEnd(ctx, sid, "done"); err != nil { t.Fatal(err) }
	evts, from, to, err = svc.SessionChanges(ctx, sid)
	// expect: 2 events ordered, to == HEAD, ended_at set
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/memory/ -run TestSessionStartEndEvents -v`
Expected: FAIL (`SessionChanges` undefined)

- [ ] **Step 3: Write minimal implementation**
  - `SessionStart`/`SessionStartByClientID`: after the `sessions` insert, capture `git rev-parse HEAD` in `directory` (best-effort, empty outside a repo) into `from_commit`; insert `session_start` event at sequence 0 with `timestamp` UTC.
  - `SessionEnd`/`SessionSummary`: capture HEAD into `to_commit`, set `ended_at`, insert `session_end` event at max+1.
  - `nextSequence` helper: `SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id=?` — called inside the writer's transaction (Task 4 generalizes this).
  - `changes.go`: `SessionChanges` selects events ordered by sequence plus `from_commit`/`to_commit` from `sessions`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/memory/ -run TestSessionStartEndEvents -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memory/service.go skillgrid-cli/internal/mnemonic/memory/changes.go skillgrid-cli/internal/mnemonic/memory/changes_test.go
git commit -m "feat(mnemonic): session start/end events plus SessionChanges"
```

---

### Task 3: Tool-call events + counters + sensitive matcher

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/skills.go` (`HookPayload` ~:255, `RunHook` ~:384, add `post_tool_use` branch)
- Create: `skillgrid-cli/internal/mnemonic/memory/sensitive.go` (matcher), extend `changes_test.go` (or new `hooks_events_test.go`)
- Test: event-ordering + counter + sensitivity tests

**Interfaces:**
- Consumes: Task 2's event writer + `nextSequence`; existing `hookSessionStart/Stop/PreEdit` handlers stay
- Produces: `post_tool_use` hook → `file_read`/`file_write`/`command_exec`/`tool_use` rows + counter bumps; `isSensitivePath(path) bool`
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 — justified: single hook branch

**SATISFIES:** tool-call-stream, sensitive-redaction

- [ ] **Step 1: Write the failing tests**

```go
func TestPostToolUseAppendsOrderedEvents(t *testing.T) {
	// start session; RunHook("post_tool_use", {File: "a.go", ToolName: "Write"}) twice + once {Command: "go test ./...", ToolName: "Shell"}
	// expect sequences 1,2,3 with action_types file_write,file_write,command_exec
	// expect sessions.files_written==2, sessions.commands_exec==1
}

func TestSensitiveWriteRedacted(t *testing.T) {
	// RunHook post_tool_use with File "config/.env", content "SECRET=abc"
	// expect is_sensitive==1, sensitive_actions==1, payload has hash+preview, not "SECRET=abc"
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mnemonic/memory/ -run 'TestPostToolUse|TestSensitiveWrite' -v`
Expected: FAIL (unknown hook type / no matcher)

- [ ] **Step 3: Write minimal implementation**
  - Extend `HookPayload` with `ActionType, ToolName, Command, ResultStatus, ContentHash, ContentPreview string` fields (JSON-tagged, omitempty).
  - `RunHook` new `post_tool_use` branch: map `ToolName` → `action_type` (Write→file_write, Read→file_read, Shell→command_exec, else tool_use); run `isSensitivePath(path)` (match `.env`, `*.pem`, `*.key`, `*secret*`, `.ssh/`, `.aws/` — gryph set); insert event + bump counter in ONE transaction with `nextSequence`.
  - `sensitive.go`: `isSensitivePath` + `redactPreview` (hash via SHA-256, preview truncated to 200 chars, env-style `KEY=...` values masked).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mnemonic/memory/ -v -run 'TestPostToolUse|TestSensitiveWrite'`
Expected: PASS; then the full `memory` package suite.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memory/skills.go skillgrid-cli/internal/mnemonic/memory/sensitive.go skillgrid-cli/internal/mnemonic/memory/hooks_events_test.go
git commit -m "feat(mnemonic): post_tool_use events, counters, sensitive redaction"
```

---

### Task 4: Commit events from the post-commit flow

**Files:**
- Modify: post-commit caller (`hooks/checkpoint-state.sh` post-commit path or the CLI commit hook — reuse whichever invokes `skillgrid` today; smallest change: extend the existing `skillgrid handoff record` call-site to emit a `commit` event via the new ingest endpoint instead)
- Test: `skillgrid-cli/internal/mnemonic/memory/commit_events_test.go`

**Interfaces:**
- Consumes: Task 2's event writer; existing `[skillgrid-context]` block format (unchanged — still written by work-unit-commits)
- Produces: `commit` events with sha + parsed Task/Decisions/Remaining/Tried payload
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 — justified: one call-site swap

**SATISFIES:** commit-events

- [ ] **Step 1: Write the failing test**

```go
func TestCommitEventParsesContextBlock(t *testing.T) {
	// temp repo, one commit with [skillgrid-context] block; invoke the record path
	// expect session_events row action_type=commit, commit==sha, payload JSON has task/decisions/remaining
	// second commit WITHOUT a block → row with empty payload, no error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mnemonic/memory/ -run TestCommitEventParsesContextBlock -v`
Expected: FAIL (no record path)

- [ ] **Step 3: Write minimal implementation** — parse `git log -1 --pretty=%B` for the block (same field semantics as the old `context_field`), insert `commit` event on the session resolved from the hook payload (fall back to latest active session for the repo when no session id is passed).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mnemonic/memory/ -run TestCommitEventParsesContextBlock -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memory/commit_events_test.go <record-path-file>
git commit -m "feat(mnemonic): commit events with parsed skillgrid-context"
```

---

### Task 5: Read path — CLI `skillgrid session <id>` + MCP tool

**Files:**
- Modify: `skillgrid-cli/cmd/skillgrid/session.go` (add `session <id> [--show-diff]`; the old handoff/resume/status subcommands are deleted in Task 7 — this task only ADDS the new read)
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_session_events.go` (read tool, e.g. `session_changes`)
- Test: CLI test + MCP tool test

**Interfaces:**
- Consumes: `SessionChanges` (Task 2); `git diff --stat from..to` (reuse `handoff/snapshot.go` `changedFiles` logic — copy, don't import the deleted package's future)
- Produces: human + `--json` output; MCP tool contract for the resume skill
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 2 (CLI + MCP over one function) — justified seam use

**SATISFIES:** resume-from-events

- [ ] **Step 1: Write the failing tests** — CLI: `session <id>` prints ordered events; `--show-diff` includes `git diff --stat from..to`; unknown id errors. MCP: tool returns events JSON for a seeded session.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/skillgrid/ -run TestSessionShow ./internal/mnemonic/mcp/ -run TestSessionChangesTool -v`
Expected: FAIL (no subcommand / no tool)

- [ ] **Step 3: Write minimal implementation** — list events in sequence order; `--show-diff` shells `git -C <repoDir> diff --stat from to` (empty range → note, not error).

- [ ] **Step 4: Run tests to verify they pass**

Run: same selectors
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/session.go skillgrid-cli/internal/mnemonic/mcp/tools_session_events.go <test-files>
git commit -m "feat(mnemonic): session show CLI plus session_changes MCP tool"
```

---

### Task 6: Delete the Handoff Hub Go package + CLI command

**Files:**
- Delete: `skillgrid-cli/internal/mnemonic/handoff/` (snapshot, checkpoint, refs, rollup, context, handoff + tests), `skillgrid-cli/cmd/skillgrid/handoff.go` (+`handoff_test.go`, `mem_handoff_test.go`), strip the `handoff` dispatch from `main.go`
- Test: existing hub tests go RED by deletion; add `TestHandoffCommandGone` asserting `skillgrid handoff` exits unknown-subcommand

**Interfaces:**
- Consumes: Tasks 2–5 green (the replacement reads exist)
- Produces: no `handoff` subcommand; no `internal/mnemonic/handoff` importers (verify with `rg`)
- Seam: none
- Deletion test: the task IS a deletion — verify `rg "mnemonic/handoff" skillgrid-cli --glob '*.go'` returns nothing
- Adapters: 1 — justified: pure removal

**SATISFIES:** old-surfaces-removed (CLI half)

- [ ] **Step 1: Write the failing test**

```go
func TestHandoffCommandGone(t *testing.T) {
	// run the CLI argv ["handoff", "status"]; expect exit 2 / unknown-subcommand on stderr
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/skillgrid/ -run TestHandoffCommandGone -v`
Expected: FAIL (command still exists)

- [ ] **Step 3: Delete the package + command + dispatch**

- [ ] **Step 4: Run test to verify it passes** + `go build ./...` + full `cmd/skillgrid` suite

- [ ] **Step 5: Commit**

```bash
git add -A skillgrid-cli/cmd/skillgrid/ skillgrid-cli/internal/mnemonic/handoff/
git commit -m "refactor(mnemonic): delete handoff hub package and CLI"
```

---

### Task 7: Delete Hub MCP tools + session relay tools

> ⚠ one-way: removing 9 published MCP tools breaks existing callers (UI, skills, external clients). STOP for explicit approval before committing this task.

**Files:**
- Delete: `skillgrid-cli/internal/mnemonic/mcp/tools_handoff.go` (5 tools), `tools_session_handoff.go` (`session_handoff`/`session_resume`), `tools_session_status.go` (`session_status`/`knowledge_compact`); remove `registerHandoffTools` calls in `mcp/server.go:66,99`
- Test: tool-registry test asserting the 9 names absent and `session_changes` present

**Interfaces:**
- Consumes: Task 5's `session_changes` tool (the resume path replacement)
- Produces: slimmer tool registry
- Seam: none
- Deletion test: the task IS a deletion
- Adapters: 1 — justified: pure removal

**SATISFIES:** old-surfaces-removed (MCP half)

- [ ] **Step 1: Write the failing registry test** (absent 9 + present 1)
- [ ] **Step 2: Run test to verify it fails** (tools still registered)
- [ ] **Step 3: Delete the files + deregister** (STOP — get approval first)
- [ ] **Step 4: Run test to verify it passes** + full `mcp` package suite
- [ ] **Step 5: Commit** — `git commit -m "refactor(mnemonic): delete hub and relay MCP tools"`

---

### Task 8: Delete relay, HTTP, envelope, UI, bash snapshot/restore

**Files:**
- Delete: `skillgrid-cli/internal/mnemonic/relay/` (relay, cleave, status, compact, watchdog + tests), `cmd/skillgrid/session.go` handoff/resume/status (keep the Task 5 `session <id>` read), `internal/mnemonic/http/handoff.go` (+test, strip `activity.go`), `memory/envelope.go:165` HandoffRefs field, `handoff/checkpoint.go:278 detectHandoffFile` (already gone with Task 6 — verify), `skillgrid-ui/.../sessions/HandoffsPane.tsx`, `ChangesPane.tsx`, `api.ts` handoff calls
- Modify: `hooks/checkpoint-state.sh` — delete `cmd_snapshot`/`cmd_restore` + `skillgrid handoff record` mirror; keep `guard`/`guard-msg`/`post-check`
- Test: `scripts/test-hooks.sh` updated (drop snapshot/restore cases or repoint to SessionEnd capture); `rg` checks for `session_handoffs|session_archives|change_snapshots|checkpoints|handoff_refs|cleave` in Go sources return nothing

**Interfaces:**
- Consumes: Tasks 6–7 (Hub gone first so the mirror call has nothing to call)
- Produces: no relay/cleave/HTTP-hub code; leaner `checkpoint-state.sh`
- Seam: none
- Deletion test: the task IS a deletion
- Adapters: 1 — justified: coordinated removal wave

**SATISFIES:** old-surfaces-removed (remainder)

- [ ] **Step 1: Write the failing checks** — `checkpoint-state.sh snapshot` must exit unknown-subcommand; `rg` for the five table names in Go sources must be empty
- [ ] **Step 2: Run checks to verify they fail** (subcommands still exist; table names still referenced)
- [ ] **Step 3: Delete + modify** (bash edit + `git rm`s; UI file deletions)
- [ ] **Step 4: Run checks to verify they pass** + `bash scripts/test-hooks.sh` (22→ adjusted count) + `go build ./...`
- [ ] **Step 5: Commit** — `git commit -m "refactor: delete cleave relay, hub HTTP/UI, checkpoint snapshot/restore"`

---

### Task 9: Repoint skills to the events model

**Files:**
- Modify (14): `work-unit-commits/SKILL.md` + `references/state-schema.md` + `templates/checkpoint.md`, `resume/SKILL.md`, `using-skillgrid/SKILL.md`, `subagent-execution/SKILL.md` + `references/setup.md` + `references/implementer-prompt.md`, `simple-execution/SKILL.md`, `ship/SKILL.md` (drop `handoff archive`), `slicing/SKILL.md`, `onboarding/SKILL.md` + `templates/config.yaml` (`state_file`), `_shared/conventions/sdd-structure.md`, `mnemonic/references/memory.md`
- Test: grep checks (no `checkpoint.json`, `checkpoint-state.sh snapshot|restore`, `handoff verify|record|archive|checkpoint`, `.cleave` refs); regenerate site partial

**Interfaces:**
- Consumes: Tasks 5 (read path the resume skill now documents), 6–8 (removed surfaces)
- Produces: coherent skill instructions bound to the events model
- Seam: none (docs)
- Deletion test: pass-through
- Adapters: 1 — justified: single reviewable docs wave

**SATISFIES:** resume-from-events (skill half), old-surfaces-removed (docs half)

- [ ] **Step 1: Write the failing check** — script asserting zero stale refs across the 14 files + `sdd-structure.md` sdd/ tree without `checkpoint.json`
- [ ] **Step 2: Run check to verify it fails**
- [ ] **Step 3: Rewrite** — resume = latest session's events via `session_changes` + `to_commit`; work-unit-commits keeps the `[skillgrid-context]` block (now feeding `commit` events) minus the resume-handle section; ship drops `handoff archive`; onboarding drops `state_file`
- [ ] **Step 4: Run check to verify it passes** + regenerate site (`python3 scripts/gen-guide-content.py`)
- [ ] **Step 5: Commit** — `git commit -m "docs: repoint skills to session events model"`

---

### Task 10: Per-plan `.skillgrid/sdd/` layout + scripts

**Files:**
- Create: `sdd-workspace`, `task-brief`, `review-package` (adapted from superpowers, repathed `.superpowers/sdd` → `.skillgrid/sdd`)
- Test: `scripts/test-sdd-workspace.sh` (new): distinct plan files → distinct dirs; artifacts land per-plan; parent `.gitignore` self-ignores (`git status` clean)

**Interfaces:**
- Consumes: briefing's superpowers layout decision; slug = `.skillgrid/specs/<YYYY-MM-DD-topic>/` basename
- Produces: `progress.md` (line 1 names plan), `task-N-brief.md`, `task-N-report.md`, `review-<base7>..<head7>.diff`
- Seam: none (scripts)
- Deletion test: pass-through, delete it
- Adapters: 1 — justified: one script family

**SATISFIES:** per-plan-workspace

- [ ] **Step 1: Write the failing test** (`scripts/test-sdd-workspace.sh` per the cases above)
- [ ] **Step 2: Run test to verify it fails** (scripts absent)
- [ ] **Step 3: Write the three scripts** (port superpowers' `sdd-workspace`/`task-brief`/`review-package`; exit 2 bad args, exit 3 task-not-found)
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit** — `git commit -m "feat(sdd): per-plan workspace scripts"`

---

### Task 11: Migration 041 — drop the old tables

> ⚠ one-way: dropping five tables erases Hub UI history (reconstructable from `git log`, but the index is gone). STOP for explicit approval before committing this task.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/041_drop_handoff_tables.sql`
- Test: migrate a DB seeded by 039/019, assert the five tables + six indexes absent and `sessions`/`session_events` intact

**Interfaces:**
- Consumes: Tasks 6–9 (zero readers of the old tables — verify with `rg` first)
- Produces: clean schema
- Seam: none
- Deletion test: the task IS a deletion
- Adapters: 1 — justified: pure removal

**SATISFIES:** old-surfaces-removed (schema half)

- [ ] **Step 1: Write the failing test** (tables present → assertions fail)
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write `041_drop_handoff_tables.sql`** (STOP — get approval first)

```sql
-- 041: drop the handoff hub (039) + cleave relay (019) tables. The session
-- events layer (040) is the index now; git log remains the durable record.
DROP TABLE IF EXISTS change_snapshots;
DROP TABLE IF EXISTS checkpoints;
DROP TABLE IF EXISTS handoff_refs;
DROP TABLE IF EXISTS session_handoffs;
DROP TABLE IF EXISTS session_archives;
DROP INDEX IF EXISTS idx_snapshots_project_time;
DROP INDEX IF EXISTS idx_checkpoints_project_status;
DROP INDEX IF EXISTS idx_handoff_refs_project;
DROP INDEX IF EXISTS idx_handoffs_source;
DROP INDEX IF EXISTS idx_handoffs_status;
DROP INDEX IF EXISTS idx_archives_session;
```

- [ ] **Step 4: Run test to verify it passes** + full store suite
- [ ] **Step 5: Commit** — `git commit -m "refactor(mnemonic): drop handoff hub and relay tables"`

---

## Self-Review

**1. Spec coverage:** briefing §Data model → Tasks 1, 11; §Capture bridge (endpoint, adapters already in repo, lifecycle map) → Tasks 2–4 (adapters exist; endpoint + lifecycle wiring is Tasks 2–4); §Read path → Task 5; §Go removals table → Tasks 6–8 row-for-row; §Bash → Task 8; §Skills (14 files) → Task 9; §sdd layout → Task 10; §Execution order → task numbering matches; §Risks (bridge critical path, sequence atomicity) → covered by Task 3's transaction requirement and the door check. Gap: briefing's backfill note (replay `git log` for open sessions) has no task — add as Step 6 of Task 4 if open sessions exist at cutover, else drop with a note. Gap: `.cursor/hooks.json` + `hooks/mnemonic-hook.sh` from the capture-bridge plan were deferred earlier ("plugins/ committed" covers opencode/kilo/cursor rule files; the Cursor `hooks.json` command-hook pair is not in `plugins/`) — Task 10's scripts do not cover it; add it to Task 4's scope or a follow-up change. Recording here: **follow-up change** (Cursor command hooks), not this blueprint.

**2. Must-haves coverage:** every briefing requirement maps to a truth/artifact/link above; `backstop` tags on tool-call ordering and sensitive redaction (both need live hook runtime).

**3. One-way-door completeness:** 041 drops + 9-tool MCP removal flagged in Must-Haves and tagged on Tasks 7, 11. The `core.hooksPath` repoint is already done (not a task).
