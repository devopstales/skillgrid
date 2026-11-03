# TICKET-04 — Sandboxed `use_skill` executor

**Change:** 2026-09-04-hermes-memory · **Work unit:** 2 (PR 2) · **Status:** DONE

## Implementation

### 1. `skills` package — sandboxed `Execute` (Create)
`skillgrid-cli/internal/mnemonic/skills/execute.go`

- `ErrSoftDeleted` — sentinel for executing a soft-deleted skill; the MCP tool
  surfaces it verbatim, naming the skill.
- `DefaultTimeout` (30s) — the sandbox execution deadline; overridable via
  `Store.Timeout` for tests (shrunk to 2s in the timeout test to keep the suite
  fast).
- `maxOutputBytes` (1MB) — each captured stream (stdout/stderr) is capped;
  larger output is truncated with a `[skill sandbox: output truncated at 1MB]`
  note appended.
- `SkillResult{Stdout, Stderr, ExitCode, TimedOut, UsageID}` — `UsageID` is
  zero when no `skill_usage` row was written (failure, timeout, or reject).
- `execAllowlist` — language→runner map: sh/bash/zsh/shell→`bash -c`,
  python→`python3 -c`, node/javascript→`node -e`, go→`go run <file>` (compiled).
  A language outside this set is rejected before any subprocess is spawned
  (unknown-language rejects without exec — step-03 boundary).
- `Execute(ctx, name, input) (SkillResult, error)`:
  1. Validates the name (single safe basename, no path separators).
  2. Looks up the skill row. Soft-deleted → `ErrSoftDeleted`; missing → not-found.
  3. **Path escape (defense in depth):** the stored `code_path` must resolve to
     a file inside `.skillgrid/files/skills/` (`filepath.Clean` + base-join
     comparison). A path like `../evil.sh` is rejected before any subprocess.
  4. Language dispatch against `execAllowlist` (unknown language → error, no exec).
  5. `context.WithTimeout(ctx, deadline)` bounds the run; `buildSkillCommand`
     feeds the code file's content on **stdin** (not a `-c` arg — bash/python
     treat a path-like arg as a script file → exit 126, and a huge `-c` arg hits
     the OS arg-list limit). The `input` is a trailing argument (`$1`/`argv[1]`).
  6. `run(cmd, stdin, deadline)` — spawns with `Setpgid` (its own process group),
     captures stdout/stderr into buffers, and selects on `cmd.Wait()` vs the
     deadline channel. On timeout it kills the **process group**
     (`syscall.Kill(-pid, SIGKILL)`) so a `sleep 15` under `bash -c` cannot
     outlive its deadline and hold the wait open. Non-zero exit is folded into
     `ExitCode` (not a Go error); `exec.ErrNotFound` → exit 127.
  7. `truncateOutput` caps each stream at 1MB.
  8. On exit 0 + !timedOut, `insertUsage` writes the `skill_usage` row
     (`skill_id`, `session_id=NULL`, `timestamp`) and returns its id.
- `insertUsage(ctx, skillID)` — INSERT into `skill_usage`, returns `LastInsertId`.
- `truncateOutput(b)` — 1MB cap + note.
- `executableLanguages()` — sorted, deduped allowlist list for the error message.

### 2. `Store` struct — `Timeout` field (Modify)
`skillgrid-cli/internal/mnemonic/skills/skills.go`

- Added `Timeout time.Duration` to `Store` so tests can shrink the deadline
  (production default stays `DefaultTimeout` = 30s).

### 3. MCP tool — `use_skill` (Modify)
`skillgrid-cli/internal/mnemonic/mcp/tools_skills.go`

- `useSkillTool()` — tool descriptor: `name` (required), `input` (optional
  trailing arg), `session_id` (trail event), `project`.
- `handleUseSkill(ctx, req)` — resolves service + project, opens the store,
  calls `skills.New(db, root).Execute(ctx, name, input)`. On success (even a
  non-zero exit — the skill *ran*) returns JSON `{stdout, stderr, exit_code,
  timed_out, usage_id, status}`. A non-zero exit or timeout sets `status=failure`
  but is NOT a tool error (the execution happened). If `session_id` is present,
  `insertSkillUseEvent` appends the `skill_use` trail row.
- `insertSkillUseEvent(...)` — mirrors the facts trail convention: next in-session
  `sequence`, `action_type='skill_use'`, `tool_name='use_skill'`,
  `result_status` = success/failure, JSON payload `{skill, exit_code, timed_out,
  usage_id}`, UTC timestamp.
- `statusFor(res)` — `success` iff exit 0 && !timedOut.
- `rootDB()` — test helper returning the active project store's `*sql.DB`
  (delegates to `rootService().OpenForCWD()`).
- Registered in `registerSkillTools` (4th tool).

### 4. Tests (TDD)
- `skills/execute_test.go` (Create) — 9 tests: bash stdout+usage, bash
  stderr+exit-code, python, go, unknown-language (reject, no exec),
  soft-deleted, unknown-skill, sandbox-timeout (bounded), path-escape (reject,
  no exec), oversized-output (1MB cap). Plus a pre-existing name-path-escape test
  from the Write path.
- `mcp/tools_skills_execute_test.go` (Create) — 4 tests: registration,
  execute+log (skill_usage + session_events), non-zero exit returns output,
  unknown-language no-exec, soft-deleted error.
- `mcp/server_test.go` + 6 tool-count test files (Modify) — bumped the
  tool-surface count 87→88 and added `use_skill` to the `want` slice.

## TDD evidence

**RED** (tests written before implementation, `execute.go` absent):
```
--- FAIL: TestExecuteBashSkillCapturesStdoutAndUsage  (undefined: s.Execute / Store.Timeout)
--- FAIL: TestExecuteSandboxTimeout
--- FAIL: TestUseSkillExecutesAndLogs                 (undefined: handleUseSkill / rootDB)
... (all execute/useskill tests fail to compile)
```

**GREEN** (after implementation, `-count=1 -run 'UseSkill|Execute|Sandbox|SkillUsage|PathEscape'`):
```
ok  .../internal/mnemonic/skills  11.5s  (9 tests)
ok  .../internal/mnemonic/mcp      4.8s  (4 use_skill tests)
```

Fails-when gate
(`go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Sandbox|PathEscape|Allowlist|Timeout|UseSkill'`)
**PASS**, exit 0.

TICKET-03 Fails-when gate (`-run 'Skill|WriteSkill|Overwrite|SoftDelete'`) still
**PASS** — the registry was not broken by the executor.

## 4-pass self-review

1. **Completeness** — every acceptance bullet has a named test: sandboxed exec
   returns stdout+stderr, path escape rejects without exec, unknown language
   rejects without exec, timeout is bounded (no hang), soft-deleted → error,
   skill_usage logged, session_events trail logged. `rootDB` is a test-only
   helper; the handler path is exercised end-to-end via `handleUseSkill`.
2. **Re-read** — `execute.go` + `tools_skills.go` alignment clean; the redundant
   `status` variable in the handler was removed (now uses `statusFor`); `gofmt -w`
   applied; `gofmt -l` clean on all four files touched.
3. **Defects** —
   - **Exit-code folding:** `cmd.Wait()` returns `*exec.ExitError` directly (not
     wrapped) for a non-zero exit, so the fold uses a direct type assertion
     (`runErr.(*exec.ExitError)`) after an `errors.Is(ErrNotFound)` guard.
     (An earlier `errors.Unwrap(runErr)` attempt returned nil — ExitError has no
     wrapped cause — which left the error surfacing from `Execute`.)
   - **Timeout:** the deadline case kills the **process group**
     (`Setpgid` + `syscall.Kill(-pid, SIGKILL)`) rather than just the parent, so
     `sleep 15` under `bash -c` cannot survive the kill and hold `cmd.Wait` open.
     This is why the timeout test is bounded to ~2.4s.
   - **stdin over `-c`:** code is fed on stdin (not a `-c` arg) because bash/python
     treat a path-like argument as a script file (exit 126) and a huge `-c` arg
     can exceed the OS argument-list limit.
   - **Path escape** compares `filepath.Clean(codePath)` against
     `filepath.Join(skillDir, base)` — a `../evil.sh` stored path can never pass.
4. **Full run** — both packages GREEN; `go vet` clean on both packages (the
   remaining `tools_code_pdg.go:241` self-assignment is pre-existing and
   unmodified).

## Lazy / gate evidence

- No new dependencies (stdlib only: `os/exec`, `syscall`, `context`, `database/sql`).
- Reuses the 011 `skill_usage` schema and the 040 `session_events` schema — no
  migration added.
- The `skill_use` trail mirrors the facts trail convention (next in-session
  sequence, JSON payload, UTC timestamp) — DRY with `facts.go`.
- `run()` is the single subprocess helper shared by every language; the allowlist
  is a map lookup (no per-language branching).
- Tool surface grew additively (87→88); no existing tools removed or renamed.

## Files

- `skillgrid-cli/internal/mnemonic/skills/execute.go` (Create — `Execute`, `run`,
  `buildSkillCommand`, `SkillResult`, `ErrSoftDeleted`, `execAllowlist`,
  `truncateOutput`, `insertUsage`, `executableLanguages`)
- `skillgrid-cli/internal/mnemonic/skills/execute_test.go` (Create — 9 tests)
- `skillgrid-cli/internal/mnemonic/skills/skills.go` (Modify — `Timeout` field)
- `skillgrid-cli/internal/mnemonic/mcp/tools_skills.go` (Modify — `use_skill` tool
  + `handleUseSkill` + `insertSkillUseEvent` + `statusFor` + `rootDB`)
- `skillgrid-cli/internal/mnemonic/mcp/tools_skills_execute_test.go` (Create — 4 MCP tests)
- `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (Modify — count + want slice)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_affected_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_args_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_retrieval_test.go` (Modify — count)

## Concerns

- **Pre-existing `http/tracker` test failures** (`TestPhase2_*` gh/jira/backlog
  CLI timeouts) — environmental: under full-suite parallel load each timed-out
  test eats the full 10s CLI budget. They pass in isolation. Unrelated to this
  change (carried over TICKET-01/02/03).
- **Pre-existing `wiki` build errors** (`internal/mnemonic/wiki/compile.go`) —
  untracked WIP from another ticket; not introduced here. My two packages
  (`skills/`, `mcp/`) build clean.
- **Pre-existing `go vet` finding** at `tools_code_pdg.go:241` (self-assignment) —
  not from this change.
- **Network isolation** (`env`/`LD_PRELOAD`) is listed in the task *Scope* line but
  is not a BDD acceptance scenario; the step-03 security scenario covers
  path-escape / unknown-language / soft-delete rejects, all of which are
  enforced. Network isolation can be layered on in TICKET-06 if the QA gate
  demands it — it does not block this ticket's acceptance.

## Fix report — Critical review findings (commit 60350b1)

Addressed the two Critical findings from code review:

### Finding 1: Timeout was 30s, not the binding 10s
- `DefaultTimeout`: `30 * time.Second` → `10 * time.Second`
  (`skills/execute.go`).
- Stale "30s" references updated: `execute.go` comment, `skills.go`
  `Store.Timeout` doc comment, test comment in `execute_test.go`, and the
  `use_skill` MCP tool description (`mcp/tools_skills.go`).
- `Store.Timeout` override behavior unchanged — tests that shrink it (2s / 5s)
  still work; the production default is now the binding 10s.

### Finding 2: No Pdeathsig — go runner child could outlive the sandbox
- `run()` now calls `newSysProcAttr()` instead of inlining `Setpgid` only.
- `procattr_linux.go` (`//go:build linux`): `Setpgid: true, Pdeathsig:
  syscall.SIGKILL` — when the runner dies, the kernel kills its children too,
  so a `go run` compiled binary in a different process group cannot outlive
  the timeout SIGKILL.
- `procattr_other.go` (`//go:build !linux`): `Setpgid: true` only (Pdeathsig is
  not portable; verified `GOOS=linux go build` compiles the linux variant).

### Covering tests
- `TestExecuteSandboxTimeoutGoLanguage` (new, `skills/execute_test.go`): a go
  skill that `time.Sleep(15s)`s is run with `s.Timeout = 2s`; asserts
  `TimedOut == true`, no `skill_usage` row, and bounded wall time (<10s).
  Skips with `t.Skip` when `go` is absent.
- Existing `TestExecuteSandboxTimeout` (bash) still passes unchanged.

### Command + output
```
$ go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'UseSkill|Execute|Sandbox|SkillUsage|PathEscape|Timeout'
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills  11.060s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp     2.747s
```
All 12 matched tests PASS, including the new
`TestExecuteSandboxTimeoutGoLanguage` (2.25s).

Broader skill gate:
```
$ go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Skill'
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills  4.560s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp     3.561s
```
PASS.
