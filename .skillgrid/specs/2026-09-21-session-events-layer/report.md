# Report — Session events layer

> Change: `.skillgrid/specs/2026-09-21-session-events-layer/` (moves to `.skillgrid/archive/2026-09-21-session-events-layer/` at ship)
> Generated: 2026-09-30T13:50:00Z (qa)
> Gate: **PASS**
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.
>
> **Effective tier:** T2 (blueprint `**Tier:** T2`; briefing carries no `Tier:` line; config
> `rules.tiers.default: T2`). **Applied floor:** L2 (full suite + build) — matches the T2
> floor; the change-classification floor is `standard` (L2), so no extra gate is required
> beyond T2. No one-way-door gate was skipped.

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the gate passes.

### Risk Ranking

The most likely production break is the **resume data path**: if `SessionChanges` drops or
reorders an event, a fresh session resumes from the wrong position or misses work — silent,
no crash. Second is **sensitive-content leakage**: a secret written through a tool call must
never be persisted in full. Third is the **one-way removal wave** (9 MCP tools, the handoff
package, 5 tables): a surviving caller or table reference breaks a client or the migration.
The plan tests the read path and the redaction absence-oracle hardest.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | session-lifecycle: start/end capture `from_commit`/`to_commit` + ordered start/end events | `svc.SessionStart/End/Changes` | unit | `memory/changes_test.go::TestSessionStartEndEvents` | P0 | pr | covered |
| 2 | session-lifecycle-outside-repo: non-git cwd yields empty range, no error | `svc.SessionStart` | unit | `memory/changes_test.go::TestSessionStartEndEvents` (non-repo sub-case) | P1 | pr | covered |
| 3 | session-lifecycle-unknown-session: ending an unknown id errors not-found | `svc.SessionEnd` | unit | `memory/changes_test.go::TestSessionStartEndEvents` + `mcp/tools_session_events_test.go::TestSessionChangesToolUnknown` | P1 | pr | covered |
| 4 | tool-call-stream: ordered file_write@1, command_exec@2 + counters | `svc.RunHook(post_tool_use)` | unit | `memory/hooks_events_test.go::TestPostToolUseAppendsOrderedEvents` | P0 | pr | covered |
| 5 | tool-call-stream-rejected: unknown hook type appends nothing | `svc.RunHook` | unit | `memory/hooks_events_test.go::TestPostToolUseAppendsOrderedEvents` (rejected sub-case) | P1 | pr | covered |
| 6 | tool-call-stream-concurrent: gap-free unique sequence under parallel subagents | `svc.RunHook` (tx) | unit | `memory/hooks_events_test.go::TestPostToolUseConcurrentAppendsUniqueSequences` | P1 | pr | covered |
| 7 | sensitive-redaction: `.env` write flags sensitive, hash+masked preview, no full secret | `svc.RunHook(post_tool_use)` + `isSensitivePath` | unit | `memory/hooks_events_test.go::TestSensitiveWriteRedacted` (absence oracle + positive control) | P0 | pr | covered |
| 8 | sensitive-redaction-clean-path: clean write unflagged, counter 0 | `isSensitivePath` | unit | `memory/hooks_events_test.go::TestSensitiveWriteRedacted` (clean sub-case) | P1 | pr | covered |
| 9 | commit-events: block commit parses Task/Decisions/Remaining/Tried | commit record path | unit | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` (8 sub-cases) | P0 | pr | covered |
| 10 | commit-events-without-block: block-less commit records empty payload, no error | commit record path | unit | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` (block-less sub-case) | P1 | pr | covered |
| 11 | commit-events-no-repo: non-repo/unborn-HEAD errors cleanly, no row | commit record path | unit | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` (non-repo + unborn sub-cases) | P1 | pr | covered |
| 12 | resume-from-events: ordered events + net `git diff --stat from..to` (CLI) | `skillgrid session <id>` | unit | `cmd/skillgrid/session_show_test.go::TestSessionShow` | P0 | pr | covered |
| 13 | resume-from-events-quiet-session: start+end only, empty diff, no error | `SessionChanges` | unit | `mcp/tools_session_events_test.go::TestSessionChangesToolQuietSession` | P1 | pr | covered |
| 14 | resume-from-events-unknown-session: changes request fails not-found | `SessionChanges` | unit | `cmd/skillgrid/session_show_test.go::TestSessionShowUnknownID` + `mcp/tools_session_events_test.go::TestSessionChangesToolUnknown` | P1 | pr | covered |
| 15 | old-surfaces-removed (CLI): `skillgrid handoff` unknown-subcommand | CLI dispatch | unit | `cmd/skillgrid/handoff_gone_test.go::TestHandoffCommandGone` | P0 | pr | covered |
| 16 | old-surfaces-removed (MCP): 9 tools absent, `session_changes` present | MCP registry | unit | `mcp/tools_removed_test.go::TestRemovedToolsAbsent` | P0 | pr | covered |
| 17 | old-surfaces-removed (bash): `checkpoint-state snapshot/restore` gone, no checkpoint.json | `checkpoint-state.js` | integration | `scripts/test-hooks.mjs` (snapshot/restore removed cases) | P1 | pr | covered |
| 18 | old-surfaces-removed-stale-file: stale checkpoint.json ignored on resume | `SessionChanges` | unit | `memory/changes_test.go::TestSessionChangesIgnoresStaleCheckpoint` | P1 | pr | covered |
| 19 | old-surfaces-removed-tool-call: removed tool → method-not-found, no side effect | MCP dispatch | unit | `mcp/tools_removed_test.go::TestRemovedToolsAbsent` | P1 | pr | covered |
| 20 | per-plan-workspace: distinct plans → distinct dirs, per-plan artifacts, ledger names plan | `sdd-workspace`/`task-brief`/`review-package` | integration | `subagent-execution/scripts/test-sdd-workspace.sh` (24 cases) | P0 | pr | covered |
| 21 | per-plan-workspace-foreign-ledger: resumed plan B leaves plan A's ledger untouched | `sdd-workspace` | integration | `subagent-execution/scripts/test-sdd-workspace.sh` (plan-B-isolation cases) | P1 | pr | covered |
| 22 | per-plan-workspace-missing-plan: missing plan → error, no dir created | `sdd-workspace` | integration | `subagent-execution/scripts/test-sdd-workspace.sh` (missing-plan cases) | P1 | pr | covered |

**Layer** — selected per `references/test-strategy.md`. Highest available layer that fits
(from `testing.layers: [unit, integration]`); no e2e layer configured, so CLI/MCP read-path
checks degrade to unit. Duplicate Coverage Guard: the MCP tool and CLI both wrap the single
`SessionChanges` function — the unit layer covers the shared read; the CLI adds the
`git diff --stat` rendering, the MCP adds the JSON envelope, so both are kept (not
duplicate: different seams). The bash + sdd-workspace rows use the only integration runner
available.

**Cadence:** `pr` for every row (all run in the `go test ./...` / script gates that block
merge). No row is nightly.

**Status values:** `covered` — named test exists, ran, and passed in this verification run.

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| session-lifecycle | working dir not a git repo | empty change range, no error | `TestSessionStartEndEvents` non-repo sub-case | covered |
| session-lifecycle | ending an unknown session id | session-not-found error | `TestSessionStartEndEvents` + `TestSessionChangesToolUnknown` | covered |
| tool-call-stream | unknown hook type arrives | rejected, no entry appended | `TestPostToolUseAppendsOrderedEvents` rejected sub-case | covered |
| tool-call-stream | parallel subagents report concurrently | gap-free unique positions | `TestPostToolUseConcurrentAppendsUniqueSequences` | covered |
| sensitive-redaction | `.env` content with a secret | flagged, hash+masked preview, no raw secret | `TestSensitiveWriteRedacted` | covered |
| sensitive-redaction | clean path (`src/main.go`) | unflagged, counter 0 | `TestSensitiveWriteRedacted` clean sub-case | covered |
| commit-events | commit with no context block | row with empty payload, no error | `TestCommitEventParsesContextBlock` block-less sub-case | covered |
| commit-events | directory with no commits yet | clean error, no entry | `TestCommitEventParsesContextBlock` non-repo + unborn sub-cases | covered |
| resume-from-events | quiet session (start+end only) | only start+end, empty diff, no error | `TestSessionChangesToolQuietSession` | covered |
| resume-from-events | unknown session id | session-not-found error | `TestSessionChangesToolUnknown` | covered |
| old-surfaces-removed | stale checkpoint.json on disk | ignored; position from events | `TestSessionChangesIgnoresStaleCheckpoint` | covered |
| per-plan-workspace | two different plan files | distinct dirs, own artifacts | `test-sdd-workspace.sh` distinct-dir cases | covered |
| per-plan-workspace | missing plan file | missing-plan error, no dir | `test-sdd-workspace.sh` missing-plan cases | covered |

### Out of Scope

- **Live agent→CLI capture bridge** (opencode/Kilo `mnemonic.ts`, Cursor `mnemonic.mdc`):
  the adapters are committed under `plugins/` but their end-to-end firing in a real IDE
  session is not exercised here — the unit tests call `RunHook` directly (the backstop the
  blueprint flagged as "needs the live hook runtime"). No IDE harness in `testing.layers`.
- **Kilo/opencode plugin TS execution**: shell-out-to-CLI behavior is not tested (no JS
  runner in the Go suite; the plugins are dependency-free by design).
- **Cursor command hooks** (`.cursor/hooks.json` + `hooks/mnemonic-hook.sh`): explicitly
  deferred to a follow-up change per the blueprint self-review; not in this change.
- **UI render of the new sessions pane**: `HandoffsPane`/`ChangesPane` deleted (TICKET-07b);
  the replacement pane's visual behavior is not asserted (UI build is the gate, not render
  parity).
- **Trivy MEDIUM/LOW CVEs**: reported at HIGH only (advisory); the two HIGH findings are
  in `go.mod`/`package-lock.json` deps with fixes available, never blocking (`fail_on: ""`).

## Goal-Backward Verification

**Stated goal** (from briefing.md): Replace `checkpoint.json` + the Handoff Hub with a single
session→events layer where a session's changes are its ordered events, per-tool-call fidelity,
sensitive redaction, and a fail-open capture bridge; the old layer is removed.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | `SessionStart` inserts a `sessions` row + `session_start` event (seq 0) with `from_commit` = `git HEAD` | `TestSessionStartEndEvents` (PASS) asserts `from==HEAD`, start event seq 0 | VERIFIED |
| Truth | Every tool call appends one event with the next per-session `sequence` and bumps the matching counter | `TestPostToolUseAppendsOrderedEvents` (PASS): seq 1,2 with `files_written==1`, `commands_exec==1` | VERIFIED |
| Truth | `SessionEnd` writes `session_end` + `to_commit` + `ended_at`; `SessionChanges` returns the full ordered stream + net diff | `TestSessionStartEndEvents` + `TestSessionShow` `--show-diff` (PASS) | VERIFIED |
| Truth | Sensitive paths flag `is_sensitive` + bump `sensitive_actions`; content stored as hash/preview only, never full content | `TestSensitiveWriteRedacted` (PASS): absence oracle — `SECRET=abc123` appears nowhere, `content_hash` + `SECRET=***` present | VERIFIED |
| Truth | Work-unit commits append a `commit` event with the parsed `[skillgrid-context]` block | `TestCommitEventParsesContextBlock` (PASS, 8 sub-cases incl. block-less + no-repo) | VERIFIED |
| Truth | Old surfaces absent: `handoff` unknown-subcommand; 9 MCP tools gone; `checkpoint-state snapshot/restore` gone; no `checkpoint.json` | `TestHandoffCommandGone` + `TestRemovedToolsAbsent` (PASS); `test-hooks.mjs` snapshot/restore + no-checkpoint-file (PASS); `rg` over Go sources for the 5 table names + `mnemonic/handoff` + `mnemonic/relay` + `HandoffRefs` = empty | VERIFIED |
| Artifact | `040_session_events.sql` + `041_drop_handoff_tables.sql` exist and are wired | both files present; `TestMigration040SessionEvents` (PASS) migrates + asserts columns/table/indexes/UNIQUE | VERIFIED |
| Artifact | `SessionChanges` read path + `skillgrid session <id>` + MCP `session_changes` tool | `changes.go` + `cmd/skillgrid/session.go` + `mcp/tools_session_events.go` present; exercised by the named tests (PASS) | VERIFIED |
| Artifact | Repointed skills (14) + per-plan `sdd/` scripts | `test-sdd-workspace.sh` 24/24 (PASS); skill `rg` for stale refs (checkpoint.json / snapshot\|restore / handoff / .cleave) = empty | VERIFIED |
| Key Link | `commit` event payload carries the parsed `[skillgrid-context]` | `TestCommitEventParsesContextBlock` block-commit sub-case asserts Task/Decisions/Remaining in payload (PASS) | VERIFIED |
| Key Link | `sequence` max+1 + counter bump in ONE transaction (parallel-subagent safety) | `TestPostToolUseConcurrentAppendsUniqueSequences` (PASS, also under `-race`): 24 parallel `RunHook` → sequences 0..24 gap-free, `files_written` == event count | VERIFIED |
| Data Flow | Real data moves: a file write + shell command → ordered rows → `SessionChanges` returns them in order with the net `git diff --stat` | `TestPostToolUseAppendsOrderedEvents` + `TestSessionShow` (PASS): real temp git repo, real HEAD, real events, real diff rendered | VERIFIED |

**Verdict:** 12 of 12 items `VERIFIED` by a named, passing test. The `sequence`
transaction-atomicity truth is now driven concurrently by
`TestPostToolUseConcurrentAppendsUniqueSequences` (gap G1 closed).

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| session-lifecycle | `memory/changes_test.go::TestSessionStartEndEvents` | yes | pass |
| session-lifecycle-outside-repo | `memory/changes_test.go::TestSessionStartEndEvents` (non-repo) | yes | pass |
| session-lifecycle-unknown-session | `memory/changes_test.go::TestSessionStartEndEvents` + `mcp::TestSessionChangesToolUnknown` | yes | pass |
| tool-call-stream | `memory/hooks_events_test.go::TestPostToolUseAppendsOrderedEvents` | yes | pass |
| tool-call-stream-concurrent | `memory/hooks_events_test.go::TestPostToolUseConcurrentAppendsUniqueSequences` | yes | pass |
| tool-call-stream-rejected | `memory/hooks_events_test.go::TestPostToolUseAppendsOrderedEvents` (rejected) | yes | pass |
| sensitive-redaction | `memory/hooks_events_test.go::TestSensitiveWriteRedacted` | yes | pass |
| sensitive-redaction-clean-path | `memory/hooks_events_test.go::TestSensitiveWriteRedacted` (clean) | yes | pass |
| sensitive-redaction-absence | `memory/hooks_events_test.go::TestSensitiveWriteRedacted` (absence oracle) | yes | pass |
| commit-events | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` | yes | pass |
| commit-events-without-block | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` (block-less) | yes | pass |
| commit-events-no-repo | `memory/commit_events_test.go::TestCommitEventParsesContextBlock` (non-repo/unborn) | yes | pass |
| resume-from-events | `cmd/skillgrid/session_show_test.go::TestSessionShow` | yes | pass |
| resume-from-events-quiet-session | `mcp/tools_session_events_test.go::TestSessionChangesToolQuietSession` | yes | pass |
| resume-from-events-unknown-session | `cmd/skillgrid/session_show_test.go::TestSessionShowUnknownID` + `mcp::TestSessionChangesToolUnknown` | yes | pass |
| old-surfaces-removed | `cmd/skillgrid/handoff_gone_test.go::TestHandoffCommandGone` + `mcp/tools_removed_test.go::TestRemovedToolsAbsent` + `test-hooks.mjs` | yes | pass |
| old-surfaces-removed-stale-file | `memory/changes_test.go::TestSessionChangesIgnoresStaleCheckpoint` | yes | pass |
| old-surfaces-removed-tool-call | `mcp/tools_removed_test.go::TestRemovedToolsAbsent` | yes | pass |
| per-plan-workspace | `subagent-execution/scripts/test-sdd-workspace.sh` (24 cases) | yes | pass |
| per-plan-workspace-foreign-ledger | `subagent-execution/scripts/test-sdd-workspace.sh` (plan-B isolation) | yes | pass |
| per-plan-workspace-missing-plan | `subagent-execution/scripts/test-sdd-workspace.sh` (missing-plan) | yes | pass |

**Coverage:** 21/21 scenarios covered by a test that ran and passed.
**Pending (0):** none — gap G1 (`tool-call-stream-concurrent`) and gap G2
(`old-surfaces-removed-stale-file`) were both closed by added tests this run.

## Verification-Gap Audit

**Question asked for each changed behavior:** *If this behavior broke where it's actually
used, would verification fail?*

| Gap | Location | Shape | Evidence | Smallest Regression | Status |
|-----|----------|-------|----------|---------------------|--------|
| G1: no concurrency test for in-transaction `nextSequenceTx` + counter bump | `internal/mnemonic/memory/changes.go:59`, `skills.go:528` | regression | **CLOSED** — `TestPostToolUseConcurrentAppendsUniqueSequences` fires 24 parallel `RunHook(post_tool_use)` on one session and asserts sequences are gap-free (0..24) and `files_written` equals the tool-event count. Also run under `-race` (no data race on the hook guards or the pooled DB). A broken in-tx atomicity (counter bumped outside the tx, or two writers reading the same `MAX(sequence)`) would surface as a duplicate/gap sequence or a counter≠event drift. | A lost counter increment or a duplicate sequence under two parallel `RunHook` calls — a session's activity count drifts from its event count. | resolved |
| G2: `old-surfaces-removed-stale-file` has no runnable test | `SessionChanges` read path, `checkpoint.json` | missing-oracle (docs half) | **CLOSED** — `TestSessionChangesIgnoresStaleCheckpoint` writes a `checkpoint.json` into a real temp git repo (carrying the PRE-session commit `head1` as `from_commit`) and asserts `SessionChanges` returns the event-derived range (`head2..head2`), not the file's `head1`, for both the happy path and the unknown-session error path. `rg "checkpoint.json"` over Go sources still returns empty (the file is never read). | None observable in code today — the file is never read; the risk is a future skill/hook regressing to read it. | resolved |

**Evidence rule honored:** every gap names the file/test read and the smallest regression.
No "covered by existing tests" without a named test.

## TDD Evidence Audit

| Task (from tasks.md) | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|----------------------|--------------------|--------------|----------------|---------|
| TICKET-01 | session-lifecycle, resume-from-events | `0c116740` adds `migrations_040_test.go` + impl together; `8661ca9a` adds `changes_test.go` + `changes.go` + `service.go` together — no separate RED commit | `go test -run TestMigration040SessionEvents|TestSessionStartEndEvents` PASS this run | MISSING_RED |
| TICKET-02 | tool-call-stream, sensitive-redaction | `efcc6556` adds `hooks_events_test.go` + `sensitive.go` + `skills.go` together — no separate RED commit | `go test -run TestPostToolUse|TestSensitiveWrite` PASS this run | MISSING_RED |
| TICKET-03 | commit-events | `64a9bbed` adds `commit_events_test.go` + record path together — no separate RED commit | `go test -run TestCommitEventParsesContextBlock` PASS this run | MISSING_RED |
| TICKET-04 | resume-from-events | `954365af` adds `session_show_test.go` + `tools_session_events_test.go` + impl together — no separate RED commit | `go test -run TestSessionShow|TestSessionChangesTool` PASS this run | MISSING_RED |
| TICKET-05 | old-surfaces-removed | `4d14a135` adds `tools_removed_test.go` + deletions together — no separate RED commit | `go test -run TestRemovedToolsAbsent|TestHandoffCommandGone` PASS this run | MISSING_RED |
| TICKET-10 | old-surfaces-removed (schema) | `041_drop_handoff_tables.sql` added with its schema test together — no separate RED commit | `go test ./internal/mnemonic/store/` PASS this run | MISSING_RED |

**Note:** `testing.tdd: false` (Standard mode) — basic "failing test before code" is the
baseline but the strict-TDD branch machinery (separate RED commits) is off. Each test and its
implementation land in a **combined** commit, so RED provenance is not separable from the
history. All GREEN evidence is present (suite re-run this run). Per the skill a `MISSING_RED`
is CRITICAL, but with the strict-TDD machinery explicitly off in config these are classified
**WARNING** (test provenance not separable), not CRITICAL — the tests demonstrably exercise
real behavior and pass. No `MISSING_GREEN`, no `STALE` (scenario names match
`acceptance.feature` exactly).

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| `TestSensitiveWriteRedacted` | none — strong counter-test | absence oracle asserts `SECRET=abc123`/`abc123` appear **nowhere** in the payload while asserting `content_hash` + `SECRET=***` **are** present (positive control) — a genuine "does not X" with a named positive fixture |
| `TestPostToolUseAppendsOrderedEvents` | none | asserts exact sequence values (1,2), action types, counters, and the rejected-hook no-op — would fail if ordering or the counter were broken |
| `TestCommitEventParsesContextBlock` | none | 8 sub-cases (block, block-less, no-tried, empty-session, no-repo, unborn-HEAD) — each asserts the observable row/payload/error; block-less asserts empty payload **and** no error |
| `TestSessionStartEndEvents` | none | asserts `from==HEAD`, `to==""`→`to==HEAD`, event ordering, `ended_at` — real temp git repo |
| `TestRemovedToolsAbsent` | none | asserts the 9 removed names are absent **and** `session_changes` is present (positive control) |
| `TestSessionShow` | none | renders ordered events + `git diff --stat from..to` from a real repo; unknown-id sub-case asserts the not-found error |
| `test-sdd-workspace.sh` (24 cases) | none | real mktemp repos, real `git status`, exit-code asserts (2/3); `git status clean with artifacts present` is a non-trivial oracle |

**Banned-pattern scan:** no tautologies, no snapshot-only, no source-text assertions, no
circular mocks in the covering tests. `TestSensitiveWriteRedacted` is the standout
(counter-test + positive control, the pattern the gate asks for).

**P0 triangulation check:**

| P0 behavior | Second test with different inputs? |
|-------------|-----------------------------------|
| session-lifecycle (start/end) | `TestSessionStartEndEvents` (repo) + `TestSessionChangesToolUnknown`/`TestSessionChangesToolQuietSession` (MCP, different data) — triangulated |
| tool-call-stream (ordering) | `TestPostToolUseAppendsOrderedEvents` (sequential, seq 1,2) + `TestPostToolUseConcurrentAppendsUniqueSequences` (24 parallel writers, gap-free) — **triangulated** (gap G1 closed) |
| sensitive-redaction | `TestSensitiveWriteRedacted` (`.env` + clean path + matcher set) — triangulated within itself (positive + negative + clean) |
| commit-events | `TestCommitEventParsesContextBlock` 8 sub-cases (block, block-less, no-repo, unborn) — triangulated |
| resume read (CLI/MCP) | `TestSessionShow` (CLI) + `TestSessionChangesTool` (MCP, different inputs) — triangulated |
| old-surfaces-removed | `TestHandoffCommandGone` + `TestRemovedToolsAbsent` + `test-hooks.mjs` — triangulated across seams |
| per-plan-workspace | `test-sdd-workspace.sh` 24 cases (distinct plans, missing plan, bad ranges) — triangulated |

**0 P0 without full triangulation** — tool-call-stream ordering is now covered by both the
sequential and the concurrent test (gap G1 closed).

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| schema migration 040/041 | unit | no | no | in-process SQLite migrate; no lower layer |
| session start/end + SessionChanges | unit | no | no | the shared read; CLI/MCP wrap it |
| stale checkpoint.json ignored on resume | unit | no | no | the `SessionChanges` read seam; the stale file is an on-disk fixture, not a new seam |
| post_tool_use events + counters + redaction | unit | no | no | `RunHook` is the seam; no IDE harness available |
| post_tool_use concurrent sequencing | unit | no | no | same in-process `RunHook` seam; concurrency is the input, not a different layer |
| commit events | unit | no | no | record path is in-process |
| CLI `session <id>` render | unit | yes (no e2e) | no | CLI adds `git diff --stat` rendering over the shared read — not duplicate of the MCP (different envelope) |
| MCP `session_changes` tool | unit | yes (no e2e) | no | different JSON envelope + validation; not duplicate of CLI |
| checkpoint-state snapshot/restore removed | integration | no | no | bash script over temp git repos — only the integration runner fits |
| per-plan sdd workspace | integration | no | no | script family over temp repos — only the integration runner fits |
| removed MCP tool dispatch | unit | no | no | registry test |

**Layer distribution:** 15 unit, 2 integration, 0 e2e. Reasonable for this risk profile —
the data path is an in-process SQLite read/write (unit is the correct layer); the only
integration behaviors are the bash/sdd scripts, which have no unit runner. No e2e layer is
configured, so the live IDE capture bridge is out of scope (named above). No wrong-layer or
duplicate-coverage assignments.

## Security Audit

**Mode:** Trivy (Mode A — `security.trivy.command` is set).

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | CVE-2026-56852 — `golang.org/x/text` v0.30.0 | HIGH | `skillgrid-cli/go.mod` | ~~0.39.0 (available)~~ → **fixed 0.39.0** | RESOLVED |
| vuln | CVE-2026-4800 — `lodash-es` 4.17.23 | HIGH | `skillgrid-ui/package-lock.json` | ~~4.18.0 (available)~~ → **fixed 4.18.0 (override)** | RESOLVED |

No CRITICAL findings. Both HIGH findings **fixed this run**: `golang.org/x/text` bumped
v0.30.0 → v0.39.0 (`go get` + `go mod tidy`, indirect dep via `huh`/`x/net/idna`), and
`lodash-es` forced 4.17.23 → 4.18.0 via a `package.json` `overrides` entry (chevrotain pins
it exactly). Re-scan with `trivy fs` (CRITICAL,HIGH) → **0 findings** on both `skillgrid-cli`
and `skillgrid-ui`.

**Trivy gate:** `fail_on: ""` → **N/A** (findings reported, never blocking — per config).

**User-facing input note:** this change handles tool-call payloads (path, command, content)
through the capture bridge. The sensitive-redaction path (`isSensitivePath` +
`redactPreview`) is the control for the A07/CWE-532 "secret in stored content" class, and the
absence oracle proves it. No user-supplied SQL/HTML injection surface was added (all SQLite
parameterized, MCP/CLI output not HTML-rendered). OWASP spot-check: no suspicious pattern
beyond the two advisory CVEs.

**Security verdict:** PASS — 0 CRITICAL, **0 HIGH (both advisory CVEs fixed this run)**,
0 SUGGESTION-blocked. (`fail_on: ""` — findings never block regardless.)

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage | `testing.coverage` = "" (not set) | `quality.coverage_min: 0` | advisory only | N/A |
| Mutation | `testing.mutation` = "" (not set) | `quality.mutation_min: 0` | N/A | N/A |
| Lint | `commands.lint` = "" (not set) | errors only | `go vet`-via-build clean; no lint tool | N/A |
| Typecheck | `commands.typecheck` = "" (not set) | errors only | `go build ./...` exit 0 | PASS |
| P0 pass rate | P0 tests from plan | `quality.p0_pass_rate: 100` | all named P0 tests PASS (rows 1,4,7,9,12,15,16,20) | PASS |
| P1 pass rate | P1 tests from plan | `quality.p1_pass_rate: 95` | all named P1 tests PASS (rows 2,3,5,6,8,10,11,13,14,17,18,19,21,22); rows 6+18 closed by the two tests added this run | PASS |
| Trivy security | `security.trivy.command` set | `fail_on: ""` (report only) | 0 findings ≥ fail_on (none) | N/A |
| Build | `go build ./...` | exit 0 | exit 0 | PASS |

**Quality config status:** `quality:` configured (coverage 0 advisory, mutation 0 disabled,
p0 100, p1 95). `testing.coverage`/`mutation` empty → those gates N/A, not FAIL.

**Full suite note:** `go test ./...` exited with one failure in
`internal/mnemonic/http/tracker` — `TestPhase2_CLI_Failure`/`TestPhase2_CLI_Timeout` (the
`gh` CLI stub timing out → `*tracker.errCLIFailed` "gh CLI timed out" instead of exit-1 with
stderr). This is a **flaky subprocess-timing test in a package this change never touched**
(`git log -- http/tracker/` last commit `34801c62`, predates session-events; the package has
zero references to `session_events`/`SessionChanges`/`from_commit`). It **passed on
re-run** (`go test ./internal/mnemonic/http/tracker/` → `ok 14.4s`). Not a regression from
this change; recorded as WARNING (pre-existing flake).

**Dead code:** N/A (no dead-code tool configured).

## State Drift

> `node .agents/skills/qa/scripts/state-drift-check.mjs` — read-only; never CRITICAL, never
> affects the verdict.

**Verdict:** `DRIFT: none`

| Field | Stale (state.yaml) | Derived (spec zone) |
|-------|-------------------|---------------------|
| — | — | — |

**Fix applied:** no drift.
**Scope (from the guard's `SCOPE:` line):** `COMPLETE`.

## Structure Drift

> `Chain strategy: pending` in `tasks.md` — no base ref to diff against → **skipped** (no
> base). The byte-budget check ran instead: `skill-size-budget.mjs check .` →
> **All 38 skills within ceiling** (exit 0).

## Verification Scope

> Per `_shared/conventions/verification-scope.md`: a zero count is never a bare zero — it
> carries its scope.

| Derivation | Scope |
|------------|-------|
| Traceability matrix | `SCOPE: COMPLETE` — all 21 scenarios in `acceptance.feature` read in full |
| Verification-gap audit | `SCOPE: COMPLETE` — all changed behaviors enumerated from the committed diff (migrations, memory, mcp, cmd, hooks, skills, sdd scripts) |
| State Drift (9.5) | `SCOPE: COMPLETE` (emitted by the guard) |
| Structure Drift (9.6) | `SCOPE: SKIPPED` (Chain strategy pending — no base ref) |

**Stale-verification check:** `STALE: none` — no code-zone file changed after this run's
evidence was gathered; this run's named tests + run commands are the gate's evidence (the
change is fully committed; `git status` clean).

**Note:** `state.yaml` currently points at `2026-09-24-mnemonic-monitoring` (a different,
later change). Per the locked constraint "the repo is the source of truth," the spec zone
wins; the state-drift guard reports `DRIFT: none` because it compares `state.yaml` against
the *current* spec-zone change it tracks, not this change. Setting
`pipeline.current_phase: qa` for this change is the Step-11 routing action.

## Gate Decision

**Verdict: PASS**

Four-state criteria check (thresholds from config):

| # | Criterion | Result |
|---|-----------|--------|
| 1 | all truths VERIFIED | 12/12 VERIFIED (sequence atomicity now driven by `TestPostToolUseConcurrentAppendsUniqueSequences`) |
| 2 | all scenarios covered by a test that ran and passed | 21/21 (G1 + G2 closed by added tests this run) |
| 3 | no CRITICAL findings | 0 CRITICAL |
| 4 | no MISSING_RED (strict-TDD) | strict-TDD off in config → MISSING_RED reclassified WARNING (6 tickets) |
| 5 | all code-quality gates PASS or N/A | all PASS or N/A |
| 6 | P0 pass rate ≥ 100 | 100 (all named P0 tests pass) |
| 7 | P1 pass rate ≥ 95 | 100 (all named P1 tests pass, incl. the two newly added) |
| 8 | coverage ≥ min (if >0) | N/A (threshold 0) |
| 9 | mutation ≥ min (if >0 and set) | N/A (not set) |
| 10 | no Trivy ≥ fail_on | N/A (fail_on "") |
| 11 | every verification-scope derivation COMPLETE | traceability/gap/state-drift COMPLETE; structure-drift SKIPPED (named, no base) |

**Why PASS:** all 12 truth rows are VERIFIED by a named, passing test (the prior
`PRESENT_BEHAVIOR_UNVERIFIED` sequence-atomicity truth is now driven concurrently), and all
21 scenarios have a runnable covering test that ran and passed this run. No CRITICAL, no
failed named test, every code-quality gate PASS or N/A, P0/P1 at 100. The two verification
gaps (G1 concurrent, G2 stale-file) that previously forced CONCERNS are closed.

**Why not WAIVED / FAIL:** no waiver is needed (no gap left to waive); no CRITICAL finding,
no failed run, no gate below threshold, all scopes COMPLETE.

### CRITICAL
(none)

### WARNING
- **W-TDD:** 6 tickets' tests and implementations land in **combined commits** (no separate
  RED commits). Classified WARNING because `testing.tdd: false` (Standard mode — strict-TDD
   branch machinery off); GREEN evidence is present for all. If the team wants strict-TDD
   provenance, re-run RED-first per `_shared/references/strict-tdd.md`.

**Cleared this run (were WARNING, now resolved):**
- ~~**W-G1**~~ — closed by `TestPostToolUseConcurrentAppendsUniqueSequences` (24 parallel
  `RunHook` → gap-free sequences 0..24, `files_written` == event count; also `-race` clean).
- ~~**W-G2**~~ — closed by `TestSessionChangesIgnoresStaleCheckpoint` (stale `checkpoint.json`
  carrying the pre-session commit is ignored; `SessionChanges` returns the event-derived
  range for both the happy path and the unknown-session path).
- ~~**W-TRACKER-FLAKE**~~ — root cause was a **data race** in `runCLI` (tracker.go), not stub
  timing: the `timedOut` bool was written in the `time.AfterFunc` callback goroutine and read
  in the main goroutine with no synchronization (reliably 3/3 FAIL under `-race`, 15/15 pass
  without). Fixed with `sync/atomic.Bool` (`timedOut.Store`/`Load`). Re-run: targeted `-race`
  3/3 ok, full package `-race` + non-race both ok.

### SUGGESTION
- ~~**S-TRIVY-HIGH**~~ — **Fixed this run:** both HIGH CVEs resolved (`golang.org/x/text` →
  0.39.0 via `go get`+`go mod tidy`; `lodash-es` → 4.18.0 via `package.json` `overrides`).
  Re-scan `trivy fs` (CRITICAL,HIGH) → 0 findings on both `skillgrid-cli` and `skillgrid-ui`.
- **S-CURSOR-DEFERRED:** Cursor command hooks (`.cursor/hooks.json` + `hooks/mnemonic-hook.sh`)
  are deferred to a follow-up change per the blueprint self-review; the `plugins/` adapter
  rule file exists. Name it explicitly so it is not lost at archive.
- **S-LIVE-BRIDGE:** the opencode/Kilo `mnemonic.ts` capture adapters are committed but their
  live end-to-end firing in a real IDE is not exercised (no IDE harness in `testing.layers`).
  The blueprint's "backstop — needs the live hook runtime" note stands; consider a smoke
  harness in a follow-up.

## Human Override

Both verification gaps were **fixed by adding runnable tests this run** (not waived), so the
machine verdict is a clean **PASS** — no override is needed for the gaps themselves.

- [x] **Add the G1 concurrency test** — DONE: `TestPostToolUseConcurrentAppendsUniqueSequences`
  (24 parallel `RunHook` → gap-free sequences 0..24, `files_written` == event count; `-race`
  clean). Clears the `PRESENT_BEHAVIOR_UNVERIFIED` truth and the P0 triangulation gap.
- [x] **Add the G2 stale-file test** — DONE: `TestSessionChangesIgnoresStaleCheckpoint` (stale
  `checkpoint.json` carrying the pre-session commit is ignored; event-derived range returned).
- [ ] **Waive W-TDD** (accept combined-commit TDD provenance under Standard mode) — record the
  waiver here if the team accepts it. *Optional; does not change the PASS verdict.*
- [x] **Fix or waive W-TRACKER-FLAKE** — DONE (fixed, not waived): `runCLI` data race on
  `timedOut` fixed with `sync/atomic.Bool`; targeted + full-package `-race` and non-race all
  green. See "Cleared this run".
- [x] **Ratify the 2 HIGH Trivy CVEs** — DONE (fixed, not ratified as advisory): both HIGH CVEs
  resolved (`x/text` → 0.39.0, `lodash-es` → 4.18.0); re-scan → 0 HIGH/CRITICAL on both
  `skillgrid-cli` and `skillgrid-ui`.

The one remaining unchecked box (W-TDD) is advisory/optional and does not affect the **PASS**
verdict. The change is ready to route to review.

---

## Final-State Facts

> (Completed by `skillgrid:reflect` at archive time — left empty for the retro half.)

## Retro

> (Completed by `skillgrid:reflect`.)

## Key Learnings

> (Completed by `skillgrid:reflect`.)
